package aipaas_multiplex

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type clienttest struct {
	cid     string
	status  int
	conn    *Conn
	results chan Message
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
	f, err := os.Open(`/Users/sjliu/go/src/git.xfyun.cn/AIaaS/webgate-aipaas/aclient_example/audio/fa178_spk1_iat0007b857@dx174965b19f67389822.wav`)
	if err != nil {
		panic(err)
	}
	buf := make([]byte, 25600)
	seq := 0
	for {

		n, err := f.Read(buf)
		if err != nil {
			if err == io.EOF {
				c.status = 2
			} else {
				panic(err)
			}
		}

		req := map[string]interface{}{
			"header": map[string]interface{}{
				"app_id": "kjsheng2222",
				"cid":    c.cid,
				"status": c.status,
			},
			"parameter": map[string]interface{}{
				"passivefeaonline": map[string]interface{}{
					"func": "clusteringAudio", "groupId": "END_G999", "clusteringAudioRes": map[string]interface{}{"encoding": "utf8", "compress": "raw", "format": "json"},
				},
			},
			"payload": map[string]interface{}{
				"resource": map[string]interface{}{
					"status": c.status, "encoding": "raw", "seq": seq, "sample_rate": 16000, "channels": 1, "bit_depth": 16,
					"audio": base64.StdEncoding.EncodeToString(buf[:n]),
				},
			},
		}

		msg, _ := json.Marshal(req)
		if err := c.conn.Write(msg); err != nil {
			fmt.Println(err)
		}
		seq++
		if c.status == 0 {
			c.status = 1
		}
		if c.status == 2 {
			fmt.Println("send all the data", "cid", c.cid)
			return
		}
		time.Sleep(0 * time.Millisecond)
	}
}

var cidseed int64 = 0

func genCid() string {
	return strconv.Itoa(int(atomic.AddInt64(&cidseed, 1)))
}

func TestNewPool(t *testing.T) {
	dail := websocket.Dialer{}
	p := NewPool(func() (*websocket.Conn, error) {
		conn, _, err := dail.Dial("ws://172.31.103.99:8885/v1/private/passivefeaonline?stream_mode=multiplex", nil)
		return conn, err
	}, func(data []byte) (Message, error) {
		var i messageMap
		err := json.Unmarshal(data, &i)
		return i, err
	}, 5, func(err error) {
		fmt.Println("handler error", err)
	})

	conn, err := p.GetConn()
	if err != nil {
		panic(err)
	}
	wg := sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			timeout := 5 * time.Second
			timer := time.NewTimer(timeout)
			client := &clienttest{
				cid:     genCid(),
				conn:    conn,
				results: make(chan Message, 10),
			}
			conn.BindClient(client)
			defer conn.UnBindClient(client)
			go client.start()
			for {
				select {
				case r := <-client.results:
					fmt.Println("get result:", r)
					if r.(messageMap).Status() == 2 {
						fmt.Println("receive all the data", ",cid=", client.cid)
						return
					}
					timer.Reset(timeout)
				case <-timer.C:
					fmt.Println("context deadline exceed")
					return
				}
			}

		}()
	}

	wg.Wait()

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

/*
 {"header": {"app_id": "kjsheng2222", "status": 2, "cid": "x"}, "parameter": {"passivefeaonline": {"func": "clusteringAudio", "groupId": "END_G999", "clusteringAudioRes": {"encoding": "utf8", "compress": "raw", "format": "json"}}}, "payload": {"resource": {"audio": "", "status": 2, "encoding": "raw", "seq": 1, "sample_rate": 16000, "channels": 1, "bit_depth": 16}}}
 {"header":{"app_id":"4cc5779a","cid":"1","status":2},"parameter":{"passivefeaonline":{"clusteringAudioRes":{"compress":"raw","encoding":"utf8","format":"json"},"func":"clusteringAudio","groupId":"END_G999"}},"payload":{"resource":{"audio":"","bit_depth":16,"channels":1,"encoding":"raw","sample_rate":16000,"seq":1,"status":2}}}

*/



