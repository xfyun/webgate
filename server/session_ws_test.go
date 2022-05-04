package server

import (
	"expvar"
	"fmt"
	"github.com/google/btree"
	"github.com/zserge/metric"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"testing"
	"time"
)

type treeMap struct {
	tree *btree.BTree
}

func newTreeMap() *treeMap {
	return &treeMap{
		tree: btree.New(2),
	}
}

func (t *treeMap) Set(key string, val interface{}) {
	t.tree.ReplaceOrInsert(&elem{
		key: key,
		val: val,
	})
}

func (t *treeMap) Get(key string) interface{} {
	d := t.tree.Get(&elem{key: key}).(*elem)
	return d.val
}

type item string

func (i item) Less(than btree.Item) bool {
	return string(i) < string(than.(item))
}

type elem struct {
	key string
	val interface{}
}

func (e *elem) Less(than btree.Item) bool {
	t := than.(*elem)
	return e.key < t.key
}

func TestTrree(t *testing.T) {
	bt := newTreeMap()
	bt.Set("1", 1)
	bt.Set("1", 3)
	bt.Set("1", 5)
	fmt.Println(bt.Get("1"))
	fmt.Println(bt.tree.Len())
}
func BenchmarkBtree(b *testing.B) {
	bt := newTreeMap()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 30; j++ {
			bt.Set(strconv.Itoa(i), j)
		}
	}

}

func fibrec(n int) int {
	if n <= 1 {
		return n
	}
	return fibrec(n-1) + fibrec(n-2)
}

func TestCheckAppIdMaMetric(t *testing.T) {
	// Fibonacci: how long it takes and how many calls were made
	expvar.Publish("fib:rec:sec", metric.NewHistogram("120s1s", "15m10s", "1h1m"))
	expvar.Publish("fib:rec:count", metric.NewCounter("120s1s", "15m10s", "1h1m"))

	// Random numbers always look nice on graphs
	expvar.Publish("random:gauge", metric.NewGauge("60s1s"))
	expvar.Publish("random:hist", metric.NewHistogram("2m1s", "15m30s", "1h1m"))

	// Some Go internal metrics
	expvar.Publish("go:numgoroutine", metric.NewGauge("2m1s", "15m30s", "1h1m"))
	expvar.Publish("go:numcgocall", metric.NewGauge("2m1s", "15m30s", "1h1m"))
	expvar.Publish("go:alloc", metric.NewGauge("2m1s", "15m30s", "1h1m"))
	expvar.Publish("go:alloctotal", metric.NewGauge("2m1s", "15m30s", "1h1m"))

	go func() {
		for range time.Tick(123 * time.Millisecond) {
			expvar.Get("random:gauge").(metric.Metric).Add(rand.Float64())
			expvar.Get("random:hist").(metric.Metric).Add(rand.Float64() * 100)
		}
	}()
	go func() {
		for range time.Tick(100 * time.Millisecond) {
			m := &runtime.MemStats{}
			runtime.ReadMemStats(m)
			expvar.Get("go:numgoroutine").(metric.Metric).Add(float64(runtime.NumGoroutine()))
			expvar.Get("go:numcgocall").(metric.Metric).Add(float64(runtime.NumCgoCall()))
			expvar.Get("go:alloc").(metric.Metric).Add(float64(m.Alloc) / 1000000)
			expvar.Get("go:alloctotal").(metric.Metric).Add(float64(m.TotalAlloc) / 1000000)
		}
	}()
	http.Handle("/debug/metrics", metric.Handler(metric.Exposed))
	http.HandleFunc("/fibrec", func(w http.ResponseWriter, r *http.Request) {
		expvar.Get("fib:rec:count").(metric.Metric).Add(1)
		start := time.Now()
		fmt.Fprintf(w, "%d", fibrec(40))
		expvar.Get("fib:rec:sec").(metric.Metric).Add(float64(time.Now().Sub(start)) / float64(time.Second))
	})
	fmt.Println("Listen on :8000")
	http.ListenAndServe(":8000", nil)
}

