package server

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/xfyun/webgate-aipaas/common"
	"github.com/xfyun/webgate-aipaas/conf"
	"github.com/xfyun/webgate-aipaas/schemas"
	xsf "github.com/xfyun/xsf/client"
	"sync"
	"time"
)

type MultipleSession struct {
	sessions    map[string]*WsSession
	ctx         *gin.Context
	schema      *schemas.AISchema
	conf        *conf.Config
	logger      *xsf.Logger
	conn        *websocket.Conn
	readTimeout int
	lock        *sync.Mutex
	timer       int //
}

func NewMultipleSession() *MultipleSession {
	return &MultipleSession{
		sessions: map[string]*WsSession{},
	}
}

func (m *MultipleSession) reset() {
	m.conn.SetReadDeadline(time.Now().Add(time.Duration(m.readTimeout) * time.Second))
}

func (m *MultipleSession) writeMessage(typ int, data []byte) {
	m.lock.Lock()
	m.conn.WriteMessage(typ, data)
	m.lock.Unlock()
}

func (m *MultipleSession) writeClose(msg string) {
	m.writeMessage(websocket.CloseMessage, websocket.FormatCloseMessage(1000, msg))
}

func (m *MultipleSession) writeError(code ErrorCode, info string, cid string, sid string) {
	resp := &ErrorResp{
		Header: Header{
			Code:    code,
			Message: info,
			Sid:     sid,
			Status:  0,
			Cid:     cid,
		},
	}
	data, _ := json.Marshal(resp)
	m.writeMessage(websocket.TextMessage, data)
}

func (m *MultipleSession) CloseSession() {
	time.Sleep(1 * time.Second)
	for _, session := range m.sessions {
		session.CloseMulitplex()
	}
	m.conn.Close()
}

func (m *MultipleSession) checkAndClearInactiveSession() {
	m.timer++
	if m.timer&(0xff) == 0 { // 每255 帧检查一次,清理掉不活跃的session
		now := time.Now()
		for cid, session := range m.sessions {
			if int(now.Sub(session.lastActiveTime).Seconds()) > m.readTimeout+10 {
				session.CloseMulitplex()
				delete(m.sessions, cid)
			}
		}
	}
}

func (m *MultipleSession) Do(ctx *gin.Context, schema *schemas.AISchema, conf *conf.Config, logger *xsf.Logger, conn *websocket.Conn) {
	m.ctx = ctx
	m.schema = schema
	m.conf = conf
	m.logger = logger
	m.conn = conn
	m.readTimeout = conf.Session.TimeoutInterver
	if rdt := m.schema.Meta.GetReadTimeout(); rdt > 5 {
		m.readTimeout = rdt
	}
	if m.readTimeout <= 5 {
		m.readTimeout = 15
	}
	m.lock = &sync.Mutex{}

	for {
		m.reset()
		_, msg, err := m.conn.ReadMessage()
		if err != nil {
			logger.Errorw("read message in multiplex error", "err", err)
			m.writeClose("timeout")
			return
		}
		m.checkAndClearInactiveSession()
		cid, sess, code, info := m.handleMessage(msg)
		if code != 0 {
			if sess != nil {
				sess.WriteError(code, info)
			} else {
				m.writeError(code, info, cid, common.NewSid(m.schema.Meta.GetSub()))
			}
		}

	}
}

//@reture1 cid
//@return4 errorInfo
func (m *MultipleSession) handleMessage(msg []byte) (string, *WsSession, ErrorCode, string) {
	in := map[string]interface{}{}
	err := jsoniter.Unmarshal(msg, &in)
	if err != nil {
		return "", nil, ErrorCodeGetUpCall, "parse request json error,"
	}
	//
	cid := getConnId(in)
	sess := m.sessions[cid]
	if sess == nil {
		//status,ok:=getStatus(in)
		//if !ok || status !=0  {
		//	return cid,nil,ErrorCodeInvalidSessionHandle,"invalid '$.header.status' value at begin frame"
		//}
		sess = NewWsSession(m.ctx, m.schema, m.conf, m.logger, m.conn, m.lock)
		sess.cid = cid
		m.sessions[cid] = sess
		sess.lastActiveTime = time.Now()
	}

	if int(time.Since(sess.lastActiveTime).Seconds()) > m.readTimeout {
		return cid, sess, ErrorCodeConnRead, "session read timeout #mu"
	}

	// 重置活跃时间
	sess.lastActiveTime = time.Now()
	//schema 校验
	//if m.schema.InputSchema != nil {
	//	if err := m.schema.InputSchema.Validate(in); err != nil {
	//		return cid, sess, ErrorCodeGetUpCall, fmt.Sprintf("schema validate error:" + err.Error())
	//	}
	//}

	code, retinfo := sess.sendMessage(in)
	return cid, sess, code, retinfo
}

func getConnId(in map[string]interface{}) string {
	header, ok := in["header"].(map[string]interface{})
	if !ok {
		return ""
	}
	cid, _ := header[keyConnId].(string)
	delete(header, keyConnId) // 从header 中删除cid，否则schema 校验会不通过
	return cid
}

func getStatus(in map[string]interface{}) (int, bool) {
	header, ok := in["header"].(map[string]interface{})
	if !ok {
		return 0, false
	}
	status, ok := header["status"].(float64)
	if !ok {
		return 0, false
	}
	return int(status), true
}
