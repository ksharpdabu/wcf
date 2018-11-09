package exchange

import (
	"log"
	"net"
	"sync"
	"testing"
	"time"
)

func onlyOnce() OnDataTransfer {
	v := true
	return func(b []byte) ([]byte, bool) {
		if v {
			v = false
			return reduceOne(b), true
		}
		return b, false
	}
}

func reduceOne(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	return b[1:]
}

func listen(wg *sync.WaitGroup, wgall *sync.WaitGroup) {
	defer wgall.Done()
	l, err := net.Listen("tcp", "127.0.0.1:9999")
	wg.Done()
	if err != nil {
		log.Fatal(err)
	}

	conn, err := l.Accept()
	if err != nil {
		log.Fatal(err)
	}
	conn = NewExchangeConn(conn, nil, []byte("mno"), nil, nil)
	_, err = conn.Write([]byte("hehe"))
	if err != nil {
		log.Fatal(err)
	}
	buf := make([]byte, 1024)
	index := 0
	conn.SetDeadline(time.Now().Add(2 * time.Second))
	for {
		cnt, err := conn.Read(buf[index:])
		if err != nil {
			break
		}
		index += cnt
	}
	log.Printf("server recv:%s\n", string(buf[:index]))
	conn.Close()
	l.Close()
}

func connect(wgall *sync.WaitGroup) {
	defer wgall.Done()
	conn, err := net.DialTimeout("tcp", "127.0.0.1:9999", 2*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	conn = NewExchangeConn(conn, []byte("abc"), []byte("defg"), onlyOnce(), onlyOnce())
	_, err = conn.Write([]byte("haha"))
	if err != nil {
		log.Fatal(err)
	}
	buf := make([]byte, 1024)
	conn.SetDeadline(time.Now().Add(2 * time.Second))
	index := 0
	for {
		cnt, err := conn.Read(buf[index:])
		if err != nil {
			break
		}
		index += cnt
	}
	log.Printf("client recv:%s\n", string(buf[:index]))
}

func TestExchangeExample(t *testing.T) {
	var wg sync.WaitGroup
	var wgall sync.WaitGroup
	wg.Add(1)
	wgall.Add(2)
	go listen(&wg, &wgall)
	wg.Wait()
	connect(&wgall)
	wgall.Wait()
}
