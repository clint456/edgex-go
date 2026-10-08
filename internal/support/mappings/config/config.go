/*******************************************************************************
* @microservice: support-notifications
* @author: Luo Qinwen, HY
* @version: 0.0.1
*******************************************************************************/
package config

import (
	bootstrapConfig "github.com/edgexfoundry/go-mod-bootstrap/v4/config"
)

type ConfigurationStruct struct {
	Writable   WritableInfo
	Clients    bootstrapConfig.ClientsCollection
	Database   bootstrapConfig.Database
	Registry   bootstrapConfig.RegistryInfo
	Service    bootstrapConfig.ServiceInfo
	MessageBus bootstrapConfig.MessageBusInfo
}

type WritableInfo struct {
	LogLevel        string
	InsecureSecrets bootstrapConfig.InsecureSecrets
	Telemetry       bootstrapConfig.TelemetryInfo
	ChangeTopic     string
}

// 更新配置结构体的内容
func (c *ConfigurationStruct) UpdateFromRaw(rawConfig interface{}) bool {
	configuration, ok := rawConfig.(*ConfigurationStruct)
	if ok {
		*c = *configuration
	}
	return ok
}

// 返回 WritableInfo 的指针
func (c *ConfigurationStruct) EmptyWritablePtr() interface{} {
	return &WritableInfo{}
}

// 更新 WritableInfo 的内容
func (c *ConfigurationStruct) GetWritablePtr() any {
	return &c.Writable
}

// 更新 WritableInfo 的内容
func (c *ConfigurationStruct) UpdateWritableFromRaw(rawWritable interface{}) bool {
	writable, ok := rawWritable.(*WritableInfo)
	if ok {
		c.Writable = *writable
	}
	return ok
}

func (c *ConfigurationStruct) GetBootstrap() bootstrapConfig.BootstrapConfiguration {
	return bootstrapConfig.BootstrapConfiguration{
		Clients:    &c.Clients,
		Database:   &c.Database,
		Registry:   &c.Registry,
		Service:    &c.Service,
		MessageBus: &c.MessageBus,
	}
}

// GetLogLevel returns the current ConfigurationStruct's log level.
func (c *ConfigurationStruct) GetLogLevel() string {
	return c.Writable.LogLevel
}

// GetRegistryInfo returns the RegistryInfo from the ConfigurationStruct.
func (c *ConfigurationStruct) GetRegistryInfo() bootstrapConfig.RegistryInfo {
	return c.Registry
}

// GetDatabaseInfo returns a database information.
func (c *ConfigurationStruct) GetDatabaseInfo() bootstrapConfig.Database {
	return c.Database
}

// GetInsecureSecrets returns the service's InsecureSecrets.
func (c *ConfigurationStruct) GetInsecureSecrets() bootstrapConfig.InsecureSecrets {
	return c.Writable.InsecureSecrets
}

// GetTelemetryInfo returns the service's Telemetry settings.
func (c *ConfigurationStruct) GetTelemetryInfo() *bootstrapConfig.TelemetryInfo {
	return &c.Writable.Telemetry
}
