
### Oauth2 认证

#### 使用access_token请求服务端AI能力：

**请求参数**

- access_token :有效的token，获取方式见[access_token 获取](#oauth2-access_token-)

**请求示例**

https://ws-api.xfyun.cn/oauth2/v2/iat?access_token=oSBxV2rGkAlTmZZ3ksL6JtO7732QyyJ7

请求认证失败会返回 401 状态码。或者其他的4xx系列状态码<br>
失败响应示例：

```
{"error_description":"The access token is invalid or has expired","error":"invalid_token"}
```

#### Oauth2 access_token 获取

**POST** 请求，请求地址为： https://ws-api.xfyun.cn/oauth2/token

请求参数：

- grant_type: 固定为 client_credentials<br>
- client_id: 平台申请的api_key<br>
- client_secret: 平台申请的api_secret<br>

例如：<br>
https://ws-api.xfyun.cn/oauth2/token?grant_type=client_credentials&client_id=9eda6546894b0ab3f1d8ab4e53f5ee49&client_secret=fcc1f2699768f3373d1622e9dab9780d



示例代码 bash

````
curl -X POST "https://ws-api.xfyun.cn/oauth2/token?grant_type=client_credentials&client_id=9eda6546894b0ab3f1d8ab4e53f5ee49&client_secret=fcc1f2699768f3373d1622e9dab9780d"
````

成功返回响应，http状态码为200， 响应主体为json格式：

- access_token: 用于请求服务端的AI能力
- expires_in: token有效期 ，单位秒,一般为30 分钟

示例：<br>
```json
{"token_type":"bearer","access_token":"oSBxV2rGkAlTmZZ3ksL6JtO7732QyyJ7","expires_in":1800}
```

请求失败时会返回错误，http 状态码为400 或者其他的4xx系列错误码

错误响应示例如下：
```json
{"error_description":"Invalid client authentication","error":"invalid_client"}
```


#### access_token 使用问题

- access_token 存放于数据库中，多地机房无法共享。
- access_token ttl 到期后会自动清理，无需担心数据库存在过多的垃圾。
- oauth2 获取access_token 使用的client_id 与 client_secret 需要单独创建,无法直接使用api_key 与 api_secret。


**解决方案**
 
- 定时从数据库中拉取最新的access_token,同步到各地。kong 本身也是通过这种方式实时拉取最新的数据更新缓存，一般间隔是5s，因此并不会对数据库造成压力。
- 与apikey 同步不一样的是，access_token 的生产机房是三地机房，需要三地相互同步。
- client_id 与 client_secret 前期可以手动创建，后期可以自动从 api_key 与 api_secret 中同步