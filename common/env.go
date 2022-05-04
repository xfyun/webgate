package common

import (
	"runtime"
	"os"
	"io"
	"fmt"
)

func Setenv(k,v string)  (err error){
	var envfileName string
	if runtime.GOOS == "windows"{
		envfileName  = "C:/users/admin/watchdog-env"
	}else{
		envfileName = "/etc/watchdog-env"
	}
	var file *os.File
	file,err=os.OpenFile(envfileName,os.O_WRONLY,0666)
	if err != nil{
		file,err = os.Create(envfileName)
		if err !=nil{
			return
		}
	}
	defer file.Close()
	n,err:=file.Seek(0,io.SeekEnd)
	if err !=nil{
		return
	}
	_,err = file.WriteAt([]byte(fmt.Sprintf("export %s=%s\n",k,v)),n)

	return
}
