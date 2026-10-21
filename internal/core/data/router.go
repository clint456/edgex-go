//
// Copyright (C) 2021-2025 IOTech Ltd
// Copyright (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package data

import (
	"github.com/clint456/edgex-go"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/controller"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/handlers"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"

	dataController "github.com/clint456/edgex-go/internal/core/data/controller/http"

	"github.com/labstack/echo/v4"
)

func LoadRestRoutes(r *echo.Echo, dic *di.Container, serviceName string) {
	authenticationHook := handlers.AutoConfigAuthenticationFunc(dic)

	// Common
	_ = controller.NewCommonController(dic, r, serviceName, edgex.Version)

	// Events
	ec := dataController.NewEventController(dic)
	r.POST(common.ApiEventServiceNameProfileNameNodeNameSourceNameRoute, ec.AddEvent, authenticationHook)
	r.GET(common.ApiEventIdRoute, ec.EventById, authenticationHook)
	r.DELETE(common.ApiEventIdRoute, ec.DeleteEventById, authenticationHook)
	r.GET(common.ApiEventCountRoute, ec.EventTotalCount, authenticationHook)
	r.GET(common.ApiEventCountByNodeNameRoute, ec.EventCountByNodeName, authenticationHook)
	r.GET(common.ApiAllEventRoute, ec.AllEvents, authenticationHook)
	r.GET(common.ApiEventByNodeNameRoute, ec.EventsByNodeName, authenticationHook)
	r.DELETE(common.ApiEventByNodeNameRoute, ec.DeleteEventsByNodeName, authenticationHook)
	r.GET(common.ApiEventByTimeRangeRoute, ec.EventsByTimeRange, authenticationHook)
	r.DELETE(common.ApiEventByAgeRoute, ec.DeleteEventsByAge, authenticationHook)

	// Readings
	rc := dataController.NewReadingController(dic)
	r.GET(common.ApiReadingCountRoute, rc.ReadingTotalCount, authenticationHook)
	r.GET(common.ApiAllReadingRoute, rc.AllReadings, authenticationHook)
	r.GET(common.ApiReadingByNodeNameRoute, rc.ReadingsByNodeName, authenticationHook)
	r.GET(common.ApiReadingByTimeRangeRoute, rc.ReadingsByTimeRange, authenticationHook)
	r.GET(common.ApiReadingByResourceNameRoute, rc.ReadingsByResourceName, authenticationHook)
	r.GET(common.ApiReadingCountByNodeNameRoute, rc.ReadingCountByNodeName, authenticationHook)
	r.GET(common.ApiReadingByResourceNameAndTimeRangeRoute, rc.ReadingsByResourceNameAndTimeRange, authenticationHook)
	r.GET(common.ApiReadingByNodeNameAndResourceNameRoute, rc.ReadingsByNodeNameAndResourceName, authenticationHook)
	r.GET(common.ApiReadingByNodeNameAndResourceNameAndTimeRangeRoute, rc.ReadingsByNodeNameAndResourceNameAndTimeRange, authenticationHook)
	r.GET(common.ApiReadingByNodeNameAndTimeRangeRoute, rc.ReadingsByNodeNameAndResourceNamesAndTimeRange, authenticationHook)
}
