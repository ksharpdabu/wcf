package net_utils

import (
	"net"
	"context"
	"time"
	"sync"
	"io"
	//"github.com/sirupsen/logrus"
	"strconv"
	"regexp"
	"strings"
	"errors"
	"fmt"
)

func IsDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

//src read, src write, dst read, dst write, src read err, src write err, dst read err, dst write err
func Pipe(src net.Conn, dst net.Conn,
	srcReadBuffer []byte, dstReadBuffer []byte,
		ctx context.Context, cancel context.CancelFunc,
			timeout time.Duration) (uint64, uint64, uint64, uint64, error, error, error, error) {
	var wg sync.WaitGroup
	wg.Add(2)
	//src read, src write, dst read, dst write
	var sr, sw, dr, dw uint64
	//src read err, src write err, dst read err, dst write err
	var sre, swe, dre, dwe error
	go func() {
		defer func() {
			cancel()
			wg.Done()
		} ()
		for {
			src.SetReadDeadline(time.Now().Add(timeout))
			srcRead, dstWrite, srcReadErr, dstWriteErr := DataCopy(src, dst, srcReadBuffer)
			sr += uint64(srcRead)
			dw += uint64(dstWrite)
			sre = srcReadErr
			dwe = dstWriteErr
			//if srcRead == 0 || dstWrite == 0 {
			//	return
			//}
			if srcReadErr == nil && dstWriteErr == nil {
				continue
			}
			if srcReadErr == io.EOF || dstWriteErr == io.EOF {
				return
			} else if err, ok := srcReadErr.(net.Error); ok && err.Timeout() {
				if IsDone(ctx) {
					return
				}
			} else {
				return
			}
		}
	}()
	go func() {
		defer func() {
			cancel()
			wg.Done()
		}()
		for {
			dst.SetReadDeadline(time.Now().Add(timeout))
			dstRead, srcWrite, dstReadErr, srcWriteErr := DataCopy(dst, src, dstReadBuffer)
			dr += uint64(dstRead)
			sw += uint64(srcWrite)
			dre = dstReadErr
			swe = srcWriteErr
			//if dstRead == 0 || srcWrite == 0 {
			//	return
			//}
			if dstReadErr == nil && srcWriteErr == nil {
				//logrus.Infof("re:%v, we:%v, r:%d, w:%d", dstReadErr, srcWriteErr, dstRead, srcWrite)
				continue
			}
			//logrus.Infof("xre:%v, we:%v, r:%d, w:%d", dstReadErr, srcWriteErr, dstRead, srcWrite)
			if dstReadErr == io.EOF || srcWriteErr == io.EOF {
				return
			} else if err, ok := dstReadErr.(net.Error); ok && err.Timeout() {
				if IsDone(ctx) {
					//logrus.Infof("connection not done:conn:%s, err:%v", dst.RemoteAddr(), err)
					return
				} else {
					//logrus.Infof("connection not done, conn:%s, err:%v", dst.RemoteAddr(), err)
				}
			} else {
				return
			}
		}
	}()
	wg.Wait()
	return sr, sw, dr, dw, sre, swe, dre, dwe
}

func DataCopy(src net.Conn, dst net.Conn, buffer []byte) (int, int, error, error) {
	cnt, err := src.Read(buffer)
	if err != nil {
		return 0, 0, err, nil
	}
	data := buffer[:cnt]
	total := len(data)
	index := 0
	for ; index < total; {
		wcnt, werr := dst.Write(data[index:])
		if werr != nil {
			return cnt, wcnt, nil, werr
		}
		index += wcnt
	}
	return cnt, cnt, nil, nil
}

func CopyTo(src net.Conn, dst net.Conn) (int, int, error, error) {
	//defer func() {
	//	err := recover()
	//	if err != nil {
	//		log.Fatal("copy write panic, err:%v", err)
	//	}
	//} ()
	buf := make([]byte, 64)
	readCnt := 0
	writeCnt := 0
	for {
		cnt, rerr := src.Read(buf)
		if rerr != nil {
			return readCnt, writeCnt, rerr, nil
		}
		readCnt += cnt

		data := buf[0:cnt]
		writeIndex := 0
		writeTotal := len(data)
		for ; writeIndex < writeTotal; {
			wcur, werr := dst.Write(data[writeIndex:])
			if werr != nil {
				return readCnt, writeCnt, rerr, werr
			}
			writeCnt += wcur
			writeIndex += wcur
		}
	}
}

func RecvSpecLen(conn net.Conn, buf []byte) error {
	total := len(buf)
	index := 0
	for ; index < total; {
		cur, err := conn.Read(buf[index:])
		//log.Printf("Read:%v, client:%s", buf[index:index + cur], conn.RemoteAddr())
		if err != nil {
			return err
		}
		index += cur
	}
	return nil
}

func SendSpecLen(conn net.Conn, buf[]byte) error {
	total := len(buf)
	index := 0
	for ; index < total; {
		cur, err := conn.Write(buf[index:])
		//log.Printf("Send:%v, client:%s", buf[index:index+cur], conn.RemoteAddr())
		if err != nil {
			return err
		}
		index += cur
	}
	return nil
}

//schema, domain, port
func GetUrlInfo(req string) (error, string, string, int) {
	req = strings.ToLower(req)
	reg := regexp.MustCompile("^(https?)?(://)?([-a-zA-Z0-9\\.]+):?(\\d*).*")
	parsed := reg.FindStringSubmatch(req)
	if len(parsed) != 5 {
		return errors.New(fmt.Sprintf("parse fail, data:%v", parsed)), "", "", 0
	}
	schema := parsed[1]
	host := parsed[3]
	sport := parsed[4]
	if len(host) == 0 {
		return errors.New(fmt.Sprintf("invalid host, url:%s", req)), "", "", 0
	}
	var port = 80
	if len(sport) != 0 {
		tmp, e := strconv.ParseInt(sport, 10, 16)
		if e != nil {
			return errors.New(fmt.Sprintf("parse port fail, portstr:%s, url:%s", sport, req)), "", "", 0
		}
		port = int(tmp)
	} else {
		if len(schema) == 0 || schema == "http" {
			port = 80
		} else if schema == "https" {
			port = 443
		} else {
			return errors.New(fmt.Sprintf("invalid schema:%s, url:%s", schema, req)), "", "", 0
		}
	}
	return nil, schema, host, port
}
