/*
******************************************************************************
* @microservice: support-notifications
* @author: Luo Qinwen, HY
* @version: 0.0.1
* @description:
职责：维护南北向映射
业务接口：
1.查询：
正向查询
反向查询
批量查询
全量查询
按NorthDeiceName查询
按NorthProfileName查询
按SouthDeviceName查询
按SouthProfileName查询
2.添加单条映射
3.更新已有映射
4.删除映射
按NorthDeiceName删除
按NorthProfileName删除
按SouthDeviceName删除
按SouthProfileName删除
5.映射配置变更监听
******************************************************************************
*/
package mappings

import (
	"context"

	"github.com/clint456/edgex-go"
	pkgHandlers "github.com/clint456/edgex-go/internal/pkg/bootstrap/handlers"
	mappingsConfig "github.com/clint456/edgex-go/internal/support/mappings/config"
	"github.com/clint456/edgex-go/internal/support/mappings/container"
	"github.com/clint456/edgex-go/internal/support/mappings/embed"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/flags"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/handlers"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/interfaces"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/startup"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/config"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"

	"github.com/labstack/echo/v4"
)

func Main(ctx context.Context, cancel context.CancelFunc, router *echo.Echo, args []string) {
	const supportMappingsServiceKey = "support-mappings"
	// 开始启动计时器
	startupTimer := startup.NewStartUpTimer(supportMappingsServiceKey)
	// 解析命令行参数
	f := flags.New()
	f.Parse(args)
	// 初始化配置结构体
	configuration := &mappingsConfig.ConfigurationStruct{}
	// 创建依赖注入容器
	dic := di.NewContainer(di.ServiceConstructorMap{
		container.ConfigurationName: func(get di.Get) interface{} {
			return configuration
		},
	})
	// 初始化 HTTP 服务器和数据库处理器
	httpServer := handlers.NewHttpServer(router, true, supportMappingsServiceKey)
	dbHandler := pkgHandlers.NewDatabase(httpServer, configuration, container.DBClientInterfaceName, embed.SchemaName,
		supportMappingsServiceKey, edgex.Version, embed.SQLFiles)
	// 运行启动流程
	bootstrap.Run(
		ctx,
		cancel,
		f,
		supportMappingsServiceKey,
		common.ConfigStemCore,
		configuration,
		startupTimer,
		dic,
		true,
		config.ServiceTypeOther,
		[]interfaces.BootstrapHandler{
			handlers.NewClientsBootstrap().BootstrapHandler, // 初始化客户端
			dbHandler.BootstrapHandler,                      // 初始化数据库
			handlers.MessagingBootstrapHandler,              // 初始化消息传递
			handlers.NewServiceMetrics(supportMappingsServiceKey).BootstrapHandler,
			NewBootstrap(router, supportMappingsServiceKey).BootstrapHandler,
			httpServer.BootstrapHandler, // 初始化 HTTP 服务器
			handlers.NewStartMessage(supportMappingsServiceKey, edgex.Version).BootstrapHandler,
		})
}
