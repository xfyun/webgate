package server

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/xfyun/sonar"
	"github.com/xfyun/webgate-aipaas/common"
	"github.com/xfyun/webgate-aipaas/conf"
	"github.com/xfyun/webgate-aipaas/pb"
	"github.com/xfyun/webgate-aipaas/schemas"
	xsf "github.com/xfyun/xsf/client"
	"github.com/xfyun/xsf/utils"
	"io"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type WsSession struct {
	Sid            string
	AppId          string
	Uid            string
	ServiceId      string
	RawServiceId   string
	MagicServiceId string
	Conn           *websocket.Conn
	buffer         []byte
	Sub            string
	lock           *sync.Mutex
	ReadTimeout    int
	SessonTimeout  int
	CallService    string
	CallType       int
	Ctx            *gin.Context
	schema         *schemas.AISchema
	//session        string
	sessionContext *schemas.Context
	sonar          *sonar.MetricData
	span           *utils.Span
	startTime      time.Time
	Status         int32
	errorCode      int
	logger         *xsf.Logger
	conf           *conf.Config
	firstout       int
	lastout        int
	firstin        int
	lastin         int
	CloudId        string
	index          int
	handlers       HandlerChain
	targetSub      string
	cid            string
	lastActiveTime time.Time
	sessionAddress string
	sessionCall    bool
	routerInfo     string
}

func (s *WsSession) Run() {
	n := len(s.handlers)
	for s.index < n {
		s.handlers[s.index](s)
		s.index++
	}
}

func (s *WsSession) Next() {
	s.index++
	s.Run()
}

func (s *WsSession) Abort() {
	s.index = len(s.handlers)
}

const (
	CtxKeyClientIp = "client_ip"
	CtxKeyKongIp   = "kong_ip"
	CtxKeyHost     = "host"
)

func NewWsSession(ctx *gin.Context, schema *schemas.AISchema, conf *conf.Config, logger *xsf.Logger, conn *websocket.Conn, lock *sync.Mutex) *WsSession {
	meta := schema.Meta
	sid := common.NewSid(meta.GetSub())
	//sess := sessionPool.GetSession()
	sess := &WsSession{}
	sess.Sid = sid
	sess.Sub = meta.GetSub()
	sess.Conn = conn
	sess.buffer = bytePool.Get()
	sess.SessonTimeout = conf.Session.SessionTimeout
	sess.ReadTimeout = conf.Session.TimeoutInterver
	calls := conf.Server.MockService
	sess.lock = lock
	if calls != "" {
		sess.CallService = calls
	} else {
		sess.CallService = meta.GetCallService()
	}
	sess.CallType = meta.GetCallType()
	sess.Ctx = ctx
	sess.Status = 0
	sess.startTime = time.Now()
	sess.schema = schema
	sess.cid = ""
	if sess.sessionContext != nil {
		sess.sessionContext.SeqNo = 1
		sess.sessionContext.Header = nil
		sess.sessionContext.Session = nil
		sess.sessionContext.Sync = false
	} else {
		sess.sessionContext = &schemas.Context{SeqNo: 1, Sync: false, IsStream: true}
	}

	sess.logger = logger
	sess.conf = conf
	sess.sonar = nil
	sess.errorCode = 0
	sess.ServiceId = meta.GetServiceId()
	sess.RawServiceId = sess.ServiceId
	sess.MagicServiceId = ""
	if t := meta.GetSessonTimeout(); t > 0 {
		sess.SessonTimeout = t
	}
	if t := meta.GetReadTimeout(); t > 0 {
		sess.ReadTimeout = t
	}

	if sess.ReadTimeout == 0 {
		sess.ReadTimeout = 15
	}

	if sess.SessonTimeout == 0 {
		sess.SessonTimeout = 180
	}
	sess.firstout = 0
	sess.lastout = 0
	sess.firstin = 0
	sess.lastin = 0
	sess.sessionAddress = ""
	sess.routerInfo = ""
	aiSessGroup.Set(sid, sess)

	return sess
}

// 从websocket 连接中读取数据，重用buffer ，提升性能
func (t *WsSession) readMessage() (int, []byte, error) {
	var r io.Reader
	messageType, r, err := t.Conn.NextReader()
	if err != nil {
		return messageType, nil, err
	}
	ed := 0 // buffer 尾
	for {
		n, err := r.Read(t.buffer[ed:])
		ed += n
		if err != nil {
			if err == io.EOF {
				return messageType, t.buffer[:ed], nil
			}
			return messageType, nil, err
		}
		if ed == len(t.buffer) { // reader 的 长度== 扩容
			old := t.buffer
			t.buffer = make([]byte, len(old)*2) // buffer cap double
			copy(t.buffer, old)
		}
	}
	//b,err:=ioutil.ReadAll(r)
	return messageType, t.buffer[:ed], err
}

// 向websocket 写消息
func (s *WsSession) WriteMessage(message interface{}) {
	data, _ := jsoniter.Marshal(message)
	s.lock.Lock()
	defer s.lock.Unlock()
	s.Conn.WriteMessage(websocket.TextMessage, data)
}

func (s *WsSession) WriteError(code ErrorCode, msg string) {
	// 忽略10101 等错误码，不返回给用户
	for _, c := range s.conf.Server.IgnoreRespCodes {
		if c == int(code) {
			return
		}
	}
	s.WriteMessage(&ErrorResp{
		Header: Header{
			Code:    code,
			Message: msg,
			Sid:     s.Sid,
			Cid:     s.cid,
		},
		//Code:    code,
		//Message: msg,
		//Sid:     s.Sid,
	})
}

func (s *WsSession) WriteSuccess(payload interface{}, status int, headers map[string]string) {
	rss := NewSuccessResp(s.Sid, payload, status, s.cid)
	for key, val := range s.schema.BuildResponseHeader(headers) {
		rss.SetHeader(key, val)
	}
	s.WriteMessage(rss)
}

func (s *WsSession) WriteSuccessWithGuiderStatus(payload interface{}, status int, guiderStatus string, headers map[string]string) {
	wfStatus := 0
	if guiderStatus != "" {
		wfStatus, _ = strconv.Atoi(guiderStatus)
	}
	//rsp := SuccessResp{
	//	Header: Header{
	//		Code:    0,
	//		Message: "success",
	//		Sid:     s.Sid,
	//		Status:  status,
	//		Cid:     s.cid,
	//	},
	//	Payload:  payload,
	//	WfStatus: wfStatus,
	//}
	rss := NewSuccessResp(s.Sid, payload, status, s.cid)
	rss.WfStatus = wfStatus
	for key, val := range s.schema.BuildResponseHeader(headers) {
		rss.SetHeader(key, val)
	}
	s.WriteMessage(rss)
}

// 写close 帧
func (s *WsSession) WriteClose(reason string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason))

}

func (s *WsSession) formatArgs(kvs []interface{}) []interface{} {

	common := []interface{}{"sid", s.Sid, "app_id", s.AppId, "uid", s.Uid, "serviceId", s.ServiceId, "couldId", s.CloudId}
	args := make([]interface{}, len(kvs)+len(common))
	copy(args, common)
	copy(args[len(common):], kvs)
	return args
}

func (s *WsSession) Errorw(msg string, kvs ...interface{}) {
	//s.logger.Errorw("","")

	s.logger.Errorw(msg, s.formatArgs(kvs)...)
}

func (s *WsSession) Infow(msg string, kvs ...interface{}) {
	if s.conf.Log.Level == "error" { //日志级别为error 时就不用复制args了
		return
	}
	s.logger.Infow(msg, s.formatArgs(kvs)...)
}

func (s *WsSession) Debugw(msg string, kvs ...interface{}) {
	if s.conf.Log.Level == "error" {
		return
	}
	s.logger.Debugw(msg, s.formatArgs(kvs)...)
}

func (s *WsSession) StartSpan() {
	s.span = utils.NewSpan(utils.SrvSpan).Start()
	s.span.WithName("webgate-aipaas").WithTag("sub", s.Sub).WithRetTag("0").WithTag("sid", s.Sid)
	s.span.WithTag("goroutines", strconv.Itoa(runtime.NumGoroutine()))
}

func (s *WsSession) SpanTagString(k, v string) {
	s.span.WithTag(k, v)
}

func (s *WsSession) SpanMeta() string {
	return s.span.Meta()
}

func (s *WsSession) SpanTagErr(err string) {
	s.span.WithErrorTag(err)
}

func (s *WsSession) setError(code int) {
	if s.errorCode == 0 {
		s.errorCode = code
	}
}
func (s *WsSession) SetError(code int) {
	if s.errorCode == 0 {
		s.errorCode = code
	}
}

// 检查session是否超时
func (s *WsSession) Alive() bool {
	if time.Since(s.startTime) > time.Duration(s.SessonTimeout)*time.Second {
		return false
	}
	return true
}

func (s *WsSession) SchemaCheck(o interface{}) error {
	if s.conf.Schema.Enable && s.schema != nil {
		if err := s.schema.Validate(o); err != nil {
			return err
		}
	}
	return nil
}

func Handle(s *WsSession) {
	for {
		s.ResetReadDeadline()
		_, msg, err := s.readMessage()
		if err != nil {
			//if !websocket.IsCloseError(err,websocket.CloseNormalClosure,websocket.CloseNoStatusReceived,websocket.CloseAbnormalClosure){
			s.StartSpan()
			s.SpanTagString("sessionCloseErr", fmt.Sprintf("connection close: err:%s ; cost:%d", err.Error(), time.Since(s.startTime)/time.Second))
			s.SpanTagString("appid", s.AppId)
			s.Errorw("read message error:", "error", err.Error(), "cost", time.Since(s.startTime))
			s.FlushSpan()
			s.WriteClose("time out")
			//}
			return
		}

		if !s.Alive() {
			s.StartSpan()
			s.SpanTagString("sessionCloseErr", fmt.Sprintf("session timeout! used:%d   allow:%d", time.Since(s.startTime)/time.Second, s.SessonTimeout))
			s.SpanTagString("appid", s.AppId)
			s.FlushSpan()
			s.Errorw("session timeout")
			s.WriteError(ErrorCodeSetReadDeadline, "session timeout")
			s.WriteClose("session timeout")
			return
		}

		code, info := s.handleAIMessage(msg)
		if code != 0 {
			s.WriteError(code, info)
			time.Sleep(5 * time.Millisecond)
			s.WriteClose(info)
			s.setError(int(code))

			return
		}
	}
}

func (s *WsSession) StartSonar() {
	if !s.conf.Server.EnableSonar {
		return
	}
	s.sonar = sonar.NewMetricWithNamePort(sonar.TYPE_GAUGE,
		"sps", XsfCallBackAddr,
		s.Sub,
		s.conf.Xsf.ServerPort,
		"vagus_null")

}

//{route_key}___{service_id}
func (s *WsSession) FlushSonar() {
	if s.sonar != nil {
		sub := s.targetSub
		if sub == "" {
			sub = s.Sub
		}

		ent := s.ServiceId
		if s.MagicServiceId != "" {
			ent = s.MagicServiceId
		}
		//if s.routerInfo != ""{
		//	//sub = "category_"+ sub
		//	//ent = s.routerInfo + "___" +ent
		//}else{
		//	//sub = "ase"
		//}

		s.sonar.Tag(sonar.KV{"sid", s.Sid}).
			Tag(sonar.KV{"from", s.conf.Server.From}).
			Tag(sonar.KV{"sub", sub}).
			Tag(sonar.KV{"cluster", s.conf.Server.Cluster}).
			Tag(sonar.KV{"dc", s.conf.Xsf.Location}).
			Tag(sonar.KV{"appid", s.AppId}).
			Tag(sonar.KV{"uid", s.Uid})

		s.sonar.Tag(sonar.KV{"finalEnt", ent})
		s.sonar.Tag(sonar.KV{"routekey", s.routerInfo})
		//metPtr.TagDS("vagus_null").Tag(sonar.KV{"end", "true"})
		s.sonar.TagDS("vagus_null")
		s.SonarTagInt("sret", s.errorCode)
		s.SonarTagInt("ret", s.errorCode)
		s.SonarTagString("end", "true")
		if s.CloudId != "" {
			s.SonarTagString("cloud_id", s.CloudId)
		} else {
			s.SonarTagString("cloud_id", "ai_cloud")
		}

		s.sonar.Flush()

	}
}

func (s *WsSession) SonarTagString(key string, v string) {
	if s.sonar != nil {
		s.sonar.Tag(sonar.KV{
			Key:   key,
			Value: v,
		})
	}
}

func (s *WsSession) SonarTagInt(key string, v int) {
	if s.sonar != nil {
		s.sonar.Tag(sonar.KV{
			Key:   key,
			Value: strconv.Itoa(v),
		})
	}
}

// close 只能被执行一次
func (s *WsSession) CloseSession() {
	s.SonarTagInt("firstout", s.firstout)
	s.SonarTagInt("lastout", s.lastout)
	s.SonarTagInt("firstin", s.firstin)
	s.SonarTagInt("lastin", s.lastin)
	s.FlushSonar()
	aiSessGroup.Delete(s.Sid)
	time.Sleep(1 * time.Second)
	s.Conn.Close()
	bytePool.Put(s.buffer)
	if s.errorCode == 0 && s.Status != 2 {
		s.SendException()
	}
	//sessionPool.PutSession(s)
}

func (s *WsSession) CloseMulitplex() {
	s.SonarTagInt("firstout", s.firstout)
	s.SonarTagInt("lastout", s.lastout)
	s.SonarTagInt("firstin", s.firstin)
	s.SonarTagInt("lastin", s.lastin)

	s.FlushSonar()
	aiSessGroup.Delete(s.Sid)
	//time.Sleep(1 * time.Second)
	//s.Conn.Close()
	bytePool.Put(s.buffer)
	if s.errorCode == 0 && s.Status != 2 {
		s.SendException()
	}
	//sessionPool.PutSession(s)
}

// close 只能被执行一次
func (s *WsSession) CloseHttpSession() {
	s.SonarTagInt("firstout", s.firstout)
	s.SonarTagInt("lastout", s.lastout)
	s.FlushSonar()
	bytePool.Put(s.buffer)
	aiSessGroup.Delete(s.Sid)
	//sessionPool.PutSession(s)
}

func (s *WsSession) SendException() {
	if s.conf.Server.Mock {
		return
	}
	s.Errorw("session close unexpected, send exception to atmos")
	s.StartSpan()
	defer s.FlushSpan()
	biz := &pb.ServerBiz{
		GlobalRoute: &pb.GlobalRoute{
			Headers: s.sessionContext.Header,
		},
		UpCall: &pb.UpCall{
			Call:         s.Sub,
			SeqNo:        2,
			From:         s.conf.Xsf.From,
			Sync:         false,
			SessionState: s.schema.Meta.GetSesssionStat(s.sessionContext),
			BusinessArgs: nil,
			Session:      s.sessionContext.Session,
			//Ple:          nil,
			DataList: nil,
		},
	}

	data, err := proto.Marshal(biz)
	if err != nil {
		s.Errorw("pbMarshal error while send exception", "error", err)
		return
	}

	//xsfCALLER := xsf.NewCaller(xsfClient)
	//xsfCALLER.WithRetry(1)
	req := xsf.NewReq()
	req.SetTraceID(s.SpanMeta())
	req.Append(data, nil)

	_, code, err := s.call(req, "exception")
	if err != nil || code != 0 {
		s.Errorw("send exception error", "code", code, "error", err.Error())
	}

}

func (s *WsSession) FlushSpan() {
	if s.span == nil {
		return
	}
	s.SpanTagString("ret", strconv.Itoa(s.errorCode))
	s.span.End().Flush()
}

func bizToString(biz *pb.ServerBiz) string {
	busiStr := map[string]string{}
	for k, v := range biz.GetUpCall().GetBusinessArgs() {
		busiStr[k+"args"] = common.MapstrToString(v.GetBusinessArgs())
		ples := make(map[string]string)
		for acpt, desc := range v.GetPle() {
			ple := common.NewStringBuilder()
			ple.AppendIfNotEmpty("attr", common.MapstrToString(desc.GetAttribute()))
			ple.AppendIfNotEmpty("accept", desc.GetName())
			ple.AppendIfNotEmpty("data_type", desc.GetDataType().String())
			ples[acpt] = ple.ToString()
		}
		busiStr[k+"ple"] = common.MapstrToString(ples)
	}
	m := map[string]interface{}{
		"header":   common.MapstrToString(biz.GetGlobalRoute().GetHeaders()),
		"business": common.MapstrToString(busiStr),
		//"payload":  common.MapToString(payload),
	}
	return common.MapToString(m)
}

func bizPayloadString(biz *pb.ServerBiz) string {
	//payload := map[string]string{}
	sb := strings.Builder{}
	for _, v := range biz.GetUpCall().GetDataList() {
		sb.WriteString(v.GetMeta().GetName())
		sb.WriteString(" dataLen:")
		sb.WriteString(strconv.Itoa(len(v.GetData())))
		//sb.WriteString("service:")
		//sb.WriteString(v.GetMeta().GetServiceName())
		sb.WriteString(" dataType:")
		sb.WriteString(pb.MetaDesc_DataType_name[int32(v.GetMeta().GetDataType())])
		sb.WriteString(" attr:")
		sb.WriteString(common.MapstrToString(v.GetMeta().GetAttribute()))
		//payload[v.GetMeta().GetName()] = fmt.Sprintf("service=%s,dataLen=%d,dataType=%v,attr=%s", v.GetMeta().GetServiceName(), len(v.GetData()), v.GetMeta().GetDataType(), common.String(v.GetMeta().GetAttribute()))
		sb.WriteString(" | ")
	}
	return sb.String()
}

//s.Ctx.GetHeader("X-Consumer-Username")
func CheckAppIdMatching(readAppid, consumerAppid string, enabled bool) bool {
	if !enabled {
		return true
	}
	//if realAppid=="" 那么可能不是走kong，或者kong没有开启鉴权，也放过
	if readAppid == "" {
		return true
	}
	if readAppid != consumerAppid {
		return false
	}
	return true
}

func (s *WsSession) sendMessage(in map[string]interface{}) (ErrorCode, string) {
	cloudId := s.Ctx.GetString(KeyCloudId)
	s.CloudId = cloudId
	//sonar
	if s.sonar == nil {
		s.StartSonar()
	}
	if s.Status == StatusBegin {
		if s.schema.Meta.IsCategory() {
			subServiceId, routerInfo := s.schema.GetSubServiceId(in)
			s.routerInfo = routerInfo
			// 子serviceId 不为空，使用子serviceID ，并在 sub=ase 时获取子schema
			if subServiceId != "" {
				s.ServiceId = subServiceId
				s.SpanTagString("useMappedServiceId", "true")
				s.SpanTagString("routerInfo", routerInfo)
				sc := schemas.GetSchemaByServiceId(subServiceId, cloudId)
				s.Debugw("companion route", "subSrv", subServiceId, "routeInfo", routerInfo)
				if sc == nil {
					return ErrorSubServiceNotFound, fmt.Sprintf("sub service not found:%s", subServiceId)
				}
				s.schema = sc
			} else {
				return ErrorSubServiceNotFound, fmt.Sprintf("no category route find")
			}
		}
	}
	clientIp := s.Ctx.GetString(CtxKeyClientIp)
	// 组装pb
	biz, err := s.schema.ResolveServerBiz(in, s.sessionContext)
	if err != nil {
		s.Errorw("resolve server biz error", "error", err.Error())
		return ErrorCodeGetUpCall, "resolve request error:" + err.Error()
	}

	header := biz.GetGlobalRoute().GetHeaders()
	//currentStatus := header["status"]
	s.lastin = currentTimeMills()

	//if currentStatus == "2" {
	//	//s.SonarTagInt("lastin", currentTimeMills())
	//}

	header[KeyCloudId] = cloudId
	if s.ServiceId != "" {
		header[KeyServiceId] = s.ServiceId
	}
	header[KeySub] = s.Sub
	header["routekey"] = s.routerInfo
	if s.Status == StatusBegin {
		header["route_param"] = s.routerInfo
		header[KeySid] = s.Sid
		header[KeyClientIp] = clientIp
		header[KeyCallBackAddr] = XsfCallBackAddr
		header[KeyTraceId] = s.SpanMeta()
		header[KeyServiceId] = s.ServiceId
		header[KeyRoute] = s.Ctx.Request.URL.Path
		s.SpanTagString(KeyCloudId, cloudId)
		s.SpanTagString("path", s.Ctx.Request.URL.Path)
		//s.CloudId = cloudId
		s.AppId = header[KeyAppId]
		if s.AppId == "" {
			s.AppId = header["app_id"]
			header[KeyAppId] = s.AppId
		}
		s.Uid = header[KeyUid]
		s.sessionContext.Header = header
		if s.AppId == "" {
			return ErrorInvalidAppid, "app_id cannot be empty"
		}

		s.Ctx.Set(KeyAppId, s.AppId)
		s.Ctx.Set(KeySid, s.Sid)
		s.Ctx.Set(KeyUid, s.Uid)
		realAppId := s.Ctx.GetHeader("X-Consumer-Username")
		bs := bizToString(biz)
		s.SpanTagString("first_frame", bs)
		//s.SonarTagInt("firstin", currentTimeMills())
		s.firstin = currentTimeMills()
		s.Debugw("first_frame", "biz", bs)
		if !CheckAppIdMatching(realAppId, s.AppId, s.conf.Auth.EnableAppidCheck) {
			s.Errorw("app_id and api_key does not match", "want", realAppId, "got", s.AppId)
			return ErrorInvalidAppid, "app_id and api_key does not match"
		}
		s.sessionCall = false
		if s.CallType == CallWithHash {
			s.SpanTagString("useHash", "true")
			//xsfCALLER.WithHashKey(s.Sid)
			s.sessionCall = true
		} else {
			for _, service := range s.conf.Xsf.HashServices {
				if service == s.CallService {
					//xsfCALLER.WithHashKey(s.Sid)
					s.sessionCall = true
					s.SpanTagString("useHash", "true")
					break
				}
			}
		}

	}
	// 做参数校验
	if err := s.SchemaCheck(in); err != nil {
		s.Errorw("schema validate error", "error", err.Error())
		return ErrorCodeGetUpCall, err.Error()
	}

	//header[KeySeqNo] = common.String(s.sessionContext.SeqNo)
	// log
	bizstr := bizPayloadString(biz)
	s.Debugw("resolved biz", "data", bizstr, "header", header)
	s.SpanTagString("serviceId", s.ServiceId)
	s.SpanTagString("route_param", s.routerInfo)
	// trace
	s.SpanTagString("clientIp", clientIp)
	s.SpanTagString("appid", s.AppId)
	s.SpanTagString("kongIp", s.Ctx.GetString(CtxKeyKongIp))
	s.SpanTagString("host", s.Ctx.GetString(CtxKeyHost))
	s.SpanTagString("path", s.Ctx.Request.URL.Path)

	s.SpanTagString("uprouterId", XsfCallBackAddr)
	s.SpanTagString("uid", s.Uid)
	s.SpanTagString("sess_stat", strconv.Itoa(int(s.schema.Meta.GetSesssionStat(s.sessionContext))))
	s.SpanTagString("header", common.MapstrToString(biz.GetGlobalRoute().GetHeaders()))
	s.SpanTagString("seqNo", strconv.Itoa(int(s.sessionContext.SeqNo)))
	s.SpanTagString("bizData", bizstr)

	// send request to atmos
	if s.conf.Server.Mock {
		s.MockCall(header)
		s.sessionContext.SeqNo++
		return 0, ""
	}
	upResult, respheader, bizE := s.SendAIBizByXsf(biz, s.CallType)

	if upResult != nil {
		if s.targetSub != "" {
			sess := upResult.GetSession()
			if sess != nil {
				s.targetSub = sess["aipaas_sub"]
			}
		}

	}

	s.SpanTagString("aipaas_sub", s.targetSub)
	s.SpanTagString("servieId2", s.MagicServiceId)
	if bizE != nil {
		s.Errorw("send biz error", "error", bizE.Message, "code", bizE.Code, "call", s.CallService)
		s.SpanTagErr(bizE.Error())
		return bizE.Code, bizE.Message
	}
	s.ResolveUpResult(upResult, respheader)
	s.sessionContext.SeqNo++
	return 0, ""
}

func (s *WsSession) handleAIMessage(msg []byte) (ErrorCode, string) {
	s.StartSpan()
	defer s.FlushSpan()

	in := map[string]interface{}{}
	err := jsoniter.Unmarshal(msg, &in)
	if err != nil {
		s.Errorw("json unmarshal error", "error", err.Error(), "json_data", common.ToString(msg))
		return ErrorCodeGetUpCall, "parse request json error"
	}
	// schema 校验，校验请求参数的合法性
	//if err := s.SchemaCheck(in); err != nil {
	//	s.Errorw("schema validate error", "error", err.Error(), "data", common.ToString(msg))
	//	return ErrorCodeGetUpCall, err.Error()
	//}
	return s.sendMessage(in)
}

//func bizToLogString(biz *pb.ServerBiz)string{
//	for srv, args := range biz.GetUpCall().GetBusinessArgs() {
//
//	}
//}

func (s *WsSession) ResolveUpResult(upr *pb.UpResult, header map[string]string) {
	if s.targetSub == "" {
		if upr.GetSession() != nil {
			s.targetSub = upr.GetSession()[KeyAipaaSSUb]
		}
	}
	if s.sessionContext.Session == nil {
		s.sessionContext.Session = upr.GetSession()
	}
	s.Debugw("up result", "session", upr.GetSession())
	if len(upr.GetDataList()) == 0 {
		if s.Status == StatusBegin {
			s.WriteSuccess(nil, 0, header)
			s.Status = StatusContinue
		}
		return
	}

	if s.Status == StatusBegin {
		s.Status = StatusContinue
	}

	result := s.schema.ResolveUpResult(upr)
	s.WriteSuccess(result, int(upr.GetStatus()), header)

}

func (s *WsSession) ResetReadDeadline() {
	s.Conn.SetReadDeadline(time.Now().Add(time.Duration(s.ReadTimeout) * time.Second))
}

var mockData = []byte("this is mocked data")

func (s *WsSession) MockCall(header map[string]string) {
	status := header["status"]
	if s.sessionContext.SeqNo == 0 || status == "2" || s.sessionContext.SeqNo%10 == 0 {
		data := map[string]interface{}{
			"result": map[string]interface{}{
				"text": "1234",
			},
		}
		s.WriteSuccess(data, common.IntFromString(status), nil)
		s.Status = StatusEnd
	}
}

type SendBizError struct {
	Code    ErrorCode
	Message string
}

func (e *SendBizError) Error() string {
	return fmt.Sprintf("%d|%s", e.Code, e.Message)
}

func NewSendBizError(code ErrorCode, msg string) *SendBizError {
	return &SendBizError{
		Code:    code,
		Message: msg,
	}
}

func (s *WsSession) call(req *xsf.Req, op string) (res *xsf.Res, code int32, err error) {
	xsfCALLER := xsf.NewCaller(xsfClient)
	xsfCALLER.WithRetry(s.conf.Xsf.CallRetry)
	if !s.sessionCall || s.sessionAddress == "" {
		res, code, err = xsfCALLER.Call(s.CallService, op, req, time.Duration(5)*time.Second)
		if res != nil && s.sessionAddress == "" {
			s.sessionAddress, _ = res.GetPeerIp()
			s.SpanTagString("serviceAddr", s.sessionAddress)
		}
	} else {
		res, code, err = xsfCALLER.CallWithAddr(s.CallService, op, s.sessionAddress, req, time.Duration(5)*time.Second)
		s.SpanTagString("callWithAddr", s.sessionAddress)
	}
	return
}

func (s *WsSession) SendAIBizByXsf(biz *pb.ServerBiz, callType int) (*pb.UpResult, map[string]string, *SendBizError) {
	cl1 := time.Now()
	data, err := proto.Marshal(biz)
	if err != nil {
		s.Errorw("pbMarshal error", "error", err)
		return nil, nil, NewSendBizError(ErrorCodeGetUpCall, "pb marshal error"+err.Error())
	}
	cl2 := time.Now()
	s.SpanTagString("pbMarshalCost", strconv.Itoa(int(cl2.Sub(cl1).Nanoseconds())))
	//初始化回调者
	//xsfCALLER := xsf.NewCaller(xsfClient)
	//
	//xsfCALLER.WithRetry(s.conf.Xsf.CallRetry)

	//初始化发送参数
	req := xsf.NewReq()
	req.SetTraceID(s.SpanMeta())
	//req.Session(s.Sid)

	req.Append(data, nil)

	var res *xsf.Res
	var code int32
	//var err error
	res, code, err = s.call(req, "req")
	//if !s.sessionCall || s.sessionAddress == "" {
	//	res, code, err = xsfCALLER.Call(s.CallService, "req", req, time.Duration(5)*time.Second)
	//	if res != nil && s.sessionAddress == "" {
	//		s.sessionAddress, _ = res.GetPeerIp()
	//		s.SpanTagString("serviceAddr", s.sessionAddress)
	//	}
	//} else {
	//	res, code, err = xsfCALLER.CallWithAddr(s.CallService, "req", s.sessionAddress, req, time.Duration(5)*time.Second)
	//	s.SpanTagString("callWithAddr", s.sessionAddress)
	//}
	//if res != nil {
	//	s.session = res.Session()
	//}
	if err != nil {
		s.Errorw(":send request error", "error", err.Error(), "code", code)
		return nil, nil, NewSendBizError(ErrorCode(code), "send request to backend error:"+err.Error())
	}
	cl3 := time.Now()
	s.SpanTagString("xsfCallCost", strconv.Itoa(int(cl3.Sub(cl2).Nanoseconds())))

	//解析响应结果
	respMsg := &pb.ServerBiz{}
	err = proto.Unmarshal(res.GetData()[0].Data, respMsg)
	if err != nil {
		s.Errorw("proto.Unmarshal up result error", "error", err.Error())
		return nil, nil, NewSendBizError(ErrorCodeJSONParsing, "invalid up result message")
	}

	header := respMsg.GetGlobalRoute().GetHeaders()
	if header != nil {
		serviceId := header["service_id"]
		if serviceId != "" && s.MagicServiceId == "" {
			s.MagicServiceId = serviceId
			s.ServiceId = serviceId
		}
	}

	if respMsg.GetUpResult().GetRet() != 0 {
		s.Errorw("get up result error", "error", respMsg.GetUpResult().GetErrInfo(), "code", respMsg.GetUpResult().GetRet())
		return respMsg.GetUpResult(), header, NewSendBizError(ErrorCode(respMsg.GetUpResult().GetRet()), respMsg.GetUpResult().GetErrInfo())
	}
	//s.Debugw("success send request to backend")
	return respMsg.GetUpResult(), header, nil
}

func (s *WsSession) readBody(body io.Reader) ([]byte, error) {
	bf := bytes.NewBuffer(s.buffer[:0])
	_, err := bf.ReadFrom(body)
	if err != nil {
		if err == io.EOF {
			return bf.Bytes(), nil
		}
		return nil, err
	}
	return bf.Bytes(), nil
}

// once
var (
	aiSessGroup *AISessionGroup
)

type AISessionGroup struct {
	lock     sync.RWMutex
	sess     map[string]*WsSession
	interval time.Duration
}

func InitSessionGroup(interval int) {
	aiSessGroup = &AISessionGroup{sess: map[string]*WsSession{}, interval: time.Duration(interval) * time.Second}
}

func (g *AISessionGroup) Get(sid string) *WsSession {
	g.lock.RLock()
	defer g.lock.RUnlock()
	return g.sess[sid]
}

func (g *AISessionGroup) Set(sid string, sess *WsSession) {
	g.lock.Lock()
	g.sess[sid] = sess
	g.lock.Unlock()
}

func (g *AISessionGroup) Delete(sid string) {
	g.lock.Lock()
	delete(g.sess, sid)
	g.lock.Unlock()
}

//attention read only ，if write in this function ，will occur dead lock
func (g *AISessionGroup) Range(f func(sid string, sess *WsSession) bool) {
	g.lock.RLock()
	defer g.lock.RUnlock()
	for k, v := range g.sess {
		if !f(k, v) {
			return
		}
	}
}

func (g *AISessionGroup) CheckIdleInBackground() {
	if g.interval == 0 {
		g.interval = 30 * time.Second
	}

	go func() {
		for range time.Tick(g.interval) {
			g.lock.RLock()
			deleted := make([]string, 0, 10)
			for sid, s := range g.sess {
				if !s.Alive() {
					deleted = append(deleted, sid)
				}
			}
			g.lock.RUnlock()
			for _, sid := range deleted {
				g.Delete(sid)
			}
		}
	}()

}
