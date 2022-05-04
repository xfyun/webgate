package conf

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"testing"
)

type Str struct {
	CallService string `toml:"callService"`
	Client      struct {
		Name string `toml:"name"`
	} `toml:"client"`
}

func Test_C(t *testing.T) {
	var s Config
	_, err := toml.DecodeFile("app.toml", &s)

	if err != nil {
		panic(err)
	}
	fmt.Println(s)
}

type A struct {

}

type e error

func (a A)Say()string  {
	return "aaaa"
}
func TestInit(t *testing.T) {
	//Init()


}


func TestRegexp(t *testing.T) {
	type A struct {
		B string
		C int
	}

	a:=&A{B:"sdfsdf",C:12}
	ca:=*a
	cap:=&ca
	cap.B="heeee"
	cap.C=2
	fmt.Println(a)

}


func TestInterface(t *testing.T){
	var a string
	for i:=0;i<10000000;i++{
		testin(a)
	}
}

var c string
func testin(v string){

}
