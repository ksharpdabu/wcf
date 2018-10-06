package relay

import (
	"time"
	"net"
	"wcf/relay/msg"
	"github.com/golang/protobuf/proto"
	"net_utils"
	"errors"
	"fmt"
	"bytes"
)

type RelayClientConn struct {
	net.Conn
	token uint32
}

func(this *RelayClientConn) GetToken() uint32 {
	return this.token
}

type RelayAddress struct {
	Addr string
	AddrType int32
	Name string
	Port uint16
}

type RelayConfig struct {
	User string
	Pwd string
	Address RelayAddress
	RelayType int32   //refer consts.go OP_TYPE_xxx
}

type RelayFrameConn struct {
	net.Conn
	rbuf bytes.Buffer
	wbuf bytes.Buffer
	rdbuf bytes.Buffer
	rtmp []byte
}

func WrapRelayFrameConn(conn net.Conn, rbuf []byte, wbuf []byte) *RelayFrameConn {
	cn := &RelayFrameConn{}
	cn.Conn = conn
	if rbuf != nil {
		cn.rbuf.Write(rbuf)
	}
	if wbuf != nil {
		cn.wbuf.Write(wbuf)
	}
	cn.rtmp = make([]byte, MAX_BYTE_PER_PACKET)
	return cn
}

func(this *RelayFrameConn) Read(b []byte) (int, error) {
	if this.rdbuf.Len() != 0 {
		cnt := copy(b, this.rdbuf.Bytes())
		this.rdbuf.Next(cnt)
		return cnt, nil
	}
	var total = 0
	var cerr error
	for {
		total, cerr = CheckRelayPacketReay(this.rbuf.Bytes())
		if total < 0 || cerr != nil {
			return 0, errors.New(fmt.Sprintf("packet check fail, v:%d, cerr:%v", total, cerr))
		}
		if total > 0 {
			break
		}
		cnt, err := this.Conn.Read(this.rtmp)
		if err != nil {
			return cnt, err
		}
		this.rbuf.Write(this.rtmp[:cnt])
	}
	raw, err := GetPacketData(this.rbuf.Bytes()[:total])
	this.rbuf.Next(total)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("get packet data fail, err:%v", err))
	}
	cnt := copy(b, raw)
	if len(raw) > cnt {
		this.rdbuf.Write(raw[cnt:])
	}
	return cnt, nil
}

func(this *RelayFrameConn) Write(b []byte) (int, error) {
	pkt, err := BuildDataPacket(b)
	if err != nil {
		return 0, err
	}
	err = net_utils.SendSpecLen(this.Conn, pkt)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func(this *RelayFrameConn) Close() error {
	if this.rbuf.Len() != 0 || this.wbuf.Len() != 0 {
		return errors.New(fmt.Sprintf("buffer spare, rs:%d, ws:%d", this.rbuf.Len(), this.wbuf.Len()))
	}
	return this.Conn.Close()
}

func Dial(addr string, config *RelayConfig) (*RelayClientConn, error) {
	return DialWithTimeout(addr, time.Hour, config)  //应该差不多效果了吧, 一个钟都连不上, 那只能玩个蛋蛋了。。
}

func buildAuthReq(config *RelayConfig) []byte {
	req := msg.AuthMsgReq{}
	req.Pwd = proto.String(config.Pwd)
	req.User = proto.String(config.User)
	req.Address = &msg.RelayAddress{
		AddressType:proto.Int32(config.Address.AddrType),
		Address:proto.String(config.Address.Addr),
		Name:proto.String(config.Address.Name),
		Port:proto.Uint32(uint32(config.Address.Port)),
	}
	req.OpType = proto.Int32(config.RelayType)

	data, _ := proto.Marshal(&req)
	return data
}

func doAuth(conn net.Conn, config *RelayConfig) (error, uint32) {
	data := BuildMsgFrame(buildAuthReq(config))
	err := net_utils.SendSpecLen(conn, data)
	if err != nil {
		return errors.New(fmt.Sprintf("send auth pkg to svr fail, err:%v, conn:%s, datalen:%d", err, conn.RemoteAddr(), len(data))), 0
	}
	authBuffer := make([]byte, 128)
	data, err = RecvOneMsg(conn, authBuffer)
	if err != nil {
		return errors.New(fmt.Sprintf("recv auth rsp from svr fail, err:%v, conn:%s", err, conn.RemoteAddr())), 0
	}
	msg := msg.AuthMsgRsp{}
	err = proto.Unmarshal(data, &msg)
	if err != nil {
		return errors.New(fmt.Sprintf("decode auth rsp fail, err:%v, conn:%s", err, conn.RemoteAddr())), 0
	}
	if msg.GetResult() != 0 {
		return errors.New(fmt.Sprintf("auth fail, user:%s, pwd:%s, result:%d", config.User, config.Pwd, msg.GetResult())), 0
	}

	return nil, msg.GetToken()
}

func WrapConnection(conn net.Conn, config *RelayConfig) (*RelayClientConn, error) {
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	err, token := doAuth(conn, config)
	conn.SetDeadline(time.Time{})
	if err != nil {
		return nil, err
	}
	return &RelayClientConn{Conn:WrapRelayFrameConn(conn, nil, nil), token:token}, nil
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
