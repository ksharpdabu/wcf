package http

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net_utils"
	"wcf/redirect"
)

func init() {
	redirect.Regist("http", ParseHTTPArgs, ProcessHTTP)
}

type HTTPParam struct {
	RedirectHost []string `json:"redirect"`
}

func ParseHTTPArgs(data []byte) (interface{}, error) {
	param := &HTTPParam{}
	err := json.Unmarshal(data, param)
	if err != nil {
		return nil, err
	}
	return param, nil
}

func buildHTTPRespHeader(code int, headers http.Header) []byte {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("HTTP/1.1 %d OK\r\n", code))
	for k, v := range headers {
		if len(v) == 0 {
			continue
		}
		buffer.WriteString(fmt.Sprintf("%s: %s\r\n", k, v[0]))
	}
	buffer.WriteString("\r\n")
	return buffer.Bytes()
}

func ProcessHTTP(conn net.Conn, extra interface{}) (int64, int64, error) {
	param := extra.(*HTTPParam)
	defer func() {
		if err := conn.Close(); err != nil {
			fmt.Printf("http close conn fail, err:%v, conn:%s\n", err, conn.RemoteAddr())
		}
	}()
	bio := bufio.NewReader(conn)
	req, err := http.ReadRequest(bio)
	if err != nil {
		return 0, 0, fmt.Errorf("parse request fail, err:%v", err)
	}
	uri := param.RedirectHost[rand.Int()%len(param.RedirectHost)] + "/" + req.RequestURI
	newReq, err := http.NewRequest(req.Method, uri, req.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("create new request fail, err:%v, url:%s", err, uri)
	}
	client := &http.Client{}
	rsp, err := client.Do(newReq)
	if err != nil {
		return 0, 0, fmt.Errorf("do request fail, err:%v, url:%s", err, uri)
	}
	defer rsp.Body.Close()
	headers := buildHTTPRespHeader(rsp.StatusCode, rsp.Header)
	if err := net_utils.SendSpecLen(conn, headers); err != nil {
		return 0, 0, fmt.Errorf("write http rsp header fail, err:%v", err)
	}
	_, w, rerr, werr := net_utils.CopyTo(conn, rsp.Body)
	if (rerr != nil && rerr != io.EOF) || (werr != nil && werr != io.EOF) {
		err = errors.New(fmt.Sprintf("transfer data fail, rerr:%v, werr:%v, url:%s", rerr, werr, uri))
	}
	return w, w, err
}
