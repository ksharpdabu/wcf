package comp

import (
	"compress/gzip"
	"net"
)

func init() {
	//目前使用会产生死循环,先不提供
	//
	//mix_layer.Regist("comp", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
	//	return Wrap(key, conn)
	//})
}

type Comp struct {
	net.Conn
	key string
	r   *gzip.Reader
	w   *gzip.Writer
}

func (this *Comp) SetKey(key string) {
	this.key = key
}

func (this *Comp) Read(b []byte) (int, error) {
	if this.r == nil {
		r, err := gzip.NewReader(this.Conn)
		if err != nil {
			return 0, err
		}
		this.r = r
	}
	return this.r.Read(b)
}

func (this *Comp) Write(b []byte) (n int, err error) {
	cnt, err := this.w.Write(b)
	if err != nil {
		return cnt, err
	}
	err = this.w.Flush()
	return cnt, err
}

func Wrap(key string, conn net.Conn) (*Comp, error) {
	//_, err := gzip.NewReader(conn)
	//if err != nil {
	//	return nil, errors.New(fmt.Sprintf("create gz reader fail, err:%v", err))
	//}
	writer := gzip.NewWriter(conn)
	return &Comp{Conn: conn, key: key, r: nil, w: writer}, nil
}
