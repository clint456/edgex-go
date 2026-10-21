//
// Copyright (C) 2020-2024 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"fmt"

	"github.com/clint456/edgex-go/internal/core/metadata/container"
	"github.com/clint456/edgex-go/internal/core/metadata/infrastructure/interfaces"
	"github.com/clint456/edgex-go/internal/pkg/correlation"
	"github.com/clint456/edgex-go/internal/pkg/utils"
	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/requests"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
)

// The AddNodeService function accepts the new node service model from the controller function
// and then invokes AddNodeService function of infrastructure layer to add new node service
func AddNodeService(d models.NodeService, ctx context.Context, dic *di.Container) (id string, err errors.EdgeX) {
	dbClient := container.DBClientFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)

	correlationId := correlation.FromContext(ctx)
	addedNodeService, err := dbClient.AddNodeService(d)
	if err != nil {
		return "", errors.NewCommonEdgeXWrapper(err)
	}

	lc.Debugf(
		"NodeService created on DB successfully. NodeService ID: %s, Correlation-ID: %s ",
		addedNodeService.Id,
		correlationId,
	)
	NodeServiceDTO := dtos.FromNodeServiceModelToDTO(d)
	go publishSystemEvent(common.NodeServiceSystemEventType, common.SystemEventActionAdd, d.Name, NodeServiceDTO, ctx, dic)
	return addedNodeService.Id, nil
}

// NodeServiceByName query the node service by name
func NodeServiceByName(name string, ctx context.Context, dic *di.Container) (nodeService dtos.NodeService, err errors.EdgeX) {
	if name == "" {
		return nodeService, errors.NewCommonEdgeX(errors.KindContractInvalid, "name is empty", nil)
	}
	dbClient := container.DBClientFrom(dic.Get)
	ds, err := dbClient.NodeServiceByName(name)
	if err != nil {
		return nodeService, errors.NewCommonEdgeXWrapper(err)
	}
	nodeService = dtos.FromNodeServiceModelToDTO(ds)
	return nodeService, nil
}

// PatchNodeService executes the PATCH operation with the node service DTO to replace the old data
func PatchNodeService(dto dtos.UpdateNodeService, ctx context.Context, dic *di.Container) errors.EdgeX {
	dbClient := container.DBClientFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)

	nodeService, err := nodeServiceByDTO(dbClient, dto)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	requests.ReplaceNodeServiceModelFieldsWithDTO(&nodeService, dto)

	err = dbClient.UpdateNodeService(nodeService)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	lc.Debugf(
		"NodeService patched on DB successfully. Correlation-ID: %s ",
		correlation.FromContext(ctx),
	)
	NodeServiceDTO := dtos.FromNodeServiceModelToDTO(nodeService)
	go publishSystemEvent(common.NodeServiceSystemEventType, common.SystemEventActionUpdate, nodeService.Name, NodeServiceDTO, ctx, dic)
	return nil
}

func nodeServiceByDTO(dbClient interfaces.DBClient, dto dtos.UpdateNodeService) (nodeService models.NodeService, err errors.EdgeX) {
	// The ID or Name is required by DTO and the DTO also accepts empty string ID if the Name is provided
	if dto.Id != nil && *dto.Id != "" {
		nodeService, err = dbClient.NodeServiceById(*dto.Id)
		if err != nil {
			return nodeService, errors.NewCommonEdgeXWrapper(err)
		}
	} else {
		nodeService, err = dbClient.NodeServiceByName(*dto.Name)
		if err != nil {
			return nodeService, errors.NewCommonEdgeXWrapper(err)
		}
	}
	if dto.Name != nil && *dto.Name != nodeService.Name {
		return nodeService, errors.NewCommonEdgeX(errors.KindContractInvalid, fmt.Sprintf("node service name '%s' not match the exsting '%s' ", *dto.Name, nodeService.Name), nil)
	}
	return nodeService, nil
}

// DeleteNodeServiceByName delete the node service by name
func DeleteNodeServiceByName(name string, ctx context.Context, dic *di.Container) errors.EdgeX {
	if name == "" {
		return errors.NewCommonEdgeX(errors.KindContractInvalid, "name is empty", nil)
	}
	dbClient := container.DBClientFrom(dic.Get)
	nodeService, err := dbClient.NodeServiceByName(name)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	// Check the associated Node and ProvisionWatcher existence
	nodes, edgeXErr := dbClient.NodesByServiceName(0, 1, name)
	if edgeXErr != nil {
		return errors.NewCommonEdgeXWrapper(edgeXErr)
	}
	if len(nodes) > 0 {
		return errors.NewCommonEdgeX(errors.KindStatusConflict, "fail to delete the node service when associated node exists", nil)
	}
	provisionWatchers, edgeXErr := dbClient.ProvisionWatchersByServiceName(0, 1, name)
	if edgeXErr != nil {
		return errors.NewCommonEdgeXWrapper(edgeXErr)
	}
	if len(provisionWatchers) > 0 {
		return errors.NewCommonEdgeX(errors.KindStatusConflict, "fail to delete the node service when associated provisionWatcher exists", nil)
	}

	err = dbClient.DeleteNodeServiceByName(name)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	NodeServiceDTO := dtos.FromNodeServiceModelToDTO(nodeService)
	go publishSystemEvent(common.NodeServiceSystemEventType, common.SystemEventActionDelete, nodeService.Name, NodeServiceDTO, ctx, dic)
	return nil
}

// AllNodeServices query the node services with labels, offset, and limit
func AllNodeServices(offset int, limit int, labels []string, ctx context.Context, dic *di.Container) (nodeServices []dtos.NodeService, totalCount int64, err errors.EdgeX) {
	dbClient := container.DBClientFrom(dic.Get)

	totalCount, err = dbClient.NodeServiceCountByLabels(labels)
	if err != nil {
		return nodeServices, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	cont, err := utils.CheckCountRange(totalCount, offset, limit)
	if !cont {
		return []dtos.NodeService{}, totalCount, err
	}

	services, err := dbClient.AllNodeServices(offset, limit, labels)
	if err != nil {
		return nodeServices, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	nodeServices = make([]dtos.NodeService, len(services))
	for i, s := range services {
		nodeServices[i] = dtos.FromNodeServiceModelToDTO(s)
	}
	return nodeServices, totalCount, nil
}
