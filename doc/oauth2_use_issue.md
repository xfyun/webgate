#### access_token 使用问题

- access_token 存放于数据库中，多地机房无法共享。
- access_token ttl 到期后会自动清理，无需担心
- oauth2 获取access_token 使用的client_id 与 client_secret 需要单独创建,无法直接使用api_key 与 api_secret。

#### 解决方案：
 
- 定时从数据库中拉取最新的access_token,同步到各地。kong 本身也是通过这种方式实时拉取最新的数据更新缓存。