// Copyright (C) 2023 Intel Inc.
// Copyright (C) 2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package messaging

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	config2 "github.com/edgexfoundry/go-mod-bootstrap/v4/config"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	mocks2 "github.com/edgexfoundry/go-mod-core-contracts/v4/clients/interfaces/mocks"
	lcMocks "github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger/mocks"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/responses"
	"github.com/edgexfoundry/go-mod-messaging/v4/messaging/mocks"
	"github.com/edgexfoundry/go-mod-messaging/v4/pkg/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/clint456/edgex-go/internal/core/command/config"
	"github.com/clint456/edgex-go/internal/core/command/container"
)

var expectedResponseTopicPrefix = "edgex/response"
var expectedProfileName = "TestProfile"
var baseTopic = "edgex"

func TestSubscribeCommandRequests(t *testing.T) {
	wg := sync.WaitGroup{}
	expectedServiceName := "node-simple"
	expectedRequestId := uuid.NewString()
	expectedCorrelationId := uuid.NewString()
	expectedNode := "node1"
	expectedResource := "resource"
	expectedMethod := "get"
	expectedNodeResponseTopicPrefix := strings.Join([]string{expectedResponseTopicPrefix, expectedServiceName}, "/")
	expectedCommandResponseTopic := strings.Join([]string{expectedResponseTopicPrefix, common.CoreCommandServiceKey, expectedRequestId}, "/")
	expectedCommandRequestSubscribeTopic := common.BuildTopic(baseTopic, common.CoreCommandRequestSubscribeTopic)
	expectedCommandRequestReceivedTopic := common.BuildTopic(strings.Replace(expectedCommandRequestSubscribeTopic, "/#", "", 1),
		expectedServiceName, expectedNode, expectedResource, expectedMethod)
	expectedCommandRequestRequestTopic := common.BuildTopic(baseTopic, common.CoreCommandNodeRequestPublishTopic, expectedServiceName, expectedNode, expectedResource, expectedMethod)
	mockLogger := &lcMocks.LoggingClient{}
	mockNodeClient := &mocks2.NodeClient{}
	mockProfileClient := &mocks2.ProfileClient{}
	mockNodeServiceClient := &mocks2.NodeServiceClient{}
	mockMessaging := &mocks.MessageClient{}

	mockLogger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)

	mockMessaging.On("Subscribe", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		topics := args.Get(0).([]types.TopicChannel)
		require.Len(t, topics, 1)
		require.Equal(t, expectedCommandRequestSubscribeTopic, topics[0].Topic)
		wg.Add(1)
		go func() {
			defer wg.Done()
			topics[0].Messages <- types.MessageEnvelope{
				RequestID:     expectedRequestId,
				CorrelationID: expectedCorrelationId,
				ReceivedTopic: expectedCommandRequestReceivedTopic,
			}
			time.Sleep(time.Second * 1)
		}()
	}).Return(nil)

	mockMessaging.On("Request", mock.Anything, expectedCommandRequestRequestTopic, expectedNodeResponseTopicPrefix, mock.Anything).Run(func(args mock.Arguments) {
	}).Return(&types.MessageEnvelope{
		RequestID:     expectedRequestId,
		CorrelationID: expectedCorrelationId,
		ContentType:   common.ContentTypeJSON,
		Payload:       "This is my payload",
	}, nil)

	mockMessaging.On("Publish", mock.Anything, expectedCommandResponseTopic).Run(func(args mock.Arguments) {
		response := args.Get(0).(types.MessageEnvelope)
		assert.Equal(t, expectedRequestId, response.RequestID)
		assert.Equal(t, expectedCorrelationId, response.CorrelationID)
		assert.Equal(t, common.ContentTypeJSON, response.ContentType)
		assert.NotEmpty(t, response.Payload)
	}).Return(nil)

	mockNodeClient.On("NodeByName", mock.Anything, expectedNode).Return(
		responses.NodeResponse{
			Node: dtos.Node{
				ProfileName: expectedProfileName,
				ServiceName: expectedServiceName,
			},
		},
		nil)

	mockNodeServiceClient.On("NodeServiceByName", mock.Anything, expectedServiceName).Return(
		responses.NodeServiceResponse{
			Service: dtos.NodeService{
				Name: expectedServiceName,
			},
		},
		nil)

	dic := di.NewContainer(di.ServiceConstructorMap{
		container.ConfigurationName: func(get di.Get) interface{} {
			return &config.ConfigurationStruct{
				MessageBus: config2.MessageBusInfo{
					BaseTopicPrefix: "edgex",
				},
			}
		},
		bootstrapContainer.LoggingClientInterfaceName: func(get di.Get) interface{} {
			return mockLogger
		},
		bootstrapContainer.MessagingClientName: func(get di.Get) interface{} {
			return mockMessaging
		},
		bootstrapContainer.NodeClientName: func(get di.Get) interface{} {
			return mockNodeClient
		},
		bootstrapContainer.ProfileClientName: func(get di.Get) interface{} {
			return mockProfileClient
		},
		bootstrapContainer.NodeServiceClientName: func(get di.Get) interface{} {
			return mockNodeServiceClient
		},
		bootstrapContainer.ConfigurationInterfaceName: func(get di.Get) interface{} {
			return &config.ConfigurationStruct{
				Service: config2.ServiceInfo{EnableNameFieldEscape: false},
			}
		},
	})

	err := SubscribeCommandRequests(context.Background(), time.Second*5, dic)
	require.NoError(t, err)

	wg.Wait()

	mockMessaging.AssertExpectations(t)
}

func TestSubscribeCommandQueryRequests(t *testing.T) {
	wg := sync.WaitGroup{}
	expectedRequestId := uuid.NewString()
	expectedCorrelationId := uuid.NewString()
	expectedResponseTopic := strings.Join([]string{expectedResponseTopicPrefix, common.CoreCommandServiceKey, expectedRequestId}, "/")

	tests := []struct {
		Name             string
		ExpectedNodeName string
	}{
		{"By Node", "Node1"},
		{"All Nodes", "All"},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {

			mockLogger := &lcMocks.LoggingClient{}
			mockNodeClient := &mocks2.NodeClient{}
			mockProfileClient := &mocks2.ProfileClient{}
			mockMessaging := &mocks.MessageClient{}

			mockLogger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			mockLogger.On("Infof", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			mockLogger.On("Errorf", mock.Anything).Run(func(args mock.Arguments) {
				require.Fail(t, "Errorf not expected")
			})
			mockLogger.On("Error", mock.Anything).Run(func(args mock.Arguments) {
				require.Fail(t, "Error not expected")
			})

			expectedSubscribeTopic := common.BuildTopic(baseTopic, common.CoreCommandQueryRequestSubscribeTopic)
			expectedReceivedTopic := common.BuildTopic(strings.Replace(expectedSubscribeTopic, "/#", "", 1), test.ExpectedNodeName)

			mockNodeClient.On("NodeByName", mock.Anything, test.ExpectedNodeName).Return(
				responses.NodeResponse{
					Node: dtos.Node{
						ProfileName: expectedProfileName,
					},
				},
				nil)

			mockNodeClient.On("AllNodes", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
				responses.MultiNodesResponse{
					Nodes: []dtos.Node{
						{
							ProfileName: expectedProfileName,
						},
					},
				},
				nil)

			mockProfileClient.On("ProfileByName", mock.Anything, expectedProfileName).Return(
				responses.ProfileResponse{},
				nil)

			mockMessaging.On("Subscribe", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				topics := args.Get(0).([]types.TopicChannel)
				require.Len(t, topics, 1)
				require.Equal(t, expectedSubscribeTopic, topics[0].Topic)
				wg.Add(1)
				go func() {
					defer wg.Done()
					topics[0].Messages <- types.MessageEnvelope{
						RequestID:     expectedRequestId,
						CorrelationID: expectedCorrelationId,
						ReceivedTopic: expectedReceivedTopic,
					}
					time.Sleep(time.Second * 1)
				}()
			}).Return(nil)
			mockMessaging.On("Publish", mock.Anything, expectedResponseTopic).Run(func(args mock.Arguments) {
				response := args.Get(0).(types.MessageEnvelope)
				assert.Equal(t, expectedRequestId, response.RequestID)
				assert.Equal(t, expectedCorrelationId, response.CorrelationID)
				assert.Equal(t, common.ContentTypeJSON, response.ContentType)
				assert.NotEmpty(t, response.Payload)
			}).Return(nil)

			dic := di.NewContainer(di.ServiceConstructorMap{
				container.ConfigurationName: func(get di.Get) interface{} {
					return &config.ConfigurationStruct{
						MessageBus: config2.MessageBusInfo{
							BaseTopicPrefix: "edgex",
						},
					}
				},
				bootstrapContainer.LoggingClientInterfaceName: func(get di.Get) interface{} {
					return mockLogger
				},
				bootstrapContainer.MessagingClientName: func(get di.Get) interface{} {
					return mockMessaging
				},
				bootstrapContainer.NodeClientName: func(get di.Get) interface{} {
					return mockNodeClient
				},
				bootstrapContainer.ProfileClientName: func(get di.Get) interface{} {
					return mockProfileClient
				},
			})

			err := SubscribeCommandQueryRequests(context.Background(), dic)
			require.NoError(t, err)

			wg.Wait()

			mockMessaging.AssertExpectations(t)

		})
	}
}
