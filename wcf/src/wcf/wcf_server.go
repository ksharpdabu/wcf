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
	"sync/atomic"
	"sync"
	"time"
	"mix_delegate"
	"transport_delegate"
	"wcf/visit_delegate"
	"wcf/visit"
	"wcf/redirect_delegate"
)

type RemoteServer struct {
	config *ServerConfig
	userinfo *UserHolder
	host *check.Rule
	visitor visit.Visitor
	reportQueue chan *visit.VisitInfo
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
	//路由配置
	host, err := check.NewRule(cli.config.Host)
	if err != nil {
		log.Errorf("Load rule fail, err:%v, host:%s", err, cli.config.Host)
		cli.host, err = check.NewRule("")
		if err != nil {
			panic("new rule fail")
		}
	} else {
		cli.host = host
	}
	//上报配置
	if config.ReportVisit.Enable {
		var err error
		var visitor visit.Visitor
		for {
			if visitor, err = visit_delegate.Get(config.ReportVisit.Visitor); err != nil {
				log.Errorf("Load visitor fail err:%v, name:%s", err, config.ReportVisit.Visitor)
				break
			}
			if err = visitor.Init(config.ReportVisit.VisitorConfig); err != nil {
				log.Errorf("Visitor init fail, err:%v, config:%s", err, config.ReportVisit.VisitorConfig)
				break
			}
			cli.visitor = visitor
			cli.reportQueue = make(chan *visit.VisitInfo, 2000)
			go cli.asyncReport()
			log.Infof("Load report visitor info success, name:%s", config.ReportVisit.Visitor)
			break
		}

	}
	return cli
}

func(this *RemoteServer) asyncReport() {
	if this.visitor == nil {
		return
	}
	for {
		info := <- this.reportQueue
		this.visitor.OnView(info)
	}
}

func(this *RemoteServer) handleErrConnect(conn *relay.RelayConn, sessionid uint32) {
	defer func() {
		conn.Close()
	}()
	log.Infof("Conn:%s reach err branch, errmsg:%s, sessionid:%d", conn.RemoteAddr(), conn.GetHandshakeErrmsg(), sessionid)
	if !this.config.Redirect.Enable {
		log.Errorf("Redirect config not set, close it, sessionid:%d, conn:%s", sessionid, conn.RemoteAddr())
		return
	}
	r, w, err := redirect_delegate.Process(this.config.Redirect.Redirector, conn)
	log.Infof("Process err redirect success, redirector:%s, r:%d, w:%d, err:%v, conn:%s, session:%d", this.config.Redirect.Redirector, r, w, err, conn.RemoteAddr(), sessionid)
}

//上报当前的用户访问信息
func(this *RemoteServer) report(user string, from string, visitHost string,
	read int64, write int64,
		start time.Time, end time.Time, connectCost int64, logger *log.Entry) {
	if this.visitor == nil {
		return
	}
	visi := &visit.VisitInfo{
		Name:user,
		From:from,
		Host:visitHost,
		Read:read,
		Write:write,
		Start:start,
		End:end,
		ConnectCost:connectCost,
	}
	select {
		case this.reportQueue <- visi:
			break
		default:
			logger.Errorf("Queue full, skip report user:%s visit info, host:%s", user, visitHost)
			break
	}
}

func(this *RemoteServer) handleProxy(conn *relay.RelayConn, sessionid uint32) {
	defer conn.Close()
	visitStart := time.Now()
	logger := log.WithFields(log.Fields{
		"local": conn.RemoteAddr(),
		"remote": conn.GetTargetAddress(),
		"user": conn.GetUser(),
		"id": sessionid,
		"token": conn.GetToken(),
	})
	if conn.GetHandshakeResult() != true {
		this.handleErrConnect(conn, sessionid)
		return
	}
	ui := this.userinfo.GetUserInfo(conn.GetUser())
	uiCtx := GetOrCreateContext(ui)
	var curconn int
	var ok bool
	if curconn, ok = uiCtx.Limit.TryAcqure(); !ok {
		logger.Infof("User:%s reach max connection:%d, close it", ui.User, ui.MaxConnection)
		return
	}
	defer uiCtx.Limit.Release()
	logger.Infof("Recv new connection from remote success, current connections:%d", curconn)
	var remote net.Conn
	var err error
	var address string
	if conn.GetTargetOPType() == proxy.OP_TYPE_FORWARD {
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
	var cost1 int64
	var cost2 int64
	remote, err, cost1 = transport_delegate.Dial("tcp", address, this.config.Timeout)
	if err != nil {
		remote, err, cost2 = transport_delegate.Dial("tcp", address, this.config.Timeout / 2)
	}
	if err != nil {
		logger.Errorf("Connect to remote svr failed, err:%s, remote addr:%s, conn:%s, cost:%dms", err, address, conn.RemoteAddr(), cost1 + cost2)
		return
	}
	logger.Infof("Connect to remote svr success, target:%s, cost:%dms", remote.RemoteAddr(), cost1 + cost2)
	defer func() {
		remote.Close()
	}()
	rbuf := make([]byte, relay.ONE_PER_BUFFER_SIZE)
	wbuf := make([]byte, relay.MAX_BYTE_PER_PACKET)
	ctx, cancel := context.WithCancel(context.Background())
	var transconn net.Conn = conn
	if ui.Speed.Enable {
		transconn = NewSpeedConn(conn, ui.Speed.PerConn.Read, ui.Speed.PerConn.Write)
	}
	sr, sw, dr, dw, sre, swe, dre, dwe := net_utils.Pipe(transconn, remote, rbuf, wbuf, ctx, cancel, this.config.Timeout)
	logger.Infof("Data transfer finish, br:%d, bw:%d, pr:%d, pw:%d, bre:%+v, bwe:%+v, pre:%+v, pwe:%+v",
		sr, sw, dr, dw, sre, swe, dre, dwe)
	this.report(conn.GetUser(), conn.RemoteAddr().String(), address, int64(sr), int64(sw), visitStart, time.Now(), cost1 + cost2, logger)
}

func(this *RemoteServer) Start() error {
	var wg sync.WaitGroup
	wg.Add(len(this.config.Localaddr))
	for _, v := range this.config.Localaddr {
		binder, err := transport_delegate.Bind(v.Protocol, v.Address)
		if err != nil {
			log.Errorf("Bind local svr fail, err:%v, local addr:%s, protocol:%s", err, v.Address, v.Protocol)
			return err
		}
		acceptor, err := relay.WrapListener(binder)
		if err != nil {
			log.Errorf("Relay wrap listener fail, err:%v, protocol:%s, localaddr:%s", err, v.Protocol, v.Address)
			return err
		}
		acceptor.AddMixWrap(func(conn net.Conn) (mix_layer.MixConn, error) {
			return mix_delegate.Wrap(this.config.Encrypt, this.config.Key, conn)
		})
		acceptor.OnAuth = func(user, pwd string) bool {
			return this.userinfo.Check(user, pwd)
		}
		err = acceptor.Start()
		if err != nil {
			log.Errorf("Start relay acceptor fail, protocol:%s, addr:%s, err:%v", v.Protocol, v.Address, err)
			return err
		}
		log.Infof("Start relay acceptor success, protocol:%s, addr:%s", v.Protocol, v.Address)
		var sessionid uint32 = 0
		go func() {
			defer func() {
				wg.Done()
			}()
			for {
				cli, err := acceptor.Accept()
				if err != nil {
					log.Errorf("Recv client from remote fail, err:%v, continue", err)
					continue
				}
				sess := atomic.AddUint32(&sessionid, 1)
				go this.handleProxy(cli, sess)
			}
		}()
	}
	wg.Wait()
	return nil
}