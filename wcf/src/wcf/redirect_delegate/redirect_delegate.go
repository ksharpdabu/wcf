package redirect_delegate

import "wcf/redirect"
import (
	"net"
	_ "wcf/redirect/http"
	_ "wcf/redirect/raw"
	_ "wcf/redirect/timeout"
)

func InitAll(file string) error {
	return redirect.InitAll(file)
}

func Process(name string, conn net.Conn) (int64, int64, error) {
	return redirect.Process(name, conn)
}
