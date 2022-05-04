package common

import (
	"fmt"
	"sync"
)

type Queue interface {
	Add(data interface{})
	Remove() interface{}
	Top() interface{}
	Size() int
}

//优先队列
type PriorityQueue struct {
	lesss      func(data []interface{}, i, j int) bool
	ExpandSize int
	nodes      []interface{}
	N          int
}

func NewPrioQueue(lesss func(data []interface{}, i, j int) bool) Queue {
	p := &PriorityQueue{N: 1, ExpandSize: 10,nodes:make([]interface{},0,10)} //0 数组0号位置不存放数据
	p.lesss = lesss
	return p
}
//上浮
func (p *PriorityQueue) swim(k int) {
	for k > 1 && p.less(k, k/2) {
		p.exch(k/2, k)
		k /= 2
	}
}
//下沉
func (p *PriorityQueue) sink(k int) {
	if p.N == 1 {
		return
	}
	for 2*k < p.N {
		j := 2 * k
		if j+1 < p.N && p.less(j+1, j) {
			j = j + 1
		}

		if p.less(j, k) {
			p.exch(k, j)
			k = j
		} else {
			break
		}
	}
}

func (p *PriorityQueue) Add(data interface{}) {

	if len(p.nodes) <= p.N {
		apd := make([]interface{},len(p.nodes)+5)
		p.nodes = append(p.nodes, apd...)
	}
	p.nodes[p.N] = data
	p.swim(p.N)
	p.N++
}

func (p *PriorityQueue) Remove() interface{} {
	if p.N == 1 {
		return nil
	}
	data := p.nodes[1]
	p.nodes[1] = p.nodes[p.N-1]
	p.N--
	p.sink(1)
	return data
}

func (p *PriorityQueue) Top() interface{} {
	if p.N <= 1 {
		return nil
	}
	return p.nodes[1]
}

func (p *PriorityQueue) Size() int {
	return p.N - 1
}

func (p *PriorityQueue) less(a, b int) bool {
	return p.lesss(p.nodes, a, b)
}



func (p *PriorityQueue) show() {

	for i := 1; i < len(p.nodes); i++ {
		fmt.Print(p.nodes[i], " ")
	}

	fmt.Println()
}

func (p *PriorityQueue) exch(i, j int) {
	p.nodes[i], p.nodes[j] = p.nodes[j], p.nodes[i]
}

func NewBlockingPriorityQueue(lesss func(data []interface{}, i, j int) bool) Queue {
	pq := &BlockingPriorityQueue{
		pq:   NewPrioQueue(lesss),
		cond: sync.NewCond(&sync.Mutex{}),
	}
	return pq
}
//阻塞的优先队列，remove 时如果队列为空，则阻塞直到有元素进入队列
//并发安全
type BlockingPriorityQueue struct {
	cond    *sync.Cond
	pq      Queue
	hasdata bool
}

func (c *BlockingPriorityQueue) Add(f interface{}) {
	c.cond.L.Lock()
	c.pq.Add(f)
	c.cond.L.Unlock()
	c.cond.Signal()
}

func (c *BlockingPriorityQueue) Remove() interface{} {
	c.cond.L.Lock()
	for c.Size() == 0 {
		c.cond.Wait()
	}
	d := c.pq.Remove()
	c.cond.L.Unlock()
	return d
}

func (c *BlockingPriorityQueue) Top() interface{} {
	c.cond.L.Lock()
	d:=c.Top()
	c.cond.L.Unlock()
	return d
}


func (c *BlockingPriorityQueue) Size() int {
	return c.pq.Size()
}
