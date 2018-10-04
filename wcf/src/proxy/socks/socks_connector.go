package socks

import (
	"net"
	"time"
	"errors"
	"fmt"
	"strconv"
	"encoding/hex"
	"proxy"
	"encoding/binary"
	"net_utils"
)

type SocksClient struct {
	net.Conn
	rbuf []byte
}

func(this *SocksClient) Read(b []byte) (int, error) {
	if len(this.rbuf) != 0 {
		cnt := copy(b, this.rbuf)
		if cnt == len(this.rbuf) {
			this.rbuf = nil
		} else {
			this.rbuf = this.rbuf[cnt:]
		}
		return cnt, nil
	}
	return this.Conn.Read(b)
}

func BuildSocks5Handshake() []byte {
	return []byte { 0x5, 0x1, 0x0 }
}

func CheckSocks5HandshakeResult(result []byte) (int, error) {
	if len(result) < 2 {
		return 0, errors.New(fmt.Sprintf("invalid result len:%d, hex:%s", len(result), hex.EncodeToString(result)))
	}
	if result[0] != 0x5 {
		return -1, errors.New(fmt.Sprintf("invalid socks ver:%d", result[0]))
	}
	if result[1] != 0x0 {
		return -2, errors.New(fmt.Sprintf("invalid method seleced:%d", result[1]))
	}
	return 2, nil
}

func BuildSocks5Req(host string, port uint16) ([]byte, error) {
	addrType := 0
	var v net.IP
	var addr []byte
	if v = net.ParseIP(host); v == nil {
		addr = make([]byte, 1 + len(host))
		addr[0] = byte(len(host))
		copy(addr[1:], host)
		addrType = proxy.ADDR_TYPE_DOMAIN
	} else if ip := v.To4(); ip != nil {
		addrType = proxy.ADDR_TYPE_IPV4
		addr = make([]byte, 4)
		copy(addr, []byte(ip)[:4])
	} else if ip := v.To16(); ip != nil {
		addrType = proxy.ADDR_TYPE_IPV6
		addr = make([]byte, 16)
		copy(addr, []byte(ip)[:16])
	} else {
		return nil, errors.New(fmt.Sprintf("invalid host:%s", host))
	}
	req := []byte{ 0x5, 0x1, 0x0 }
	req = append(req, byte(addrType))
	req = append(req, addr...)
	pb := make([]byte, 2)
	binary.BigEndian.PutUint16(pb, port)
	req = append(req, pb...)
	return req, nil
}

func CheckSocks5RspResult(data []byte) (int, error) {
	if len(data) < 10 {
		return 0, errors.New(fmt.Sprintf("buffer len invalid, at least 10, but got:%d", len(data)))
	}
	if data[1] != 0x0 {
		return -1, errors.New(fmt.Sprintf("socks handle shake fail, rsp:%d", data[1]))
	}
	total := 0
	if data[3] == proxy.ADDR_TYPE_DOMAIN {
		total = 4 + 1 + int(data[4]) + 2
	} else if data[3] == proxy.ADDR_TYPE_IPV4 {
		total = 4 + 4 + 2
	} else if data[3] == proxy.ADDR_TYPE_IPV6 {
		total = 4 + 16 + 2
	} else {
		return -2, errors.New(fmt.Sprintf("invalid atyp:%d, data:%s", data[3], hex.EncodeToString(data)))
	}
	if len(data) < total {
		return 0, errors.New(fmt.Sprintf("need more buffer, acquire:%d, but got:%d", total, len(data)))
	}
	return total, nil
}

func handleShake(addr string, conn net.Conn) (*SocksClient, error) {
	host, sport, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	port, err := strconv.ParseUint(sport, 10, 16)
	if err != nil {
		return nil, err
	}
	reqBuf, err := BuildSocks5Req(host, uint16(port))
	if err != nil {
		return nil, errors.New(fmt.Sprintf("build sock req fail, err:%v", err))
	}
	var totalbuf []byte
	totalbuf = append(totalbuf, BuildSocks5Handshake()...)
	totalbuf = append(totalbuf, reqBuf...)
	err = net_utils.SendSpecLen(conn, totalbuf)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("send hand shake/req fail, err:%v, hex:%s", err, hex.EncodeToString(totalbuf)))
	}
	buffer := make([]byte, 1024)
	index := 0
	hasCheckHandshake := false
	for {
		cnt, err := conn.Read(buffer[index:])
		if err != nil || cnt == 0 {
			return nil, errors.New(fmt.Sprintf("Read buffer fail, err:%v, conn:%s", err, conn.RemoteAddr()))
		}
		index += cnt
		if !hasCheckHandshake {
			if index < 2 {
				continue
			}
			cret, cerr := CheckSocks5HandshakeResult(buffer[0:2])
			if cret < 0 || cerr != nil {
				return nil, errors.New(fmt.Sprintf("check hand shake fail, ret:%d, err:%v", cret, cerr))
			}
			hasCheckHandshake = true
			buffer = buffer[2:]
			index -= 2
		}
		cret, cerr := CheckSocks5RspResult(buffer[:index])
		if cret == 0 {
			continue
		}
		if cret < 0 {
			return nil, errors.New(fmt.Sprintf("check socks rsp fail, ret:%d, err:%v", cret, cerr))
		}
		buffer = buffer[cret:]
		break
	}
	cli := &SocksClient{Conn:conn, rbuf:buffer}
	return cli, nil
}

func Dial(addr string, proxy string) (net.Conn, error) {
	return DialWithTimeout(addr, proxy, 1 * time.Hour)
}

func DialWithTimeout(addr string, proxy string, timeout time.Duration) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", proxy, timeout)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("dial err:%v, target:%s", err, proxy))
	}
	conn.SetDeadline(time.Now().Add(timeout))
	newConn, err := handleShake(addr, conn)
	conn.SetDeadline(time.Time{})
	if err != nil {
		conn.Close()
		return nil, err
	}
	return newConn, nil
}