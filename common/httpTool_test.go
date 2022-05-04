package common

import (
	"fmt"
	"net/http"
	"testing"
)

func TestNewRequest(t *testing.T) {
	req:=NewRequest("GET","http://ws-api.xfyun.cn",nil).
		SetClient(http.DefaultClient).Do()
	if req.Errors() != nil{
		panic(req.Errors())
	}


	fmt.Println(string(req.ReadBody()),req.IsSuccess())
}