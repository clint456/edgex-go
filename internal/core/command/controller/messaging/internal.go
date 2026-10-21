//
// Copyright (C) 2022-2023 IOTech Ltd
// Copyright (C) 2023 Intel Inc.
//
// SPDX-License-Identifier: Apache-2.0

package messaging

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-messaging/v4/messaging"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"

	"github.com/edgexfoundry/go-mod-messaging/v4/pkg/types"

	"github.com/clint456/edgex-go/internal/core/command/container"
)

// SubscribeCommandRequests subscribes command requests from EdgeX service (e.g., Application Service)
// and forwards them to the appropriate Node Service via internal MessageBus
func SubscribeCommandRequests(ctx context.Context, requestTimeout time.Duration, dic *di.Container) errors.EdgeX {
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	baseTopic := container.ConfigurationFrom(dic.Get).MessageBus.GetBaseTopicPrefix()
	requestCommandTopic := common.BuildTopic(baseTopic, common.CoreCommandRequestSubscribeTopic)

	messages := make(chan types.MessageEnvelope)
	messageErrors := make(chan error)
	topics := []types.TopicChannel{
		{
			Topic:    requestCommandTopic,
			Messages: messages,
		},
	}

	messageBus := bootstrapContainer.MessagingClientFrom(dic.Get)
	err := messageBus.Subscribe(topics, messageErrors)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				lc.Infof("Exiting waiting for MessageBus '%s' topic messages", requestCommandTopic)
				return
			case err = <-messageErrors:
				lc.Error(err.Error())
			case requestEnvelope := <-messages:
				go func() {
					processCommandRequest(messageBus, requestEnvelope, baseTopic, requestTimeout, lc, dic)
				}()
			}
		}
	}()

	return nil
}

func processCommandRequest(
	messageBus messaging.MessageClient,
	requestEnvelope types.MessageEnvelope,
	baseTopic string,
	requestTimeout time.Duration,
	lc logger.LoggingClient,
	dic *di.Container) {
	var err error
	config := container.ConfigurationFrom(dic.Get)

	lc.Debugf("Command node request received on internal MessageBus. Topic: %s, Request-id: %s, Correlation-id: %s", requestEnvelope.ReceivedTopic, requestEnvelope.RequestID, requestEnvelope.CorrelationID)

	if len(strings.TrimSpace(requestEnvelope.RequestID)) == 0 {
		lc.Errorf("RequestId not set in Command request received on internal MessageBus")
		lc.Warn("Not publishing error message back due to insufficient information to publish on response topic")
		return
	}

	// internal response topic scheme: <ResponseTopicPrefix>/<service-name>/<request-id>
	internalResponseTopic := common.BuildTopic(baseTopic, common.ResponseTopic, common.CoreCommandServiceKey, requestEnvelope.RequestID)
	topicLevels := strings.Split(requestEnvelope.ReceivedTopic, "/")
	length := len(topicLevels)
	if length < 3 {
		err = fmt.Errorf("invalid internal command request topic scheme. Expected request topic scheme with >=3 levels: '<NodeRequestTopicPrefix>/<node-name>/<command-name>/<method>'")
		lc.Error(err.Error())
		responseEnvelope := types.NewMessageEnvelopeWithError(requestEnvelope.RequestID, err.Error())
		err = messageBus.Publish(responseEnvelope, internalResponseTopic)
		if err != nil {
			lc.Errorf("Could not publish to topic '%s': %s", internalResponseTopic, err.Error())
		}
		return
	}

	// expected internal command request/response topic scheme: #/<node>/<command-name>/<method>
	nodeName, err := url.PathUnescape(topicLevels[length-3])
	if err != nil {
		lc.Errorf("Failed to unescape node name from '%s': %s", topicLevels[length-3], err.Error())
		lc.Warn("Not publishing error message back due to insufficient information on response topic")
		return
	}
	commandName, err := url.PathUnescape(topicLevels[length-2])
	if err != nil {
		err = fmt.Errorf("failed to unescape command name from '%s': %s", topicLevels[length-2], err.Error())
		lc.Error(err.Error())
		responseEnvelope := types.NewMessageEnvelopeWithError(requestEnvelope.RequestID, err.Error())
		err = messageBus.Publish(responseEnvelope, internalResponseTopic)
		if err != nil {
			lc.Errorf("Could not publish to topic '%s': %s", internalResponseTopic, err.Error())
		}
		return
	}
	method := topicLevels[length-1]
	if !strings.EqualFold(method, "get") && !strings.EqualFold(method, "set") {
		err = fmt.Errorf("unknown request method: %s, only 'get' or 'set' is allowed", method)
		lc.Error(err.Error())
		responseEnvelope := types.NewMessageEnvelopeWithError(requestEnvelope.RequestID, err.Error())
		err = messageBus.Publish(responseEnvelope, internalResponseTopic)
		if err != nil {
			lc.Errorf("Could not publish to topic '%s': %s", internalResponseTopic, err.Error())
		}
		return
	}

	topicPrefix := common.BuildTopic(baseTopic, common.CoreCommandNodeRequestPublishTopic)
	// internal command request topic scheme: <NodeRequestTopicPrefix>/<node-service>/<node>/<command-name>/<method>
	nodeServiceName, err := retrieveServiceNameByNode(nodeName, dic)
	if err != nil {
		err = fmt.Errorf("invalid request topic: %s", err.Error())
		lc.Error(err.Error())
		responseEnvelope := types.NewMessageEnvelopeWithError(requestEnvelope.RequestID, err.Error())
		err = messageBus.Publish(responseEnvelope, internalResponseTopic)
		if err != nil {
			lc.Errorf("Could not publish to topic '%s': %s", internalResponseTopic, err.Error())
		}
		return
	}

	err = validateGetCommandQueryParameters(requestEnvelope.QueryParams)
	if err != nil {
		lc.Errorf(err.Error())
		responseEnvelope := types.NewMessageEnvelopeWithError(requestEnvelope.RequestID, err.Error())
		err = messageBus.Publish(responseEnvelope, internalResponseTopic)
		if err != nil {
			lc.Errorf("Could not publish to topic '%s': %s", internalResponseTopic, err.Error())
		}
		return
	}

	nodeRequestTopic := common.NewPathBuilder().EnableNameFieldEscape(config.Service.EnableNameFieldEscape).
		SetPath(topicPrefix).SetNameFieldPath(nodeServiceName).SetNameFieldPath(nodeName).SetNameFieldPath(commandName).SetPath(method).BuildPath()
	nodeResponseTopicPrefix := common.NewPathBuilder().EnableNameFieldEscape(config.Service.EnableNameFieldEscape).
		SetPath(baseTopic).SetPath(common.ResponseTopic).SetNameFieldPath(nodeServiceName).BuildPath()

	lc.Debugf("Sending Command Node Request to internal MessageBus. Topic: %s, Correlation-id: %s", nodeRequestTopic, requestEnvelope.CorrelationID)
	lc.Debugf("Expecting response on topic: %s/%s", nodeResponseTopicPrefix, requestEnvelope.RequestID)

	response, err := messageBus.Request(requestEnvelope, nodeRequestTopic, nodeResponseTopicPrefix, requestTimeout)
	if err != nil {
		lc.Errorf("Request to topic '%s' failed: %s", nodeRequestTopic, err.Error())
		return
	}

	// original request is from internal MessageBus
	err = messageBus.Publish(*response, internalResponseTopic)
	if err != nil {
		lc.Errorf("Could not publish to internal MessageBus topic '%s': %s", internalResponseTopic, err.Error())
		return
	}

	lc.Debugf("Command response sent to internal MessageBus. Topic: %s, Correlation-id: %s", internalResponseTopic, response.CorrelationID)
}

// SubscribeCommandQueryRequests subscribes command query requests from EdgeX service (e.g., Application Service)
// via internal MessageBus
func SubscribeCommandQueryRequests(ctx context.Context, dic *di.Container) errors.EdgeX {
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	baseTopic := container.ConfigurationFrom(dic.Get).MessageBus.GetBaseTopicPrefix()
	queryRequestTopic := common.BuildTopic(baseTopic, common.CoreCommandQueryRequestSubscribeTopic)

	messages := make(chan types.MessageEnvelope)
	messageErrors := make(chan error)
	topics := []types.TopicChannel{
		{
			Topic:    queryRequestTopic,
			Messages: messages,
		},
	}

	messageBus := bootstrapContainer.MessagingClientFrom(dic.Get)

	lc.Infof("Subscribing to internal command query requests on topic: %s", queryRequestTopic)

	err := messageBus.Subscribe(topics, messageErrors)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				lc.Infof("Exiting waiting for MessageBus '%s' topic messages", queryRequestTopic)
				return
			case err = <-messageErrors:
				lc.Error(err.Error())
			case requestEnvelope := <-messages:
				go func() {
					processCommandQueryRequest(messageBus, requestEnvelope, baseTopic, lc, dic)
				}()
			}
		}
	}()

	return nil
}

func processCommandQueryRequest(
	messageBus messaging.MessageClient,
	requestEnvelope types.MessageEnvelope,
	baseTopic string,
	lc logger.LoggingClient,
	dic *di.Container,
) {
	lc.Debugf("Command query request received on internal MessageBus. Topic: %s, Request-id: %s, Correlation-id: %s", requestEnvelope.ReceivedTopic, requestEnvelope.RequestID, requestEnvelope.CorrelationID)

	if len(strings.TrimSpace(requestEnvelope.RequestID)) == 0 {
		lc.Errorf("RequestId not set in Command request received on internal MessageBus")
		lc.Warn("Not publishing error message back due to insufficient information to publish on response topic")
		return
	}

	// example topic scheme: /commandquery/request/<node>
	// nodeName is expected to be at last topic level.
	topicLevels := strings.Split(requestEnvelope.ReceivedTopic, "/")
	nodeName, err := url.PathUnescape(topicLevels[len(topicLevels)-1])
	if err != nil {
		lc.Errorf("Failed to unescape node name '%s': %s", nodeName, err.Error())
		lc.Warn("Not publishing error message back due to insufficient information on response topic")
		return
	}
	if strings.EqualFold(nodeName, common.All) {
		nodeName = common.All
	}

	responseEnvelope, err := getCommandQueryResponseEnvelope(requestEnvelope, nodeName, dic)
	if err != nil {
		lc.Error(err.Error())
		responseEnvelope = types.NewMessageEnvelopeWithError(requestEnvelope.RequestID, err.Error())
	}

	// internal response topic scheme: <ResponseTopicPrefix>/<service-name>/<request-id>
	internalQueryResponseTopic := common.BuildTopic(baseTopic, common.ResponseTopic, common.CoreCommandServiceKey, requestEnvelope.RequestID)
	lc.Debugf("Responding to command query request on topic: %s", internalQueryResponseTopic)

	err = messageBus.Publish(responseEnvelope, internalQueryResponseTopic)
	if err != nil {
		lc.Errorf("Could not publish to topic '%s': %s", internalQueryResponseTopic, err.Error())
		return
	}

	lc.Debugf("Command query response sent to internal MessageBus. Topic: %s, Correlation-id: %s", internalQueryResponseTopic, requestEnvelope.CorrelationID)
}
