package relay

import (
	"net"
	"errors"
	"net_utils"
	"fmt"
	"wcf/relay/msg"
	"github.com/golang/protobuf/proto"
	"math/rand"
	"time"
	"mix_layer"
	"strings"
	"transport_delegate"
	//log "github.com/sirupsen/logrus"
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
	handshake bool
	errmsg error
}

func(this *RelayConn) GetHandshakeErrmsg() error {
	return this.errmsg
}

func(this *RelayConn) GetHandshakeResult() bool {
	return this.handshake
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

func Bind(protocol string, address string) (*RelayAcceptor, error) {
	protocol = strings.ToLower(protocol)
	var listener net.Listener
	var err error
	listener, err = transport_delegate.Bind(protocol, address)
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

func(this *RelayAcceptor) doHandshake(conn net.Conn) (*RelayConn, error) {
	cn := &RelayConn{}
	cn.Conn = nil
	cn.token = rand.Uint32()
	cn.handshake = false
	buf := make([]byte, 1024)
	index := 0
	total := len(buf)
	var ckResult int
	var ckErr error
	var auth msg.AuthMsgReq
	for {
		for ; index < total ;{
			cnt, err := conn.Read(buf[index:])
			if err != nil {
				return nil, errors.New(fmt.Sprintf("relay conn recv buf head fail, err:%v, conn:%s", err, conn.RemoteAddr()))
			}
			index += cnt
			ckResult, ckErr = CheckRelayPacketReadyWithLength(buf[:index], MAX_BYTE_AUTH_PACKET)
			if ckResult != 0 {
				break
			}
		}
		if ckResult > 0 {
			var raw []byte
			raw, ckErr = GetPacketData(buf[:ckResult])
			if ckErr == nil {
				ckErr = proto.Unmarshal(raw, &auth)
				if ckErr == nil {
					break
				}
			}
		}
		cn.Conn = &ExtraConn{conn, buf[:index]}
		cn.errmsg = ckErr
		return cn, nil
	}
	result := this.OnAuth(auth.GetUser(), auth.GetPwd())
	if !result {
		net_utils.SendSpecLen(conn, BuildDataPacket(BuildAuthRspMsg(int32(msg.AuthResult_AUTH_USER_PWD_INVALID), cn.token)))
		return nil, errors.New(fmt.Sprintf("invalid user/pwd, user:%s, pwd:%s, conn:%s",
			auth.GetUser(), auth.GetPwd(), conn.RemoteAddr()))
	}
	if len(auth.GetAddress().GetAddress()) == 0 || len(auth.GetAddress().GetName()) == 0 || auth.GetAddress().GetPort() == 0 {
		net_utils.SendSpecLen(conn, BuildDataPacket(BuildAuthRspMsg(int32(msg.AuthResult_AUTH_INVALID_ADDRESS), cn.token)))
		return nil, errors.New(fmt.Sprintf("invalid address:%s/name:%s/port:%d, user:%s, conn:%s",
			auth.GetAddress().GetAddress(), auth.GetAddress().GetName(), auth.GetAddress().GetPort(),
				auth.GetUser(), conn.RemoteAddr()))
	}
	err := net_utils.SendSpecLen(conn, BuildDataPacket(BuildAuthRspMsg(int32(msg.AuthResult_AUTH_OK), cn.token)))
	if err != nil {
		return nil, errors.New(fmt.Sprintf("send auth succ rsp to client fail, user:%s, err:%v, conn:%s", auth.GetUser(), err, conn.RemoteAddr()))
	}
	var spare []byte
	if ckResult == index {
		spare = nil
	} else {
		spare = buf[ckResult:index]
	}
	cn.targetAddress = auth.GetAddress().GetAddress()
	cn.targetType = auth.GetAddress().GetAddressType()
	cn.user = auth.GetUser()
	cn.targetName = auth.GetAddress().GetName()
	cn.targetPort = auth.GetAddress().GetPort()
	cn.targetOPType = auth.GetOpType()
	cn.Conn = WrapRelayFrameConn(conn, spare, nil)
	cn.handshake = true
	return cn, nil
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
