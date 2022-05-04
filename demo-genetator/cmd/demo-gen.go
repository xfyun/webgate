package main

import (
	"flag"
	demo_genetator "github.com/xfyun/webgate-aipaas/demo-genetator"
	"io/ioutil"
	"os"
)

var (
	input = "stdin"
)

func init() {
	flag.StringVar(&input, "f", input, "input schema file")
}

func main() {
	flag.Parse()
	var data []byte
	var err error
	if input == "stdin" {
		data, err = ioutil.ReadAll(os.Stdin)
	} else {
		data, err = ioutil.ReadFile(input)
	}
	if err != nil {
		panic(err)
	}
	err = demo_genetator.GenDemo(data, os.Stdout)
	if err != nil {
		panic(err)
	}
}
