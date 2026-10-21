/*******************************************************************************
 * Copyright 2017 Dell Inc.
 * Copyright (c) 2019 Intel Corporation
 * Copyright (C) 2023 IOTech Ltd
 * Copyright (C) 2025 IOTech Ltd
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 *******************************************************************************/

package data

import (
	"context"
	"sync"

	"github.com/clint456/edgex-go/internal/core/data/application"
	dataCache "github.com/clint456/edgex-go/internal/core/data/cache"
	"github.com/clint456/edgex-go/internal/core/data/container"
	"github.com/clint456/edgex-go/internal/core/data/controller/messaging"
	"github.com/clint456/edgex-go/internal/pkg/cache"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/startup"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"

	"github.com/labstack/echo/v4"
)

// Bootstrap contains references to dependencies required by the BootstrapHandler.
type Bootstrap struct {
	router      *echo.Echo
	serviceName string
}

// NewBootstrap is a factory method that returns an initialized Bootstrap receiver struct.
func NewBootstrap(router *echo.Echo, serviceName string) *Bootstrap {
	return &Bootstrap{
		router:      router,
		serviceName: serviceName,
	}
}

// BootstrapHandler fulfills the BootstrapHandler contract and performs initialization needed by the data service.
func (b *Bootstrap) BootstrapHandler(ctx context.Context, wg *sync.WaitGroup, startupTimer startup.Timer, dic *di.Container) bool {
	LoadRestRoutes(b.router, dic, b.serviceName)

	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	app := application.CoreDataAppFrom(dic.Get)

	err := messaging.SubscribeEvents(ctx, dic)
	if err != nil {
		lc.Errorf("Failed to subscribe events from message bus, %v", err)
		return false
	}
	err = messaging.SubscribeSystemEvents(ctx, dic)
	if err != nil {
		lc.Errorf("Failed to subscribe system events from message bus, %v", err)
		return false
	}

	err = initNodeCache(ctx, dic)
	if err != nil {
		lc.Errorf("Failed to init node cache, %v", err)
		return false
	}

	if err = initNodeInfoCache(dic); err != nil {
		lc.Errorf("Failed to init node info cache, %v", err)
		return false
	}

	err = app.AsyncPurgeEvent(ctx, dic)
	if err != nil {
		lc.Errorf("Failed to run event purging process, %v", err)
	}

	return true
}

func initNodeCache(ctx context.Context, dic *di.Container) errors.EdgeX {
	dc := bootstrapContainer.NodeClientFrom(dic.Get)
	if dc == nil {
		return errors.NewCommonEdgeX(errors.KindServerError, "nil NodeClient returned", nil)
	}
	nodeStore := cache.NodeStore(dic)
	dic.Update(di.ServiceConstructorMap{
		container.NodeStoreInterfaceName: func(get di.Get) interface{} {
			return nodeStore
		},
	})

	nodes, err := dc.AllNodes(ctx, nil, 0, -1)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	for _, d := range nodes.Nodes {
		nodeStore.Add(dtos.ToNodeModel(d))
	}
	return nil
}

func initNodeInfoCache(dic *di.Container) errors.EdgeX {
	dbClient := container.DBClientFrom(dic.Get)
	nodeInfos, err := dbClient.AllNodeInfos(0, -1)
	if err != nil {
		return errors.NewCommonEdgeX(errors.Kind(err), "failed to get node infos from db", err)
	}

	nodeInfoCache := dataCache.NewNodeInfoCache(dic, nodeInfos)
	dic.Update(di.ServiceConstructorMap{
		container.NodeInfoCacheInterfaceName: func(get di.Get) interface{} {
			return nodeInfoCache
		},
	})

	return nil
}
