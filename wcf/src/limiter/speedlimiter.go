package limiter

import (
	"github.com/juju/ratelimit"
	"io"
	"net"
	"time"
)

type SpeedConn struct {
	net.Conn
	reader io.Reader
	writer io.Writer
}

func (this *SpeedConn) Read(b []byte) (int, error) {
	return this.reader.Read(b)
}

func (this *SpeedConn) Write(b []byte) (int, error) {
	return this.writer.Write(b)
}

//单位为KB/s
func NewSpeedConn(conn net.Conn, rMax int64, wMax int64) net.Conn {
	rMax = rMax * 1024 * 4 / 3
	wMax = wMax * 1024 * 4 / 3
	r := ratelimit.NewBucketWithQuantum(100*time.Millisecond, rMax, rMax/10)
	w := ratelimit.NewBucketWithQuantum(100*time.Millisecond, wMax, wMax/10)
	return &SpeedConn{conn, ratelimit.Reader(conn, r), ratelimit.Writer(conn, w)}
}
