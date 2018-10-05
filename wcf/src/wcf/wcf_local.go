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
)

type LocalClient struct {
	config *LocalConfig
	noProxy *check.Host
	black *check.Host
	lb *lb.LoadBalance
}

func NewClient(config *LocalConfig) *LocalClient {
	cli := &LocalClient{}
	cli.config = config
	if len(cli.config.Host.BlackHost) != 0 {
		blk, err := check.NewRule(cli.config.Host.BlackHost)
		if err != nil {
			log.Errorf("Load black host fail, err:%v, file:%s", err, cli.config.Host.BlackHost)
			cli.black, _ = check.NewRule("")
		} else {
			cli.black = blk
		}
	}
	if len(cli.config.Host.NoProxyHost) != 0 {
		nop, err := check.NewRule(cli.config.Host.NoProxyHost)
		if err != nil {
			log.Errorf("Load no proxy host fail, err:%v, file:%s", err, cli.config.Host.NoProxyHost)
			cli.noProxy, _ = check.NewRule("")
		} else {
			cli.noProxy = nop
		}
	}
	cli.lb = lb.New(cli.config.Lbinfo.MaxErrCnt, cli.config.Lbinfo.MaxFailTime)
	for _, v := range cli.config.Proxyaddr {
		log.Infof("Add addr:%s weight:%d to load balance", v.Addr, v.Weight)
		cli.lb.Add(v.Addr, v.Weight)
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
	needUpdate := false
	if !this.noProxy.IsSubExists(conn.GetTargetName()) {
		newConnAddr, err := this.lb.Get()
		if err != nil {
			logger.Errorf("Get balance ip fail, err:%v, conn:%s", err, conn.RemoteAddr())
			conn.Close()
			return
		}
		connAddr = newConnAddr
		needUpdate = true

	} else {
		connAddr = conn.GetTargetAddress()
	}
	remote, err = net.DialTimeout("tcp", connAddr, this.config.Timeout)
	if needUpdate {
		logger.Infof("Update addr:%s as t:%t", connAddr, err == nil)
		this.lb.Update(connAddr, err == nil)
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
	logger.Infof("Connect to proxy svr success, target:%s, token:%d", remote.RemoteAddr(), token)
	ctx, cancel := context.WithCancel(context.Background())
	rbuf := make([]byte, relay.MAX_BYTE_PER_PACKET)
	wbuf := make([]byte, relay.ONE_PER_BUFFER_SIZE)
	sr, sw, dr, dw, sre, swe, dre, dwe := net_utils.Pipe(conn, remote, rbuf, wbuf, ctx, cancel, this.config.Timeout)
	logger.Infof("Data transfer finish, br:%d, bw:%d, pr:%d, pw:%d, bre:%v, bwe:%v, pre:%v, pwe:%v",
		sr, sw, dr, dw, sre, swe, dre, dwe)
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
			return !this.black.IsSubExists(addr), addr, port, addrType
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

