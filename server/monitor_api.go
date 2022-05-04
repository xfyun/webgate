package server

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"runtime"
)

const (
	RunTime = "info"
	Health  = "health"
	GC      = "gc"
	Conn    = "conn"
	Version = "1.1.0_12"
)

type MonitorResp struct {
	//Code int `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

var (
	NotFoundResp = &MonitorResp{"Not Found", nil}
)

var handlerMap = map[string]func(ctx *gin.Context){
	RunTime: handleRuntimeInfo,
	Health:  handleHealthCheck,
	GC:      handlerGc,
	//Conn:handleConnectionInfo,
	"version": versionHandler,
}

func versionHandler(ctx *gin.Context) {
	ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
		"version": Version,
	})
}

func MonitorHandler(ctx *gin.Context) {
	option, _ := ctx.Params.Get("option")
	handler := handlerMap[option]
	if handler == nil {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"opts": []string{RunTime, Health, GC, Conn},
		})
		return
	}
	handler(ctx)
}

func handleRuntimeInfo(c *gin.Context) {
	countMap, total := getActiveClients()
	resp := &MonitorResp{
		//Code:http.StatusOK,
		Message: "ok",
		Data: map[string]interface{}{
			"clients": map[string]interface{}{
				"total":  total,
				"detail": countMap,
			},
			"app_ids":        getAppids(),
			"go_routine_num": runtime.NumGoroutine(),
			"num_cpu":        runtime.NumCPU(),
			"num_cgo_call":   runtime.NumCgoCall(),
		},
	}
	c.AbortWithStatusJSON(http.StatusOK, resp)

}

type ConnInfo struct {
	Appid   string `json:"appid"`
	ConnNum int    `json:"conn_num"`
}

//specialize s-works tarmac sl7
//获取appid 连接信息
func handleConnectionInfo(ctx *gin.Context) {
	//ctx.AbortWithStatusJSON(http.StatusOK,gin.H{
	//	""
	//})
}

func handleHealthCheck(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusOK, nil)
}

func handlerGc(c *gin.Context) {
	runtime.GC()
	c.AbortWithStatusJSON(http.StatusOK, map[string]interface{}{
		"message": "gc success",
	})
}

func getActiveClients() (map[string]int, int) {
	total := 0
	var count = map[string]int{}
	aiSessGroup.Range(func(sid string, sess *WsSession) bool {
		total++
		count[sess.Sub]++

		return true
	})
	return count, total
}

func getAppids() map[string]int {
	m := make(map[string]int)
	aiSessGroup.Range(func(sid string, sess *WsSession) bool {
		m[sess.AppId]++
		return true
	})
	return m

}

//杀掉appid上过多的连接
func HandlerKill(ctx *gin.Context) {
	//appid:=ctx.Param("appid")
	//reverse:=ctx.Param("remain")
	//num,err:=strconv.Atoi(reverse)
	////剩下
	//if err !=nil{
	//	ctx.AbortWithStatusJSON(http.StatusBadRequest,gin.H{
	//		"message":"remain num:"+reverse+" is not a vaild number",
	//	})
	//	return
	//}
	////ConnTransManager.Kill(appid,num)
	//ctx.AbortWithStatusJSON(http.StatusBadRequest,gin.H{
	//	"message":fmt.Sprintf("success kill connnection of %s,remain connection:%d",appid,ConnTransManager.GetCount(appid)),
	//})

}

//获取appid的当前接数
func HandlerGetConncection(ctx *gin.Context) {
	//appid:=ctx.Param("appid")
	//count:=ConnTransManager.GetCount(appid)
	//ctx.AbortWithStatusJSON(http.StatusOK,gin.H{
	//	"appid":appid,
	//	"active_conns":count,
	//})
}

//
