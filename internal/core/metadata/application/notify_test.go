//
// Copyright (C) 2022 Intel
// Copyright (C) 2023-2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	goErrors "errors"
	"fmt"
	"testing"

	"github.com/clint456/edgex-go/internal/core/metadata/config"
	"github.com/clint456/edgex-go/internal/core/metadata/container"
	bootstrapConfig "github.com/edgexfoundry/go-mod-bootstrap/v4/config"
	mocks2 "github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger/mocks"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-messaging/v4/messaging/mocks"
	"github.com/edgexfoundry/go-mod-messaging/v4/pkg/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
)

func TestPublishSystemEvent(t *testing.T) {
	TestProfileName := "onvif-camera"
	TestNodeServiceName := "Node-onvif-camera"
	TestNodeName := "Camera-Node"

	expectedNode := dtos.Node{
		Name:        TestNodeName,
		Id:          uuid.NewString(),
		ServiceName: TestNodeServiceName,
		ProfileName: TestProfileName,
	}

	expectedProfile := dtos.Profile{
		ProfileBasicInfo: dtos.ProfileBasicInfo{
			Id:   uuid.NewString(),
			Name: TestProfileName,
		},
	}

	expectedCorrelationID := uuid.NewString()
	expectedPublishTopicPrefix := "edgex/system-events"

	tests := []struct {
		Name          string
		Type          string
		Action        string
		Owner         string
		PubError      bool
		ClientMissing bool
	}{
		{"Node Add", common.NodeSystemEventType, common.SystemEventActionAdd, TestNodeServiceName, false, false},
		{"Node Update", common.NodeSystemEventType, common.SystemEventActionUpdate, TestNodeServiceName, false, false},
		{"Node Delete", common.NodeSystemEventType, common.SystemEventActionDelete, TestNodeServiceName, false, false},
		{"Node Profile Add", common.ProfileSystemEventType, common.SystemEventActionAdd, common.CoreMetaDataServiceKey, false, false},
		{"Node Profile Update", common.ProfileSystemEventType, common.SystemEventActionUpdate, TestNodeServiceName, false, false},
		{"Node Profile Delete", common.ProfileSystemEventType, common.SystemEventActionDelete, common.CoreMetaDataServiceKey, false, false},
		{"Client Missing Error", common.NodeSystemEventType, common.SystemEventActionAdd, TestNodeServiceName, false, true},
		{"Publish Error", common.NodeSystemEventType, common.SystemEventActionAdd, TestNodeServiceName, true, false},
	}

	pubErrMsg := errors.NewCommonEdgeXWrapper(goErrors.New("publish failed"))
	mockLogger := &mocks2.LoggingClient{}

	dic := di.NewContainer(di.ServiceConstructorMap{
		container.ConfigurationName: func(get di.Get) interface{} {
			return &config.ConfigurationStruct{Service: bootstrapConfig.ServiceInfo{EnableNameFieldEscape: true}}
		},
		bootstrapContainer.LoggingClientInterfaceName: func(get di.Get) interface{} {
			return mockLogger
		},
	})

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			validatePublishCallFunc := func(envelope types.MessageEnvelope, topic string) error {
				assert.Equal(t, common.ContentTypeJSON, envelope.ContentType)
				assert.Equal(t, expectedCorrelationID, envelope.CorrelationID)
				require.NotEmpty(t, envelope.Payload)
				systemEvent, err := types.GetMsgPayload[dtos.SystemEvent](envelope)
				require.NoError(t, err)

				switch test.Type {
				case common.NodeSystemEventType:
					actualNode := dtos.Node{}
					err = systemEvent.DecodeDetails(&actualNode)
					require.NoError(t, err)
					assert.Equal(t, expectedNode.Name, actualNode.Name)
					assert.Equal(t, expectedNode.Id, actualNode.Id)
					assert.Equal(t, expectedNode.ServiceName, actualNode.ServiceName)
					assert.Equal(t, expectedNode.ProfileName, actualNode.ProfileName)
				case common.ProfileSystemEventType:
					actualProfile := dtos.Profile{}
					err = systemEvent.DecodeDetails(&actualProfile)
					require.NoError(t, err)
					assert.Equal(t, expectedProfile.Name, actualProfile.Name)
					assert.Equal(t, expectedProfile.Id, actualProfile.Id)
				}

				assert.Equal(t, common.ApiVersion, systemEvent.ApiVersion)
				assert.Equal(t, test.Type, systemEvent.Type)
				assert.Equal(t, test.Action, systemEvent.Action)
				assert.Equal(t, common.CoreMetaDataServiceKey, systemEvent.Source)
				assert.Equal(t, test.Owner, systemEvent.Owner)
				assert.NotZero(t, systemEvent.Timestamp)

				return nil
			}

			mockClient := &mocks.MessageClient{}
			mockLogger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

			if test.PubError {
				mockClient.On("Publish", mock.Anything, mock.Anything).Return(pubErrMsg)
				mockLogger.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			} else {
				mockClient.On("Publish", mock.Anything, mock.Anything).Return(validatePublishCallFunc)
			}

			if test.ClientMissing {
				mockLogger.On("Errorf", mock.Anything, mock.Anything, mock.Anything).Return()
				dic.Update(di.ServiceConstructorMap{
					bootstrapContainer.MessagingClientName: func(get di.Get) interface{} {
						return nil
					},
				})
			} else {
				dic.Update(di.ServiceConstructorMap{
					bootstrapContainer.MessagingClientName: func(get di.Get) interface{} {
						return mockClient
					},
				})
			}

			// Use CBOR to make sure publisher overrides with JSON properly
			// lint:ignore SA1029 legacy
			// nolint:staticcheck // See golangci-lint #741
			ctx := context.WithValue(context.Background(), common.ContentType, common.ContentTypeCBOR)
			// lint:ignore SA1029 legacy
			// nolint:staticcheck // See golangci-lint #741
			ctx = context.WithValue(ctx, common.CorrelationHeader, expectedCorrelationID)
			var expectedDetails any
			switch test.Type {
			case common.NodeSystemEventType:
				expectedDetails = expectedNode
			case common.ProfileSystemEventType:
				expectedDetails = expectedProfile
			}

			publishSystemEvent(test.Type, test.Action, test.Owner, expectedDetails, ctx, dic)

			if test.ClientMissing {
				mockLogger.AssertCalled(t, "Errorf", mock.Anything, mock.Anything, errNoMessagingClient)
				return
			}

			expectedTopic := fmt.Sprintf("%s/%s/%s/%s/%s/%s",
				expectedPublishTopicPrefix,
				common.CoreMetaDataServiceKey,
				test.Type,
				test.Action,
				common.URLEncode(test.Owner),
				common.URLEncode(expectedNode.ProfileName))
			mockClient.AssertCalled(t, "Publish", mock.Anything, expectedTopic)

			if test.PubError {
				mockLogger.AssertCalled(t, "Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, pubErrMsg)
			}
		})
	}
}
