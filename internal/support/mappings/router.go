/*******************************************************************************
* @microservice: support-notifications
* @author: Luo Qinwen, HY
* @version: 0.0.1
*******************************************************************************/
package mappings

import (
	"github.com/clint456/edgex-go"
	mappingHTTP "github.com/clint456/edgex-go/internal/support/mappings/controller/http"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/controller"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/handlers"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/labstack/echo/v4"
)

func LoadRestRoutes(r *echo.Echo, dic *di.Container, serviceName string) {
	authenticationHook := handlers.AutoConfigAuthenticationFunc(dic)
	_ = controller.NewCommonController(dic, r, serviceName, edgex.Version)
	mc := mappingHTTP.NewMappingController(dic)
	const route = common.ApiBase + "/mapping"
	r.POST(route, mc.Add, authenticationHook)                       // 处理添加映射的请求
	r.PUT(route, mc.Update, authenticationHook)                     // 处理更新映射的请求
	r.GET(route+"/all", mc.All, authenticationHook)                 // 处理查询所有映射的请求
	r.GET(route+"/forward", mc.Forward, authenticationHook)         // 处理正向查询映射的请求
	r.GET(route+"/reverse", mc.Reverse, authenticationHook)         // 处理反向查询映射的请求
	r.POST(route+"/batch/:direction", mc.Batch, authenticationHook) // 处理批量查询映射的请求
	r.GET(route+"/north/device/:northDeviceName", mc.ByNorthDeviceName, authenticationHook)
	r.GET(route+"/north/profile/:northProfileName", mc.ByNorthProfileName, authenticationHook)
	r.GET(route+"/south/device/:southDeviceName", mc.BySouthDeviceName, authenticationHook)
	r.GET(route+"/south/profile/:southProfileName", mc.BySouthProfileName, authenticationHook)
	r.DELETE(route+"/north/device/:northDeviceName", mc.DeleteByNorthDeviceName, authenticationHook)
	r.DELETE(route+"/north/profile/:northProfileName", mc.DeleteByNorthProfileName, authenticationHook)
	r.DELETE(route+"/south/device/:southDeviceName", mc.DeleteBySouthDeviceName, authenticationHook)
	r.DELETE(route+"/south/profile/:southProfileName", mc.DeleteBySouthProfileName, authenticationHook)
	r.DELETE(route+"/:id", mc.Delete, authenticationHook)
}
