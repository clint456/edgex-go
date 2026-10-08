/*******************************************************************************
* @microservice: support-notifications
* @author: Luo Qinwen, HY
* @version: 0.0.1
*******************************************************************************/
package mappings

import (
	"context"
	"sync"

	"github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/startup"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/labstack/echo/v4"
)

type Bootstrap struct {
	router      *echo.Echo
	serviceName string
}

func NewBootstrap(router *echo.Echo, serviceName string) *Bootstrap {
	return &Bootstrap{
		router:      router,
		serviceName: serviceName,
	}
}

func (b *Bootstrap) BootstrapHandler(ctx context.Context, wg *sync.WaitGroup, _ startup.Timer, dic *di.Container) bool {
	LoadRestRoutes(b.router, dic, b.serviceName)

	// 添加其他自定义启动逻辑

	return true
}
