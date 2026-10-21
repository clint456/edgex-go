//
// Copyright (C) 2021-2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	commandContainer "github.com/clint456/edgex-go/internal/core/command/container"
	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	commonDTO "github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/responses"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
)

// AllCommands query commands by offset, and limit
func AllCommands(offset int, limit int, dic *di.Container) (nodeCoreCommands []dtos.NodeCoreCommand, totalCount int64, err errors.EdgeX) {
	// retrieve node information through Metadata NodeClient
	dc := bootstrapContainer.NodeClientFrom(dic.Get)
	if dc == nil {
		return nodeCoreCommands, totalCount, errors.NewCommonEdgeX(errors.KindServerError, "nil NodeClient returned", nil)
	}
	multiNodesResponse, err := dc.AllNodes(context.Background(), nil, offset, limit)
	if err != nil {
		return nodeCoreCommands, totalCount, errors.NewCommonEdgeXWrapper(err)
	}

	// retrieve node profile information through Metadata ProfileClient
	dpc := bootstrapContainer.ProfileClientFrom(dic.Get)
	if dpc == nil {
		return nodeCoreCommands, totalCount, errors.NewCommonEdgeX(errors.KindServerError, "nil ProfileClient returned", nil)
	}

	// Prepare the url for command
	configuration := commandContainer.ConfigurationFrom(dic.Get)
	serviceUrl := configuration.Service.Url()

	for _, node := range multiNodesResponse.Nodes {
		if len(node.ProfileName) == 0 {
			// if the profile is not set, skip the profile query
			continue
		}
		profileResponse, err := dpc.ProfileByName(context.Background(), node.ProfileName)
		if err != nil {
			return nodeCoreCommands, totalCount, errors.NewCommonEdgeXWrapper(err)
		}
		commands, err := dtos.BuildCoreCommands(node.Name, serviceUrl, profileResponse.Profile)
		if err != nil {
			return nil, totalCount, errors.NewCommonEdgeXWrapper(err)
		}
		nodeCoreCommands = append(nodeCoreCommands, dtos.NodeCoreCommand{
			NodeName:     node.Name,
			ProfileName:  node.ProfileName,
			CoreCommands: commands,
		})
	}
	return nodeCoreCommands, multiNodesResponse.TotalCount, nil
}

// IssueGetCommandByName issues the specified get(read) command referenced by the command name to the node/sensor, also
// referenced by name.
func IssueGetCommandByName(nodeName string, commandName string, queryParams string, dic *di.Container) (res *responses.EventResponse, err errors.EdgeX) {
	if nodeName == "" {
		return res, errors.NewCommonEdgeX(errors.KindContractInvalid, "node name cannot be empty", nil)
	}

	if commandName == "" {
		return res, errors.NewCommonEdgeX(errors.KindContractInvalid, "command name cannot be empty", nil)
	}

	// retrieve node information through Metadata NodeClient
	dc := bootstrapContainer.NodeClientFrom(dic.Get)
	if dc == nil {
		return res, errors.NewCommonEdgeX(errors.KindServerError, "nil NodeClient returned", nil)
	}
	nodeResponse, err := dc.NodeByName(context.Background(), nodeName)
	if err != nil {
		return res, errors.NewCommonEdgeXWrapper(err)
	}

	// retrieve node service information through Metadata NodeClient
	dsc := bootstrapContainer.NodeServiceClientFrom(dic.Get)
	if dsc == nil {
		return res, errors.NewCommonEdgeX(errors.KindServerError, "nil NodeServiceClient returned", nil)
	}
	nodeServiceResponse, err := dsc.NodeServiceByName(context.Background(), nodeResponse.Node.ServiceName)
	if err != nil {
		return res, errors.NewCommonEdgeXWrapper(err)
	}

	// Issue command by passing the base address of node service into NodeServiceCommandClient
	dscc := bootstrapContainer.NodeServiceCommandClientFrom(dic.Get)
	if dscc == nil {
		return res, errors.NewCommonEdgeX(errors.KindServerError, "nil NodeServiceCommandClient returned", nil)
	}
	res, err = dscc.GetCommand(context.Background(), nodeServiceResponse.Service.BaseAddress, nodeName, commandName, queryParams)
	if err != nil {
		return res, errors.NewCommonEdgeXWrapper(err)
	}

	return res, nil
}

// IssueSetCommandByName issues the specified set(write) command referenced by the command name to the node/sensor, also
// referenced by name.
func IssueSetCommandByName(nodeName string, commandName string, queryParams string, settings map[string]interface{}, dic *di.Container) (response commonDTO.BaseResponse, err errors.EdgeX) {
	if nodeName == "" {
		return response, errors.NewCommonEdgeX(errors.KindContractInvalid, "node name cannot be empty", nil)
	}

	if commandName == "" {
		return response, errors.NewCommonEdgeX(errors.KindContractInvalid, "command name cannot be empty", nil)
	}

	// retrieve node information through Metadata NodeClient
	dc := bootstrapContainer.NodeClientFrom(dic.Get)
	if dc == nil {
		return response, errors.NewCommonEdgeX(errors.KindServerError, "nil NodeClient returned", nil)
	}
	nodeResponse, err := dc.NodeByName(context.Background(), nodeName)
	if err != nil {
		return response, errors.NewCommonEdgeXWrapper(err)
	}

	// retrieve node service information through Metadata NodeClient
	dsc := bootstrapContainer.NodeServiceClientFrom(dic.Get)
	if dsc == nil {
		return response, errors.NewCommonEdgeX(errors.KindServerError, "nil NodeServiceClient returned", nil)
	}
	nodeServiceResponse, err := dsc.NodeServiceByName(context.Background(), nodeResponse.Node.ServiceName)
	if err != nil {
		return response, errors.NewCommonEdgeXWrapper(err)
	}

	// Issue command by passing the base address of node service into NodeServiceCommandClient
	dscc := bootstrapContainer.NodeServiceCommandClientFrom(dic.Get)
	if dscc == nil {
		return response, errors.NewCommonEdgeX(errors.KindServerError, "nil NodeServiceCommandClient returned", nil)
	}
	return dscc.SetCommandWithObject(context.Background(), nodeServiceResponse.Service.BaseAddress, nodeName, commandName, queryParams, settings)
}

// CommandsByNodeName query coreCommands with node name
func CommandsByNodeName(name string, dic *di.Container) (nodeCoreCommand dtos.NodeCoreCommand, err errors.EdgeX) {
	if name == "" {
		return nodeCoreCommand, errors.NewCommonEdgeX(errors.KindContractInvalid, "node name is empty", nil)
	}

	// retrieve node information through Metadata NodeClient
	dc := bootstrapContainer.NodeClientFrom(dic.Get)
	if dc == nil {
		return nodeCoreCommand, errors.NewCommonEdgeX(errors.KindServerError, "nil NodeClient returned", nil)
	}
	nodeResponse, err := dc.NodeByName(context.Background(), name)
	if err != nil {
		return nodeCoreCommand, errors.NewCommonEdgeXWrapper(err)
	}

	// retrieve node profile information through Metadata ProfileClient
	dpc := bootstrapContainer.ProfileClientFrom(dic.Get)
	if dpc == nil {
		return nodeCoreCommand, errors.NewCommonEdgeX(errors.KindServerError, "nil ProfileClient returned", nil)
	}
	profileResponse, err := dpc.ProfileByName(context.Background(), nodeResponse.Node.ProfileName)
	if err != nil {
		return nodeCoreCommand, errors.NewCommonEdgeXWrapper(err)
	}

	// Prepare the url for command
	configuration := commandContainer.ConfigurationFrom(dic.Get)
	serviceUrl := configuration.Service.Url()

	commands, err := dtos.BuildCoreCommands(nodeResponse.Node.Name, serviceUrl, profileResponse.Profile)
	if err != nil {
		return nodeCoreCommand, errors.NewCommonEdgeXWrapper(err)
	}

	nodeCoreCommand = dtos.NodeCoreCommand{
		NodeName:     nodeResponse.Node.Name,
		ProfileName:  nodeResponse.Node.ProfileName,
		CoreCommands: commands,
	}
	return nodeCoreCommand, nil
}
