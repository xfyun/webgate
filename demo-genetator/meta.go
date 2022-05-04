package demo_genetator

import (
	"encoding/json"
	"fmt"
	"io"
	"text/template"
)
type Field struct {
	Name  string //app_id
	Value string //
}

type Parameter struct {
	Name       string
	Fields     []Field
	Accepts    []Field
	AcceptName string
}

type Payload struct {
	Name   string
	Fields []Field
}

type Schema struct {
	Header    []Field
	Parameter []Parameter
	Payload   []Payload
}

func generate(tlp string, sc *Schema, out io.Writer) error {
	tp, err := template.New("java").Parse(tlp)
	if err != nil {
		return err
	}
	return tp.Execute(out, sc)
}

type SchemaTlp struct {
	SchemaInput *Object `json:"schemainput"`
}

type Object struct {
	Type       string             `json:"type"`
	Properties map[string]*Object `json:"properties,omitempty"`
	Enum       []interface{}      `json:"enum,omitempty" `
	Maximum    int                `json:"maximum"`
	Minimum    int                `json:"minimum"`
	Required   []string           `json:"required"`
}

var (
	ignoreParams = map[string]bool{
		"directEngIp": true,
	}

	demoValue = map[string]string{
		"app_id": `appId`,
		"mac":    `"6c:92:bf:65:c6:14"`,
		"imei":   `"866402031869366"`,
		"status": "status",
		"seq":    "seq++",
	}
)

type Opt struct {
	Required []string
	UseRequired bool
	UseDemoValue bool
}

func parseFields(objs map[string]*Object,opt Opt) []Field {
	fds := make([]Field, 0, len(objs))
conn:
	for name, object := range objs {
		if ignoreParams[name] {
			continue
		}
		f := Field{
			Name: name,
		}
		if val := demoValue[name]; opt.UseDemoValue && val != "" {
			f.Value = val
			fds = append(fds, f)
			continue
		}

		if opt.UseRequired && len(object.Enum)==0{
			con:= false
			for _, key := range opt.Required {
				if key == name{
					con = true
				}
			}
			if !con{
				continue
			}
		}

		switch object.Type {
		case "string":
			if len(object.Enum) > 0 {
				f.Value = fmt.Sprintf(`"%v"`, object.Enum[0])
			} else {
				f.Value = `""`
			}
		case "integer":
			if len(object.Enum) > 0 {
				f.Value = fmt.Sprintf(`%v`, object.Enum[0])
			} else {
				f.Value = fmt.Sprintf("%v", object.Minimum)
			}
		case "boolean":
			f.Value = "false"
		case "number":
			if len(object.Enum) > 0 {
				f.Value = fmt.Sprintf(`%v`, object.Enum[0])
			} else {
				f.Value = fmt.Sprintf("%v", object.Minimum)
			}
		default:
			continue conn
		}
		fds = append(fds, f)
	}
	return fds
}

func parseParameters(o map[string]*Object,opt Opt) []Parameter {
	ps := make([]Parameter, 0, len(o))
	for name, object := range o {
		p := Parameter{
			Name:       name,
			Fields:     parseFields(object.Properties,opt),
			Accepts:    nil,
			AcceptName: "",
		}
		for key, pro := range object.Properties {
			if pro.Type == "object" {
				p.AcceptName = key
				p.Accepts = parseFields(pro.Properties,opt)
			}
		}
		ps = append(ps, p)
	}
	return ps
}

func parsePayload(o map[string]*Object,opt Opt) []Payload {
	pads := []Payload{}
	for name, object := range o {
		pd := Payload{
			Name: name,
		}
		fds := parseFields(object.Properties,opt)
		for i, fd := range fds {
			if fd.Name == "audio" || fd.Name == "image" || fd.Name == "text" || fd.Name == "video" {
				fds[i] = Field{
					Name:  fd.Name,
					Value: `Base64.getEncoder().encodeToString(Arrays.copyOf(frame, n > 0 ? n : 0))`,
				}
			}
		}
		pd.Fields = fds
		pads = append(pads, pd)
	}

	return pads
}

func parseSchema(o *Object) *Schema {
	return &Schema{
		Header:    parseFields(o.Properties["header"].Properties,Opt{UseDemoValue: true}),
		Parameter: parseParameters(o.Properties["parameter"].Properties,Opt{Required: o.Required,UseRequired: true}),
		Payload:   parsePayload(o.Properties["payload"].Properties,Opt{UseDemoValue: true}),
	}
}

func GenDemo(in []byte, out io.Writer) error {
	schema := &SchemaTlp{}
	err := json.Unmarshal(in, schema)
	if err != nil {
		return err
	}
	scc := parseSchema(schema.SchemaInput)
	return generate(javatlp, scc, out)
}
