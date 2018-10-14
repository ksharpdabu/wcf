# wcf
代理工具, 支持socks代理, http代理, 流量中转, 以前用shadowsocks的时候总是需要用其他工具才能生成http代理, 所以这次写这个东西, 直接加入http代理支持。

[TOC]

## 编译方式
**简单点可以使用wcf目录下的build.bat进行编译, 缺少的依赖项需要自己手动go get获取。**
```shell
git clone https://github.com/xxxsen/wcf.git --depth=1
#进到工程根目录(bin, pkg, src所在的目录为根目录)
cd wcf/wcf
#将wcf工程加到GOPATH变量中
export GOPATH=$GOPATH:`pwd`

#生成本地端
cd src/wcf/cmd/local
go build

#生成远程端
cd src/wcf/cmd/server 
go build
```
编译完会在local和server目录下分别生成对应的可执行文件, 如果不知道怎么编译的,可以使用releases目录下已经生成好的文件。

## 服务配置
### 本地端
```json
{
	"localaddr": [
		{"name":"socks", "address":"127.0.0.1:8010"},
		{"name":"http", "address":"127.0.0.1:8011"},
		{"name":"forward", "address":"127.0.0.1:8012"}
	],
	"loadbalance":{
		"enable":true, 
		"max_errcnt":2,
		"max_failtime":30
	},	
	"proxyaddr":[
		{"addr":"127.0.0.1:8020", "weight":100, "protocol":"tcp"}
	],	
	"user":"test",
	"pwd":"xxx",
	"timeout":5,
	"host":"d:/host.rule",
	"encrypt":"xor",
	"key":"hellotest",
	"transport":"d:/transport.json"
}
```
* localaddr 为本地监听的地址, 目前支持3种代理, socks5, http, forward(透传)
* loadbalance 负载均衡模块
* * max_errcnt 连接错误多少次会被踢掉
* * max_failtime 连接被踢掉后多久重新可用, 单位为秒
* proxyaddr 为远程server的地址, 及其权重信息
* * addr 远程服务器地址
* * weight 权重信息
* * protocol 使用的协议
* user/pwd 鉴权用的用户名和密码
* timeout 链接超时时间, 单位是秒
* host 这个是用来配置本地host的, 一行一个配置,由域名, 操作类型, 替换域名(可选)组成, 例如baidu.com,proxy[,google.com], 分3种操作类型,block, proxy, direct, 分别代表黑名单(禁止链接), 走代理, 直连, 具体的可以看下面的配置
* encrypt 加密方式, 目前只有xor, comp, 想了下, 貌似只要混淆就能FQ, 所以就只搞了这2种
* key 加密的key
* transport 协议配置, 同server端的配置, 可以共用一个

#### host配置
```host.rule 
#一行一个配置, 井号开头的为注释
#可以只配置域名,操作类型, 也可以配置替换域名
#支持cidr,server端请务必将内网的地址给block掉, 不然会有安全风险
#替换的域名只能是域名或者ip, 不能为cidr
#配置的域名不只影响自身, 还会影响其子域名
#如下面的几行
127.0.0.0/8,direct
192.168.0.0/16,direct
baidu.com,block
www.test.com,direct,127.0.0.1
google.com,proxy
```

### 远程端
```json
{
	"localaddr":[{"address":"127.0.0.1:8020", "protocol":"tcp"}, {"address":"127.0.0.1:8021", "protocol":"kcp"}],
	"timeout":5,
	"userinfo":"D:/GoProj/wcf_proj/src/wcf/cmd/server/userinfo.dat",
	"encrypt":"xor",
	"key":"hellotest",
	"host":"d:/host.rule",
	"transport":"d:/transport.json",
	"err_redirect":[
		{"protocol":"tcp", "address":"127.0.0.1:36000"}
	],
	"report":{
		"enable":true,
		"visitor":"json",
		"visitor_config":"./visitor.json"
	}
}
```
* localaddr 本地监听地址, 这里是服务端的监听地址, 如果要公网使用, 这里要填为0.0.0.0:8020
* timeout 链接超时时间, 单位为秒
* userinfo 用户配置文件, 下面说明
* encrypt/key 加密方式与加密key, 需要保持与客户端一致
* host 同client配置
* transport 协议配置, 正常来说可以不用管, 有个transport.json 直接指定就可以了。
* err_redirect 用于当协议错误的时候进行转发
* * protocol 转发使用的协议
* * address 转发到此地址上
* report 用于上报用户的访问数据
* * enable 是否启用
* * visitor 使用的观察者, 目前能使用的有json和sqlite3, 观察者的配置可以看后面
* * visitor_config 配置观察者需要的数据的文件

#### 观察者配置

##### sqlite3 观察者的配置
以json进行配置, 主要有下面几个项

```json
{
	"init_template":"create table if not exists visit_record_%s (username VARCHAR(64) NOT NULL,host VARCHAR(1024) NOT NULL,user_from VARCHAR(32) NOT NULL,start_time DATE,end_time DATE,read_cnt BIGINT,write_cnt BIGINT,connect_cost int)",
	"insert_template":"insert into visit_record_%s(username, host, user_from, start_time, end_time, read_cnt, write_cnt, connect_cost) values(?, ?, ?, ?, ?, ?, ?, ?)",
	"store_location":"./visit.db",
	"pre_create_n_day":3
}
```

* init_template 初始化建表模板
* insert_template 插入语句的模板
* store_location DB文件存储的位置
* pre_create_n_day 预创建N天的表格

正常来说, 如果要进行配置的话, 只需要配置store_location 用于指定db的位置就行了。

##### json 观察者配置
```json
{
	"store":"./visit.jsonline"
}
```
就只有一个store的项, 用于输出jsonline的存储位置


#### 用户配置信息说明
以json line方式进行配置, 一行一个用户。
```json
{"user":"test", "pwd":"xxx", "forward":{"enable":true, "address":"127.0.0.1:8000"}, "max_conn":100}
{"user":"hello", "pwd":"world", "forward":{"enable":false}}
{"user":"xxxtc", "pwd":"hahaa"}
```

* user 用户名
* pwd 密码
* forward 当为透传的时候才使用
* * enable 启用透传
* * address 链接指向
* max_conn 指定该用户最大的链接数,避免把server端拖崩, 不建议设置太小, 会导致用户打不开页面。

## 启动命令
```
#本地端 
./local --config=./local.json
#远程端 
./server --config=./server.json
```
