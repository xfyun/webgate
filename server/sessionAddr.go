package server

import (
	"encoding/base64"
	"github.com/xfyun/webgate-aipaas/common"
)

type Session struct {
	CallAddr string
	Sid      string
}

var (
	encoder = base64.NewEncoding("ABGVWXCDEFefghijIJPQRSTUrstuvwxyzklmnopqH0123YZabcd45678KLMNO9-_")
)

func encodeSession(s Session) string {
	return encoder.EncodeToString([]byte(s.CallAddr + "," + s.Sid))
}
func DecodeSession(s string) (res Session, err error) {
	if s == "" {
		return res, nil
	}
	data, err := encoder.DecodeString(s)
	if err != nil {
		return res, err
	}
	remote, sid, ok := common.Cut(common.ToString(data), ",")
	if !ok {
		return res, NewHttpError(ErrorCodeGetUpCall, 400, "invalid session")
	}
	res.CallAddr = remote
	res.Sid = sid
	return res, nil
}
