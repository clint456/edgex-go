//
// Copyright (C) 2024-2026 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"testing"

	"github.com/clint456/edgex-go/internal/core/metadata/config"
	"github.com/clint456/edgex-go/internal/core/metadata/container"
	"github.com/clint456/edgex-go/internal/core/metadata/infrastructure/interfaces/mocks"
	"github.com/clint456/edgex-go/internal/pkg/correlation"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	profile             = "test-profile"
	notFountProfileName = "notFoundProfile"
	source1             = "source1"
	source2             = "resource2"
	command1            = "command1"
	command2            = "command2"
	profileModel        = models.Profile{
		Name:      profile,
		Resources: []models.Resource{{Name: source1}, {Name: source2}},
		Commands:  []models.Command{{Name: command1}, {Name: command2}},
	}
)

func TestValidateParentProfileAndAutoEvents(t *testing.T) {
	dic := di.NewContainer(di.ServiceConstructorMap{})
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("ProfileByName", profile).Return(profileModel, nil)
	dbClientMock.On("ProfileByName", notFountProfileName).Return(models.Profile{}, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "not found", nil))
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	tests := []struct {
		name          string
		node          models.Node
		errorExpected bool
	}{
		{"empty profile",
			models.Node{},
			false,
		},
		{"not found profile",
			models.Node{
				ProfileName: notFountProfileName,
			},
			true,
		},
		{"no auto events",
			models.Node{
				ProfileName: profile,
			},
			false,
		},
		{"resource exist",
			models.Node{
				ProfileName: profile,
				AutoEvents:  []models.AutoEvent{{SourceName: source1, Interval: "1s"}},
			},
			false,
		},
		{"command exist",
			models.Node{
				ProfileName: profile,
				AutoEvents:  []models.AutoEvent{{SourceName: source1, Interval: "1s"}, {SourceName: command1, Interval: "1s"}},
			},
			false,
		},
		{"resource not exist",
			models.Node{
				ProfileName: profile,
				AutoEvents:  []models.AutoEvent{{SourceName: "notFoundSource", Interval: "1s"}},
			},
			true,
		},
		{"interval format not valid",
			models.Node{
				ProfileName: profile,
				AutoEvents:  []models.AutoEvent{{SourceName: source1, Interval: "1"}},
			},
			true,
		},
		{"no profile",
			models.Node{
				AutoEvents: []models.AutoEvent{{SourceName: source1, Interval: "1s"}},
			},
			false,
		},
		{"resource match regex",
			models.Node{
				ProfileName: profile,
				AutoEvents:  []models.AutoEvent{{SourceName: "res.*", Interval: "1s"}},
			},
			false,
		},
		{"is own parent",
			models.Node{
				ProfileName: profile,
				Parent:      "me",
				Name:        "me",
			},
			true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := validateParentProfileAndAutoEvent(dic, testCase.node)
			if testCase.errorExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateParentProfileAndAutoEventsDropInvalid(t *testing.T) {
	dic := di.NewContainer(di.ServiceConstructorMap{})
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("ProfileByName", profile).Return(profileModel, nil)
	dbClientMock.On("ProfileByName", notFountProfileName).Return(models.Profile{}, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "not found", nil))
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
		bootstrapContainer.LoggingClientInterfaceName: func(get di.Get) interface{} {
			return logger.NewMockClient()
		},
	})

	tests := []struct {
		name               string
		node               models.Node
		expectedAutoEvents int
		errorExpected      bool
	}{
		{"empty profile",
			models.Node{},
			0,
			false,
		},
		{"not found profile",
			models.Node{
				ProfileName: notFountProfileName,
			},
			0,
			true,
		},
		{"no auto events",
			models.Node{
				ProfileName: profile,
			},
			0,
			false,
		},
		{"is own parent",
			models.Node{
				ProfileName: profile,
				Parent:      "me",
				Name:        "me",
				AutoEvents:  []models.AutoEvent{{SourceName: source1, Interval: "1s"}},
			},
			0,
			true,
		},
		{"all valid",
			models.Node{
				ProfileName: profile,
				AutoEvents:  []models.AutoEvent{{SourceName: source1, Interval: "1s"}},
			},
			1,
			false,
		},
		{"one invalid",
			models.Node{
				ProfileName: profile,
				AutoEvents: []models.AutoEvent{
					{SourceName: source1, Interval: "1s"},
					{SourceName: "invalid", Interval: "1s"},
				},
			},
			1,
			false,
		},
		{"all invalid",
			models.Node{
				ProfileName: profile,
				AutoEvents: []models.AutoEvent{
					{SourceName: "invalid1", Interval: "1s"},
					{SourceName: "invalid2", Interval: "1s"},
				},
			},
			0,
			false,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			d, err := validateParentProfileAndAutoEventDropInvalid(dic, testCase.node)
			if testCase.errorExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expectedAutoEvents, len(d.AutoEvents))
			}
		})
	}
}

func TestForceAddNode(t *testing.T) {
	invalidNodeName := "invalidNode"
	validNodeName := "validNode"
	invalidNodeName2 := "invalidNode2"
	invalidNode := models.Node{Name: invalidNodeName}
	invalidNode2 := models.Node{Name: invalidNodeName2}
	returnedNode := models.Node{Name: validNodeName}

	dic := di.NewContainer(di.ServiceConstructorMap{
		bootstrapContainer.LoggingClientInterfaceName: func(get di.Get) interface{} {
			return logger.NewMockClient()
		},
		container.ConfigurationName: func(get di.Get) interface{} {
			return &config.ConfigurationStruct{
				Writable: config.WritableInfo{
					LogLevel: "DEBUG",
				},
			}
		},
	})

	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("NodeByName", invalidNodeName).Return(models.Node{}, errors.NewCommonEdgeX(errors.KindDatabaseError, "failed to query", nil))
	dbClientMock.On("NodeByName", validNodeName).Return(returnedNode, nil)
	dbClientMock.On("NodeByName", invalidNodeName2).Return(invalidNode2, nil)
	dbClientMock.On("UpdateNode", returnedNode).Return(nil)
	dbClientMock.On("UpdateNode", invalidNode2).Return(errors.NewCommonEdgeX(errors.KindDatabaseError, "failed to update", nil))
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	tests := []struct {
		name          string
		node          models.Node
		errorExpected bool
	}{
		{"invalid - NodeByName error", invalidNode, true},
		{"valid", returnedNode, false},
		{"invalid - UpdateNode error", invalidNode2, true},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, _ := correlation.FromContextOrNew(context.Background())
			result, err := updateNode(testCase.node, ctx, dic)
			if testCase.errorExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, returnedNode.Id, result)
			}
		})
	}
}

func TestNodeById(t *testing.T) {
	validId := "82eb2e26-0f24-48aa-ae4c-de9dac3fb9bc"
	notFoundId := "00000000-0000-0000-0000-000000000000"
	returnedNode := models.Node{Id: validId, Name: "testNode"}

	dic := di.NewContainer(di.ServiceConstructorMap{
		bootstrapContainer.LoggingClientInterfaceName: func(get di.Get) interface{} {
			return logger.NewMockClient()
		},
	})

	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("NodeById", validId).Return(returnedNode, nil)
	dbClientMock.On("NodeById", notFoundId).Return(models.Node{}, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "node doesn't exist in the database", nil))
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	tests := []struct {
		name          string
		id            string
		errorExpected bool
	}{
		{"Valid - find node by id", validId, false},
		{"Invalid - empty id", "", true},
		{"Invalid - node not found by id", notFoundId, true},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			node, err := NodeById(testCase.id, dic)
			if testCase.errorExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, validId, node.Id)
				assert.Equal(t, returnedNode.Name, node.Name)
			}
		})
	}
}
