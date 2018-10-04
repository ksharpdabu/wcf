package http

import (
	"net"
	"time"
	"fmt"
	"errors"
	"net_utils"
	"strings"
	"bytes"
	"encoding/hex"
	"strconv"
	"proxy"
)

func init() {
	proxy.RegistClient("http", func(addr string, proxy string, timeout time.Duration) (net.Conn, error) {
		return DialWithTimeout(addr, proxy, timeout)
	})
}

type HttpClient struct {
	net.Conn
	rbuf []byte
}

func(this *HttpClient) Read(b []byte) (int, error) {
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

var CONNECT_STRING = "CONNECT %s HTTP/1.0\r\n\r\n"

func handleShake(addr string, conn net.Conn) (*HttpClient, error) {
	err := net_utils.SendSpecLen(conn, []byte(fmt.Sprintf(CONNECT_STRING, addr)))
	if err != nil {
		return nil, errors.New(fmt.Sprintf("send connect req fail, err:%v", err))
	}
	buf := make([]byte, 2048)
	index := 0
	total := len(buf)
	var loc int = -1
	for ; index < total; {
		cnt, err := conn.Read(buf[index:])
		if err != nil {
			return nil, errors.New(fmt.Sprintf("recv connect rsp fail, err:%v", err))
		}
		index += cnt
		loc = bytes.Index(buf[0:index], []byte(HTTP_END))
		if loc < 0 {
			continue
		}
		break
	}
	if loc < 0 {
		return nil, errors.New(fmt.Sprintf("not found http end from buf, buf:%s", hex.EncodeToString(buf[:index])))
	}
	codeinfo := strings.SplitN(string(buf[:index]), " ", 3)
	if len(codeinfo) != 3 {
		return nil, errors.New(fmt.Sprintf("invalid http rsp line, code len:%d", len(codeinfo)))
	}
	code, err := strconv.ParseInt(codeinfo[1], 10, 32)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("recv http code invalid, may not num, err:%v", err))
	}
	if code != 200 {
		return nil, errors.New(fmt.Sprintf("recv invalid rsp code:%d", code))
	}
	cli := &HttpClient{Conn:conn}
	if loc + 4 < index {
		cli.rbuf = buf[loc + 4:]
	}
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