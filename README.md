# wcf
代理工具, 支持socks代理, http代理, 流量中转, 以前用shadowsocks的时候总是需要用其他工具才能生成http代理, 所以这次写这个东西, 直接加入http代理支持。

## 编译方式
```shell
git clone https://github.com/xxxsen/wcf.git
cd wcf 
#将wcf工程加到GOPATH变量中
export GOPATH=$GOPATH:`pwd`

#生成本地端
cd src/wcf/cmd/local
go build

#生成远程端
cd src/wcf/cmd/server 
go build
```
编译完会在local和server目录下分别生成对应的可执行文件

## 服务配置
### 本地端
```json
{
	"localaddr": [
		{"name":"socks", "address":"127.0.0.1:8010"},
		{"name":"http", "address":"127.0.0.1:8011"},
		{"name":"forward", "address":"127.0.0.1:8012"}
	],
	"proxyaddr":"127.0.0.1:8020",
	"user":"test",
	"pwd":"xxx",
	"timeout":5,
	"host":{
		"black":"D:/GoProj/wcf_proj/src/wcf/cmd/local/black.rule",
		"no_proxy":"D:/GoProj/wcf_proj/src/wcf/cmd/local/no_proxy.rule"
	},
	"encrypt":"xor",
	"key":"hellotest"
}
```
* localaddr 为本地监听的地址, 目前支持3种代理, socks5, http, forward(透传)
* proxyaddr 为远程server的地址, 上面写的是127.0.0.1:8020, 需要改成你实际的服务器地址
* user/pwd 鉴权用的用户名和密码
* timeout 链接超时时间, 单位是秒
* host 这个是用来配置本地host的, black为黑名单域名, 在这个文件内的域名都会被reset. no_proxy为不进行代理的域名, 在这个名单内的地址, 会直接在本地进行请求而不走远程server. 这2个的配置方式都是一行一个域名.
* encrypt 加密方式, 目前只有xor, comp, 想了下, 貌似只要混淆就能FQ, 所以就只搞了这2种
* key 加密的key

### 远程端
```json
{
	"localaddr":"127.0.0.1:8020",
	"timeout":5,
	"userinfo":"D:/GoProj/wcf_proj/src/wcf/cmd/server/userinfo.dat",
	"encrypt":"xor",
	"key":"hellotest",
	"secure_check":true
}
```
* localaddr 本地监听地址, 这里是服务端的监听地址, 如果要公网使用, 这里要填为0.0.0.0:8020
* timeout 链接超时时间, 单位为秒
* userinfo 用户配置文件, 下面说明
* encrypt/key 加密方式与加密key, 需要保持与客户端一致
* secure_check 安全检查, 目前只做了简单的内网ip检查

#### 用户配置信息说明
以json line方式进行配置, 一行一个用户。
```json
{"user":"test", "pwd":"xxx", "forward":{"enable":true, "address":"127.0.0.1:8000"}}
{"user":"hello", "pwd":"world", "forward":{"enable":false}}
{"user":"xxxtc", "pwd":"hahaa"}
```

* user 用户名
* pwd 密码
* forward 当为透传的时候才使用
* * enable 启用透传
* * address 链接指向

## 启动命令
```
#本地端 
./local --config=./local.json
#远程端 
./server --config=./server.json
```
