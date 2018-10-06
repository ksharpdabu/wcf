package relay

import (
	"net"
	"context"
	"errors"
	"net_utils"
	"fmt"
	"encoding/binary"
	"wcf/relay/msg"
	"github.com/golang/protobuf/proto"
	"math/rand"
	"time"
	"mix_layer"
)

const MAX_BYTE_AUTH_PACKET uint32 = 128
const MAX_BYTE_PER_PACKET uint32 = 64 * 1024
const ONE_PER_BUFFER_SIZE uint32 = MAX_BYTE_PER_PACKET + 1024

type AuthFunc func(user, pwd string) bool
type MixWrapFunc func(conn net.Conn) (mix_layer.MixConn, error)

type connrecv struct {
	conn *RelayConn
	err error
}

type RelayAcceptor struct {
	listener net.Listener
	OnAuth AuthFunc
	connectionList chan *connrecv
	mixFunc MixWrapFunc
}

func(this *RelayAcceptor) AddMixWrap(fun MixWrapFunc) {
	this.mixFunc = fun
}

type RelayConn struct {
	targetAddress string
	targetType int32
	targetName string
	targetPort uint32
	targetOPType int32
	net.Conn
	token uint32
	user string
}

func(this *RelayConn) GetTargetName() string {
	return this.targetName
}

func(this *RelayConn) GetTargetPort() uint32 {
	return this.targetPort
}

func(this *RelayConn) GetTargetOPType() int32 {
	return this.targetOPType
}

func(this *RelayConn) GetUser() string {
	return this.user
}

func(this *RelayConn) GetTargetAddress() string {
	return this.targetAddress
}

func(this *RelayConn) GetTargetType() int32 {
	return this.targetType
}

func(this *RelayConn) GetToken() uint32 {
	return this.token
}

func newRelayAcceptor() *RelayAcceptor {
	return &RelayAcceptor{connectionList:make(chan *connrecv, 5)}
}

func Bind(address string) (*RelayAcceptor, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	return WrapListener(listener)
}

func WrapListener(listener net.Listener) (*RelayAcceptor, error) {
	ra := newRelayAcceptor()
	ra.listener = listener
	return ra, nil
}

func isDone(ctx context.Context) bool {
	select {
	case <- ctx.Done():
		return true
	default:
		return false
	}
}

func BuildMsgFrame(data []byte) []byte {
	msg := make([]byte, len(data) + 4)
	binary.BigEndian.PutUint32(msg, uint32(len(data)) + 4)
	copy(msg[4:], data)
	return msg
}

func RecvOneMsg(conn net.Conn, buffer []byte) ([]byte, error) {
	err := net_utils.RecvSpecLen(conn, buffer[0:4])
	if err != nil {
		return nil, err
	}
	total := binary.BigEndian.Uint32(buffer[0:4])
	if total >= MAX_BYTE_PER_PACKET {
		return nil, errors.New(fmt.Sprintf("invalid pkg size:%d", total))
	}
	if total > uint32(len(buffer)) {
		return nil, errors.New(fmt.Sprintf("buffer too small, pkg size:%d, buffer size:%d", total, len(buffer)))
	}
	err = net_utils.RecvSpecLen(conn, buffer[0:total - 4])
	if err != nil {
		return nil, err
	}
	return buffer[0:total - 4], nil
}

func(this *RelayAcceptor) buildAuthRsp(result int32, token uint32) []byte {
	rsp := msg.AuthMsgRsp{}
	rsp.Result = proto.Int32(result)
	rsp.Token = proto.Uint32(token)
	data, _ := proto.Marshal(&rsp)
	return data
}

func(this *RelayAcceptor) doHandshake(conn net.Conn) (*RelayConn, error) {
	buf := make([]byte, 512)
	err := net_utils.RecvSpecLen(conn, buf[0:4])
	if err != nil {
		return nil, errors.New(fmt.Sprintf("relay conn recv buf head fail, err:%v, conn:%s", err, conn.RemoteAddr()))
	}
	total := binary.BigEndian.Uint32(buf)
	if total >= MAX_BYTE_AUTH_PACKET || total == 0 {
		return nil, errors.New(fmt.Sprintf("relay conn recv a invalid pkg, pkg len:%d, conn:%s", total, conn.RemoteAddr()))
	}
	err = net_utils.RecvSpecLen(conn, buf[0:total - 4])
	if err != nil {
		return nil, errors.New(fmt.Sprintf("relay conn recv auth pkg fail, err:%v, head total, conn:%s", err, total, conn.RemoteAddr()))
	}
	reqmsg := msg.AuthMsgReq{}
	err = proto.Unmarshal(buf[0:total - 4], &reqmsg)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("decode auth packet body fail, err:%v, pkt len:%d, conn:%s", err, total, conn.RemoteAddr()))
	}
	result := this.OnAuth(reqmsg.GetUser(), reqmsg.GetPwd())
	token := rand.Uint32()
	if !result {
		net_utils.SendSpecLen(conn, BuildMsgFrame(this.buildAuthRsp(int32(msg.AuthResult_AUTH_USER_PWD_INVALID), token)))
		return nil, errors.New(fmt.Sprintf("invalid user/pwd, user:%s, pwd:%s, conn:%s",
			reqmsg.GetUser(), reqmsg.GetPwd(), conn.RemoteAddr()))
	}
	if len(reqmsg.GetAddress().GetAddress()) == 0 {
		net_utils.SendSpecLen(conn, BuildMsgFrame(this.buildAuthRsp(int32(msg.AuthResult_AUTH_INVALID_ADDRESS), token)))
		return nil, errors.New(fmt.Sprintf("invalid address:%s, user:%s, conn:%s",
			reqmsg.GetAddress().GetAddress(), reqmsg.GetUser(), conn.RemoteAddr()))
	}
	err = net_utils.SendSpecLen(conn, BuildMsgFrame(this.buildAuthRsp(int32(msg.AuthResult_AUTH_OK), token)))
	if err != nil {
		return nil, errors.New(fmt.Sprintf("send auth succ rsp to client fail, user:%s, err:%v, conn:%s", reqmsg.GetUser(), err, conn.RemoteAddr()))
	}
	return &RelayConn{
		targetAddress:reqmsg.GetAddress().GetAddress(),
		targetType:reqmsg.GetAddress().GetAddressType(),
		token:token,
		Conn:WrapRelayFrameConn(conn, nil, nil),
		user:reqmsg.GetUser(),
		targetName:reqmsg.Address.GetName(),
		targetPort:reqmsg.Address.GetPort(),
		targetOPType:reqmsg.GetOpType(),
	}, nil
}

func(this *RelayAcceptor) Start() error {
	if this.listener == nil {
		return errors.New("no listener")
	}
	go func() {
		for {
			conn, err := this.listener.Accept()
			if err != nil {
				this.connectionList <- &connrecv{ nil, err }
			}
			go func() {
				if this.mixFunc != nil {
					tconn, err  := this.mixFunc(conn)
					if err != nil {
						this.connectionList <- &connrecv{nil, err}
						conn.Close()
					} else {
						conn = tconn
					}
				}
				conn.SetDeadline(time.Now().Add(10 * time.Second))
				client, err := this.doHandshake(conn)
				conn.SetDeadline(time.Time{})
				if err != nil {
					this.connectionList <- &connrecv{nil, err}
					conn.Close()
				} else {
					this.connectionList <- &connrecv{client, nil}
				}
			}()
		}
	}()
	return nil
}

func(this *RelayAcceptor) Accept() (*RelayConn, error) {
	if this.listener == nil {
		return nil, errors.New("no listener")
	}
	cli := <-this.connectionList
	return cli.conn, cli.err
}
