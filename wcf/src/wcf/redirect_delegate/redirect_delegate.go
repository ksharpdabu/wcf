package redirect_delegate

import "wcf/redirect"
import (
	_ "wcf/redirect/http"
	_ "wcf/redirect/raw"
	_ "wcf/redirect/timeout"
	"net"
)

func InitAll(file string) error {
	return redirect.InitAll(file)
}

func Process(name string, conn net.Conn) (int64, int64, error) {
	return redirect.Process(name, conn)
}