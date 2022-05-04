package main

/*
aipaas 多路复用使用文档
多路复用模式主要是为了复用一个websocket connection ，能够在一路 websocket 连接上同时开启多路会话。
因此需要一个额外的参数来标示一个连接上的数据是属于哪一个会话。
服务端使用cid 参数来标示一路会话的数据,如下
{
	"header"{
		"cid" :  "123456"
	}
	"parameter":{...}
	"payload":{...}
}

- 开启多路复用模式需要新增额外的url 参数：stream_mode=multiplex
- 每一帧数据都要发送cid 参数；需要保障每一路会话的cid 是同样的，同时保证其和其他的会话的cid不一样
- 一个连接建议最多并发开启15路会话。太多的话可能会导致延迟增加
*/

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	apiKey = "94a179ef19bd9f8c5c5a3ac1060016f7"

	apiSecret = "HY7xfGGKO3ilByAE9MHDGS9ByvsNg0gO"
	//requestUrl = "ws://172.31.103.99:8889/v1/private/passivefeaonline"
	//requestUrl = "ws://172.31.103.99:8889/v1/private/testroute"
	requestUrl = "ws://127.0.0.1:8889/v1/private/xist"
)

func initFlags() {
	flag.StringVar(&apiKey, "ak", apiKey, "apiKey")
	flag.StringVar(&apiSecret, "as", apiSecret, "apiSecret")
	flag.StringVar(&requestUrl, "ul", requestUrl, "request url ")

	flag.Parse()
}

type Context struct {
	context context.Context
	cf      context.CancelFunc
}

func newContext(timeout time.Duration) *Context {
	c, f := context.WithTimeout(context.Background(), timeout)
	return &Context{
		context: c,
		cf:      f,
	}
}

func (c *Context) Done() <-chan struct{} {
	return c.context.Done()
}

func (c *Context) Stop() {
	c.cf()
}

func main() {
	initFlags()
	dail := websocket.Dialer{}
	poolConnectionSize := 1

	p := NewPool(func() (*websocket.Conn, error) {
		// 构建鉴权url，开启多路复用模式，需要添加query 参数 stream_mode=multiplex
		conn, rsp, err := dail.Dial(fmt.Sprintf("%s", assembleAuthUrl(requestUrl, apiKey, apiSecret)), nil)
		if err != nil {
			if rsp != nil {
				bs, _ := ioutil.ReadAll(rsp.Body)
				return nil, fmt.Errorf("%s,%w", string(bs), err)
			}
		}
		return conn, err
	}, func(data []byte) (Message, error) {
		var i messageMap
		err := json.Unmarshal(data, &i)
		return i, err
	}, poolConnectionSize, func(err error) {
		fmt.Println("handler error", err)
	})
	//获取一个连接

	wg := sync.WaitGroup{}
	for i := 0; i < 2; i++ {
		conn, err := p.GetConn()
		if err != nil {
			panic(err)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			timeout := 5 * time.Second
			timer := time.NewTimer(timeout)
			client := &clienttest{
				cid:     genCid(),
				conn:    conn,
				results: make(chan Message, 10),
				timer:   timer,
				timeout: timeout,
			}
			// 绑定client 到连接上
			conn.BindClient(client)
			// 会话结束解除绑定
			defer conn.UnBindClient(client)
			// 开启协程发送请求数据
			go client.start()
			// 获取响应结果
			for {
				select {
				case r := <-client.results:
					fmt.Println("get result:", r)
					// 已经拿到status=2 的最后一帧数据，直接return
					if r.(messageMap).Status() == 2 {
						fmt.Println("receive all the data", ",cid=", client.cid)
						return
					}
					timer.Reset(timeout) // 获取到响应数据，重置定时器
				case <-timer.C:
					fmt.Println("context deadline exceed", "cid", client.cid)
					return
				}
			}

		}()
	}

	wg.Wait()

}

type Client interface {
	Cid() string
	OnMessage(msg Message)
	OnError(err error)
}

type Message interface {
	Cid() string
}

type MessageNew func(data []byte) (Message, error)

type DailFunc func() (*websocket.Conn, error)

// client 实现
type clienttest struct {
	cid     string
	status  int
	conn    *Conn
	results chan Message
	timer   *time.Timer
	timeout time.Duration
}

func (c *clienttest) Cid() string {
	return c.cid
}

func (c *clienttest) OnMessage(msg Message) {
	//fmt.Println("cid:",c.cid,msg)
	c.results <- msg
}

func (c *clienttest) OnError(err error) {
	fmt.Println("onError", err)
}

func (c *clienttest) start() {
	f, err := os.Open(`/Users/sjliu/Downloads/iat000e4ade@dx178623039d4060d802.opus-wb`)
	if err != nil {
		panic(err)
	}
	buf := make([]byte, 1280)
	c.status = 0
	seq := 0
	for {
		c.timer.Reset(c.timeout)
		_, err := f.Read(buf)
		if err != nil {
			if err == io.EOF {
				c.status = 2
			} else {
				panic(err)
			}
		}

		//keyService := "ist"
		//keyAccept := "result"
		//keyStreamId := "input"
		//keyData := "audio"

		req := map[string]interface{}{
			"header": map[string]interface{}{
				"app_id": "4CC5779A", //appid 必须带上，只需第一帧发送
				"uid":    "77607d78ccc5424986cdbe4039879b14",
				"status": c.status,
				"cid":    c.Cid(),
			},
			//"parameter": map[string]interface{}{ //business 参数，只需一帧发送
			//	keyService: map[string]interface{}{
			//		"dwa": "wpgs",
			//		keyAccept: map[string]interface{}{
			//			"encoding": "utf8",
			//		},
			//	},
			//},
			//"payload": map[string]interface{}{
			//	keyStreamId: map[string]interface{}{
			//		keyData:       base64.StdEncoding.EncodeToString(buf[:n]),
			//		"encoding":    "opus-wb",
			//		"sample_rate": 16000,
			//		"status":      c.status,
			//		"seq":         seq,
			//	},
			//},
		}

		msg, _ := json.Marshal(req)
		if err := c.conn.Write(msg); err != nil {
			fmt.Println(err)
		}
		seq++

		if c.status != 0 {
			delete(req, "parameter")
		}

		if c.status == 0 {
			c.status = 1
		}
		if c.status == 2 {
			fmt.Println("send all the data", "cid", c.cid)
			return
		}
		time.Sleep(40 * time.Millisecond)
	}
}

var cidseed int64 = 0

func genCid() string {
	return strconv.Itoa(int(atomic.AddInt64(&cidseed, 1)))
}

type messageMap map[string]interface{}

func (m messageMap) Cid() string {
	hd, ok := m["header"].(map[string]interface{})
	if ok {
		cid, _ := hd["cid"].(string)
		return cid
	}
	return ""
}

func (m messageMap) Status() int {
	hd, ok := m["header"].(map[string]interface{})
	if ok {
		cid, _ := hd["status"].(float64)
		return int(cid)
	}
	return 0
}

type Pool struct {
	conn           []*Conn
	size           int // 连接池大小
	idx            int64
	newRespMessage MessageNew
	dail           DailFunc
	errHandler     ErrorHandler
}

type ErrorHandler func(err error)

func NewPool(dail DailFunc, newRespMessage MessageNew, poolSize int, eh ErrorHandler) *Pool {
	if poolSize <= 0 {
		poolSize = 5
	}
	p := &Pool{
		conn:           nil,
		size:           poolSize,
		idx:            0,
		newRespMessage: newRespMessage,
		dail:           dail,
		errHandler:     eh,
	}
	p.init()
	return p
}

func (p *Pool) GetConn() (*Conn, error) {
	conn := p.conn[int(atomic.AddInt64(&p.idx, 1))%len(p.conn)]
	if conn.stat == statDisConnected {
		if err := conn.reset(); err != nil {
			p.handlerError(err)
			return nil, err
		}
	}
	return conn, nil
}

func (p *Pool) handlerError(err error) {
	if p.errHandler != nil {
		p.errHandler(err)
	}
}

func (p *Pool) init() {
	for i := 0; i < p.size; i++ {
		c := NewConn(p.dail, p.newRespMessage)
		c.pool = p
		p.conn = append(p.conn, c)
	}
}

const (
	statDisConnected int32 = iota
	statConnected
)

type Conn struct {
	conn           *websocket.Conn
	lock           sync.Mutex
	dail           DailFunc
	stat           int32 //0
	newRespMessage MessageNew
	clients        sync.Map //cid: Client
	pool           *Pool
}

func NewConn(dail DailFunc, newRespMessage MessageNew) *Conn {
	conn := &Conn{dail: dail, stat: statDisConnected, newRespMessage: newRespMessage}
	conn.reset()
	return conn
}

func (c *Conn) BindClient(client Client) {
	c.clients.Store(client.Cid(), client)
}

func (c *Conn) UnBindClient(client Client) {
	c.clients.Delete(client.Cid())
}

func (c *Conn) reset() (err error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if atomic.LoadInt32(&c.stat) == statConnected {
		return nil
	}
	for i := 0; i < 3; i++ {
		c.conn, err = c.dail()
		if err != nil {
			continue
		}
		atomic.StoreInt32(&c.stat, statConnected)
		break
	}
	if atomic.LoadInt32(&c.stat) != statConnected {
		return
	}
	go c.startReadMessage()
	return nil
}

func (c *Conn) sendErrorToClients(err error) {
	c.clients.Range(func(key, value interface{}) bool {
		value.(Client).OnError(err)
		return true
	})
}

func (c *Conn) RangeClients(f func(cid string, cli Client) bool) {
	c.clients.Range(func(key, value interface{}) bool {
		return f(key.(string), value.(Client))
	})
}

func (c *Conn) startReadMessage() {
	defer c.conn.Close()
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			atomic.StoreInt32(&c.stat, statDisConnected)
			c.sendErrorToClients(err)
			return
		}
		smsg, err := c.newRespMessage(msg)
		if err != nil {
			atomic.StoreInt32(&c.stat, statDisConnected)
			c.pool.handlerError(err)
			return
		}
		client, _ := c.clients.Load(smsg.Cid())
		if client != nil {
			client.(Client).OnMessage(smsg)
		} else {
			c.pool.handlerError(fmt.Errorf("receive msg from:unknow client:msg:%v", smsg))
		}
	}
}

func (c *Conn) Write(msg []byte) error {
	if atomic.LoadInt32(&c.stat) == statDisConnected {
		if err := c.reset(); err != nil {
			return err
		}
	}
	c.lock.Lock()
	err := c.conn.WriteMessage(websocket.TextMessage, msg)
	c.lock.Unlock()
	return err
}

func (c *Conn) Close() {
	c.conn.Close()
}

//创建鉴权url  apikey 即 hmac username
func assembleAuthUrl(hosturl string, apiKey, apiSecret string) string {
	ul, err := url.Parse(hosturl)
	if err != nil {
		fmt.Println(err)
	}
	//签名时间
	date := time.Now().UTC().Format(time.RFC1123)
	host := ul.Host
	//date = "Tue, 28 May 2019 09:10:42 MST"
	//参与签名的字段 host ,date, request-line
	signString := []string{"host: " + host, "date: " + date, "GET " + ul.Path + " HTTP/1.1"}
	//拼接签名字符串
	sgin := strings.Join(signString, "\n")
	//签名结果
	sha := HmacWithShaTobase64(sgin, apiSecret)
	//构建请求参数 此时不需要urlencoding
	authUrl := fmt.Sprintf("api_key=\"%s\",algorithm=\"%s\",headers=\"%s\",signature=\"%s\"", apiKey,
		"hmac-sha256", "host date request-line", sha)
	//将请求参数使用base64编码
	authorization := base64.StdEncoding.EncodeToString([]byte(authUrl))
	v := url.Values{}
	v.Add("host", host)
	v.Add("date", date)
	v.Add("authorization", authorization)
	v.Add("stream_mode", "multiplex")
	callurl := hosturl + "?" + v.Encode()
	return callurl
}

func HmacWithShaTobase64(data, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	encodeData := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(encodeData)
}
