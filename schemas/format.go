package schemas

import (
	"encoding/json"
	"fmt"
	"github.com/xfyun/webgate-aipaas/common"
)

//json
//string;gbk
func formatData(t string, data []byte, ops map[string]string) (interface{}, error) {
	switch t {
	case "json":
		if json.Valid(data) {
			return json.RawMessage(data), nil
		}
		return common.EncodingTobase64String(data), fmt.Errorf("engine output is not json")

	case "string":
		if in(ops["encoding"], "utf8", "utf-8", "UTF8", "UTF-8") {
			return string(data), nil
		}
	}
	return common.EncodingTobase64String(data), nil
}

func or(ss ...string) string {
	for _, v := range ss {
		if v != "" {
			return v
		}
	}
	return ""
}

func in(s string, ss ...string) bool {
	for _, v := range ss {
		if s == v {
			return true
		}
	}
	return false
}
