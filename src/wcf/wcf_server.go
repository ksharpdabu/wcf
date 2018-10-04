package wcf

import (
	"wcf/relay"
	log "github.com/sirupsen/logrus"
	"net"
	"context"
	"mix_layer"
	"fmt"
	"proxy"
	"net_utils"
)

type RemoteServer struct {
	config *ServerConfig
	acceptor *relay.RelayAcceptor
	userinfo *UserHolder
}

func NewServer(config *ServerConfig) *RemoteServer {
	cli := &RemoteServer{}
	cli.config = config
	ui, err := NewUserHolder(cli.config.Userinfo)
	if err != nil {
		log.Errorf("Load user info fail, err:%v, file:%s", err, cli.config.Userinfo)
		cli.userinfo, _ = NewUserHolder("")
	} else {
		cli.userinfo = ui
	}
	return cli
}

func(this *RemoteServer) isInnerIP(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}
	if v := ip.To4(); v != nil { //CHECK V4
		if (v[0] == 10) {
			return true
		} else if (v[0] == 172 && v[1] > 15 && v[1] < 32) {
			return true
		} else if (v[0] == 192 && v[1] == 168) {
			return true
		}

	} else { //CHECK V6

	}
	return false
}

func(this *RemoteServer) secureCheck(name string, port uint32) bool {
	ip := net.ParseIP(name)
	var lst []net.IP
	if ip == nil {
		ns, err := net.LookupHost(name)
		if err != nil {
			log.Errorf("Resolve ip address fail, err:%v, name:%s, pass", err, name)
			return true
		}
		for _, item := range ns {
			v := net.ParseIP(item)
			if v != nil {
				lst = append(lst, v)
			} else {
				log.Errorf("Resolve ip but can not parse, ip:%s, name:%s, err:$v", item, name, err)
				return false
			}
		}
	} else {
		lst = append(lst, ip)
	}
	for _, ip := range lst {
		if this.isInnerIP(ip) {
			return false
		}
	}
	return true
}

func(this *RemoteServer) handleProxy(conn *relay.RelayConn, sessionid uint32) {
	logger := log.WithFields(log.Fields{
		"local": conn.RemoteAddr(),
		"remote": conn.GetTargetAddress(),
		"user": conn.GetUser(),
		"id": sessionid,
		"token": conn.GetToken(),
	})
	logger.Infof("Recv new connection from remote")
	var remote net.Conn
	var err error
	var address string
	if conn.GetTargetOPType() == proxy.OP_TYPE_FORWARD {
		ui := this.userinfo.GetUserInfo(conn.GetUser())
		if !ui.Forward.EnableForward || len(ui.Forward.ForwardAddr) == 0{
			logger.Errorf("User no allaw use forward option or forward addr empty, skip, user:%s, addr:%s, conn:%s", ui.User, ui.Forward.ForwardAddr, conn.RemoteAddr())
			return
		}
		address = ui.Forward.ForwardAddr

	} else { //default proxy
		address = fmt.Sprintf("%s:%d", conn.GetTargetName(), conn.GetTargetPort())
		if this.config.EnableSecureCheck {
			if !this.secureCheck(conn.GetTargetName(), conn.GetTargetPort()) {
				logger.Errorf("User:%s use addr:%s:%d could not pass secure check.", conn.GetUser(), conn.GetTargetName(), conn.GetTargetPort())
				conn.Close()
				return
			}
		}
	}
	remote, err = net.DialTimeout("tcp", address, this.config.Timeout)
	if err != nil {
		conn.Close()
		logger.Errorf("Connect to remote svr failed, err:%s, remote addr:%s, conn:%s", err, address, conn.RemoteAddr())
		return
	}
	logger.Infof("Connect to remote svr success, target:%s", remote.RemoteAddr())
	defer func() {
		conn.Close()
		remote.Close()
	}()
	rbuf := make([]byte, relay.ONE_PER_BUFFER_SIZE)
	wbuf := make([]byte, relay.MAX_BYTE_PER_PACKET)
	ctx, cancel := context.WithCancel(context.Background())
	sr, sw, dr, dw, sre, swe, dre, dwe := net_utils.Pipe(conn, remote, rbuf, wbuf, ctx, cancel, this.config.Timeout)
	logger.Infof("Data transfer finish, br:%d, bw:%d, pr:%d, pw:%d, bre:%v, bwe:%v, pre:%v, pwe:%v",
		sr, sw, dr, dw, sre, swe, dre, dwe)
}

func(this *RemoteServer) Start() error {
	acceptor, err := relay.Bind(this.config.Localaddr)
	if err != nil {
		log.Errorf("Bind local svr fail, err:%v, local addr:%s", err, this.config.Localaddr)
		return err
	}
	this.acceptor = acceptor
	this.acceptor.AddMixWrap(func(conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.Wrap(this.config.Encrypt, this.config.Key, conn)
	})
	this.acceptor.OnAuth = func(user, pwd string) bool {
		return this.userinfo.Check(user, pwd)
	}
	err = this.acceptor.Start()
	if err != nil {
		log.Errorf("Start relay acceptor fail, err:%v", err)
		return err
	}
	var sessionid uint32 = 0
	for {
		cli, err := this.acceptor.Accept()
		if err != nil {
			log.Errorf("Recv client from remote fail, err:%v, continue", err)
			continue
		}
		sessionid++
		go this.handleProxy(cli, sessionid)
	}
}