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
	"math/rand"
	_ "github.com/mattn/go-sqlite3"
	"database/sql"
	"io/ioutil"
)

type RemoteServer struct {
	config *ServerConfig
	userinfo *UserHolder
	host *check.Rule
	db *sql.DB
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
		log.Errorf("Load rule fail, err:%v, host:%s", err, cli.config.Host)
		cli.host, err = check.NewRule("")
		if err != nil {
			panic("new rule fail")
		}
	} else {
		cli.host = host
	}
	if config.ReportVisit.Enable {
		for {
			db, err := sql.Open("sqlite3", config.ReportVisit.DBFile)
			if err != nil {
				log.Errorf("Open visit record db fail, err:%v, file:%s", err, config.ReportVisit.DBFile)
				break
			}
			data, err := ioutil.ReadFile(config.ReportVisit.SQLFILE)
			if err != nil {
				log.Errorf("Load db init sql file fail, err:%v, file:%s", err, config.ReportVisit.SQLFILE)
				break
			}
			_, err = db.Exec(string(data))
			if err != nil {
				log.Errorf("Exec init sql fail, err:%v, data:%s", err, string(data))
				break
			}
			log.Infof("Init visit record db success, file:%s", config.ReportVisit.DBFile)
			cli.db = db
			break
		}
	}
	return cli
}

func(this *RemoteServer) handleErrConnect(conn *relay.RelayConn, sessionid uint32) {
	defer func() {
		conn.Close()
	}()
	var connaddr string
	var protocol string
	if len(this.config.ErrConnect) != 0 {
		item := this.config.ErrConnect[rand.Intn(len(this.config.ErrConnect))]
		connaddr = item.Address
		protocol = item.Protocol
	}
	logger := log.WithFields(log.Fields{
		"local": conn.RemoteAddr(),
		"remote":connaddr,
		"protocol":protocol,
		"type":"err_conn",
		"id":sessionid,
		"token":conn.GetToken(),
	})
	logger.Infof("Recv invalid connection from remote")
	if len(this.config.ErrConnect) == 0 {
		//如果没有配置异常连接地址, 那么默认卡住1~30秒后自动关闭
		logger.Errorf("No err redirect domain found, time and close!")
		time.Sleep(time.Duration(1 + rand.Int63n(29)) *time.Second)
		return
	}
	remote, err, dur := transport_delegate.Dial(protocol, connaddr, this.config.Timeout)
	if err != nil {
		logger.Errorf("Dial confuse domain fail, err:%v, domain:%s, protocol:%s, cost:%dms", err, connaddr, protocol, dur)
		return
	}
	defer func() {
		remote.Close()
	}()
	logger.Infof("Dial confuse domain success, dst conn:%s, protocol:%s, domain:%s, cost:%dms", remote.RemoteAddr(), protocol, connaddr, dur)
	ctx, cancel := context.WithCancel(context.Background())
	rbuf := make([]byte, relay.MAX_BYTE_PER_PACKET)
	wbuf := make([]byte, relay.MAX_BYTE_PER_PACKET)
	sr, sw, dr, dw, sre, swe, dre, dwe := net_utils.Pipe(conn, remote, rbuf, wbuf, ctx, cancel, this.config.Timeout)
	logger.Infof("Confuse ata transfer finish, br:%d, bw:%d, pr:%d, pw:%d, bre:%+v, bwe:%+v, pre:%+v, pwe:%+v",
		sr, sw, dr, dw, sre, swe, dre, dwe)
}

//上报当前
func(this *RemoteServer) report(user string, from string, visitHost string,
	read int64, write int64,
		start time.Time, end time.Time, connectCost int64, logger *log.Entry) {
	if this.db == nil {
		return
	}
	prepare, err := this.db.Prepare("insert into visit_record(username, host, user_from, start_time, end_time, read_cnt, write_cnt, connect_cost) values(?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		logger.Errorf("Create visit sql fail, err:%v", err)
		return
	}
	_, err = prepare.Exec(user, visitHost, from, start, end, read, write, connectCost)
	if err != nil {
		logger.Errorf("insert into visit record db fail, err:%v", err)
	}
	prepare.Close()
}

func(this *RemoteServer) handleProxy(conn *relay.RelayConn, sessionid uint32) {
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
	logger.Infof("Recv new connection from remote success")
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
	var cost1 int64
	var cost2 int64
	remote, err, cost1 = transport_delegate.Dial("tcp", address, this.config.Timeout)
	if err != nil {
		remote, err, cost2 = transport_delegate.Dial("tcp", address, this.config.Timeout / 2)
	}
	if err != nil {
		conn.Close()
		logger.Errorf("Connect to remote svr failed, err:%s, remote addr:%s, conn:%s, cost:%dms", err, address, conn.RemoteAddr(), cost1 + cost2)
		return
	}
	logger.Infof("Connect to remote svr success, target:%s, cost:%dms", remote.RemoteAddr(), cost1 + cost2)
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