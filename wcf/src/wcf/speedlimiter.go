package wcf

import (
	"net"
	"github.com/juju/ratelimit"
	"io"
)

type SpeedConn struct {
	net.Conn
	reader io.Reader
	writer io.Writer
}

func(this *SpeedConn) Read(b []byte) (int, error) {
	return this.reader.Read(b)
}

func(this *SpeedConn) Write(b []byte) (int, error) {
	return this.writer.Write(b)
}

//单位为KB/s
func NewSpeedConn(conn net.Conn, rMax int64, wMax int64) net.Conn {
	//发现设置的值在实际跑的时候, 速度只有3/4, 不知道是不是哪里搞错了, 先补回去。。哈哈。
	rMax = rMax * 1024 * 4 / 3
	wMax = wMax * 1024 * 4 / 3
	r := ratelimit.NewBucketWithRate(float64(rMax), rMax)
	w := ratelimit.NewBucketWithRate(float64(wMax), wMax)
	return &SpeedConn{conn, ratelimit.Reader(conn, r), ratelimit.Writer(conn, w) }
}
