package relay

import (
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"net"
	"net_utils"
	"time"
	"wcf/relay/msg"
)

type RelayClientConn struct {
	net.Conn
	token uint32
}

func (this *RelayClientConn) GetToken() uint32 {
	return this.token
}

type RelayAddress struct {
	Addr     string
	AddrType int32
	Name     string
	Port     uint16
}

type RelayConfig struct {
	User      string
	Pwd       string
	Address   RelayAddress
	RelayType int32 //refer consts.go OP_TYPE_xxx
}

func Dial(addr string, config *RelayConfig) (*RelayClientConn, error) {
	return DialWithTimeout(addr, time.Hour, config) //应该差不多效果了吧, 一个钟都连不上, 那只能玩个蛋蛋了。。
}

func doAuth(conn net.Conn, config *RelayConfig) (net.Conn, error, uint32) {
	data := BuildDataPacket(BuildAuthReqMsg(config))
	err := net_utils.SendSpecLen(conn, data)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("send auth pkg to svr fail, err:%v, conn:%s, datalen:%d", err, conn.RemoteAddr(), len(data))), 0
	}
	buf := make([]byte, 1024)
	index := 0
	total := len(buf)
	var ckResult int
	var ckErr error
	for index < total {
		cnt, err := conn.Read(buf[index:])
		if err != nil {
			return nil, errors.New(fmt.Sprintf("recv auth rsp data from proxy fail, err:%v, conn:%s", err, conn.RemoteAddr())), 0
		}
		index += cnt
		ckResult, ckErr = CheckRelayPacketReadyWithLength(buf[:index], MAX_BYTE_AUTH_PACKET)
		if ckResult != 0 {
			break
		}
	}
	if ckErr != nil || ckResult <= 0 {
		return nil, errors.New(fmt.Sprintf("check packet result fail, err:%v, result:%d, conn:%s", ckErr, ckResult, conn.RemoteAddr())), 0
	}
	raw, err := GetPacketData(buf[:ckResult])
	if err != nil {
		return nil, errors.New(fmt.Sprintf("get packet data from wrap fail, err:%v, conn:%s", err, conn.RemoteAddr())), 0
	}
	msg := msg.AuthMsgRsp{}
	err = proto.Unmarshal(raw, &msg)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("decode auth rsp fail, err:%v, conn:%s", err, conn.RemoteAddr())), 0
	}
	if msg.GetResult() != 0 {
		return nil, errors.New(fmt.Sprintf("auth fail, user:%s, pwd:%s, result:%d", config.User, config.Pwd, msg.GetResult())), 0
	}
	var spare []byte
	if ckResult == index {
		spare = nil
	} else {
		spare = buf[ckResult:index]
	}
	return WrapRelayFrameConn(conn, spare, nil), nil, msg.GetToken()
}

func WrapConnection(conn net.Conn, config *RelayConfig) (*RelayClientConn, error) {
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	newConn, err, token := doAuth(conn, config)
	conn.SetDeadline(time.Time{})
	if err != nil {
		return nil, err
	}

	return &RelayClientConn{Conn: newConn, token: token}, nil
}

func DialWithTimeout(addr string, sec time.Duration, config *RelayConfig) (*RelayClientConn, error) {
	conn, err := net.DialTimeout("tcp", addr, sec)
	if err != nil {
		return nil, err
	}
	newConn, err := WrapConnection(conn, config)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return newConn, nil
}
