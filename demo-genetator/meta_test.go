package demo_genetator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func Test_generate(t *testing.T) {
	err := generate("", &Schema{
		Header: []Field{
			{
				Name:  "app_id",
				Value: `"123456"`,
			},
		},
		Parameter: []Parameter{
			{
				Name: "ist",
				Fields: []Field{
					{
						Name:  "eos",
						Value: "1000",
					},
				},
				Accepts: []Field{
					{
						Name:  "encoding",
						Value: "\"utf8\"",
					},
				},
				AcceptName: "accept",
			},
		},
		Payload: []Payload{
			{
				Name: "input",
				Fields: []Field{
					{
						Name:  "audio",
						Value: "Base64.getEncoder().encodeToString(Arrays.copyOf(frame, n > 0 ? n : 0))",
					},
				},
			},
		},
	}, os.Stdout)
	if err != nil {
		panic(err)
	}

}

func TestSc(t *testing.T) {
	sc, err := ioutil.ReadFile(`../schemas/schema_test.json`)
	if err != nil {
		panic(err)
	}

	schema := &SchemaTlp{}
	err = json.Unmarshal(sc, schema)
	if err != nil {
		panic(err)
	}
	scc := parseSchema(schema.SchemaInput)
	fmt.Println(schema)
	generate(javatlp, scc, os.Stdout)
	bs, _ := json.Marshal(schema)

	fmt.Println(string(bs))

}
