package conf

import (
	"fmt"
	common "github.com/xfyun/finder-go/common"
	com "github.com/xfyun/webgate-aipaas/common"
	"strings"
)

type SchemaConfigChangedHandle struct {
	load   loadSchemaFunc
	prefix string
}

// OnConfigFileChanged OnConfigFileChanged
func (s *SchemaConfigChangedHandle) OnConfigFileChanged(config *common.Config) bool {
	if config == nil {
		return true
	}
	if !strings.HasPrefix(config.Name, s.prefix) {
		return true
	}
	err := s.load(config.File)
	if err != nil {
		com.GetLoggerInstance().Errorw("cannot load changed schema:", "name", config.Name, "error", err, "content", string(config.File))
		return false
	} else {
		com.GetLoggerInstance().Infow("success load changed schema:", "name", config.Name, "content", string(config.File))
	}

	return true
}

func (s *SchemaConfigChangedHandle) OnError(errInfo common.ConfigErrInfo) {
	fmt.Println("配置文件出错：", errInfo)
}

//todo
func (s *SchemaConfigChangedHandle) OnConfigFilesAdded(configs map[string]*common.Config) bool {
	ok := true
	for name, file := range configs {
		if file == nil {
			continue
		}
		if strings.HasPrefix(name, s.prefix) {
			err := s.load(file.File)
			if err != nil {
				com.GetLoggerInstance().Errorw("cannot load new schema:", "name", name, "error", err, "content", string(file.File))
				ok = false
			} else {
				com.GetLoggerInstance().Infow("success load new schema", "name", name, "content", string(file.File))
			}

		}
	}
	return ok
}

func (s *SchemaConfigChangedHandle) OnConfigFilesRemoved(configNames []string) bool {

	return true
}
