package wcf

import (
	log "github.com/sirupsen/logrus"
	"net"
	"wcf/relay"
	"sync"
	"context"
	"sync/atomic"
	"check"
	"mix_layer"
	"proxy"
	"net_utils"
	"lb"
	"time"
	"github.com/xtaci/kcp-go"
)

type LocalClient struct {
	config *LocalConfig
	rule *check.Rule
	lb *lb.LoadBalance
}

func NewClient(config *LocalConfig) *LocalClient {
	cli := &LocalClient{}
	cli.config = config
	rule, err := check.NewRule(cli.config.Host)
	if err != nil {
		log.Errorf("Load host rule fail, err:%v, file:%s", err, cli.config.Host)
		cli.rule, err = check.NewRule("")
		if err != nil {
			panic("new rule fail")
		}
	} else {
		cli.rule = rule
	}
	cli.lb = lb.New(cli.config.Lbinfo.MaxErrCnt, cli.config.Lbinfo.MaxFailTime)
	for _, v := range cli.config.Proxyaddr {
		log.Infof("Add addr:%s weight:%d to load balance", v.Addr, v.Weight)
		cli.lb.Add(v.Addr, v.Weight, v.Protocol)
	}
	return cli
}

func isDone(ctx context.Context) bool {
	select {
	case <- ctx.Done():
		return true
	default:
		return false
	}
}

type RemoteDialFunc func(protocol string, addr string, timeout time.Duration) (net.Conn, error)

func(this *LocalClient) handleProxy(conn proxy.ProxyConn, sessionid uint32, network string) {
	logger := log.WithFields(log.Fields{
		"local": conn.RemoteAddr(),
		"remote": conn.GetTargetAddress(),
		"id": sessionid,
		"net": network,
	})
	logger.Infof("Recv new connection from browser")
	cfg := &relay.RelayConfig{}
	cfg.User = this.config.User
	cfg.Pwd = this.config.Pwd
	cfg.Address.AddrType = int32(conn.GetTargetType())
	cfg.Address.Addr = conn.GetTargetAddress()
	cfg.Address.Name = conn.GetTargetName()
	cfg.Address.Port = conn.GetTargetPort()
	cfg.RelayType = int32(conn.GetTargetOPType())

	var remote net.Conn
	var err error
	var connAddr string
	var protocol string
	var dialfunc RemoteDialFunc = net.DialTimeout
	needUpdate := false
	vinfo := this.rule.GetHostRule(conn.GetTargetName())
	if vinfo.HostRule == check.RULE_PROXY {
		newConnAddr, extra, err := this.lb.Get()
		if err != nil {
			logger.Errorf("Get balance ip fail, err:%v, conn:%s", err, conn.RemoteAddr())
			conn.Close()
			return
		}
		connAddr = newConnAddr
		if extra != nil {
			protocol = extra.(string)
		} else {
			protocol = "tcp"
		}
		if protocol == "kcp" {
			dialfunc = func(protocol string, addr string, timeout time.Duration) (net.Conn, error) {
				//no timeout support
				return kcp.DialWithOptions(addr, nil, 10, 3)
			}
		}
		if this.config.Lbinfo.Enable {
			needUpdate = true
		}

	} else {
		connAddr = conn.GetTargetAddress()
	}
	remote, err = dialfunc(protocol, connAddr, this.config.Timeout)
	if needUpdate {
		if this.config.Lbinfo.Enable {
			logger.Infof("Update addr:%s as t:%t", connAddr, err == nil)
			this.lb.Update(connAddr, err == nil)
		}
	}
	if err != nil {
		logger.Errorf("Dial connection to proxy/target svr fail, err:%s, svr addr:%s", err, connAddr)
		conn.Close()
		return
	}
	defer func() {
		conn.Close()
		remote.Close()
	}()
	var token uint32 = 0
	if vinfo.HostRule == check.RULE_PROXY {  //only proxy mode should wrap this layer
		newConn, err := mix_layer.Wrap(this.config.Encrypt, this.config.Key, remote)
		if err != nil {
			logger.Errorf("Wrap connection with mix method:%s fail, err:%v, conn:%s", this.config.Encrypt, err, remote.RemoteAddr())
			return
		} else {
			newWrapConn, err := relay.WrapConnection(newConn, cfg)
			if err != nil {
				logger.Errorf("Wrap connection with relay method fail, err:%v, conn:%s", err, newConn.RemoteAddr())
				return
			}
			remote = newWrapConn
			token = newWrapConn.GetToken()
		}
	}
	logger.Infof("Connect to proxy svr success, target:%s, token:%d, protocol:%s", remote.RemoteAddr(), token, protocol)
	ctx, cancel := context.WithCancel(context.Background())
	rbuf := make([]byte, relay.MAX_BYTE_PER_PACKET)
	wbuf := make([]byte, relay.ONE_PER_BUFFER_SIZE)
	sr, sw, dr, dw, sre, swe, dre, dwe := net_utils.Pipe(conn, remote, rbuf, wbuf, ctx, cancel, this.config.Timeout)
	logger.Infof("Data transfer finish, br:%d, bw:%d, pr:%d, pw:%d, bre:%+v, bwe:%+v, pre:%+v, pwe:%+v",
		sr, sw, dr, dw, sre, swe, dre, dwe)
	if sr != dw || dr != sw {
		logger.Errorf("Data transfer error, br:%d, bw:%d, pr:%d, pw:%d", sr, sw, dr, dw)
	}
}

func(this *LocalClient) Start() error {
	var wg sync.WaitGroup
	wg.Add(len(this.config.Localaddr))
	var sessionid uint32 = 0
	for _, config := range this.config.Localaddr {
		acceptor, err := proxy.Bind(config.Name, config.Address)
		if err != nil {
			wg.Done()
			log.Errorf("Bind addr:%s use protocol:%s fail, err:%v", config.Address, config.Name, err)
			continue
		}
		acceptor.AddHostHook(func(addr string, port uint16, addrType int) (bool, string, uint16, int) {
			vinfo := this.rule.GetHostRule(addr)
			rewrite := addr
			if len(vinfo.NewHostValue) != 0 {
				rewrite = vinfo.NewHostValue
			}
			if vinfo.HostRule == check.RULE_BLOCK {
				return false, rewrite, port, addrType
			}
			return true, rewrite, port, addrType
		})
		go func(netw string, acc proxy.ProxyListener, addr string) {
			defer func() {
				wg.Done()
			}()
			err = acc.Start()
			if err != nil {
				log.Errorf("Start %s acceptor fail, err:%v", netw, err)
				return
			} else {
				log.Infof("Start %s acceptor success on address:%s", netw, addr)
			}
			for {
				cli, err := acc.Accept()
				if err != nil {
					log.Errorf("Recv %s client from browser fail, err:%v, continue", netw, err)
					continue
				}
				sess := atomic.AddUint32(&sessionid, 1)
				go this.handleProxy(cli, sess, netw)
			}
		}(config.Name, acceptor, config.Address)
	}
	wg.Wait()
	return nil
}

