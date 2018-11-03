package wcf

import (
	"encoding/binary"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"hash/crc32"
	"net"
	"net_utils"
	"proxy/socks"
	"sync"
	"testing"
	"time"
)

func TestConnect(t *testing.T) {
	conn, err := net.DialTimeout("tcp", "www.pin-cong.com:443", 2*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	conn.Close()
}

func checkFrame(buf []byte) int {
	if len(buf) <= 8 {
		return 0
	}
	total := binary.BigEndian.Uint32(buf)
	if total > 1024 {
		return -1
	}
	if len(buf) < int(total) {
		return 0
	}
	return int(total)
}

func getFrameData(data []byte) ([]byte, int, error, uint32, uint32) {
	total := checkFrame(data)
	if total <= 0 {
		return nil, 0, errors.New("data invalid"), 0, 0
	}
	buf := make([]byte, total-8)
	copy(buf, data[4:total-4])
	crc := binary.BigEndian.Uint32(data[total-4:])
	c := crc32.Checksum(buf, crc32.IEEETable)
	if c != crc {
		return nil, 0, errors.New("crc invalid"), 0, 0
	}
	return buf, total, nil, crc, c
}

func buildFrameData(data []byte) []byte {
	total := 4 + len(data) + 4
	buf := make([]byte, total)
	binary.BigEndian.PutUint32(buf, uint32(total))
	copy(buf[4:], data)
	crc := crc32.Checksum(data, crc32.IEEETable)
	binary.BigEndian.PutUint32(buf[total-4:], crc)
	return buf
}

func TestSendBack(t *testing.T) {
	acceptor, err := net.Listen("tcp", "127.0.0.1:8807")
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := acceptor.Accept()
		if err != nil {
			log.Errorf("recv fail, err:%v", err)
			continue
		}
		log.Infof("recv conn:%s, time:%v", conn.RemoteAddr(), time.Now())
		go func(cn net.Conn) {
			buf := make([]byte, 1024)
			defer func() {
				cn.Close()
			}()
			for {
				start := time.Now()
				cnt, err := cn.Read(buf)
				if err != nil {
					log.Errorf("read err:%v", err)
					return
				}
				err = net_utils.SendSpecLen(cn, buf[:cnt])
				if err != nil {
					log.Errorf("write err:%v", err)
					return
				}
				log.Infof("write %d char cost:%d", cnt, time.Now().Sub(start)/time.Millisecond)
			}
		}(conn)
	}
}

func TestSRLittle(t *testing.T) {
	log.Infof("connect start:%v", time.Now())
	conn, err := socks.DialWithTimeout("127.0.0.1:8807", "127.0.0.1:8010", time.Second*3)
	log.Infof("connect cost:%v", time.Now())
	if err != nil {
		log.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer func() {
			wg.Done()
		}()
		for i := 0; i < 1000000; i++ {
			data := buildFrameData([]byte(fmt.Sprintf("helloworld:%d", i)))
			net_utils.SendSpecLen(conn, data)
			//log.Infof("send:%s", string(data[4:len(data) - 4]))
			//time.Sleep(1 * time.Millisecond)
		}
		log.Infof("send finish:%v", time.Now())
	}()
	go func() {
		defer func() {
			wg.Done()
		}()
		buf := make([]byte, 512)
		index := 0
		fRead := false
		for {
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			cnt, err := conn.Read(buf[index:])
			if err != nil {
				log.Errorf("err:%v, index:%d", err, index)
				return
			}
			if !fRead {
				fRead = true
				log.Infof("first read:%v", time.Now())
			}
			index += cnt
			for {
				datalen := checkFrame(buf[:index])
				if datalen < 0 {
					log.Fatalf("data len:%d < 0", datalen)
				}
				if datalen == 0 {
					break
				}
				raw, _, err, old, newcrc := getFrameData(buf[:datalen])
				if err != nil {
					log.Fatalf("read err:%v", err)
				}
				copy(buf, buf[datalen:])
				index -= datalen
				log.Infof("recv:%s, old:%d, new:%d", string(raw), old, newcrc)
			}
		}
	}()
	wg.Wait()
	conn.Close()
	log.Infof("conn:%s close", conn.RemoteAddr())
}

func TestStart(t *testing.T) {
	cfg := LocalConfig{}
	cfg.Timeout = 5 * time.Second
	cfg.Proxyaddr = []ProxyAddrInfo{ProxyAddrInfo{"127.0.0.1:8020", 50, "tcp"}}
	cfg.Localaddr = append(cfg.Localaddr, AddrConfig{Name: "socks", Address: "127.0.0.1:8010"})
	cfg.User = "test"
	cfg.Pwd = "xxx"
	cli := NewClient(&cfg)
	err := cli.Start()
	if err != nil {
		log.Fatal(err)
	}
}
