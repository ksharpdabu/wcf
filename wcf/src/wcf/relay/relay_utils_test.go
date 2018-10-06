package relay

import (
	"testing"
	"encoding/hex"
	"bytes"
	"net"
	"time"
	"fmt"
)

func TestBuildAndParse(t *testing.T) {
	data := []byte("hello world this is a test")
	enc, err := BuildDataPacket(data)
	if err != nil {
		t.Fatal(err)
	}
	total, err := CheckRelayPacketReay(enc)
	t.Logf("total:%d, err:%v", total, err)
	t.Logf("%s", hex.EncodeToString(enc))
	dec, err := GetPacketData(enc)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("decode:%s, enc len:%d, dec len:%d, pkg len:%d", string(dec), len(enc), len(dec), total)
}

type TestConn struct {
	rbuf bytes.Buffer
}

func(this *TestConn) Read(b []byte) (int, error) {
	cnt := copy(b, this.rbuf.Bytes())
	this.rbuf.Next(cnt)
	return cnt, nil
}

func(this *TestConn) Write(b []byte) (int, error) {
	this.rbuf.Write(b)
	return len(b), nil
}

func(this *TestConn) Close() error { return nil }
func(this *TestConn) LocalAddr() net.Addr { return nil }
func(this *TestConn) RemoteAddr() net.Addr { return nil }
func(this *TestConn) SetDeadline(t time.Time) error { return nil }
func(this *TestConn) SetReadDeadline(t time.Time) error { return nil }
func(this *TestConn) SetWriteDeadline(t time.Time) error { return nil }

func TestSendRcv(t *testing.T) {
	tconn := &TestConn{}
	conn := WrapRelayFrameConn(tconn, nil, nil)
	for i := 0; i < 3; i++ {
		cnt, err := conn.Write([]byte(fmt.Sprintf("hello world this is a test:%d", i)))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("i:%d write:%d, wbuf data:%s", i, cnt, hex.EncodeToString(tconn.rbuf.Bytes()))
		raw := make([]byte, 1024)
		cnt, err = conn.Read(raw)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("i:%d read:%d, data:%s, conn r:%s, w:%s, rd:%s, rtmp:%s", i, cnt, string(raw),
			hex.EncodeToString(conn.rbuf.Bytes()), hex.EncodeToString(conn.wbuf.Bytes()),
				hex.EncodeToString(conn.rdbuf.Bytes()), "")
	}
}
