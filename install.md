### 1 配置中心创建服务  webgate-aipaas webgate-http-aipaas 版本均为 0.1.0
在这两个服务中创建 app.toml 和xsf.toml，注意修改其中一个的端口号，防止出现端口号冲突，配置文件如下

app.toml
````
[server]
#host = "10.1.107.8"    # 和xsf.toml中要保持一致
netCard = ""
port = "8889"
mock = false
pipe_depth=5
enable_sonar = true
#ignore_sonar_codes = [10101,10223]
ignore_resp_codes= [10101]
[auth]
enable_auth=false   #是否开启鉴 权
max_date_interval=300   #鉴权时间校验偏差
enable_appid_check=true
[xsf]
from="webgate-ws"
#server_port="9027"    # xsf server端口   和xsg.toml要保持一致
location="dx"
call_retry=2    #xsf
enable_respsort=false #对响应结果排序
dc="dx"
[session]
scan_interver=60     #全局session扫描时间
timeout_interver=15   #消息发送超时间隔
conn_timeout=65000   #session超时时间
handshake_timeout=4  #handshake 超时时间
session_timeout= 65000
session_close_wait=5

[log]
file="/log/server/webgate-ws.log"
level="error"
size=100   #max log size :MB
count=10  #max log file num
caller=true #控制日志是否刷行号
batch=1 #刷日志的最小条数
async = true      #是否启用异步日志。1是0否。缺省1。
[schema]
enable=true  #是否开启schema校验
services=["iat"]
file_prefix="schema"
file_service = "webgate-schema"
file_version = "0.0.0"

[engine_schema]
enable=true  #是否开启schema校验
file_prefix="bas"
file_service = "bas-schema"
file_version = "0.0.0"

[guider_schema]
enable=true  #是否开启schema校验
file_prefix="guider"
file_service = "guider-schema"
file_version = "0.0.0"

[app_id_cloud_id]

file_prefix="app_id_cloud_id"
file_service = "aicloud"
file_version = "0.0.0"

# domain 和cloudid 关系映射配置存放地址
[domain_cloud_id]

file_prefix="domain_cloud_id"
file_service = "aicloud"
file_version = "0.0.0"
# 大类路由配置
[category_schema]
enable=true  #是否开启schema校验
file_prefix="schema"
file_service = "category_schema"
file_version = "0.0.0"


````
xsf.toml

````
#服务自身的配置
#注意此section名需对应bootConfig中的service
#----------------------------------------服务端------------------------------------------------------------
[webgate-aipaas]#已做缺省处理
host = "0.0.0.0"#若host为空，则取netcard对应的ip，若二者均为空，则取hostname对应的ip
#host = "10.1.107.8"
#netcard = "eth0"
#port = 9031
finder = 0 #缺省0
debug = 0 #缺省0

[log]#已做缺省处理
level = "error" #缺省warn
file = "log/xsfs.log" #缺省xsfs.log
#日志文件的大小，单位MB
size = 300 #缺省10
#日志文件的备份数量
count = 3 #缺省10
#日志文件的有效期，单位Day
die = 3 #缺省10
#缓存大小，单位条数,超过会丢弃
cache = 100000 #缺省-1，代表不丢数据，堆积到内存中
#批处理大小，单位条数，一次写入条数（触发写事件的条数）
batch = 160#缺省16*1024
#异步日志
async = 0 #缺省异步
#是否添加调用行信息
caller = 1 #缺省0
wash = 60 #写入磁盘的缺省时间

[trace]
host = "127.0.0.1"
port = 4545 #缺省4545
able = 1            #开启trace
dump = 0
bcluster = "dx"
idc = "dz"
deliver = true          #是否开启网络发包
watch = 1
watchport = 12332
loadts = 10
spill-able=0
backend = 10
buffer=2000
#taddrs="iat@10.1.205.151:50051;10.1.205.151:50052,tts@10.1.205.151:50052;10.1.205.151:50051"

#######------------------------------客户端-----------------------------------------------------
[webgate-ws-c]

conn-timeout = 3000
lb-mode= 0
lb-retry = 2

[sonar]#已做缺省处理,此section如不传缺省启用
#trace收集服务的地址
ds = "vagus_null"
dump = 0 #缺省0
able=1
````

### 2 配置schema
1 在配置中心webgate 所在的group 下创建4个服务， 版本号均为1.0.0
bas-schema ,webgate-schema,guider-schema,category_schema
2 在这4个服务中上传并推送同样的文件 create.txt
````shell script
create zk path
````
3 在配置中心webgate 所在服务下再创建一个服务
aicloud 版本为0.0.0 ，,并推送这两个文件
**domain_cloud_id.json**

```
{

     "c0b2460d": [
          "yangzihhhh-api.ifly-aicloud.com",
          "yangzihhhh-wsapi.ifly-aicloud.com"
     ],
     "c0f2d3f5": [
          "122-api.ifly-aicloud.com",
          "122-wsapi.ifly-aicloud.com"
     ],
     "c12692a8": [
          "21231-api.ifly-aicloud.com",
          "21231-wsapi.ifly-aicloud.com"
     ]

}
```

**app_id_cloud_id.json**
````
[
     {
          "app_id": "000005300daf",
          "cloud_id": "c76d995a"
     },
     {
          "app_id": "009557d29212",
          "cloud_id": "ce646531"
     }
}
````


### 3启动服务 websocket,仅适用于 0.1.0.14 版本以后，镜像

 webgate-ws: hub.iflytek.com/aiaas/webgate-aipaas:0.1.0.27
 webgate-http: hub.iflytek.com/aiaas/webgate-http-aipaas:0.1.0.27


#### 3.1 启动webgate-aipaas
```json
docker run --name=webgate-aipaas -d --net=host -v /etc/localtime:/etc/localtime -v /data1/log/webgate-aipaas:/log/server \
 ./webgate-aipaas --nativeBoot=false --project=AIPaaS --group=hu --service=webgate-ws --version=0.1.0 --url=http://companion.xfyun.iflytek:6868
```


#### 3.2 启动  webgate-http-aipaas
```json
docker run --name=webgate-http-aipaas -d --net=host -v /etc/localtime:/etc/localtime -v /data1/log/webgate-aipaas:/log/server \
 ./webgate-http-aipaas --nativeBoot=false --project=AIPaaS --group=hu --service=webgate-http --version=0.1.0 --url=http://companion.xfyun.iflytek:6868
```
