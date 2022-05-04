### webgate 功能点

#### 1 schema 配置文件功能点   版本：1.1.0_13

例： schema_iat.json

```json
[
  {
    "service": "iat",
    "version": "1.0",
    "call":"atmos-iat",
    "route": "/v2/iat",
    "request.data.mapping": {
      "data_type": [1],
      "rule":[
        {"dst":"$[0].format","src":"$.format"},
        {"dst":"$[0].status","src":"$.status"},
        {"dst":"$[0].encoding","src":"$.encoding"},
        {"dst":"$[0].data","src":"$.audio"}
      ]
    },
    "response.data.mapping": {
      "data_type": [0],
      "rule":[
        {"dst":"$.result","src":"$[0].data"},
        {"dst":"$.status","src":"$[0].status"}
      ]
    },
    "schema":{}
  }
]
```


##### atmos_map

示例：
```json
{
  "call": "atmos_iat",
  "atmos_map": {
      "iat": "atmos_iat",
      "tts": "atmos_tts"
  }
}
```
作用： 当atmos_map 不为空时，调用业务层的服务名优先取sub对应的服务。如果没有对应的sub，那么使用默认的call字段对应的服务

##### mock
示例：
```json
{
  "mock":true
}
```
作用：是否启用mock模式

##### trace_sample_rate
示例
```json
{
  "trace_sample_rate": 1
}
```
作用：设置trace日志的采样频率，为1 则每帧都记录，为10 则每10帧记录一个trace，其中第一第二帧必定记录。

##### collect_data
示例：
````json
{
  "collect_data": true
}
````

作用：是否把客户端请求数据的data部分记录到日志。如合成，如果开启，就会吧合成的文本记录到trace中

##### multiple_stream
示例：

````json
{
  "multiple_stream": true
}
````
作用：当客户端请求的ai能力需要传多个数据流时需要开启，开启后多个数据流的frameId将会分开统计，每个数据流的status也已客户端传的为准。

##### disable_schema
````json
{
  "disable_schema": true
}
````

作用：开启后会禁用当前能力的schema校验。如果要开启schmea校验，同时需要app.toml 中的  schema.enable字段为true，

##### disable_appid_check
```json
{
  "disable_appid_check": true
}
```
作用：开启后会不校验当前的客户端传的appid是否与 api_key对应的appid是否一致。如果需要校验，需要同时配置 app.toml 中的 auth.enable_appid_check 为true

##### request.data.mapping

示例：
```json
{
  "request.data.mapping": {
      "data_type": [1],
      "rule":[
        {"dst":"$[0].format","src":"$.format"},
        {"dst":"$[0].status","src":"$.status"},
        {"dst":"$[0].encoding","src":"$.encoding"},
        {"dst":"$[0].data","src":"$.audio"}
      ],
      "script": {
        "first": [
            {
              "if": "and(eq($.business.ent,'sms5s'),eq($.common.app_id,'123456'))",
              "then": "$.business.ent=sms16k"
            }     
        ],
        "every": []
      } 
    }
}
```
字段含义：
1. data_type: 传给业务层协议的数据流类型。  0：文本、1：音频、2：图像、3：视频，  取值为-1时，则会取客户端传的data_type。
2. rule: 协议映射规则，规则定义了如何把用户请求的json映射成业务层内部pb协议 。
内部pb协议格式为
    ```json
    {
      "data":[
        {
          "id": "",
          "frame_id": "",
          "desc_args": {},
          "encoding": "",
          "format": "",
          "data": "",
          "data_type": 1
        } 
      ]
    }
    ```
3. script: 参数处理规则，用于处理复杂的参数校验和转换的场景  script.first 仅第一帧执行， script.every: 每一帧都会执行


##### response.data.mapping

示例：
```json
{
  "response.data.mapping": {
     "data_type": [0],
       "rule":[
           {"dst":"$.result","src":"$[0].data"},
           {"dst":"$.status","src":"$[0].status"}
       ],
      "script": {
        "every": [
            "$.text=''",
            {
              "for": "_,v in $.result.ws",
              "do": "$.text=append($.text,v.cw[0].w)"
            }
        ]
      } 
    }
}
```
字段含义：
1. data_type: 对引擎的响应结果data部分作何处理返回给客户端。0：转成json格式，如果转换失败则不作任何处理，返回base64 字符串.1：转成string， 2 不作任何处理转成json格式。
2. rule: 协议映射规则，业务层内部pb协议 映射成客户端响应字段。
内部pb协议格式为
    ```json
    {
      "data":[
        {
          "id": "",
          "frame_id": "",
          "desc_args": {},
          "encoding": "",
          "format": "",
          "data": "",
          "data_type": 1
        } 
      ]
    }
    ```
3. script: 响应处理规则，一般用不到 script.every: 每一帧都会执。如例子所示，会把听写的结果解析出来放到text字段中返回给客户端

script 语法参考：[script.md](json_rule.md)

##### schema 校验自定义字段

**constVal**
示例：
```json
{
"ent":{
  "type": "string",
  "constVal": "123"
}
}
```
作用：无论客户端是否传了ent 字段，ent都会被设置为123


**defaultVal**
示例：
```json
{
"ent":{
  "type": "string",
  "defaultVal": "123"
}
}
```
作用：如果客户端没有传ent，ent会被设置为123

**replaceKey**
示例：
```json
{
"ent":{
  "type": "string",
  "replaceKey": "ent_type"
}
}
```
作用：如果客户端传了ent参数，那么ent会被替换为ent_type.


#### 2 app.toml 配置文件的功能

 配置文件示例：
 ```toml
[server]
#host = "10.1.107.8"    # 和xsf.toml中要保持一致
netCard = ""   # 监听网卡
port = "8082"  # 监听端口
pipe_depth=5   # 排序管道深度
enable_sonar = true  # 是否开启sonar 日志
ignore_sonar_codes = [10101] # 忽略的sonar错误码
ignore_resp_codes= [10101] # 忽略的响应错误码，不会返回给客户端
[auth]
enable_appid_check=true   # 是否开启appid与apikey的唯一性校验

[xsf]
from="webgate-ws"  
#server_port="9027"    # xsf server端口   和xsg.toml要保持一致
location="hu"   # 生成sid用的机房名称
call_retry=2    # 
enable_respsort=false #对响应结果排序,依赖于引擎的响应结果中的frame_id，有序
dc="hu"

[session]
scan_interver=60     #全局session扫描时间
timeout_interver=15   #消息发送超时间隔
handshake_timeout=4  #握手超时时间
session_timeout= 65  #会话超时时间
session_close_wait=5  # 会话关闭后，等待多久关闭连接

[log]
file="/log/server/webgate-ws.log"
level="error"
size=100   #max log size :MB
count=10  #max log file num
caller=true #控制日志是否刷行号
batch=100 #刷日志的最小条数
async = true      #是否启用异步日志。1是0否。缺省1。

[schema]
enable=true  #是否开启schema校验
services=["iat"]   #指定要加载的schema文件名称，如改配置会加载schema_iat.json 
```

**具体的功能点如下**

所属域|字段名称|类型|示例|是否必要|作用
--- |----   |----|---|---|---
server|netCard|string|netCart = ""|否|服务的监听网卡,一般不填。
server|port|string|port = "8080"|是|监听的端口号，
server|enable_sonar|bool|enable_sonar = true|是|是否启用sonar日志
server|ignore_sonar_codes|int array|ignore_sonar_codes = [10101]|否|忽略的sonar 统计错误码，包含在里面的错误码不会上报到sonar 上
server|ignore_resp_codes|int array|ignore_resp_codes = [10101]|否|忽略的响应错误码，包含在里面的错误码不会返回给客户端。
auth|enable_appid_check|bool|enable_appid_check = true|是|是否校验客户端上传的appid 与apikey是否对应。开启后从kong入口进入并且kong开启了xfyun-hmac-auth插件的请求则会校验它，直连webgate则不会校验
xsf|location|string|location = "dx"|是|生成的sid中的机房字段会用改配置
session|timeout_interver|int|timeout_interver = 15 |是|客户端长时间没有发送数据或者服务后端长时间有给响应时，服务端会主动会关闭连接。（单位秒）
session|session_timeout|int|session_timeout = 65 |是|允许客户端和服务端交互的最大会话时长，超过该时长的会话服务端会主动关闭。（单位秒）
schema|enable|bool|enable = true |是|是否开启schema参数校验（全局）。该配置和schema 配置文件中的disable_schema 一起决定是否开启schema校验。
schema|services|string array|services=["iat"]|是|要加载的配置中心配置文件，被加载的文件名称需要满足<br>schema_${service}.json 格式。如示例<br>服务端会加载schema_iat.json这个配置文件。被加载的配置文件需要上传到配置中心。


#### 3 多路复用（所有能力）
多路复用模式主要是为了解决server2server调用方式每次会话都要重新建立连接带来的耗时问题和客户端并发过高时维持大量的长连接带来的开销问题。
多路复用模式支持长连接保持和一个连接并发发起多路会话。

多路复用模式使用：
1 请求参数url参数需要添加： stream_mode=multiplex  开启多路复用模式。
如：
```text
ws://iat-api.xfyun.cn/v2/iat?stream_mode=multiplex&authorization=.....
```

请求参数common中需添加参数 cid，标识了当前连接上的每一路会话，每一个连接上的每一路回话都要不同。服务端会在响应结果中带回该值，便于客户端通过该字段找到对应的会话。

cid: 每一路会话在连接上的唯一标识，可以是一个随机的字符串，也可以是一个自增的字符串。

请求参数示例：
```json
{
  "common":{
    "cid":"1",
    "app_id":"123456"
  },
  "business":{
    "language":"zh_cn",
    "domain":"iat",
    "accent":"mandarin",
    "eos":20000
  },
  "data":{
    "status":0,
    "audio":"...",
    "format":"audio/L16;rate=16000",
    "encoding":"raw"
  }
}
```

响应结果示例：（开启了wpgs）

````json
{
  "cid":"1",   //该值和请求参数中的common.cid一致。
  "sid":"",
  "code":"0",
  "message":"success",
   "data": {
      "result": {},
      "status": 1
    }
  
}
````

注意事项：
1. 客户端需要保证每一路会话的每一帧数据通过同一个连接发送到服务端，并同时保证cid在每一个连接上的唯一性，连接断开时，连接上所有的未完成会话都会终止。
2. 每一个连接超过15s不发送数据时会断开连接，建议客户端自己实现连接池来管理连接。
3. 建议客户端当前会话结束时，发送一个close 帧 common.cmd=close来主动结束当前会话，异常时也要发送。close 帧如下：


close 帧：
````json
{
  "common":{
    "app_id":"123456",
    "cmd":"close",
    "cid":"1"
  },
  "data":{
       
  }
}
````


#### 4 session 重连（不支持合成能力）

一路回话难免会出现断网络连接超时导致断开的情况。对于某些应用场景，如评测，输入大量的数据而只会出一次结果，如果因为网络导致连接断开的话，
就无法获取到结果了，而且用户的输入也会浪费掉，导致体验不好。
 
 现在支持会话恢复来解决这个问题。当用户断开连接时，在服务端允许的超时时间内（一般是5s，可以使用app.toml的session.session_close_wait 配置来改变）
 重新连接并且重连的第一帧带上sid并且设置cmd=retry，就可以重新恢复会话。
 
 同时需要配置对应能力的 schema 文件中的配置  enable_retry =true ,来开启该能力的重连机制。

请求示例：
````json
{
  "common": {
    "sid": "ist00070002@dx16ef3aafdea5745882",
    "cmd": "retry",
    "app_id": "123456"
}
}
````
 