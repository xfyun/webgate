package conf

import (
	"fmt"
	finder "github.com/xfyun/finder-go"
	common "github.com/xfyun/finder-go/common"
	com "github.com/xfyun/webgate-aipaas/common"
	"github.com/xfyun/webgate-aipaas/schemas"
	"os"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

var configHandler []func(string, []byte) bool

func AddConfigChangerHander(f func(string, []byte) bool) {
	configHandler = append(configHandler, f)
}

var (
	findlerMamager      *finder.FinderManager
	schemaFinderManager *finder.FinderManager
	configChangeHandler common.ConfigChangedHandler
	SFManager           *SesssionFinderManager = &SesssionFinderManager{}
)

const ()

//集成配置中心与服务发现
func InitConfigFromCentra() (b []byte) {
	cachePath, err := os.Getwd()
	if err != nil {
		return
	}

	//缓存信息的存放路径
	cachePath += "/findercache"
	config := common.BootConfig{
		//companion地址
		CompanionUrl: Centra.CompanionUrl,
		//缓存路径
		CachePath: cachePath,
		//是否缓存服务信息
		CacheService: true,
		//是否缓存配置信息
		CacheConfig:   true,
		ExpireTimeout: 5 * time.Second,
		MeteData: &common.ServiceMeteData{
			Project: Centra.Project,
			Group:   Centra.Group,
			Service: Centra.Service,
			Version: Centra.Version,
			Address: "",
		},
	}

	f, err := finder.NewFinderWithLogger(config, nil)

	if err != nil {
		panic("init finder manager error:" + err.Error())
	}
	configChangeHandler = &ConfigChangedHandle{}
	findlerMamager = f
	subRes, err := f.ConfigFinder.UseAndSubscribeConfig([]string{APP_CONFIG}, configChangeHandler)
	if err != nil {
		panic("subscribe file err:" + err.Error() + " file:" + APP_CONFIG + "|")
	}
	conf := subRes[APP_CONFIG]
	b = conf.File
	//加载配置文件
	if err := UpdateConfInstance(b); err != nil {
		panic("cannot load app.config:" + err.Error())
	}

	confInst := GetConfInstance()
	//初始化XsfLog
	if err := com.UpdateLogger(confInst.Log.File, confInst.Log.Level, confInst.Log.Size, confInst.Log.Count, confInst.Log.Caller, confInst.Log.Batch, confInst.Log.Asyn); err != nil {
		panic(err)
	}

	LoadSchemaConf("schema", confInst.Schema.FileServce, confInst.Schema.FileVersion, confInst.Schema.FilePrefix, func(b []byte) error {
		return schemas.LoadAISchema(b)
	})
	//LoadSchemaConf("guider-schema", confInst.GuiderSchema.FileServce, confInst.GuiderSchema.FileVersion, confInst.GuiderSchema.FilePrefix, func(b []byte) error {
	//	return schemas.LoadGuiderSchema(b)
	//})
	//LoadSchemaConf("engine_schema", confInst.EngineSchema.FileServce, confInst.EngineSchema.FileVersion, confInst.EngineSchema.FilePrefix, func(b []byte) error {
	//	return schemas.LoadEngineSchema(b)
	//})
	////
	//LoadSchemaConf("domain cloud_id ref", confInst.DomainCloudId.FileServce, confInst.DomainCloudId.FileVersion, confInst.DomainCloudId.FilePrefix, func(b []byte) error {
	//	return schemas.UpdateCloudRef(b)
	//})
	//LoadSchemaConf("app_id cloud_id ref", confInst.AppIdCloudId.FileServce, confInst.AppIdCloudId.FileVersion, confInst.AppIdCloudId.FilePrefix, func(b []byte) error {
	//	return schemas.UpdateAppConf(b)
	//})
	//
	//LoadSchemaConf("category_schema", confInst.CategorySchema.FileServce, confInst.CategorySchema.FileVersion, confInst.CategorySchema.FilePrefix, func(b []byte) error {
	//	return schemas.LoadAISchema(b)
	//})
	return
}

type loadSchemaFunc func(b []byte) error

func LoadSchemaConf(name, service string, version string, prefix string, load loadSchemaFunc) {
	cachePath, err := os.Getwd()
	if err != nil {
		return
	}
	//缓存信息的存放路径
	cachePath += "/findercache"
	config := common.BootConfig{
		//companion地址
		CompanionUrl: Centra.CompanionUrl,
		//缓存路径
		CachePath: cachePath,
		//是否缓存服务信息
		CacheService: true,
		//是否缓存配置信息
		CacheConfig:   true,
		ExpireTimeout: 5 * time.Second,
		MeteData: &common.ServiceMeteData{
			Project: Centra.Project,
			Group:   Centra.Group,
			Service: service,
			Version: version,
			Address: "",
		},
	}

	//common.ConfigChangedHandler()
	//cfg:= GetConfInstance()
	f, err := finder.NewFinderWithLogger(config, nil)
	if err != nil {
		panic("init finder manager error:" + err.Error() + fmt.Sprintf("%s: service=%s,version=%s", name, service, version))
	}
	schemaFinderManager = f

	files, err := f.ConfigFinder.UseAndSubscribeWithPrefix(prefix, &SchemaConfigChangedHandle{load: load, prefix: prefix})
	if err != nil {
		panic(fmt.Sprintf("%s /%s/%s/%s/%s ：%v", name, *project, *group, service, version, err.Error()))
	}
	// 加载schema
	for _, file := range files {
		//fmt.Println(name,"file",file)
		if file == nil {
			continue
		}
		err := load(file.File)
		if err != nil {
			//panic("load schema file error:file="+file.Name+"error="+err.Error())
			fmt.Println(time.Now().Format(time.RFC3339), name, "load schema file error:file="+file.Name+"error="+err.Error(), "file=", string(file.File))
			continue
		} else {
			fmt.Println(time.Now().Format(time.RFC3339), name, "success load schema:", file.Name)
		}
	}
}

func ConsoleError(v interface{}) {
	fmt.Println("ERROR:", time.Now().Format(time.RFC3339), v)
}

func ConsoleWarn(v interface{}) {
	fmt.Println("WARN:", time.Now().Format(time.RFC3339), v)
}

var localFinderAddr string

func InitSessionServiceFind(port int) {
	conf := GetConfInstance()
	initSessionServiceFind(*service, conf.Server.Host, port)
	localFinderAddr = fmt.Sprintf("%s:%d", conf.Server.Host, port)
}

func initSessionServiceFind(service, host string, port int) {
	err := findlerMamager.ServiceFinder.RegisterServiceWithAddr(fmt.Sprintf("%s:%d", host, port), SessionServiceVersion)
	if err != nil {
		ConsoleError("register session find service error")
	} else {
		fmt.Println("register session service:", service, host, port)
	}

	srv, err := findlerMamager.ServiceFinder.UseAndSubscribeService([]common.ServiceSubscribeItem{
		{ServiceName: service, ApiVersion: SessionServiceVersion},
	}, &ServiceChangedHandle{})
	if err != nil {
		fmt.Println("subscribe service error", err)
		return
	}
	var addr []string
	for _, s := range srv {
		for _, v := range s.ProviderList {
			addr = append(addr, v.Addr)
		}
	}
	SFManager.UpdateAddr(addr)
	fmt.Println("subscribe addrs:", SFManager.GetAddrs())
}
func IsSchemaFile(key string) bool {
	return strings.HasPrefix(key, GetConfInstance().Schema.FilePrefix)
}

// ConfigChangedHandle ConfigChangedHandle
type ConfigChangedHandle struct {
}

// OnConfigFileChanged OnConfigFileChanged
func (s *ConfigChangedHandle) OnConfigFileChanged(config *common.Config) bool {
	if config.Name == APP_CONFIG {
		err := UpdateConfInstance(config.File)
		if err != nil {
			com.GetLoggerInstance().Errorw("reload config error", "error", err.Error())
			return false
		} else {
			com.GetLoggerInstance().Errorw("success load update config:app.toml")
		}
	}
	if configHandler != nil {
		for _, f := range configHandler {
			if !f(config.Name, config.File) {
				return false
			}
		}

	}
	return true
}

func (s *ConfigChangedHandle) OnError(errInfo common.ConfigErrInfo) {
	fmt.Println("配置文件出错：", errInfo)
}

//todo
func (s *ConfigChangedHandle) OnConfigFilesAdded(configs map[string]*common.Config) bool {

	return true
}

func (s *ConfigChangedHandle) OnConfigFilesRemoved(configNames []string) bool {

	return true
}

type ServiceChangedHandle struct {
}

// OnServiceInstanceConfigChanged OnServiceInstanceConfigChanged
func (s *ServiceChangedHandle) OnServiceInstanceConfigChanged(name string, apiVersion string, instance string, config *common.ServiceInstanceConfig) bool {
	fmt.Println("time", time.Now().Format(time.RFC1123))
	fmt.Println("服务实例配置信息更改开始，服务名：", name, "  版本号：", apiVersion, "  提供者实例为：", instance)
	fmt.Println("----当前配置为:  ", config.IsValid, "  ", config.UserConfig)
	fmt.Println("服务实例配置信息更改结束, 服务名：", name, "  版本号：", apiVersion, "  提供者实例为：", instance)
	config.IsValid = false
	config.UserConfig = "aasasasasasasa"
	config = nil
	return true
}

// OnServiceConfigChanged OnServiceConfigChanged
func (s *ServiceChangedHandle) OnServiceConfigChanged(name string, apiVersion string, config *common.ServiceConfig) bool {
	fmt.Println("time", time.Now().Format(time.RFC1123))
	fmt.Println("服务配置信息更改开始，服务名：", name, "  版本号：", apiVersion)
	fmt.Println("-----当前配置为: ", config.JsonConfig)
	fmt.Println("服务配置信息更改结束, 服务名：", name, "  版本号：", apiVersion)
	config.JsonConfig = "zyssss"
	config = nil
	return true
}

// OnServiceInstanceChanged OnServiceInstanceChanged
func (s *ServiceChangedHandle) OnServiceInstanceChanged(name string, apiVersion string, eventList []*common.ServiceInstanceChangedEvent) bool {
	fmt.Println("time", time.Now().Format(time.RFC1123))
	fmt.Println("服务实例变化通知开始, 服务名：", name, "  版本号：", apiVersion)
	addr := SFManager.GetAddrs()
	addrs := addr
	for eventIndex, e := range eventList {
		for _, inst := range e.ServerList {
			if e.EventType == common.INSTANCEREMOVE {
				fmt.Println("----服务提供者节点减少事件 ：", e.ServerList)
				fmt.Println("-----------减少的服务提供者节点信息:  ")
				fmt.Println("----------------------- 地址: ", inst.Addr)
				fmt.Println("----------------------- 是否有效: ", inst.Config.IsValid)
				fmt.Println("----------------------- 配置: ", inst.Config.UserConfig)

			} else {
				fmt.Println("----服务提供者节点增加事件 ：", e.ServerList)
				fmt.Println("-----------增加的服务提供者节点信息:  ")
				fmt.Println("----------------------- 地址: ", inst.Addr)
				fmt.Println("----------------------- 是否有效: ", inst.Config.IsValid)
				fmt.Println("----------------------- 配置: ", inst.Config.UserConfig)

			}

		}
		eventList[eventIndex] = nil
		switch e.EventType {
		case common.INSTANCEADDED:
			for i := 0; i < len(addr); i++ {
				for _, sl := range e.ServerList {
					addrs = append(addrs, sl.Addr)
				}
			}
		case common.INSTANCEREMOVE:
			for i := 0; i < len(addr); i++ {
				for _, sl := range e.ServerList {
					if sl.Addr == addr[i] {
						addrs = append(addrs[:i], addrs[i+1:]...)
					}
				}
			}

		}
	}
	SFManager.UpdateAddr(addrs)

	fmt.Println("服务实例变化通知结束, 服务名：", name, "  版本号：", apiVersion)
	return true
}

type SesssionFinderManager struct {
	activeAddr unsafe.Pointer
}

func (s *SesssionFinderManager) GetAddrs() []string {
	if s.activeAddr == nil {
		return []string{}
	}
	return *(*[]string)(atomic.LoadPointer(&s.activeAddr))
}

func (s *SesssionFinderManager) UpdateAddr(addr []string) {
	atomic.StorePointer(&s.activeAddr, unsafe.Pointer(&addr))
}

// 获取当前sid所在的地址集合。
func (s *SesssionFinderManager) GetSessionAddr() []string {
	return s.GetAddrs()
}

func GetSessionAddr() []string {
	return SFManager.GetSessionAddr()
}

func IsLocalAddr(addr string) bool {
	//local:=Conf.Server.Host
	//for i:=0 ;i< len(addr) ;i++{
	//	if i>= len(local){
	//		return true
	//	}
	//	if local[i] !=addr[i]{
	//		return false
	//	}
	//}
	return localFinderAddr == addr
}
