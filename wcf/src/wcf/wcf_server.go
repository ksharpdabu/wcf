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
	"check"
)

type RemoteServer struct {
	config *ServerConfig
	acceptor *relay.RelayAcceptor
	userinfo *UserHolder
	host *check.Rule
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
	host, err := check.NewRule(cli.config.Host)
	if err != nil {
		log.Errorf("load rule fail, may has err, err:%v, host:%s", err, cli.config.Host)
		return nil
	}
	cli.host = host
	return cli
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
		vinfo := this.host.GetHostRule(conn.GetTargetName())
		if vinfo.HostRule == check.RULE_BLOCK {
			logger.Errorf("User:%s visit site:%s not allow, skip", conn.GetUser(), conn.GetTargetName())
			conn.Close()
			return
		}
		if len(vinfo.NewHostValue) != 0 { //vinfo.NewHostValue must be domain or ip, could not be cidr!!!!!!!
			address = fmt.Sprintf("%s:%d", vinfo.NewHostValue, conn.GetTargetPort())
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
	logger.Infof("Data transfer finish, br:%d, bw:%d, pr:%d, pw:%d, bre:%+v, bwe:%+v, pre:%+v, pwe:%+v",
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