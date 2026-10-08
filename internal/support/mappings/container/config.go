package container

import (
	"github.com/clint456/edgex-go/internal/support/mappings/config"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
)

// 获取 ConfigurationStruct 的名称
var ConfigurationName = di.TypeInstanceToName(config.ConfigurationStruct{})

// 获取 ConfigurationStruct 的实例
func ConfigurationFrom(get di.Get) *config.ConfigurationStruct {
	return get(ConfigurationName).(*config.ConfigurationStruct)
}
