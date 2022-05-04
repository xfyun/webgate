# 0.1.1版本更新
1. 修改了鉴权协议,鉴权协议见[websocket%20api%20protocol.md](/doc/websocket%20api%20protocol.md)

2. 修改了请求参数和返回结果的部分字段，具体如下:

````
1、iat 的请求参数json对象data 中的传输数据字段由content 改为 audio
2、iat 的返回参数json对象data中的status值的意义由原来的0：识别中，1：识别结束改为1：识别中，2：识别结束，和引擎的返回结果保持了一致。
````

3. 加入了请求参数的 json schema 校验



# 1.0.3 版本更新

- **schema 映射功能扩充**

1. 上个版本的映射响应结果只支持单层路径的映射，不支持嵌套路径的映射

上版本的映射规则和响应结果
```text
 "response.data.mapping": [{
    "data_type": 0,
    "rule": {
      "$[0].data": "result",   
      "$[0].status": "status"  
    }
  }],
  
```
映射后生成的json

```text
{
    "result":{
        ...  //data具体数据
    },
    "status":1
}

```

 新版本的规则:
 
 1. 新增了$符号，代表映射后的root json。
 
 2. 新增了表达式解析，通过表达式实现json映射，构造响应结果
 
 实现上个版本响应结果的映射规则
 
 ```text
 "response.data.mapping": [{
    "data_type": 0,
    "rule": {
      "$.result": "$[0].data",   //$.result  表示把数据放到响应结果里面的data.result 字段上
      "$.status": "$[0].status"  
    }
  }],
 ```
 
3. 支持root直接映射，直接把数据映射到root json 上， 规则如下

```text
 "response.data.mapping": [{
    "data_type": 0,
    "rule": {
      "$": "$[0].data",   //$.result  表示把数据放到响应结果里面的data字段上
      "$.status":"$[0].status"  // 把status字段插入data 的json里面
    }
  }],
``` 

构造的响应结果：

```text
{
    ...   //data具体数据
    "status":1
}
```

4. 支持数组表达式

```text
 "response.data.mapping": [{
    "data_type": 0,
    "rule": {
      "$.data[0]": "$[0].data",   //把数据放到响应结果里面的data[0]字段上
      "$.data[1]":"$[0].args"  // 把数据放到响应结果里面的data[1]字段上
    }
  }],
```
构造的响应结果：

```text
{
    "data":[
        {
           ... //  data里面的数据
        },
        {
           ... // args里面的数据
        }
    ]
}
```


- **参数转换**

支持轻量级的参数转换，

将参数vad_eos 转换成eos的schema规则
```text
  "vad_eos": {
            "type": "integer",
            "minimum": 0,
            "magic":{   //定义magic属性的参数会被转换成magic.key 对应的参数
              "key":"eos",
              "enable":true  //为true参数转换才生效
            }
  }
```


- **动态ai能力增加与删除**

schema中新增或删除ai能力可以热生效。

```json
{
  "rule":[
    "$:$[0].data",
    "$.status:$[0].status"
  ]
}
```

# 1.0.5 更新

#### 1. 动态服务注册

之前的版本在服务启动时需要手动向kong的添加可用实例，服务关闭时需要手动删除实例。这种操作容易失误
并且在服务升级时容易发生服务不可用的情况。现在改为每次服务启动时自动向kong注册实例，并且在docker stop 的时候删除kong上
的实例。实现服务升级无缝切换。

#### 2. 返回结果排序

webgate-ws的所有调用都是异步的，容易造成结果乱序。新版本新增了结果排序功能。能够让一定程度乱序的帧能够有序的返回给用户
可通过 app.toml 中的xsf.enable_respsort=false 来禁用该功能。通过设置server.pipe_depth 来控制最大能容忍的乱序帧个数。默认为5

#### 3. 修复了上个版本的schema bug

1. 修复了status字段映射异常情况。
2. 修复了参数转换同名时会删除参数的情况

#### 4. 新增了appid校验功能

通过kong代理访问api时，如果apikey和appid对应不上，则会拒绝提供服务。apikey和appid可在konga控制台注册
可以通过 app.toml 中的配置 auth.enable_appid_check=false 禁用该功能

#### 5. kong的whitelist api proxy

由于需要把kong adminapi提供给开放平台，但是kong adminapi不能直接提供需要封装。因此单独开发了组件去访问kong 
[git 地址](https://git.xfyun.cn/sjliu7/kong-adminapi-proxy)  [api 文档地址](doc/kongapi.md) 
其中只有白名单管理是封装后的。其他的api都是直接通过代理访问kong-adminapi。

#### . [部署文档](doc/install.md)

# 1.0.6 版本更新

- 集成了sonar
sonar用于统计会话成功率的，每次会话sonar会上传一次数据，数据包含了本次会话的相关信息，具体如下
    
```text
{"endpoint":"10.1.107.8", "metric":"sps", "timestamp":1551843932, "value":0.000000, "counterType":"GAUGE", "step":"10", "tags":"sret=0,svc=igr,ds=vagus_null,sid=igr00d40002@uk169511b3a716b08882,end=true,dc=uk,appid=igrdail,uid=,finalEnt=igr_gray,port=9027,from=webgate,sub=igr,cluster=5s"}
```
其中sonar中的收集的主要数据放在了tags字段下，应该最少包含了 sret：本次会话的错误码，成功则为0，sid：本次会话的sid，appid，finalEnt,sub,from 字段

通过设置xsf.toml的[sonar] 配置项中的dump=1，可以让sonar 把数据保存到本地磁盘。具体位置为工作目录下的metric目录下。

- 性能优化，修改json解码库

#### [部署文档](doc/install.md)

# 1.0.7 版本更新

1 客户端超时刷新由从接收到客户端的数据触发改为接受和收到引擎的返回结果都会触发

2 修改了sonar会话失败判断逻辑，当用户主动结束会话不再报32000错误。

3 修正了schema 参数转换bug，当用户传了修正后的参数，没传修正前的参数，会导致修正后的参数失效。（如想要把参数 vad_eos 转换为eos，用户传了eos，没传vad_eos，会导致eos失效）

