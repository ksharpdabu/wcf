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
	"encrypt":"none",
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
	"localaddr":[
		{"address":"127.0.0.1:8020", "protocol":"tcp"},
		{"address":"127.0.0.1:8021", "protocol":"kcp"},
		{"address":"127.0.0.1:8022", "protocol":"tcp_tls"}
	],
	"timeout":5,
	"userinfo":"D:/GoProj/wcf_proj/src/wcf/cmd/server/userinfo.dat",
	"encrypt":"none",
	"key":"hellotest",
	"host":"d:/host.rule",
	"transport":"d:/transport.json",
	"report":{
		"enable":true,
		"visitor":"json",
		"visitor_config":"d:/visitor.json"
	},
	"redirect":{
		"enable":true,
		"redirector":"http",
		"redirect_config":"d:/redirect.json"
	}
}
```
* localaddr 本地监听地址, 这里是服务端的监听地址, 如果要公网使用, 这里要填为0.0.0.0:8020
* timeout 链接超时时间, 单位为秒
* userinfo 用户配置文件, 下面说明
* encrypt/key 加密方式与加密key, 需要保持与客户端一致
* host 同client配置
* transport 协议配置, 正常来说可以不用管, 有个transport.json 直接指定就可以了, 配置项说明见下面。
* err_redirect 用于当协议错误的时候进行转发
* * protocol 转发使用的协议
* * address 转发到此地址上
* report 用于上报用户的访问数据
* * enable 是否启用
* * visitor 使用的观察者, 目前能使用的有json和sqlite3, 观察者的配置可以看后面
* * visitor_config 配置观察者需要的数据的文件
* redirect 用于错误重定向
* * enable 是否启用, 启用的情况下, 最好不要使用混淆插件, 原因你懂的。
* * redirector 用于协议出错时重定向的具体操作, 目前有http, raw, timeout 3种, 具体配置见下面
* * * http 这个适用于server的传输协议使用tcp-tls(建议使用), tcp, 会把当前的流量解析成http, 然后将请求转发到配置的host上去, 取到回包后再返回给原先的链接。
* * * raw 这个适用于server使用tcp协议, 将流量原封不动的透传到指定的host上去, 例如后端可以是一个ssh server 也可以是一个rdp server。
* * * timeout 适用于所有的传输协议, 在到达指定时长后关闭连接。
* * redirect_config 配置所有的重定向设置的文件, 直接使用在config目录下面的redirect.json即可。

#### 传输配置(transport.json)
以json格式进行配置, 每一个协议定义一个map, 分别有2个子对象, bind和dial, 配置只会在初始化的时候加载一次并保存起来。并不是所有的协议都需要有bind和dial的参数, 如果某一项没有可以直接不填, 如果都没有, 那就配置一个空的, 例如里面的那个tcp。
```json
{
	"tcp":{},
	"kcp":{
		"bind":{
			"data_shards":10,
			"parity_shards":3
		},
		"dial":{
			"data_shards":10,
			"parity_shards":3
		}
	},
	"tcp_tls":{
		"bind":{
			"pem_file":"D:/GoProj/fake_cert/ca.crt",
			"key_file":"D:/GoProj/fake_cert/ca.key"
		},
		"dial":{
			"skip_insecure":false
		}
	}
}
```
##### 参数说明
* kcp 妈蛋, 这2个参数我都不知道干嘛的, 具体的可以去kcp的github页面看下说明, 我这里用的是默认的2个参数。
* * data_shards  
* * parity_shards
* tcp_tls 说白了就是tls, 这个协议主要是用于伪装https
* * pem_file pem文件或者crt文件都ok
* * key_file 私钥文件
* * skip_insecure 当证书错误的时候是否中断, false为不中断

#### 重定向配置
目前使用json作为配置, 结构如下, 每个重定向器都有自己的参数配置, 在server启动时进行加载。
```json
{
	"http":{
		"redirect":["https://en.cppreference.com/"]
	},
	"raw":{
		"protocol":"tcp",
		"target":"127.0.0.1:36000"
	},
	"timeout":{
		"min_duration":1,
		"max_duration":30
	}
}
```
##### 参数说明
* http
* * redirect 重定向的目标地址, 建议转到自己的博客或者其他的大型https网站上去, 可以配置多个, 重定向的时候会随机选一个, 建议只配置一个, 多了实在无意义, 应该是脑抽了才写了支持多个。
* raw
* * protocol 使用的协议, 支持transport.json中配置的所有协议。
* * target 目标地址, 按域名端口配置。
* timeout
* * min_duration 最小的时间值, 单位为秒
* * max_duration 最大的时间值, 会在这2个值中间随机取一个值, 可以配置成相同的。

#### 观察者配置

##### sqlite3 观察者的配置(需要启用CGO)
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
