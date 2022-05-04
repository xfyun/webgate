package common

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

func SetSystemEnv(k, v string) error {
	cmd := exec.Command("export", k+"="+v)
	return cmd.Run()
}

//解析出地址的端口号
func ParsePortFromAddr(addr string) (port int, err error) {
	res := strings.Split(addr, ":")
	if len(res) != 2 {
		return 0, errors.New("invalid addr:" + addr)
	}
	return strconv.Atoi(res[1])
}

func Cut(s, sep string) (before, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}
