package conf

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/xfyun/webgate-aipaas/common"
	"github.com/xfyun/xsf/utils"
	"sync/atomic"
	"unsafe"
)

const (
	Version               = "1.1.0"
	SessionServiceVersion = "1.1.0_session"
)

type (
	Config struct {
		Auth           Auth    `toml:"auth"`
		Xsf            Xsf     `toml:"xsf"`
		Session        Session `toml:"session"`
		Mysql          Mysql   `toml:"mysql"`
		Log            Log     `toml:"log"`
		Schema         Schema  `toml:"schema"`
		EngineSchema   Schema  `toml:"engine_schema"`
		GuiderSchema   Schema  `toml:"guider_schema"`
		AppIdCloudId   Schema  `toml:"app_id_cloud_id"`
		DomainCloudId  Schema  `toml:"domain_cloud_id"`
		CategorySchema Schema  `toml:"category_schema"`
		Server         Server  `toml:"server"`
	}

	Auth struct {
		EnableAuth       bool   `toml:"enable_auth"`
		EnableAppidCheck bool   `toml:"enable_appid_check"`
		SecretKey        string `toml:"secret_key"`
		MaxDateInterval  int    `toml:"max_date_interval"`
	}
	Xsf struct {
		ServerPort     string `toml:"server_port"` //xsf server端口
		CacheService   bool   `toml:"cache_service"`
		CacheConfig    bool   `toml:"cache_config"`
		CachePath      string `toml:"cache_path"`
		CallRetry      int    `toml:"call_retry"`
		Location       string `toml:"location"`
		From           string `toml:"from"`
		EnableRespsort bool   `toml:"enable_respsort"`
		Dc             string `toml:"dc"`
		XsfLocalIp     string
		SpillEnable    bool     `toml:"spill_enable"`
		HashServices   []string `toml:"hash_services"` //使用hash调用策略 的服务，比如编排，调用时所有的请求都会hash 到同一个后端节点
	}

	Session struct {
		ScanInterver     int `toml:"scan_interver"`      // session 全局扫描间隔
		TimeoutInterver  int `toml:"timeout_interver"`   // session 等待超时|连接等待超时
		HandshakeTimeout int `toml:"handshake_timeout"`  // 握手超时
		SessionCloseWait int `toml:"session_close_wait"` // session关闭等待时间
		SessionTimeout   int `toml:"session_timeout"`    // session时长限制
		ReadTimeout      int `toml:"read_timeout"`
	}

	Mysql struct {
		Mysql string `toml:"mysql"`
	}

	Log struct {
		Level  string `toml:"level"`
		File   string `toml:"file"`
		Count  int    `toml:"count"`
		Size   int    `toml:"size"`
		Caller bool   `toml:"caller"`
		Batch  int    `toml:"batch"`
		Asyn   bool   `toml:"async"`
	}

	Schema struct {
		Enable      bool     `toml:"enable"`
		Services    []string `toml:"services"` // schema_$service.json will be load
		FileServce  string   `toml:"file_service"`
		FileVersion string   `toml:"file_version"`
		FilePrefix  string   `toml:"file_prefix"`
	}

	Center struct {
		Project      string
		Group        string
		Service      string
		Version      string
		CompanionUrl string
	}

	Server struct {
		WriteFirst       bool     `toml:"write_first"`        //write response wherever first frame has result
		Mock             bool     `toml:"mock"`               // deprecated ,use schema.mapping.mock instead
		Host             string   `toml:"host"`               // listen host ,if empty, server will listen at first net card
		Mode             string   `toml:"mode"`               // release or debug
		NetCard          string   `toml:"net_card"`           //
		Port             string   `toml:"port"`               // listen port
		PipeDepth        int      `toml:"pipe_depth"`         // deprecated ,
		PipeTimeout      int      `toml:"pipe_timeout"`       // deprecated
		EnableSonar      bool     `toml:"enable_sonar"`       // true if use sonar log
		IgnoreRespCodes  []int    `toml:"ignore_resp_codes"`  // code return from upstream will no longer return to client
		IgnoreSonarCodes []int    `toml:"ignore_sonar_codes"` // c
		MaxConn          int      `toml:"max_conn"`           //deprecated
		EnableConnLimit  bool     `toml:"enable_conn_limit"`  // deprecatedd
		AdminListen      string   `toml:"admin_listen"`       // admin api listen addr
		Cluster          string   `toml:"cluster"`
		MockService      string   `toml:"mock_service"`
		From             string   `toml:"from"`
		AppIdWhiteList   []string `toml:"app_id_white_list"`
	}
)

var (
	//Conf   Config
	Centra Center
)

// config instance 并发安全
var confInstance unsafe.Pointer

func GetConfInstance() *Config {
	return (*Config)(atomic.LoadPointer(&confInstance))
}

func UpdateConfInstance(b []byte) error {
	c := &Config{}
	_, err := toml.Decode(string(b), c)
	if err != nil {
		return err
	}
	c.Init()

	ip, _ := utils.HostAdapter(c.Server.Host, c.Server.NetCard)
	c.Server.Host = ip
	c.Xsf.XsfLocalIp = ip + ":" + c.Xsf.ServerPort
	atomic.StorePointer(&confInstance, unsafe.Pointer(c))
	return nil
}

var (
	project  = flag.String("project", "AIPaaS", "config center project")
	group    = flag.String("group", "hu", "config center group")
	service  = flag.String("service", "webgate-ws-aipaas", "config center service")
	version  = flag.String("version", "0.1.0", "deprecated | config center version")
	url      = flag.String("url", "http://10.1.87.70:6868", "config center companionUrl")
	cfg      = flag.String("cfg", "app.toml", "name of config file")
	BootMode = flag.Bool("nativeBoot", false, "boot from native config ")
)

const (
	APP_CONFIG = "app.toml"
)

func InitConf() {
	flag.Parse()
	Centra = Center{
		Project:      *project,
		Group:        *group,
		Service:      *service,
		CompanionUrl: *url,
		Version:      *version,
	}

	InitConfigFromCentra()
	conf := GetConfInstance()
	//获取本机ip
	ip, _ := utils.HostAdapter(conf.Server.Host, conf.Server.NetCard)
	conf.Server.Host = ip
	conf.Xsf.XsfLocalIp = ip + ":" + conf.Xsf.ServerPort
	fmt.Printf("init conf:%+v\n", *conf)

	common.Setenv("APP_PORT", conf.Server.Port)
	common.Setenv("APP_HOST", conf.Server.Host)
	//InitSchema()
}

//获取端口号
func (server *Server) GetPort() string {
	return server.Port
}

//func IsIgnoreRespCode(code int) bool {
//	for i:=0;i< len(Conf.Server.IgnoreRespCodes);i++{
//		if Conf.Server.IgnoreRespCodes[i] == code{
//			return true
//		}
//	}
//	return false
//}
//
//func IsIgnoreSonarCode(code int) bool {
//	for i:=0;i< len(Conf.Server.IgnoreSonarCodes);i++{
//		if Conf.Server.IgnoreSonarCodes[i] == code{
//			return true
//		}
//	}
//	return false
//}

func (c *Config) Init() {
	if c.Session.SessionTimeout <= 0 {
		c.Session.SessionTimeout = 150
	}

	if c.Session.HandshakeTimeout <= 0 {
		c.Session.HandshakeTimeout = 4
	}

	if c.Session.TimeoutInterver <= 0 {
		c.Session.TimeoutInterver = 15
	}

	if c.Session.SessionCloseWait <= 0 {
		c.Session.SessionCloseWait = 5
	}

	if c.Session.ScanInterver <= 0 {
		c.Session.ScanInterver = 30
	}
	if c.Server.Cluster == "" {
		c.Server.Cluster = "5s"
	}

	if c.Server.From == "" {
		c.Server.From = "webgate-aipaas"
	}
	if c.Session.ReadTimeout <= 0 {
		c.Session.ReadTimeout = 10
	}
	c.Server.AppIdWhiteList = append(c.Server.AppIdWhiteList, "4CC5779A")
}
