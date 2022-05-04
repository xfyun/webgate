package server

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/xfyun/flange"
	"github.com/xfyun/webgate-aipaas/common"
	"github.com/xfyun/webgate-aipaas/conf"
	"github.com/xfyun/webgate-aipaas/pb"
	"github.com/xfyun/webgate-aipaas/schemas"
	xsf "github.com/xfyun/xsf/client"
	xsfs "github.com/xfyun/xsf/server"
	"github.com/xfyun/xsf/utils"
	"os"
	"runtime/debug"
	"strconv"
	"time"
)

const (
	XSF_CLIENT_NAME = "webgate-ws-c"
	LAST_MSG_STATUS = 2
)

var xsfClient *xsf.Client

//初始化Xsf客户端
func InitXsfClient(cfgName string) error {
	cfg := conf.GetConfInstance()
	cli, err := xsf.InitClient(
		XSF_CLIENT_NAME,
		getCfgMode(),
		utils.WithCfgCacheService(cfg.Xsf.CacheService),
		utils.WithCfgCacheConfig(cfg.Xsf.CacheConfig),
		utils.WithCfgCachePath(cfg.Xsf.CachePath),
		utils.WithCfgName(cfgName),
		utils.WithCfgURL(conf.Centra.CompanionUrl),
		utils.WithCfgPrj(conf.Centra.Project),
		utils.WithCfgGroup(conf.Centra.Group),
		utils.WithCfgService(conf.Centra.Service),
		utils.WithCfgVersion(conf.Centra.Version),
		utils.WithCfgSvcIp(cfg.Server.Host),
		utils.WithCfgSvcPort(func() int32 {
			a, _ := strconv.Atoi(cfg.Server.Port)
			return int32(a)
		}()),
	)
	if err != nil {
		return err
	}
	xsfClient = cli
	return nil
}

const (
	CallWithHash = 1
)

//发送请求
//func SendRequest(s *Session,sid string, biz *ServerBiz, frameId int,span *utils.Span,callType int) (*UpResult, *Error) {
//	cl1:=time.Now()
//	data, err := proto.Marshal(biz)
//	if err != nil {
//		common.Logger.Errorf("%s:ServerBiz(%+v) proto.Marshal is error.error:%s", sid, biz, err)
//		return nil, NewErrorByError(ErrorCodeJSONParsing, errors.New("parse proto data error"), sid, frameId)
//	}
//	cl2:=time.Now()
//	span.WithTag("pbMarshalCost",strconv.Itoa(int(cl2.Sub(cl1).Nanoseconds())))
//	//初始化回调者
//	xsfCALLER := xsf.NewCaller(xsfClient)
//
//	xsfCALLER.WithRetry(cfg.Xsf.CallRetry)
//	if callType == CallWithHash{
//		xsfCALLER.WithHashKey(s.Sid)
//	}
//	//初始化发送参数
//	req := xsf.NewReq()
//	req.SetTraceID(span.Meta())
//	req.Session(sid)
//	if s.reqParam !=nil{
//		for k, v := range s.reqParam {
//			req.SetParam(k,common.String(v))
//		}
//	}
//	req.Append(data, nil)
//	common.Logger.Infof("sid=%s,frameid=%d datalen=%d reqdatalen=%d,call_stat=%d", biz.GetGlobalRoute().GetTraceId(), biz.GetUpCall().GetSeqNo(), len(data), len(req.Data()),getSessStat(s.Status))
//	//common.Logger.Infof("sid=%s,frameid=%d datalen=%d reqdatalen=%d,busi=%v", biz.GetGlobalRoute().GetTraceId(), biz.GetUpCall().GetSeqNo(), len(data), len(req.Data()),biz.UpCall.BusinessArgs)
//	//发送请求
//	var res *xsf.Res
//	var code int32
//	//var err error
//	res, code, err  = xsfCALLER.Call(s.CallService, "req", req, time.Duration(5)*time.Second)
//
//	if res!=nil{
//		s.session =res.Session()
//	}
//	if err != nil {
//		common.Logger.Errorf(":send request error %v %v ", err, code)
//		return nil, NewError(int(code), err.Error(), sid, frameId)
//	}
//	cl3:=time.Now()
//	span.WithTag("xsfCallCost",strconv.Itoa(int(cl3.Sub(cl2).Nanoseconds())))
//
//	//解析响应结果
//	respMsg := &ServerBiz{}
//	err = proto.Unmarshal(res.GetData()[0].Data, respMsg)
//	if err != nil {
//		common.Logger.Errorf("%s:proto.Unmarshal is error.error:%s", sid, biz, err)
//		return nil, NewErrorByError(ErrorCodeJSONParsing, err, sid, frameId)
//	}
//
//	if respMsg.UpResult.GetRet() != 0 {
//		common.Logger.Errorf("%v:send request error %v", respMsg.UpResult.Ret, respMsg.UpResult.ErrInfo)
//		return nil, NewError(int(respMsg.UpResult.Ret), respMsg.UpResult.ErrInfo, sid, frameId)
//	}
//	common.Logger.Infof("sid=%s msgid=%d recieve %d %d", sid, respMsg.GetUpCall().GetSeqNo(), code, respMsg.GetUpResult().GetRet())
//
//	return respMsg.UpResult, nil
//}

var xsfMockHandler = &ServerHandler{}

type ServerHandler struct {
}

//启动Xsf的服务
func StartXsfServer(cfgName string) {

	//cfg:=conf.GetConfInstance()
	bc := xsfs.BootConfig{
		CfgMode: getCfgMode(),
		CfgData: xsfs.CfgMeta{
			CfgName:      cfgName,
			Project:      conf.Centra.Project,
			Group:        conf.Centra.Group,
			Service:      conf.Centra.Service,
			Version:      conf.Centra.Version,
			CompanionUrl: conf.Centra.CompanionUrl,
		},
	}
	var server = &xsfs.XsfServer{}
	//set spill enable
	go func() {
		time.Sleep(5 * time.Second)
		flange.SpillEnable = false
		fmt.Println("trace deliver ", flange.DeliverEnable)
		fmt.Println("trace dump ", flange.DumpEnable)
		fmt.Println("trace spill", flange.SpillEnable)
	}()

	go func() {
		err := (server).Run(bc, &ServerHandler{})
		if err != nil {
			panic(err)
		}
	}()

}

//func SendException(s *WsSession)  {
//	common.Logger.Infof("send Exception:sid=%s",s.Sid)
//	if cfg.Server.Mock{
//		return
//	}
//	xsfCALLER := xsf.NewCaller(xsfClient)
//	xsfCALLER.WithRetry(cfg.Xsf.CallRetry)
//	//初始化发送参数
//	req := xsf.NewReq()
//	req.Session(s.session)
//	s.SeqNo++
//	biz:=&pb.ServerBiz{
//		GlobalRoute:&pb.GlobalRoute{
//
//		},
//		UpCall:&pb.UpCall{
//			Call:s.Call,
//			SeqNo:int32(s.SeqNo),
//			From:s.From,
//			Sync:false,
//			Session:s.sessionMap,
//
//		},
//		MsgType:pb.ServerBiz_UP_CALL,
//		Version:conf.Centra.Version,
//	}
//	data,err:=proto.Marshal(biz)
//	if err !=nil{
//		common.Logger.Errorf("send exception error")
//		return
//	}
//	req.Append(data,nil)
//	xsfCALLER.Call(s.CallService,"exception",req,time.Duration(5)*time.Second)
//
//}

var XsfCallBackAddr string

//业务逆初始化接口
func (serHandler *ServerHandler) Finit() error {
	time.Sleep(15 * 1000 * time.Millisecond)
	return nil
}

func (serHandler *ServerHandler) Init(toolbox *xsfs.ToolBox) error {
	cfg := conf.GetConfInstance()
	XsfCallBackAddr = fmt.Sprintf("%s:%d", cfg.Server.Host, toolbox.NetManager.GetPort())
	fmt.Println("xsf callback addr:", XsfCallBackAddr)
	schemas.CallBackAddr = XsfCallBackAddr
	xsfs.AddKillerCheck("server", &killed{})
	return nil
}

//回调处理
func (c *ServerHandler) Call(in *xsf.Req, span *xsf.Span) (*utils.Res, error) {
	//	span = span.Next(utils.SrvSpan)
	defer func() {
		if err := recover(); err != nil {
			common.GetLoggerInstance().Errorf("panic:err=%v,stack=%s", err, common.ToString(debug.Stack()))
		}
	}()
	//span.Start()
	span.Start()
	defer span.End().Flush()
	serverBiz := getServerBiz(in, span)
	if serverBiz == nil {
		return xsf.NewRes(), nil
	}

	header := serverBiz.GetGlobalRoute().GetHeaders()
	if header == nil {
		common.GetLoggerInstance().Errorw("down call error,global route header is nil")
		return utils.NewRes(), nil
	}

	sid := header[KeySid]
	s := aiSessGroup.Get(sid)
	if s == nil {
		common.GetLoggerInstance().Errorw("session time out is nil", "sid", sid)
		return xsf.NewRes(), nil
	}

	if serverBiz.GetDownCall().GetRet() != 0 {
		s.Errorw("downcall error", "code", serverBiz.GetDownCall().GetRet(), "error", serverBiz.GetDownCall().GetErrInfo())
		s.WriteError(ErrorCode(serverBiz.GetDownCall().GetRet()), serverBiz.GetDownCall().GetErrInfo())
		s.setError(int(serverBiz.GetDownCall().GetRet()))
		return utils.NewRes(), nil
	}

	s.ResetReadDeadline()

	//重置session时间
	status := int(serverBiz.GetDownCall().GetStatus())
	if s.firstout == 0 {
		//s.SonarTagInt("firstout",currentTimeMills())
		s.firstout = currentTimeMills()
		span.WithTag("firstout", strconv.Itoa(s.firstout))
	}
	s.lastout = currentTimeMills()
	if status == StatusEnd {
		s.Status = StatusEnd
		//s.SonarTagInt("lastout",currentTimeMills())
		span.WithTag("lastout", strconv.Itoa(s.lastout))
		//span.WithTag("lalr",strconv.Itoa(s.firstout))
	}
	guiderEnd := ""
	params := in.GetAllParam()
	if params != nil {
		guiderEnd = params["guider_status"]
	}
	//s.resetConnTime()
	span.WithTag("sub", s.Sub)
	span.WithName("webgate-ws")
	span.WithTag("sid", sid).WithTag("call_type", "downcall")
	span.WithTag("status", strconv.Itoa(status))
	span.WithTag("appid", s.AppId)
	span.WithTag("code", strconv.Itoa(int(serverBiz.GetDownCall().GetRet())))
	payload := s.schema.ResolveDownResponseByBiz(serverBiz)
	s.WriteSuccessWithGuiderStatus(payload, status, guiderEnd, header)
	s.Debugw("downcall message", "status", status, "dataListLen", len(serverBiz.GetDownCall().GetDataList()))
	return xsf.NewRes(), nil

}

//解析回调的请求参数
func getServerBiz(in *xsf.Req, span *xsf.Span) *pb.ServerBiz {
	inData := in.Data()
	if inData == nil || len(inData) == 0 {
		//common.Logger.Errorf("downcall from atmos err ,data is nil")

		return nil
	}
	msg := &pb.ServerBiz{}
	if err := proto.Unmarshal(inData[0].Data(), msg); err != nil {
		//common.Logger.Errorf("reveive downcall  err %v", err)
		span.WithErrorTag("downcall:" + err.Error())
		return nil
	}
	//获取数据
	return msg
}

//获取xfs初始化客户端加载配置文件的方式
func getCfgMode() utils.CfgMode {
	//if *conf.BootMode {
	//	return utils.Native
	//}
	return utils.Centre
}

type killed struct {
}

func (k *killed) Closeout() {
	fmt.Println("server be killed.")
	//for true{
	//	num:=getCurrentSessionNum()
	//	if num==0{
	//		break
	//	}
	//	time.Sleep(200*time.Millisecond)
	//}
	os.Exit(0)
}

type CacheData struct {
}

type Result struct {
	timer   time.Timer
	results chan *CacheData
	timeout time.Duration
	handler func(data *CacheData) error
}

type ResultCache struct {
	cache map[string]*Result
}

func (c *ResultCache) start(rc *Result) {
	for {
		select {
		case result := <-rc.results:
			rc.timer.Reset(rc.timeout)
			for err := rc.handler(result); err != nil; {
				select {
				case <-rc.timer.C:

					return
				default:
				}
			}

		case <-rc.timer.C:
			rc.timer.Stop()
			close(rc.results)
			return
		}
	}
}
