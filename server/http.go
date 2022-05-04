package server

import (
	"github.com/gin-gonic/gin"
	ws "github.com/gorilla/websocket"
	"github.com/xfyun/webgate-aipaas/common"
	"github.com/xfyun/webgate-aipaas/conf"
	"github.com/xfyun/webgate-aipaas/schemas"
	"net/http"
	"strings"
	"sync"
	"time"
)

func IsWebsocket(ctx *gin.Context) bool {
	return ctx.Request.Method == "GET" && ws.IsWebSocketUpgrade(ctx.Request) &&
		ctx.GetHeader("Sec-Websocket-Key") != ""
}

func GetHostFromCtx(ctx *gin.Context) string {
	h := ""
	if host := ctx.GetHeader("X-Forwarded-Host"); host != "" {
		h = host
	} else {
		h = ctx.Request.Host
	}
	kvs := strings.Split(h, ":") // host 可能带端口号
	if len(kvs) > 0 {
		return kvs[0]
	}
	return h

}

var (
	wsUpgrader = ws.Upgrader{
		HandshakeTimeout: 5 * time.Second,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

var notFoundMessage = gin.H{"message": "not found"}

// entrance ws
func HandlerWs(ctx *gin.Context) {
	route := ctx.Request.URL.Path

	host := GetHostFromCtx(ctx)
	ctx.Set(CtxKeyHost, host)
	var sc *schemas.AISchema
	cloudId := ctx.GetString(KeyCloudId)

	sc = schemas.GetCompanionSchema(route, cloudId)
	if sc == nil {
		serviceId, _ := ctx.GetQuery("serviceId")
		if serviceId == "" {
			routeSbs := strings.Split(route, "/")
			if len(routeSbs) > 0 {
				serviceId = routeSbs[len(routeSbs)-1]
			}
		}
		// 优先通过serviceId 取schema
		if serviceId != "" {
			sc = schemas.GetSchemaByServiceId(serviceId, cloudId)
		}
		if sc == nil {
			sc = schemas.GetSchema(host, route)
		}
	}

	logger := common.GetLoggerInstance()

	if sc == nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, notFoundMessage)
		logger.Errorw("not found", "host", host, "path", route, "clientIp", ctx.ClientIP(), "app_id", ctx.GetHeader("X-Consumer-Username"))
		return
	}
	conn, err := wsUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		logger.Errorf("client(%s) request, upgrade protocol from http to websocket failed :%s", ctx.ClientIP(), err.Error())
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	ctx.Set(CtxKeyClientIp, ctx.ClientIP())
	// 获取kongIP
	ctx.Set(CtxKeyKongIp, ctx.Request.RemoteAddr)
	cfg := conf.GetConfInstance()

	streamMode, _ := ctx.GetQuery("stream_mode")
	if streamMode == "multiplex" {
		ms := NewMultipleSession()
		defer ms.CloseSession()
		ms.Do(ctx, sc, cfg, logger, conn)
		ctx.Abort()
		return
	}

	lock := sync.Mutex{}
	sess := NewWsSession(ctx, sc, cfg, logger, conn, &lock)
	defer sess.CloseSession()
	Handle(sess)
	ctx.Abort()
}

func CheckAppId(ctx *gin.Context) {
	appid := ctx.GetHeader("X-Consumer-Username")
	if appid == "" { // 没有获取到appid 直接放过
		return
	}
	domain := GetHostFromCtx(ctx)
	conf := conf.GetConfInstance()
	cloudId, ok := schemas.CheckAppIdAndCloudId(appid, domain, conf.Server.AppIdWhiteList...)
	if !ok {
		ctx.AbortWithStatusJSON(403, gin.H{
			"message": "you are not allowed to access this service",
		})
		return
	}
	ctx.Set(KeyCloudId, cloudId)
	ctx.Next()
}

func HandlerHttp(ctx *gin.Context) {
	if ctx.ContentType() != "application/json" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "content-type must be application/json"})
		return
	}
	//c:=conf.GetConfInstance()
	route := ctx.Request.URL.Path

	host := GetHostFromCtx(ctx)
	ctx.Set(CtxKeyHost, host)
	cloudId := ctx.GetString(KeyCloudId)
	var sc *schemas.AISchema

	sc = schemas.GetCompanionSchema(route, cloudId)
	if sc == nil {
		serviceId, _ := ctx.GetQuery("serviceId")
		if serviceId == "" {
			routeSbs := strings.Split(route, "/")
			if len(routeSbs) > 0 {
				serviceId = routeSbs[len(routeSbs)-1]
			}
		}
		// 优先通过serviceId 取schema
		if serviceId != "" {
			sc = schemas.GetSchemaByServiceId(serviceId, cloudId)
		}
		if sc == nil {
			sc = schemas.GetSchema(host, route)
		}
	}

	logger := common.GetLoggerInstance()

	if sc == nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, notFoundMessage)
		logger.Errorw("not found", "host", host, "path", route, "clientIp", ctx.ClientIP(), "app_id", ctx.GetHeader("X-Consumer-Username"))
		return
	}

	ctx.Set(CtxKeyClientIp, ctx.ClientIP())
	// 获取kongIP
	ctx.Set(CtxKeyKongIp, ctx.Request.RemoteAddr)
	cfg := conf.GetConfInstance()

	sess := NewHttpSession(ctx, sc, cfg, logger, "")

	sess.StartSonar()
	defer sess.CloseSession()
	err := HandleOnce(sess)
	if err != nil {
		httpCode := http.StatusInternalServerError
		code := int(ErrorServerError)
		message := "unexpected  error"
		switch err.(type) {
		case *HttpError:
			e := err.(*HttpError)
			httpCode = e.HttpCode
			code = int(e.Code)
			message = e.Message
		case *SendBizError:
			e := err.(*SendBizError)
			httpCode = http.StatusBadGateway
			code = int(e.Code)
			message = e.Message
		default:
			message = err.Error()
		}

		sess.SpanTagErr(err.Error())
		sess.SetError(code)
		refCode, ok := CodeMapping(sc.Meta.GetCodeMap(), code)
		if ok {
			httpCode = refCode
		}

		//sess.SpanTagString("ret",strconv.Itoa(code))
		ctx.AbortWithStatusJSON(httpCode, NewHttpErrorResp(sess.Sid, code, message))
		return
	}

	ctx.Abort()
}
