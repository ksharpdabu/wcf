package wcf

import (
	log "github.com/sirupsen/logrus"
	"net"
	"wcf/relay"
	"sync"
	"context"
	"sync/atomic"
	"check"
	"proxy"
	"net_utils"
	"lb"
	"time"
	"proxy_delegate"
	"mix_delegate"
	"transport_delegate"
	"fmt"
)

type LocalClient struct {
	config *LocalConfig
	rule *check.Rule
	lb *lb.LoadBalance
	smartRule *check.Rule
	hostQueue chan string
}

func(this *LocalClient) addToCheckRule(addr string) {
	select {
	case this.hostQueue <- addr:
		break
	default:
		log.Errorf("Queue full, skip check addr:%s", addr)
	}
}

func(this *LocalClient) autoCheckRule() {
	limiter := NewLimiter(200)
	for {
		select {
		case v := <-this.hostQueue:
			used, result := limiter.TryAcqure()
			log.Infof("Check host:%s, acqure result:%t, used:%d", v, result, used)
			if result {
				go func() {
					conn, err := net.DialTimeout("tcp", v, 1 * time.Second)
					if err == nil {
						conn.Close()
					}
					limiter.Release()
					if host, _, serr := net.SplitHostPort(v); serr == nil {
						var stat = check.RULE_DIRECT
						if err != nil {
							stat = check.RULE_PROXY
						}
						log.Infof("Add auto proxy rule, addr:%s, rule:%s, err:%v", host, check.HostRule2String(check.HostRule(stat)), err)
						this.smartRule.AddDomainRule(host, check.HostRule(stat))
					} else {
						log.Infof("Check host finish, but split host fail, host addr:%s", v)
					}
				}()
			}
			break
		}
	}
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
	if cli.config.SmartProxy {
		smart, _ := check.NewRuleStatic()
		cli.smartRule = smart
		cli.hostQueue = make(chan string, 200)
		go cli.autoCheckRule()
	}
	cli.lb = lb.New(cli.config.Lbinfo.MaxErrCnt, cli.config.Lbinfo.MaxFailTime)
	for _, v := range cli.config.Proxyaddr {
		log.Infof("Add addr:%s weight:%d to load balance", v.Addr, v.Weight)
		cli.lb.Add(v.Addr, v.Weight, v.Protocol)
	}
	return cli
}

type RemoteDialFunc func(protocol string, addr string, timeout time.Duration) (net.Conn, error)

func(this *LocalClient) handleProxy(conn proxy.ProxyConn, sessionid uint32, network string) {
	defer func() {
		conn.Close()
	}()
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
	//取消本地dns查询, 加快连接速度。
	rule, ck := this.rule.CheckAndGetRuleOptional(conn.GetTargetName(), false)
	if !ck && this.config.SmartProxy { //原有规则没有。。那么再走一遍智能代理的规则
		rule, ck = this.smartRule.CheckAndGetRuleOptional(conn.GetTargetName(), false)
		if !ck { //只能代理还没有, 那就走规则校验逻辑
			this.addToCheckRule(fmt.Sprintf("%s:%d", net_utils.ResolveRealAddr(conn.GetTargetName()), conn.GetTargetPort()))
		} else {
			logger.Infof("Host:%s hit smart config rule, rule:%s", conn.GetTargetName(), check.HostRule2String(rule))
		}
	} else {
		logger.Infof("Host:%s hit config rule, rule:%s", conn.GetTargetName(), check.HostRule2String(rule))
	}
	if rule == check.RULE_PROXY {
		newConnAddr, extra, err := this.lb.Get()
		protocol = extra.(string)
		if err != nil {
			logger.Errorf("Get balance ip fail, err:%v, conn:%s", err, conn.RemoteAddr())
			return
		}
		connAddr = newConnAddr
	} else {
		connAddr = fmt.Sprintf("%s:%d", net_utils.ResolveRealAddr(conn.GetTargetName()), conn.GetTargetPort())
		protocol = "tcp"
	}
	var dur int64
	remote, err, dur = transport_delegate.Dial(protocol, connAddr, this.config.Timeout)
	if this.config.Lbinfo.Enable && rule == check.RULE_PROXY {
		logger.Infof("Update addr:%s as t:%t", connAddr, err == nil)
		this.lb.Update(connAddr, err == nil)
	}
	if err != nil {
		logger.Errorf("Dial connection to target/proxy svr fail, err:%s, svr addr:%s, cost:%dms", err, connAddr, dur)
		return
	}
	defer func() {
		remote.Close()
	}()
	var token uint32 = 0
	if rule == check.RULE_PROXY {  //only proxy mode should wrap this layer
		newConn, err := mix_delegate.Wrap(this.config.Encrypt, this.config.Key, remote)
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
	logger.Infof("Connect to proxy/target svr success, target:%s, name:%s, token:%d, protocol:%s, cost:%dms",
		remote.RemoteAddr(), connAddr, token, protocol, dur)
	ctx, cancel := context.WithCancel(context.Background())
	rbuf := make([]byte, relay.MAX_BYTE_PER_PACKET)
	wbuf := make([]byte, relay.ONE_PER_BUFFER_SIZE)
	sr, sw, dr, dw, sre, swe, dre, dwe := net_utils.Pipe(conn, remote, rbuf, wbuf, ctx, cancel, this.config.Timeout)
	logger.Infof("Data transfer finish, br:%d, bw:%d, pr:%d, pw:%d, bre:%+v, bwe:%+v, pre:%+v, pwe:%+v",
		sr, sw, dr, dw, sre, swe, dre, dwe)
}

func(this *LocalClient) Start() error {
	var wg sync.WaitGroup
	wg.Add(len(this.config.Localaddr))
	var sessionid uint32 = 0
	for _, config := range this.config.Localaddr {
		acceptor, err := proxy_delegate.Bind(config.Name, config.Address)
		if err != nil {
			wg.Done()
			log.Errorf("Bind addr:%s use protocol:%s fail, err:%v", config.Address, config.Name, err)
			continue
		}
		acceptor.AddHostHook(func(addr string, port uint16, addrType int) (bool, string, uint16, int) {
			rule := this.rule.GetHostRuleOptional(addr, false)
			if rule == check.RULE_BLOCK {
				return false, addr, port, addrType
			}
			return true, addr, port, addrType
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

