package schemas

import (
	"encoding/json"
	"github.com/xfyun/webgate-aipaas/common"
	"sync/atomic"
)

/**
couldId 和 域名关系映射
*/
type AppCloud struct {
	AppId   string `json:"app_id"`
	CloudId string `json:"cloud_id"`
}

//no lock
type CloudDomainRef struct {
	refConfig atomic.Value // map[string]string   domain: clouldId
	appConfig atomic.Value //map[string][]string app_id: cloudIds
}

func (c *CloudDomainRef) getRefConfig() map[string]string {
	cfg, ok := c.refConfig.Load().(map[string]string)
	if !ok {
		return nil
	}
	return cfg
}

func (c *CloudDomainRef) getAppConf() map[string][]string {
	cfg, ok := c.appConfig.Load().(map[string][]string)
	if !ok {
		return nil
	}
	return cfg
}

func (c *CloudDomainRef) GetAppIdCloud(appid string) []string {
	cfg := c.getAppConf()
	if cfg == nil {
		return nil
	}
	return cfg[appid]
}

func (c *CloudDomainRef) GetCloudId(domain string) string {
	cfg := c.getRefConfig()
	if cfg != nil {
		return cfg[domain]
	}
	return ""
}

//@return1 cloudId
//@return2 是否校验通过
func (c *CloudDomainRef) CheckAppId(appid string, domain string, whiteList ...string) (string, bool) {
	cloudId := c.GetCloudId(domain) // domain 所属的cloud Id
	if cloudId == "" || cloudId == "0" {
		return "", true // 没有找到cloudId ，不属于专有云，直接放过
	}

	for _, wappid := range whiteList {
		if appid == wappid {
			return cloudId, true
		}
	}

	clds := c.GetAppIdCloud(appid)

	for _, cld := range clds {
		if cld == cloudId {
			return cloudId, true
		}
	}
	return cloudId, false
}

func (c *CloudDomainRef) UpdateRef(b []byte) error {
	data := map[string][]string{}
	err := json.Unmarshal(b, &data)
	if err != nil {
		return err
	}

	cfg := make(map[string]string)

	for clouldId, domains := range data {
		for _, domain := range domains {
			cfg[domain] = clouldId
		}
	}

	common.GetLoggerInstance().Warnw("success update cloud domain ref", "conf", cfg)
	c.refConfig.Store(cfg)
	return nil
}

func (c *CloudDomainRef) UpdateAppConf(data []byte) error {
	cfg := make([]AppCloud, 0, 100)
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	appConf := make(map[string][]string)

	for _, cc := range cfg {
		apids, ok := appConf[cc.AppId]
		if ok {
			apids = append(apids, cc.CloudId)
		} else {
			apids = []string{cc.CloudId}
		}
		appConf[cc.AppId] = apids
	}
	c.appConfig.Store(appConf)
	common.GetLoggerInstance().Warnw("success update appid cloudid ref", "conf", appConf)
	return nil
}

var appCloudInst = &CloudDomainRef{}

func UpdateCloudRef(b []byte) error {
	return appCloudInst.UpdateRef(b)
}

func UpdateAppConf(b []byte) error {
	return appCloudInst.UpdateAppConf(b)
}

func CheckAppIdAndCloudId(appid string, domain string, whiteList ...string) (string, bool) {
	return appCloudInst.CheckAppId(appid, domain, whiteList...)
}
