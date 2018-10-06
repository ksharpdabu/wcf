package wcf

import (
	"testing"
	"time"
	log "github.com/sirupsen/logrus"
	"net"
	"proxy/socks"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"net_utils"
	"github.com/pkg/errors"
	"sync"
)

func TestConnect(t *testing.T) {
	conn, err := net.DialTimeout("tcp", "www.pin-cong.com:443", 2 * time.Second)
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
	buf := make([]byte, total - 8)
	copy(buf, data[4:total - 4])
	crc := binary.BigEndian.Uint32(data[total - 4:])
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
	binary.BigEndian.PutUint32(buf[total - 4:], crc)
	return buf
}

func TestSRLittle(t *testing.T) {
	conn, err := socks.DialWithTimeout("sendev.cc:8807", "127.0.0.1:8010", time.Second * 2)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer func() {
			wg.Done()
		}()
		for i := 0; i < 50000; i++ {
			data := buildFrameData([]byte(fmt.Sprintf("helloworld:%d", i)))
			net_utils.SendSpecLen(conn, data)
			log.Infof("send:%s", string(data[4:len(data) - 4]))
		}
	}()
	go func() {
		defer func() {
			wg.Done()
		}()
		buf := make([]byte, 1024 * 32)
		index := 0
		conn.SetReadDeadline(time.Now().Add(30 *time.Second))
		for {
			cnt, err := conn.Read(buf[index:])
			if err != nil {
				t.Fatalf("%v, index:%d", err, index)
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
					t.Fatal(err)
				}
				copy(buf, buf[datalen:])
				index -= datalen
				log.Infof("recv:%s, old:%d, new:%d", string(raw), old, newcrc)
			}
		}
	}()
	wg.Wait()
	conn.Close()
}

func TestStart(t *testing.T) {
	cfg := LocalConfig{}
	cfg.Timeout = 5 * time.Second
	cfg.Proxyaddr = []ProxyAddrInfo {ProxyAddrInfo{"127.0.0.1:8020", 50}}
	cfg.Localaddr = append(cfg.Localaddr, AddrConfig{Name:"socks", Address:"127.0.0.1:8010"})
	cfg.User = "test"
	cfg.Pwd = "xxx"
	cli := NewClient(&cfg)
	err := cli.Start()
	if err != nil {
		log.Fatal(err)
	}
}

