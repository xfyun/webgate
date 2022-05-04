package schemas

import (
	"encoding/json"
	"fmt"
	"testing"
)



func TestGetSchema(t *testing.T) {
	sc := &AISchema{}
	err := json.Unmarshal(schema_tests,sc)
	if err != nil{
		panic(err)
	}
	err = LoadSchema(sc)
	if err != nil{
		panic(err)
	}

	rm := sc.BuildResponseHeader(map[string]string{"service_id":"hello","status":"3"})
	fmt.Println(rm)
}

var (
	schema_tests = []byte(`

{
  "meta":{
    "serviceId":"xist",
    "version":"v1.0",
    "service":[
      "xist"
    ],
    "sub":"ase",
    "call":"atmos-aipaas",
    "call_type":"0",
    "hosts":"api.xf-yun.com",
    "route":"/v1/private/xist",
    "xist":{
      "input":{
        "audio":{
          "dataType":"audio"
        }
      },
      "accept":{
        "result":{
          "dataType":"text"
        }
      }
    },
    "routeKey":[],
    "build_header": true
  },
  "schemainput":{
    "type":"object",
    "properties":{
      "header":{
        "type":"object",
        "properties":{
          "directEngIp":{
            "type":"string",
            "minLength":0,
            "maxLength":1024
          },
          "app_id":{
            "type":"string",
            "minLength":0,
            "maxLength":50
          },
          "uid":{
            "type":"string",
            "minLength":0,
            "maxLength":50
          },
          "did":{
            "type":"string",
            "minLength":0,
            "maxLength":50
          },
          "imei":{
            "type":"string",
            "minLength":0,
            "maxLength":50
          },
          "imsi":{
            "type":"string",
            "minLength":0,
            "maxLength":50
          },
          "mac":{
            "type":"string",
            "minLength":0,
            "maxLength":50
          },
          "net_type":{
            "type":"string",
            "enum":[
              "wifi",
              "2G",
              "3G",
              "4G",
              "5G"
            ]
          },
          "net_isp":{
            "type":"string",
            "enum":[
              "CMCC",
              "CUCC",
              "CTCC",
              "other"
            ]
          },
          "status":{
            "type":"integer",
            "enum":[
              0,
              1,
              2
            ]
          },
          "request_id":{
            "type":"string",
            "minLength":0,
            "maxLength":64
          }
        },
        "required":[
          "app_id",
          "status"
        ]
      },
      "parameter":{
        "type":"object",
        "properties":{
          "xist":{
            "type":"object",
            "properties":{
              "dwa": {"type": "string"},
              "eos": {"type": "integer"},
              "rf": {"type": "string"},
              "pd": {"type": "string"},
              "res_id": {"type": "string"},
              "vto": {"type": "integer"},
              "punc": {"type": "integer"},
              "nunum": {"type": "integer"},
              "pptaw": {"type": "integer"},
              "dyhotws": {"type": "integer"},
              "seg_max": {"type": "integer"},
              "spkdia": {"type": "integer"},
              "pgsnum": {"type": "integer"},
              "vad_mdn": {"type": "integer"},
              "language_type": {"type": "integer"},
              "feature_list": {"type": "string"},
              "dhw": {"type": "string"},
              "rsgid": {"type": "integer"},
              "rlang": {"type": "integer"},
              "dhw_mod": {"type": "integer"},
              "seg_min": {"type": "integer"},
              "seg_weight": {"type": "number"},
              "personalization": {"type": "object","properties": {"PERSONAL": {"type": "string"},"LM": {"type":"string"}}},
              "result":{
                "type":"object",
                "properties":{
                  "encoding":{
                    "type":"string",
                    "enum":[
                      "utf8",
                      "gb2312"
                    ]
                  },
                  "compress":{
                    "type":"string",
                    "enum":[
                      "raw",
                      "gzip"
                    ]
                  },
                  "format":{
                    "type":"string",
                    "enum":[
                      "plain",
                      "json",
                      "xml"
                    ]
                  }
                }
              }
            }
          }
        }
      },
      "payload":{
        "type":"object",
        "properties":{
          "audio":{
            "type":"object",
            "properties":{
              "encoding":{
                "type":"string",
                "enum":[
                  "lame",
                  "speex",
                  "opus",
                  "opus-wb",
                  "speex-wb",
                  "raw",
                  "ico",
                  "pcm"
                ]
              },
              "sample_rate":{
                "type":"integer",
                "enum":[
                  16000,
                  8000
                ]
              },
              "channels":{
                "type":"integer",
                "enum":[
                  1,
                  2
                ]
              },
              "bit_depth":{
                "type":"integer",
                "enum":[
                  16,
                  8
                ]
              },
              "status":{
                "type":"integer",
                "enum":[
                  0,
                  1,
                  2
                ]
              },
              "seq":{
                "type":"integer",
                "minimum":0,
                "maximum":9999999
              },
              "audio":{
                "type":"string",
                "minLength":0,
                "maxLength":10485760
              },
              "frame_size":{
                "type":"integer",
                "minimum":0,
                "maximum":1024
              }
            }
          }
        }
      }
    }
  },
  "schemaoutput":{
    "type":"object",
    
    "properties":{
      "header": {
        "type": "object",
        "properties": {
          "task_id": {
            "type": "string"
          },
			"status":{
				"type":"int"
			}
        }
      },
      "payload":{
        "type":"object",
        "properties":{
          "result":{
            "type":"object",
            "properties":{
              "encoding":{
                "type":"string",
                "enum":[
                  "utf8",
                  "gb2312"
                ]
              },
              "compress":{
                "type":"string",
                "enum":[
                  "raw",
                  "gzip"
                ]
              },
              "format":{
                "type":"string",
                "enum":[
                  "plain",
                  "json",
                  "xml"
                ]
              },
              "status":{
                "type":"integer",
                "enum":[
                  0,
                  1,
                  2
                ]
              },
              "seq":{
                "type":"integer",
                "minimum":0,
                "maximum":9999999
              },
              "text":{
                "type":"string",
                "minLength":0,
                "maxLength":1048576
              }
            }
          }
        }
      }
    }
  }
}


`)
)
