//
// Copyright (C) 2021-2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"fmt"
	"time"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/requests"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-messaging/v4/pkg/types"

	"github.com/clint456/edgex-go/internal/core/metadata/container"
)

// validateNodeCallback invoke node service's validation function for validating new or updated node
func validateNodeCallback(node dtos.Node, dic *di.Container) errors.EdgeX {
	configuration := container.ConfigurationFrom(dic.Get)
	messagingClient := bootstrapContainer.MessagingClientFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)

	requestTimeout, err := time.ParseDuration(configuration.Service.RequestTimeout)
	if err != nil {
		return errors.NewCommonEdgeX(errors.KindServerError, "failed to parse service.RequestTimeout", err)
	}

	// reusing AddNodeRequest here as it contains the protocols field and opens up
	// to other validation beyond protocols if ever needed
	addNodeRequest := requests.NewAddNodeRequest(node)

	baseTopic := configuration.MessageBus.GetBaseTopicPrefix()
	requestTopic := common.NewPathBuilder().EnableNameFieldEscape(configuration.Service.EnableNameFieldEscape).
		SetPath(baseTopic).SetNameFieldPath(node.ServiceName).SetPath(common.ValidateNodeSubscribeTopic).BuildPath()
	responseTopicPrefix := common.NewPathBuilder().EnableNameFieldEscape(configuration.Service.EnableNameFieldEscape).
		SetPath(baseTopic).SetPath(common.ResponseTopic).SetNameFieldPath(node.ServiceName).BuildPath()
	requestEnvelope := types.NewMessageEnvelopeForRequest(addNodeRequest, nil)

	lc.Debugf("Sending Node Validation request for node=%s, CorrelationId=%s to topic: %s", node.Name, requestEnvelope.CorrelationID, requestTopic)
	lc.Debugf("Waiting for Node Validation response on topic: %s/%s", responseTopicPrefix, requestEnvelope.RequestID)

	res, err := messagingClient.Request(requestEnvelope, requestTopic, responseTopicPrefix, requestTimeout)
	if err != nil {
		return errors.NewCommonEdgeX(errors.KindServiceUnavailable, fmt.Sprintf("Error sending request to topic '%s'", requestTopic), err)
	} else if res.ErrorCode == 1 {
		return errors.NewCommonEdgeX(errors.KindServerError, fmt.Sprintf("Node %s validation failed: %s", node.Name, res.Payload), nil)
	}

	lc.Debugf("Received Node Validation response for node=%s, CorrelationId=%s on topic: %s", node.Name, res.CorrelationID, res.ReceivedTopic)

	return nil
}

func publishUpdateProfileSystemEvent(profileDTO dtos.Profile, ctx context.Context, dic *di.Container) {
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	nodes, _, err := NodesByProfileName(0, -1, profileDTO.Name, dic)
	if err != nil {
		lc.Errorf("fail to query associated nodes by profile name %s, err: %v", profileDTO.Name, err)
		return
	}

	//Publish general system event regardless of associated nodes
	publishSystemEvent(common.ProfileSystemEventType, common.SystemEventActionUpdate, common.CoreMetaDataServiceKey, profileDTO, ctx, dic)
	// Publish system event for each node service
	dsMap := make(map[string]bool)
	for _, d := range nodes {
		if _, ok := dsMap[d.ServiceName]; ok {
			// skip the invoked node service
			continue
		}
		dsMap[d.ServiceName] = true

		publishSystemEvent(common.ProfileSystemEventType, common.SystemEventActionUpdate, d.ServiceName, profileDTO, ctx, dic)
	}
}

func publishSystemEvent(eventType, action, owner string, dto any, ctx context.Context, dic *di.Container) {
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	config := container.ConfigurationFrom(dic.Get)
	systemEvent := dtos.NewSystemEvent(eventType, action, common.CoreMetaDataServiceKey, owner, nil, dto)
	messagingClient := bootstrapContainer.MessagingClientFrom(dic.Get)
	if messagingClient == nil {
		lc.Errorf("unable to publish '%s' System Event: %v", eventType, errNoMessagingClient)
		return
	}

	var profileName, detailName string
	switch eventType {
	case common.NodeSystemEventType:
		if node, ok := dto.(dtos.Node); ok {
			profileName = node.ProfileName
			detailName = node.Name
		} else {
			lc.Errorf("can not convert to node DTO")
			return
		}
	case common.ProfileSystemEventType:
		if profile, ok := dto.(dtos.Profile); ok {
			profileName = profile.Name
			detailName = profile.Name
		} else {
			lc.Errorf("can not convert to node profile DTO")
			return
		}
	case common.ProvisionWatcherSystemEventType:
		if pw, ok := dto.(dtos.ProvisionWatcher); ok {
			profileName = pw.DiscoveredNode.ProfileName
			detailName = pw.Name
		} else {
			lc.Errorf("can not convert to provision watcher DTO")
		}
	case common.NodeServiceSystemEventType:
		if service, ok := dto.(dtos.NodeService); ok {
			detailName = service.Name
		} else {
			lc.Errorf("can not convert to node service DTO")
			return
		}
	default:
		lc.Errorf("unrecognized system event details")
		return
	}

	topicPathBuilder := common.NewPathBuilder().EnableNameFieldEscape(config.Service.EnableNameFieldEscape)
	publishTopic := topicPathBuilder.SetPath(config.MessageBus.GetBaseTopicPrefix()).SetPath(common.SystemEventPublishTopic).
		SetPath(systemEvent.Source).SetPath(systemEvent.Type).SetPath(systemEvent.Action).SetNameFieldPath(systemEvent.Owner).BuildPath()

	if profileName != "" {
		publishTopic = topicPathBuilder.SetNameFieldPath(profileName).BuildPath()
	}

	// make sure the Content Type is set appropriate if payload is required to be encoded
	ctx = context.WithValue(ctx, common.ContentType, common.ContentTypeJSON) //nolint: staticcheck
	envelope := types.NewMessageEnvelope(systemEvent, ctx)

	if err := messagingClient.Publish(envelope, publishTopic); err != nil {
		lc.Errorf("unable to publish '%s' System Event for %s '%s' to topic '%s': %v", action, eventType, detailName, publishTopic, err)
		return
	}

	lc.Debugf("Published the '%s' System Event for %s '%s' to topic '%s'", action, eventType, detailName, publishTopic)
}
