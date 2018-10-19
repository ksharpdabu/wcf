package http

import (
	"bufio"
	"net"
	"net/http"
	"math/rand"
	"io"
	"encoding/json"
	"wcf/redirect"
	"fmt"
	//"github.com/sirupsen/logrus"
	"bytes"
	"net_utils"
	"errors"
)

func init() {
	redirect.Regist("http", ParseHTTPArgs, ProcessHTTP)
}

type HTTPParam struct {
	RedirectHost   []string `json:"redirect"`
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
	buffer.WriteString("\r\n");
	return buffer.Bytes()
}

func ProcessHTTP(conn net.Conn, extra interface{}) (int64, int64, error) {
	param := extra.(*HTTPParam)
	defer conn.Close()
	bio := bufio.NewReader(conn)
	req, err := http.ReadRequest(bio)
	if err != nil {
		return 0, 0, errors.New(fmt.Sprintf("parse request fail, err:%v", err))
	}
	uri := param.RedirectHost[rand.Int() % len(param.RedirectHost)] + "/" + req.RequestURI
	newReq, err := http.NewRequest(req.Method, uri, req.Body);
	if err != nil {
		return 0, 0, errors.New(fmt.Sprintf("create new request fail, err:%v, url:%s", err, uri))
	}
	client := &http.Client{

	}
	rsp, err := client.Do(newReq)
	if err != nil {
		return 0, 0, errors.New(fmt.Sprintf("do request fail, err:%v, url:%s", err, uri))
	}
	defer rsp.Body.Close()
	headers := buildHTTPRespHeader(rsp.StatusCode, rsp.Header)
	if err := net_utils.SendSpecLen(conn, headers); err != nil {
		return 0, 0, errors.New(fmt.Sprintf("write http rsp header fail, err:%v", err))
	}
	//logrus.Infof("data:%+v", rsp)
	w, err := io.Copy(conn, rsp.Body)
	if err != nil {
		err = errors.New(fmt.Sprintf("transfer data fail, err:%v, url:%s", err, uri))
	}

	return w, w, err
}
