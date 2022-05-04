package schemas

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"git.iflytek.com/AIaaS/jsonschema"
	"github.com/go-datastructures/hashmap/fastinteger"
	"io/ioutil"
	"testing"
)

func TestElement_Get(t *testing.T) {
	var e = &JsonElement{}
	err := json.Unmarshal([]byte(sc), e)
	if err != nil {
		panic(err)
	}

	fmt.Println(md5sum([]byte("")))
	fmt.Println(e.Get("properties").Get("payload").Get("properties").Get("output").Get("properties").Get("encoding").Get("type").GetAsString())
	fastinteger.New(1)

}

func md5sum(data []byte) string {
	return fmt.Sprintf("%x", md5.Sum(data))
}

func TestSc(t *testing.T) {
	schema := jsonschema.Schema{}
	data, err := ioutil.ReadFile(`/Users/sjliu/go/src/git.xfyun.cn/AIaaS/webgate-aipaas/schemas/script.json`)
	if err != nil {
		panic(err)
	}
	json.Unmarshal(data, &schema)
	fmt.Println(schema.Validate(map[string]interface{}{
		"dsf": "dsf",
	}))
}

var sc = `
{
    "type": "object",
    "properties": {
      "payload": {
        "type": "object",
        "properties": {
          "output": {
            "type": "object",
            "properties": {
              "encoding": {
                "type": "string",
                "enum": [
                  "utf8",
                  "gb2312"
                ]
              },
              "compress": {
                "type": "string",
                "enum": [
                  "raw",
                  "gzip"
                ]
              },
              "format": {
                "type": "string",
                "enum": [
                  "plain",
                  "json",
                  "xml"
                ]
              },
              "status": {
                "type": "integer",
                "enum": [
                  0,
                  1,
                  2
                ]
              },
              "seq": {
                "type": "integer",
                "minimum": 0,
                "maximum": 9999999
              },
              "text": {
                "type": "string",
                "minLength": 0,
                "maxLength": 1000000
              }
            }
          }
        }
      }
    }
  }

`

type teststruct struct {
	a int
	c int
	d byte
	b byte
	e byte
	f byte
	g uint32
}

type nodeData = int

type node struct {
	next *node
	pre  *node
	data nodeData
}

// f->a -> b
//  f<-b
type ring struct {
	front *node
	back  *node
	size  int
}

func (r *ring) pushBack(v nodeData) {
	n := &node{data: v}
	bre := r.back.pre
	bre.next = n
	n.next = r.back
	n.pre = bre
	r.back.pre = n
	r.size++
}

func (r *ring) remove(n *node) {
	n.pre.next = n.next
	n.next.pre = n.pre
	r.size--
}

func (r *ring) goN(node *node, n int) *node {
	if r.size == 0 {
		return nil
	}
	i := 0
	for i < n {
		node = node.next
		if node == r.front || node == r.back {
			continue
		}
		i++
	}
	return node
}

func (r *ring) find(data nodeData) *node {
	n := r.front.next
	for n != r.back {
		if n.data == data {
			return n
		}
		n = n.next
	}
	return nil
}

func newRing(arr []nodeData) *ring {
	r := &ring{}
	r.front = new(node)
	r.back = new(node)
	r.front.next = r.back
	r.back.pre = r.front
	r.front.pre = r.back
	r.back.next = r.front
	for _, i := range arr {
		r.pushBack(i)
	}
	return r
}

func Test_tst(t *testing.T) {
	r := newRing([]nodeData{1, 2, 3, 4, 5, 6, 7})
	start := 5
	n := r.find(start)
	for i := 0; i < 7; i++ {
		node := r.goN(n, 8)
		n = node
		r.remove(node)
		fmt.Println(node.data)
	}
}
