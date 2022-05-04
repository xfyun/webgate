package main

import (
	"fmt"
	"github.com/xfyun/sonar"
	"github.com/xfyun/webgate-aipaas/common"
	"github.com/xfyun/webgate-aipaas/conf"
	"github.com/xfyun/webgate-aipaas/server"
	//"git.xfyun.cn/AIaaS/webgate-aipaas/sver"
	"github.com/DeanThompson/ginpprof"
	"github.com/gin-gonic/gin"
	ws "github.com/gorilla/websocket"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"
)

var wsUpgrader ws.Upgrader

func main() {
	Run()
}

func Run() {
	//初始化配置
	conf.InitConf()
	confInst := conf.GetConfInstance()
	stopWatiGroup := sync.WaitGroup{}

	server.InitSessionGroup(confInst.Session.ScanInterver)
	common.InitSidGenerator(confInst.Server.Host, confInst.Server.Port, confInst.Xsf.Location)
	//初始化sid生成器
	//启动XSF服务端
	server.StartXsfServer("xsf.toml")
	// 初始化XSF客户端
	err := server.InitXsfClient("xsf.toml")
	if err != nil {
		fmt.Printf("InitXsfClient is error:%s\n", err)
		common.GetLoggerInstance().Errorf("InitXsfClient is error:%s", err)
		return
	}
	sonar.Logger = common.GetLoggerInstance()

	//启动路由
	g := initGin()
	ginpprof.Wrapper(g)

	g.Use(Loggers())
	//监控路由
	g.GET("/monitor/:option", server.MonitorHandler)

	g.Use(func(context *gin.Context) {
		stopWatiGroup.Add(1)
		defer stopWatiGroup.Done()
		context.Next()
	})
	//处理请求的handler
	g.Use(server.CheckAppId)

	g.Use(handler)

	gin.SetMode(gin.ReleaseMode)
	addr := "0.0.0.0:" + confInst.Server.Port
	ls, err := net.Listen("tcp4", addr)
	if err != nil {
		panic(err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGSTOP)

	srv := http.Server{Addr: addr, Handler: g}
	lsclose := false
	go func() {
		fmt.Println("startServer at:", addr, time.Now().String())
		if err := srv.Serve(ls); err != nil {
			fmt.Println("server exit error:", err, time.Now().String())
			if !lsclose { // lsclose==true 说明是正常关闭导致的错误，如果不是，则是srv.Serve() error，需要退出服务
				panic("server start error:" + err.Error())
			}
		}
	}()

	sig := <-sigChan
	fmt.Println("receive stop signal:", sig, "try to stop service", time.Now().String())
	// 关闭listener ，拒绝新的请求，已经建立好的连接会继续处理
	lsclose = true
	go func() {
		if err = ls.Close(); err != nil {
			fmt.Println("close listener error:", err)
		}
	}()

	// 等待，一直到所有请求全部处理完毕。退出服务。
	stopWatiGroup.Wait()
	fmt.Println("webgate exit 0", time.Now().String())
	time.Sleep(100 * time.Millisecond)
	os.Exit(0)

}

func handler(ctx *gin.Context) {
	if server.IsWebsocket(ctx) {
		server.HandlerWs(ctx)
	} else {
		server.HandlerHttp(ctx)
	}
}

func initGin() *gin.Engine {
	confInst := conf.GetConfInstance()
	g := gin.New()
	if confInst.Server.Mode == "debug" {
		gin.SetMode(gin.DebugMode)
		fmt.Println("start mode = debug")
	} else {
		gin.SetMode(gin.ReleaseMode)
		g.Use(recoveryHandler)
		fmt.Println("start mode = release use recovery")
	}
	return g
}

func recoveryHandler(ctx *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			common.GetLoggerInstance().Errorw("recovery from panic", "error", err, "stack", string(debug.Stack()))

			ctx.AbortWithStatus(http.StatusBadGateway) //
		}
	}()
	ctx.Next()
}

func Loggers() gin.HandlerFunc {
	return func(context *gin.Context) {
		start := time.Now()
		context.Next()
		end := time.Now()
		common.GetLoggerInstance().Infow("finish session", "clientIp", context.ClientIP(),
			"sid", context.GetString("sid"),
			"app_id", context.GetString("appid"),
			"uid", context.GetString("uid"),
			"host", context.GetString("host"),
			"kong_ip", context.GetString("kong_ip"),
			"path", context.Request.URL.Path,
			"cost", end.Sub(start),
			"cloudId", context.GetString(server.KeyCloudId),
		)
	}
}
