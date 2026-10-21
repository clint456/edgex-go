//
// Copyright (C) 2022-2025 IOTech Ltd
// Copyright (C) 2023 Intel Inc.
//
// SPDX-License-Identifier: Apache-2.0

package messaging

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/responses"

	"github.com/edgexfoundry/go-mod-messaging/v4/pkg/types"

	"github.com/clint456/edgex-go/internal/core/command/application"
)

// retrieveServiceNameByNode validates the existence of node and node service,
// returns the service name to which the command request will be sent.
func retrieveServiceNameByNode(nodeName string, dic *di.Container) (string, error) {
	// retrieve node information through Metadata NodeClient
	dc := bootstrapContainer.NodeClientFrom(dic.Get)
	if dc == nil {
		return "", errors.New("nil Node Client")
	}
	nodeResponse, err := dc.NodeByName(context.Background(), nodeName)
	if err != nil {
		return "", fmt.Errorf("failed to get Node by name %s: %v", nodeName, err)
	}

	// retrieve node service information through Metadata NodeClient
	dsc := bootstrapContainer.NodeServiceClientFrom(dic.Get)
	if dsc == nil {
		return "", errors.New("nil NodeService Client")
	}
	nodeServiceResponse, err := dsc.NodeServiceByName(context.Background(), nodeResponse.Node.ServiceName)
	if err != nil {
		return "", fmt.Errorf("failed to get NodeService by name %s: %v", nodeResponse.Node.ServiceName, err)
	}
	return nodeServiceResponse.Service.Name, nil
}

// validateGetCommandQueryParameters validates the value is valid for node service's reserved query parameters
func validateGetCommandQueryParameters(queryParams map[string]string) error {
	if dsReturnEvent, ok := queryParams[common.ReturnEvent]; ok {
		if dsReturnEvent != common.ValueTrue && dsReturnEvent != common.ValueFalse {
			return fmt.Errorf("invalid query parameter, %s has to be '%s' or '%s'", common.ReturnEvent, common.ValueTrue, common.ValueFalse)
		}
	}
	if dsPushEvent, ok := queryParams[common.PushEvent]; ok {
		if dsPushEvent != common.ValueTrue && dsPushEvent != common.ValueFalse {
			return fmt.Errorf("invalid query parameter, %s has to be '%s' or '%s'", common.PushEvent, common.ValueTrue, common.ValueFalse)
		}
	}

	return nil
}

// getCommandQueryResponseEnvelope returns the MessageEnvelope containing the NodeCoreCommand payload bytes
func getCommandQueryResponseEnvelope(requestEnvelope types.MessageEnvelope, nodeName string, dic *di.Container) (types.MessageEnvelope, error) {
	var commandsResponse any
	var err error

	switch nodeName {
	case common.All:
		offset, limit := common.DefaultOffset, common.DefaultLimit
		if requestEnvelope.QueryParams != nil {
			if offsetRaw, ok := requestEnvelope.QueryParams[common.Offset]; ok {
				offset, err = strconv.Atoi(offsetRaw)
				if err != nil {
					return types.MessageEnvelope{}, fmt.Errorf("failed to convert 'offset' query parameter to intger: %s", err.Error())
				}
			}
			if limitRaw, ok := requestEnvelope.QueryParams[common.Limit]; ok {
				limit, err = strconv.Atoi(limitRaw)
				if err != nil {
					return types.MessageEnvelope{}, fmt.Errorf("failed to convert 'limit' query parameter to integer: %s", err.Error())
				}
			}
		}

		commands, totalCounts, edgexError := application.AllCommands(offset, limit, dic)
		if edgexError != nil {
			return types.MessageEnvelope{}, fmt.Errorf("failed to get all commands: %s", edgexError.Error())
		}

		commandsResponse = responses.NewMultiNodeCoreCommandsResponse(requestEnvelope.RequestID, "", http.StatusOK, totalCounts, commands)
	default:
		commands, edgexError := application.CommandsByNodeName(nodeName, dic)
		if edgexError != nil {
			return types.MessageEnvelope{}, fmt.Errorf("failed to get commands by node name '%s': %s", nodeName, edgexError.Error())
		}

		commandsResponse = responses.NewNodeCoreCommandResponse(requestEnvelope.RequestID, "", http.StatusOK, commands)
	}

	responseEnvelope, err := types.NewMessageEnvelopeForResponse(commandsResponse, requestEnvelope.RequestID, requestEnvelope.CorrelationID, common.ContentTypeJSON)
	if err != nil {
		return types.MessageEnvelope{}, fmt.Errorf("failed to create response MessageEnvelope: %s", err.Error())
	}

	return responseEnvelope, nil
}
