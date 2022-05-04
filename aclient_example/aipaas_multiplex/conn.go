package aipaas_multiplex

import (
	"fmt"
	"github.com/gorilla/websocket"
	"sync"
	"sync/atomic"
)

type Pool struct {
	conn           []*Conn
	size           int // 连接池大小
	idx            int64
	newRespMessage MessageNew
	dail           DailFunc
	errHandler ErrorHandler
}

type ErrorHandler func(err error)

func NewPool(dail DailFunc, newRespMessage MessageNew, poolSize int,eh ErrorHandler) *Pool {
	if poolSize <= 0 {
		poolSize = 5
	}
	p := &Pool{
		conn:           nil,
		size:           poolSize,
		idx:            0,
		newRespMessage: newRespMessage,
		dail:           dail,
		errHandler: eh,
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

func (p *Pool)handlerError(err error){
	if p.errHandler != nil{
		p.errHandler(err)
	}
}

func (p *Pool) init() {
	for i := 0; i < p.size; i++ {
		c:=NewConn(p.dail, p.newRespMessage)
		c.pool = p
		p.conn = append(p.conn, c)
	}
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
	pool *Pool
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
	if  atomic.LoadInt32(&c.stat) == statConnected {
		return nil
	}
	for i := 0; i < 3; i++ {
		c.conn, err = c.dail()
		if err != nil {
			continue
		}
		atomic.StoreInt32(&c.stat,statConnected)
		break
	}
	if atomic.LoadInt32(&c.stat)!= statConnected {
		return
	}
	go c.startReadMessage()
	return nil
}

func (c *Conn) startReadMessage() {
	defer c.conn.Close()
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			atomic.StoreInt32(&c.stat ,statDisConnected)
			return
		}
		smsg, err := c.newRespMessage(msg)
		if err != nil {
			atomic.StoreInt32(&c.stat ,statDisConnected)
			c.pool.handlerError(err)
			return
		}
		client, _ := c.clients.Load(smsg.Cid())
		if client != nil {
			client.(Client).OnMessage(smsg)
		}else{
			c.pool.handlerError(fmt.Errorf("receive msg from:unknow client:msg:%v",smsg))
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

// docker run -ti --rm
