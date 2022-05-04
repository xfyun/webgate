package common

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
)

type Request struct {
	r *http.Request
	errors []error
	client *http.Client
	statusCode int
	respBody io.ReadCloser
	respBodyBytes []byte
}

func NewRequest(method,url string,body interface{})*Request{
	r:=&Request{}
	var reader io.Reader
	switch body.(type) {
	case []byte:
		reader = bytes.NewReader(body.([]byte))
	case io.Reader:
		reader = body.(io.Reader)
	case nil:
		reader = nil
	default:
		bf:=&bytes.Buffer{}
		e:=json.NewEncoder(bf)
		err:=e.Encode(body)
		if err != nil{
			r.errors = append(r.errors,err)
		}
		reader = bf
	}
	req,err:=http.NewRequest(method,url,reader)
	if err != nil{
		r.errors = append(r.errors,err)
	}
	r.r = req
	return r
}

func (r *Request)SetClient(c *http.Client)*Request{
	r.client =c
	return r
}

func (r *Request)Do()(req *Request){
	req = r
	if len(r.errors) >0 {
		return
	}
	client:=r.client
	if client == nil{
		client = http.DefaultClient
	}
	resp,err:=client.Do(r.r)
	if err != nil{
		r.errors = append(r.errors,err)
		return
	}
	r.statusCode = resp.StatusCode
	r.respBody = resp.Body
	return
}

func (r *Request)Body()io.ReadCloser{
	return r.respBody
}

func (r *Request)ReadBody()[]byte{
	if r.respBody != nil{
		bytes,err:=ioutil.ReadAll(r.respBody)
		if err != nil{
			r.errors = append(r.errors,err)
			return nil
		}
		r.respBodyBytes = bytes
		return bytes
	}
	return nil
}

func (r *Request)StatusCode()int{
	return r.statusCode
}

func (r *Request)Errors()[]error{
	if len(r.errors) ==  0 {
		return nil
	}
	return r.errors
}

func in(code int ,cs ...int)bool{
	for _, v := range cs {
		if code == v{
			return true
		}
	}
	return false
}

func (r *Request)IsSuccess(successCodes ...int)bool{
	if len(r.errors) >0 {
		return false
	}
	if in(r.statusCode,200,201,101,204){
		return true
	}
	return in(r.statusCode,successCodes...)
}

