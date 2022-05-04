package server

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type Plugin interface {
	Exec(s *WsSession) error
}

//service _ app_id  uid

type PluginsGroup struct {
	service map[string][]Plugin
	appid   map[string][]Plugin
	uid     map[string][]Plugin
}

func (p *PluginsGroup) Set(service, appid, uid string) {
	if service == "" {

	}
}

type Func func()

type worker struct {
	taskChan    chan Func
	lastUseTime time.Time
}

func (w *worker) Run(g *GoRoutinePool) {
	for task := range w.taskChan {
		w.lastUseTime = time.Now()
		task()
		g.lock.Lock()
		g.idleWorkers = append(g.idleWorkers, w)
		g.lock.Unlock()
	}
}

type GoRoutinePool struct {
	idleWorkers []*worker
	lock        sync.Mutex
	shutdown    chan struct{}
	idleTime    time.Duration
}

func NewGoRoutinePool(idle time.Duration) *GoRoutinePool {
	p := &GoRoutinePool{
		idleWorkers: make([]*worker, 0, 100),
		lock:        sync.Mutex{},
		shutdown:    make(chan struct{}),
		idleTime:    idle,
	}
	if p.idleTime == 0 {
		p.idleTime = 20 * time.Second
	}
	go p.CheckIdle()
	return p
}

func (g *GoRoutinePool) getWorker() *worker {
	g.lock.Lock()
	//defer g.lock.Unlock()
	n := len(g.idleWorkers) - 1
	if n >= 0 {
		worker := g.idleWorkers[n]
		g.idleWorkers = g.idleWorkers[:n]
		g.lock.Unlock()
		return worker
	}
	g.lock.Unlock()
	worker := &worker{
		taskChan:    make(chan Func),
		lastUseTime: time.Now(),
	}
	go worker.Run(g)
	return worker
}

func (g *GoRoutinePool) Execute(task Func) {
	g.getWorker().taskChan <- task
}

func (g *GoRoutinePool) CheckIdle() {
	for {
		select {
		case <-g.shutdown:
			return
		case <-time.After(30 * time.Second):
		}
		g.lock.Lock()
		inActiveCounts := 0
		for _, worker := range g.idleWorkers {
			if time.Since(worker.lastUseTime) > g.idleTime {
				inActiveCounts++
			}
		}
		for i := len(g.idleWorkers) - inActiveCounts; i < len(g.idleWorkers); i++ {
			close(g.idleWorkers[i].taskChan)
			g.idleWorkers[i] = nil
		}
		g.idleWorkers = g.idleWorkers[:len(g.idleWorkers)-inActiveCounts]
		g.lock.Unlock()

	}
}

func (g *GoRoutinePool) ShutDown() {
	close(g.shutdown)
}

type PoolGroup struct {
	pools []*GoRoutinePool
	size  int
	seed  int64
}

func NewPoolGroup(size int, idle time.Duration) *PoolGroup {
	if size == 0 {
		size = runtime.NumCPU() / 2
	}
	if size == 0 {
		size = 2
	}
	pools := make([]*GoRoutinePool, size)
	for i := 0; i < len(pools); i++ {
		pools[i] = NewGoRoutinePool(idle)
	}

	return &PoolGroup{
		pools: pools,
		size:  size,
		seed:  0,
	}
}

func (p *PoolGroup) Execute(task Func) {
	idx := int(atomic.AddInt64(&p.seed, 1)) % p.size
	p.pools[idx].Execute(task)
}

func (p *PoolGroup) ShutDown() {
	for _, v := range p.pools {
		v.ShutDown()
	}
}
