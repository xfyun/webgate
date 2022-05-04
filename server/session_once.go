package server

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/proto"
	jsoniter "github.com/json-iterator/go"
	"github.com/xfyun/sonar"
	"github.com/xfyun/webgate-aipaas/common"
	"github.com/xfyun/webgate-aipaas/conf"
	"github.com/xfyun/webgate-aipaas/pb"
	"github.com/xfyun/webgate-aipaas/schemas"
	xsf "github.com/xfyun/xsf/client"
	"github.com/xfyun/xsf/utils"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"
)

type HttpSession struct {
	Sid            string
	ServiceId      string
	RawServiceId   string
	AppId          string
	Uid            string
	buffer         []byte
	Sub            string
	lock           sync.Mutex
	CallService    string
	CallType       int
	Ctx            *gin.Context
	schema         *schemas.AISchema
	sessionContext *schemas.Context
	sonar          *sonar.MetricData
	span           *utils.Span
	startTime      time.Time
	Status         int32
	errorCode      int
	logger         *xsf.Logger
	conf           *conf.Config
	CloudId        string
	targetSub      string
	routerInfo     string
	MagicServiceId string
	respHeader     map[string]string // 相应的upresult 中的header
	ClientSession  string
	ClientCallAddr string
}

func NewHttpSession(ctx *gin.Context, schema *schemas.AISchema, conf *conf.Config, logger *xsf.Logger, sid string) *HttpSession {
	meta := schema.Meta
	if sid == "" {
		sid = common.NewSid(meta.GetSub())
	}
	sess := &HttpSession{}
	sess.Sid = sid
	sess.Sub = meta.GetSub()
	sess.buffer = bytePool.Get()
	sess.CallService = meta.GetCallService()
	sess.CallType = meta.GetCallType()
	sess.Ctx = ctx
	sess.Status = 0
	sess.startTime = time.Now()
	sess.schema = schema
	sess.sessionContext = &schemas.Context{SeqNo: 1, Sync: true, IsStream: false}
	sess.logger = logger
	sess.conf = conf
	sess.sonar = nil
	sess.errorCode = 0
	sess.routerInfo = ""
	sess.AppId = ""
	sess.MagicServiceId = ""
	return sess
}

func (s *HttpSession) formatArgs(kvs []interface{}) []interface{} {

	common := []interface{}{"sid", s.Sid, "app_id", s.AppId, "uid", s.Uid, "serviceId", s.schema.Meta.GetServiceId(), "call", s.schema.Meta.GetCallService()}
	args := make([]interface{}, len(kvs)+len(common))
	copy(args, common)
	copy(args[len(common):], kvs)
	return args
}

func (s *HttpSession) Errorw(msg string, kvs ...interface{}) {
	//s.logger.Errorw("","")

	s.logger.Errorw(msg, s.formatArgs(kvs)...)
}

func (s *HttpSession) Infow(msg string, kvs ...interface{}) {
	if s.conf.Log.Level == "error" { //日志级别为error 时就不用复制args了
		return
	}
	s.logger.Infow(msg, s.formatArgs(kvs)...)
}

func (s *HttpSession) Debugw(msg string, kvs ...interface{}) {
	if s.conf.Log.Level == "error" {
		return
	}
	s.logger.Debugw(msg, s.formatArgs(kvs)...)
}

func (s *HttpSession) StartSpan() {
	s.span = utils.NewSpan(utils.SrvSpan).Start()

}

func (s *HttpSession) SpanTagString(k, v string) {
	s.span.WithTag(k, v)
}

func (s *HttpSession) SpanMeta() string {
	return s.span.Meta()
}

func (s *HttpSession) SpanTagErr(err string) {
	s.span.WithErrorTag(err)
}

func (s *HttpSession) SetError(code int) {
	if s.errorCode == 0 {
		s.errorCode = code
	}
}

func (s *HttpSession) SchemaCheck(o interface{}) error {
	if s.conf.Schema.Enable {
		if err := s.schema.Validate(o); err != nil {
			return err
		}
	}
	return nil
}

func (s *HttpSession) StartSonar() {
	if !s.conf.Server.EnableSonar {
		return
	}
	s.sonar = sonar.NewMetricWithNamePort(sonar.TYPE_GAUGE,
		"sps", XsfCallBackAddr,
		s.Sub,
		s.conf.Xsf.ServerPort,
		"vagus_null")

	s.sonar.Tag(sonar.KV{"sid", s.Sid}).
		Tag(sonar.KV{"from", "webgate-http-aipaas"}).
		Tag(sonar.KV{"cluster", s.conf.Server.Cluster}).
		Tag(sonar.KV{"dc", s.conf.Xsf.Dc}).
		Tag(sonar.KV{
			Key:   "firstin",
			Value: currentTimeMills(),
		}).Tag(sonar.KV{
		Key:   "lastin",
		Value: currentTimeMills(),
	})
}

func (s *HttpSession) FlushSonar() {
	if s.sonar != nil {

		sub := s.targetSub
		if sub == "" {
			sub = s.Sub
		}

		ent := s.ServiceId
		if s.MagicServiceId != "" {
			ent = s.MagicServiceId
		}

		s.sonar.Tag(sonar.KV{"finalEnt", ent})
		//metPtr.TagDS("vagus_null").Tag(sonar.KV{"end", "true"})
		s.sonar.TagDS("vagus_null")
		s.SonarTagString("sub", sub)
		s.SonarTagInt("sret", s.errorCode)
		s.SonarTagInt("ret", s.errorCode)
		s.SonarTagString("end", "true")
		s.SonarTagString("dc", s.conf.Xsf.Location)
		s.SonarTagString("appid", s.AppId)
		s.SonarTagString("uid", s.Uid)
		s.SonarTagString("routekey", s.routerInfo)
		s.SonarTagInt("firstout", currentTimeMills())
		s.SonarTagInt("lastout", currentTimeMills())

		if s.CloudId != "" {
			s.SonarTagString("cloud_id", s.CloudId)
		} else {
			s.SonarTagString("cloud_id", "ai_cloud")
		}

		s.sonar.Flush()
	}
}

func (s *HttpSession) SonarTagString(key string, v string) {
	if s.sonar != nil {
		s.sonar.Tag(sonar.KV{
			Key:   key,
			Value: v,
		})
	}
}

func (s *HttpSession) SonarTagInt(key string, v int) {
	if s.sonar != nil {
		s.sonar.Tag(sonar.KV{
			Key:   key,
			Value: strconv.Itoa(v),
		})
	}
}

// close 只能被执行一次
func (s *HttpSession) CloseSession() {
	s.FlushSonar()
	s.FlushSpan()
	bytePool.Put(s.buffer)
}

func (s *HttpSession) SendException() {
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

	xsfCALLER := xsf.NewCaller(xsfClient)
	xsfCALLER.WithRetry(1)
	req := xsf.NewReq()
	req.SetTraceID(s.SpanMeta())
	req.Append(data, nil)
	_, code, err := xsfCALLER.Call(s.CallService, "exception", req, time.Duration(5)*time.Second)
	if err != nil || code != 0 {
		s.Errorw("send exception error", "code", code, "error", err.Error())
	}
}

func (s *HttpSession) FlushSpan() {
	if s.span == nil {
		return
	}
	s.span.WithName("webgate-http-aipaas").WithTag("sub", s.Sub).WithTag("sid", s.Sid)
	s.span.WithTag("goroutines", strconv.Itoa(runtime.NumGoroutine()))
	s.SpanTagString("code", strconv.Itoa(s.errorCode))
	s.SpanTagString("ret", strconv.Itoa(s.errorCode))
	s.span.End().Flush()
}

func (s *HttpSession) getTimeout() time.Duration {
	st := s.schema.Meta.GetSessonTimeout()
	if st <= 0 {
		st = s.conf.Session.SessionTimeout
	}

	if st <= 0 {
		st = 30
	}
	return time.Duration(st) * time.Second
}

func (s *HttpSession) call(req *xsf.Req, op string) (res *xsf.Res, code int32, err error) {

	xsfCALLER := xsf.NewCaller(xsfClient)
	xsfCALLER.WithRetry(s.conf.Xsf.CallRetry)
	if s.ClientCallAddr == "" {
		res, code, err = xsfCALLER.Call(s.CallService, op, req, s.getTimeout())
		if res != nil {
			remoteIp, _ := res.GetPeerIp()
			s.ClientSession = encodeSession(Session{
				CallAddr: remoteIp,
				Sid:      s.Sid,
			})
		}
	} else {
		res, code, err = xsfCALLER.CallWithAddr(s.CallService, op, s.ClientCallAddr, req, time.Duration(s.conf.Session.ReadTimeout)*time.Second)
		s.SpanTagString("callWithSessionAddr", s.ClientCallAddr)
	}
	return
}

func (s *HttpSession) SendAIBizByXsf(biz *pb.ServerBiz, callType int) (*pb.UpResult, *SendBizError) {
	cl1 := time.Now()
	data, err := proto.Marshal(biz)
	if err != nil {
		s.Errorw("pbMarshal error", "error", err)
		return nil, NewSendBizError(ErrorCodeGetUpCall, "pb marshal error"+err.Error())
	}
	cl2 := time.Now()
	s.SpanTagString("pbMarshalCost", strconv.Itoa(int(cl2.Sub(cl1).Nanoseconds())))
	//初始化回调者
	xsfCALLER := xsf.NewCaller(xsfClient)

	xsfCALLER.WithRetry(s.conf.Xsf.CallRetry)
	if callType == CallWithHash {
		xsfCALLER.WithHashKey(s.Sid)
	}
	//初始化发送参数
	req := xsf.NewReq()
	req.SetTraceID(s.SpanMeta())
	//req.Session(s.Sid)

	req.Append(data, nil)
	//common.Logger.Infof("sid=%s,frameid=%d datalen=%d reqdatalen=%d,call_stat=%d", biz.GetGlobalRoute().GetTraceId(), biz.GetUpCall().GetSeqNo(), len(data), len(req.Data()),getSessStat(s.Status))
	s.Debugw("start send request to backend")
	//common.Logger.Infof("sid=%s,frameid=%d datalen=%d reqdatalen=%d,busi=%v", biz.GetGlobalRoute().GetTraceId(), biz.GetUpCall().GetSeqNo(), len(data), len(req.Data()),biz.UpCall.BusinessArgs)
	//发送请求
	var res *xsf.Res
	var code int32
	//var err error
	res, code, err = s.call(req, "req")

	//if res != nil {
	//	s.session = res.Session()
	//}
	if err != nil {
		s.Errorw(":send request error", "error", err.Error(), "code", code)
		return nil, NewSendBizError(ErrorCode(code), "send request to backend error:"+err.Error())
	}
	cl3 := time.Now()
	s.SpanTagString("xsfCallCost", strconv.Itoa(int(cl3.Sub(cl2).Nanoseconds())))

	//解析响应结果
	respMsg := &pb.ServerBiz{}
	err = proto.Unmarshal(res.GetData()[0].Data, respMsg)
	if err != nil {
		s.Errorw("proto.Unmarshal up result error", "error", err.Error())
		return nil, NewSendBizError(ErrorCodeJSONParsing, "invalid up result message")
	}

	header := respMsg.GetGlobalRoute().GetHeaders()
	if header != nil {
		serviceId := header["service_id"]
		if serviceId != "" {
			s.MagicServiceId = serviceId
		}
	}
	s.respHeader = header
	if respMsg.GetUpResult().GetRet() != 0 {
		s.Errorw("get up result error", "error", respMsg.GetUpResult().GetErrInfo(), "code", respMsg.GetUpResult().GetRet())
		return respMsg.GetUpResult(), NewSendBizError(ErrorCode(respMsg.GetUpResult().GetRet()), respMsg.GetUpResult().GetErrInfo())
	}
	s.Debugw("success send request to backend")
	return respMsg.GetUpResult(), nil
}

// once

func (s *HttpSession) readBody(body io.Reader) ([]byte, error) {
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

func getSession(in map[string]interface{}) (res string) {
	hd, ok := in["header"].(map[string]interface{})
	if !ok {
		return ""
	}
	res, _ = hd["session"].(string)
	return res
}

func HandleOnce(s *HttpSession) error {
	s.StartSpan()
	readbodyStart := time.Now()
	body, err := s.readBody(s.Ctx.Request.Body)
	if err != nil {
		s.Errorw("read body error", "error", err.Error())
		return err
	}
	request := make(map[string]interface{})
	if err := jsoniter.Unmarshal(body, &request); err != nil {
		s.Errorw("unmarshal json error", "error", err.Error(), "body", string(body), "code", ErrorCodeGetUpCall)
		return NewHttpError(ErrorCodeGetUpCall, http.StatusBadRequest, "request body must be json")
	}
	sc := s.schema
	if sc.Meta.EnableClientSession() {
		ss, err := DecodeSession(getSession(request))
		if err != nil {
			return NewHttpError(ErrorCodeGetUpCall, 400, "decode session error:"+err.Error())
		}
		s.ClientCallAddr = ss.CallAddr
		if ss.Sid != "" {
			s.Sid = ss.Sid
		}
	}

	s.SpanTagString("readBodyCost", time.Since(readbodyStart).String())
	cloudId := s.Ctx.GetString(KeyCloudId)
	s.CloudId = cloudId

	serviceId := s.schema.Meta.GetServiceId()
	s.ServiceId = serviceId
	s.RawServiceId = serviceId
	var routerInfo string
	var subServiceId string
	if s.schema.Meta.IsCategory() {
		subServiceId, routerInfo = s.schema.GetSubServiceId(request)
		// 子serviceId 不为空，使用子serviceID ，并在 sub=ase 时获取子schema
		s.routerInfo = routerInfo
		if subServiceId != "" {
			serviceId = subServiceId
			s.SpanTagString("useMapedServiceId", "true")
			s.SpanTagString("routeInfo", routerInfo)
			s.Debugw("use_mapped_service_id", "serviceId", serviceId, "routeInfo", routerInfo)
			sc := schemas.GetSchemaByServiceId(subServiceId, cloudId)
			if sc == nil {
				return NewHttpError(ErrorNotFound, 500, "sub service not found:"+subServiceId)
			}
			s.schema = sc
		} else {
			return NewHttpError(ErrorNotFound, 400, "no companion route found")
		}
	}

	s.ServiceId = serviceId

	if err := s.SchemaCheck(request); err != nil {
		s.Errorw("schema validate error", "error", err.Error(), "req", request, "code", ErrorCodeGetUpCall)
		return NewHttpError(ErrorCodeGetUpCall, http.StatusBadRequest, fmt.Sprintf("parameter schema validate error: %s", err.Error()))
	}

	biz, err := s.schema.Meta.ResolveServerBiz(request, s.sessionContext)
	if err != nil {
		s.Errorw("resolve request serverBiz error", "error", err.Error(), "code", ErrorCodeGetUpCall)
		return NewHttpError(ErrorCodeGetUpCall, http.StatusBadRequest, err.Error())
	}

	header := biz.GetGlobalRoute().GetHeaders()
	clientIp := s.Ctx.GetString(CtxKeyClientIp)

	s.SpanTagString("clientIp", clientIp)
	s.SpanTagString("cloud_id", cloudId)
	s.SpanTagString("serviceId", s.ServiceId)
	s.SpanTagString("kongIp", s.Ctx.GetString(CtxKeyKongIp))
	s.SpanTagString("host", s.Ctx.GetString(CtxKeyHost))
	s.SpanTagString("path", s.Ctx.Request.URL.Path)
	s.SpanTagString("route_param", routerInfo)

	s.SpanTagString("uprouterId", XsfCallBackAddr)
	s.SpanTagString("sess_stat", strconv.Itoa(int(s.schema.Meta.GetSesssionStat(s.sessionContext))))
	s.SpanTagString("header", common.MapstrToString(biz.GetGlobalRoute().GetHeaders()))
	bizstr := bizToString(biz)
	//s.SpanTagString("bizData", bizstr)

	if header != nil {
		header["route_param"] = routerInfo
		header["routekey"] = routerInfo
		header[KeyCloudId] = cloudId
		header[KeySid] = s.Sid
		header[KeyClientIp] = clientIp
		header[KeyCallBackAddr] = XsfCallBackAddr
		header[KeyTraceId] = s.SpanMeta()
		header[KeyServiceId] = serviceId
		header[KeyRoute] = s.Ctx.Request.URL.Path
		header[KeySub] = s.Sub
		s.AppId = header[KeyAppId]
		//s.ClientSession = header["session"]

		if s.AppId == "" {
			s.AppId = header["app_id"]
			header[KeyAppId] = s.AppId
		}
		s.Uid = header[KeyUid]
		s.sessionContext.Header = header
		if s.AppId == "" {
			return NewHttpError(ErrorInvalidAppid, http.StatusBadRequest, "appid cannot be empty")
		}
		s.Ctx.Set(KeyAppId, s.AppId)
		s.Ctx.Set(KeySid, s.Sid)
		s.Ctx.Set(KeyUid, s.Uid)
		realAppId := s.Ctx.GetHeader("X-Consumer-Username")
		s.SpanTagString("first_frame", bizstr)
		if !CheckAppIdMatching(realAppId, s.AppId, s.conf.Auth.EnableAppidCheck) {
			s.Errorw("app_id and api_key does not match", "want", realAppId, "got", s.AppId)
			return NewHttpError(ErrorInvalidAppid, http.StatusBadRequest, "app_id and api_key does not match")
		}
	}
	s.Debugw("resolved biz", "data", bizstr, "header", header, "biz", biz)

	s.SpanTagString("appid", s.AppId)
	s.SpanTagString("uid", s.Uid)

	upResult, bizE := s.SendAIBizByXsf(biz, s.schema.Meta.GetCallType())

	if upResult != nil {
		if s.targetSub != "" {
			sess := upResult.GetSession()
			if sess != nil {
				s.targetSub = sess["aipaas_sub"]
			}
		}

	}
	s.SpanTagString("targetSub", s.targetSub)

	if bizE != nil {
		s.Errorw("send data by xsf error", "error", bizE.Error())
		return bizE
	}
	resp := s.schema.ResolveUpResult(upResult)
	s.SpanTagString("serviceId2", s.ServiceId)
	s.Ctx.AbortWithStatusJSON(http.StatusOK, NewHttpSuccessResp(s.Sid, resp, s))
	s.Infow("succes handle request", "resp", resp)
	return nil
}
