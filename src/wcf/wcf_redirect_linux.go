package wcf

import (
	"net"
	log "github.com/sirupsen/logrus"
	"unsafe"
	"syscall"
	"os"
	"proxy/socks"
	"net_utils"
	"context"
)

type RedirectServer struct {
	config *RedirectConfig
	tcplistener *net.TCPListener
}

func NewRedirect(config *RedirectConfig) *RedirectServer {
	cli := &RedirectServer{}
	cli.config = config
	return cli
}

//from https://github.com/cybozu-go/transocks
const (
	// SO_ORIGINAL_DST is a Linux getsockopt optname.
	SO_ORIGINAL_DST = 80

	// IP6T_SO_ORIGINAL_DST a Linux getsockopt optname.
	IP6T_SO_ORIGINAL_DST = 80
)

func getsockopt(s int, level int, optname int, optval unsafe.Pointer, optlen *uint32) (err error) {
	_, _, e := syscall.Syscall6(
		syscall.SYS_GETSOCKOPT, uintptr(s), uintptr(level), uintptr(optname),
		uintptr(optval), uintptr(unsafe.Pointer(optlen)), 0)
	if e != 0 {
		return e
	}
	return
}

func GetOriginalDST(conn *net.TCPConn) (*net.TCPAddr, error) {
	f, err := conn.File()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fd := int(f.Fd())
	// revert to non-blocking mode.
	// see http://stackoverflow.com/a/28968431/1493661
	if err = syscall.SetNonblock(fd, true); err != nil {
		return nil, os.NewSyscallError("setnonblock", err)
	}

	v6 := conn.LocalAddr().(*net.TCPAddr).IP.To4() == nil
	if v6 {
		var addr syscall.RawSockaddrInet6
		var len uint32
		len = uint32(unsafe.Sizeof(addr))
		err = getsockopt(fd, syscall.IPPROTO_IPV6, IP6T_SO_ORIGINAL_DST,
			unsafe.Pointer(&addr), &len)
		if err != nil {
			return nil, os.NewSyscallError("getsockopt", err)
		}
		ip := make([]byte, 16)
		for i, b := range addr.Addr {
			ip[i] = b
		}
		pb := *(*[2]byte)(unsafe.Pointer(&addr.Port))
		return &net.TCPAddr{
			IP:   ip,
			Port: int(pb[0])*256 + int(pb[1]),
		}, nil
	}

	// IPv4
	var addr syscall.RawSockaddrInet4
	var len uint32
	len = uint32(unsafe.Sizeof(addr))
	err = getsockopt(fd, syscall.IPPROTO_IP, SO_ORIGINAL_DST,
		unsafe.Pointer(&addr), &len)
	if err != nil {
		return nil, os.NewSyscallError("getsockopt", err)
	}
	ip := make([]byte, 4)
	for i, b := range addr.Addr {
		ip[i] = b
	}
	pb := *(*[2]byte)(unsafe.Pointer(&addr.Port))
	return &net.TCPAddr{
		IP:   ip,
		Port: int(pb[0])*256 + int(pb[1]),
	}, nil
}

func(this *RedirectServer) handleProxy(conn net.Conn, sessionid uint32) {
	original, err := GetOriginalDST(conn.(*net.TCPConn))
	if err != nil {
		log.Errorf("Get connection original dst fail, err:%v, conn:%s", err, conn.RemoteAddr())
		conn.Close()
		return
	}
	log.Infof("Recv connection from local, addr:%s, sessionid:%d, original:%s", conn.RemoteAddr(), sessionid, original.String())
	remote, err := socks.DialWithTimeout(original.String(), this.config.Proxyaddr, this.config.Timeout)
	if err != nil {
		log.Errorf("Connect to proxy svr fail, err:%v, proxy:%s", err, this.config.Proxyaddr)
		conn.Close()
		return
	}
	log.Infof("Connect to proxy svr succ, sessionid:%d, local:%s, remote:%s", sessionid, conn.RemoteAddr(), remote.RemoteAddr())
	defer func() {
		conn.Close()
		remote.Close()
	}()
	sbuf := make([]byte, 64 * 1024)
	dbuf := make([]byte, 64 * 1024)
	ctx, cancel := context.WithCancel(context.Background())
	br, bw, pr, pw, bre, bwe, pre, pwe := net_utils.Pipe(conn, remote, sbuf, dbuf, ctx, cancel, this.config.Timeout)
	log.Infof("Transfer data finish, br:%d, bw:%d, pr:%d, pw:%d, bre:%v, bwe:%v, pre:%v, pwe:%v, session:%d, local:%s, remote:%s", br, bw, pr, pw, bre, bwe, pre, pwe, sessionid, conn.RemoteAddr(), remote.RemoteAddr())
}

func(this *RedirectServer) Start() error {
	listener, err := net.Listen("tcp", this.config.Localaddr)
	if err != nil {
		log.Errorf("Redirect svr bind addr:%s fail:err:%v", this.config.Localaddr, err)
		return err
	}
	this.tcplistener = listener.(*net.TCPListener)
	var sessionid uint32 = 0
	for {
		cli, err := this.tcplistener.Accept()
		if err != nil {
			log.Errorf("Accept client fail, err:%v", err)
			continue
		}
		sessionid++
		go this.handleProxy(cli, sessionid)
	}
}