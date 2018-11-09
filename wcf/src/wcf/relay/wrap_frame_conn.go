package relay

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net_utils"
)

type RelayFrameConn struct {
	net.Conn
	rbuf  bytes.Buffer
	wbuf  bytes.Buffer
	rdbuf bytes.Buffer
	rtmp  []byte
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

func (this *RelayFrameConn) Read(b []byte) (int, error) {
	if this.rdbuf.Len() != 0 {
		cnt := copy(b, this.rdbuf.Bytes())
		this.rdbuf.Next(cnt)
		return cnt, nil
	}
	var total = 0
	var cerr error
	for {
		total, cerr = CheckRelayPacketReady(this.rbuf.Bytes())
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

func (this *RelayFrameConn) Write(b []byte) (int, error) {
	if len(b) > 4*int(MAX_BYTE_PER_PACKET)/5 {
		b = b[:len(b)*4/5]
	}
	pkt := BuildDataPacket(b)
	err := net_utils.SendSpecLen(this.Conn, pkt)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (this *RelayFrameConn) Close() error {
	if this.rbuf.Len() != 0 || this.wbuf.Len() != 0 {
		return errors.New(fmt.Sprintf("buffer spare, rs:%d, ws:%d", this.rbuf.Len(), this.wbuf.Len()))
	}
	return this.Conn.Close()
}
