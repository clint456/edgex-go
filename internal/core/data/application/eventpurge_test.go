//
// Copyright (C) 2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"testing"
	"time"

	"github.com/clint456/edgex-go/internal/core/data/cache"
	"github.com/clint456/edgex-go/internal/core/data/config"
	"github.com/clint456/edgex-go/internal/core/data/container"
	dbMock "github.com/clint456/edgex-go/internal/core/data/infrastructure/interfaces/mocks"
	"github.com/clint456/edgex-go/internal/core/data/mocks"
	pkgCache "github.com/clint456/edgex-go/internal/pkg/cache"
	dbModels "github.com/clint456/edgex-go/internal/pkg/infrastructure/models"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	bootstrapConfig "github.com/edgexfoundry/go-mod-bootstrap/v4/config"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	lcMocks "github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger/mocks"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewEventPurge(t *testing.T) {
	nodeInfoMap := map[int]dbModels.NodeInfo{1: {NodeName: testNodeName}}
	ep := newEventPurgeExecutor(nodeInfoMap)
	assert.Equal(t, testNodeName, ep.nodeInfoMap[1].NodeName)
}

func TestPurgeMatchingNodeInfo(t *testing.T) {
	lc := logger.NewMockClient()
	tests := []struct {
		name                string
		nodeName            string
		sourceName          string
		expectedResultCount int
	}{
		{"Test cullMatchingNodeInfo using node/source defined by matched regular expressions", "^Test+", "^Source+", 0},
		{"Test cullMatchingNodeInfo using node/source defined by unmatched regular expressions", "^Not+", "^Not+", 1},
		{"Test cullMatchingNodeInfo using node/source defined by exact matched names", "Test1", "Source1", 0},
		{"Test cullMatchingNodeInfo using node/source defined by unmatched names", "Test2", "Source2", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeInfoMap := map[int]dbModels.NodeInfo{1: {NodeName: "Test1", SourceName: "Source1"}}
			ep := newEventPurgeExecutor(nodeInfoMap)
			ep.cullMatchingNodeInfo(tt.nodeName, tt.sourceName, lc)
			assert.Equal(t, tt.expectedResultCount, len(ep.nodeInfoMap))
		})
	}

}

func TestPurgeEventsByAutoEvent(t *testing.T) {
	dic := mocks.NewMockDIC()
	nodeName := "testNode"
	sourceName := "testSource"
	coreDataConfig := container.ConfigurationFrom(dic.Get)
	coreDataConfig.Retention = config.EventRetention{
		Interval:        "10m",
		DefaultMaxCap:   -1,
		DefaultMinCap:   1,
		DefaultDuration: "30m",
	}
	//readingCache := cache.NewReadingCache(dic)

	dic.Update(di.ServiceConstructorMap{
		container.ConfigurationName: func(get di.Get) interface{} {
			return coreDataConfig
		},
		//container.ReadingCacheInterfaceName: func(get di.Get) interface{} { return readingCache },
	})

	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("LatestEventByNodeNameAndSourceNameAndAgeAndOffset", nodeName, sourceName, mock.Anything, mock.Anything).Return(models.Event{}, nil)
	dbClientMock.On("DeleteEventsByAgeAndNodeNameAndSourceName", mock.Anything, nodeName, sourceName).Return(nil)
	dbClientMock.On("DeleteEventsByNodeNameAndSourceName", nodeName, sourceName).Return(nil)
	dbClientMock.On("LatestEventByNodeNameAndSourceNameAndOffset", nodeName, sourceName, mock.Anything).Return(models.Event{}, nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	nodeInfoMap := map[int]dbModels.NodeInfo{}
	ep := newEventPurgeExecutor(nodeInfoMap)

	tests := []struct {
		name      string
		autoEvent models.AutoEvent
	}{
		{"time-based event retention",
			models.AutoEvent{
				Interval:          "",
				OnChange:          false,
				OnChangeThreshold: 0,
				SourceName:        sourceName,
				Retention: models.Retention{
					MaxCap:   -1,
					MinCap:   -1,
					Duration: "10m",
				},
			},
		},
		{"time-based event retention with miniCap",
			models.AutoEvent{
				Interval:          "",
				OnChange:          false,
				OnChangeThreshold: 0,
				SourceName:        sourceName,
				Retention: models.Retention{
					MaxCap:   -1,
					MinCap:   1,
					Duration: "10m",
				},
			},
		},
		{"count-based event retention",
			models.AutoEvent{
				Interval:          "",
				OnChange:          false,
				OnChangeThreshold: 0,
				SourceName:        sourceName,
				Retention: models.Retention{
					MaxCap:   -1,
					MinCap:   -1,
					Duration: "0s",
				},
			},
		},
		{"count-based event retention with miniCap",
			models.AutoEvent{
				Interval:          "",
				OnChange:          false,
				OnChangeThreshold: 0,
				SourceName:        sourceName,
				Retention: models.Retention{
					MaxCap:   -1,
					MinCap:   1,
					Duration: "0s",
				},
			},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			nodeStore := pkgCache.NodeStore(dic)
			node := models.Node{
				Name:       nodeName,
				AutoEvents: []models.AutoEvent{testCase.autoEvent},
			}
			nodeStore.Add(node)
			dic.Update(di.ServiceConstructorMap{
				container.NodeStoreInterfaceName: func(get di.Get) interface{} {
					return nodeStore
				},
			})

			err := ep.purgeEventsByAutoEvent(dic)
			require.NoError(t, err)
			duration, parseErr := time.ParseDuration(testCase.autoEvent.Retention.Duration)
			require.NoError(t, parseErr)
			if duration > 0 && testCase.autoEvent.Retention.MinCap <= 0 {
				// time-based retention
				dbClientMock.AssertCalled(t, "DeleteEventsByAgeAndNodeNameAndSourceName", mock.Anything, nodeName, testCase.autoEvent.SourceName)
			} else if duration > 0 && testCase.autoEvent.Retention.MinCap > 0 {
				// time-based retention with miniCap
				dbClientMock.AssertCalled(t, "LatestEventByNodeNameAndSourceNameAndAgeAndOffset", nodeName, sourceName, duration.Nanoseconds(), testCase.autoEvent.Retention.MinCap)
				dbClientMock.AssertCalled(t, "DeleteEventsByAgeAndNodeNameAndSourceName", mock.Anything, nodeName, testCase.autoEvent.SourceName)
			} else if testCase.autoEvent.Retention.MinCap <= 0 {
				// count-based retention
				dbClientMock.AssertCalled(t, "DeleteEventsByNodeNameAndSourceName", nodeName, testCase.autoEvent.SourceName)
			} else {
				// count-based retention with miniCap
				dbClientMock.AssertCalled(t, "LatestEventByNodeNameAndSourceNameAndOffset", nodeName, sourceName, testCase.autoEvent.Retention.MinCap)
				dbClientMock.AssertCalled(t, "DeleteEventsByAgeAndNodeNameAndSourceName", mock.Anything, nodeName, testCase.autoEvent.SourceName)
			}

		})
	}
}

func TestPurgeEventsByNodeInfo(t *testing.T) {
	testResourceName := "test-resource"
	testNodeInfo := dbModels.NodeInfo{
		Id:           1,
		NodeName:     testNodeName,
		SourceName:   testSourceName,
		ResourceName: testResourceName,
		ValueType:    common.ValueTypeString,
	}

	failedNodeName := "failedNode"
	failedNodeInfo := testNodeInfo
	failedNodeInfo.NodeName = failedNodeName
	testDuration := "3d10h"

	isValid, duration := common.ParseDurationWithDay(testDuration)
	assert.True(t, isValid)

	dic := di.NewContainer(di.ServiceConstructorMap{})

	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("DeleteEventsByAgeAndNodeNameAndSourceName", duration.Nanoseconds(), testNodeName, testSourceName).Return(nil)
	dbClientMock.On("DeleteEventsByAgeAndNodeNameAndSourceName", duration.Nanoseconds(), failedNodeName, testSourceName).Return(errors.NewCommonEdgeX(errors.KindServerError, "failed to delete event with node name", nil))
	dbClientMock.On("EventCountByNodeNameAndSourceNameAndLimit", testNodeName, testSourceName, 1000).Return(int64(0), errors.NewCommonEdgeX(errors.KindServerError, "failed to get event count", nil))
	dbClientMock.On("EventCountByNodeNameAndSourceNameAndLimit", failedNodeName, testSourceName, 1000).Return(int64(0), errors.NewCommonEdgeX(errors.KindServerError, "failed to get event count", nil))

	mockLogger := &lcMocks.LoggingClient{}
	mockLogger.On("Errorf", "failed to execute event retention for node info with node name: '%s', source name: '%s', error: %v", failedNodeName, testSourceName, mock.Anything)
	mockLogger.On("Errorf", "failed to execute event retention for node info with node name: '%s', source name: '%s', config DefaultMinCap: '%d', config DefaultDuration: %s, error: %v",
		failedNodeName, testSourceName, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockLogger.On("Debugf", "Starting execute event retention for node name: '%s', source name: '%s', maxCap: %d, minCap: %d, duration: %s", testNodeName, testSourceName, int64(-1), int64(0), testDuration).Return(nil)
	mockLogger.On("Debugf", "Starting execute event retention for node name: '%s', source name: '%s', maxCap: %d, minCap: %d, duration: %s", failedNodeName, testSourceName, int64(-1), int64(0), testDuration).Return(nil)
	mockLogger.On("Debugf", "MinCap is disabled, purge events by duration '%d' and nodeName '%s', and sourceName '%s'", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("Debug", "Starting purge events by retention policies on default config level ......").Return(nil)

	validConfig := config.EventRetention{
		Interval:        "10m",
		DefaultMaxCap:   -1,
		DefaultMinCap:   0,
		DefaultDuration: testDuration,
	}
	invalidDurationConfig := validConfig
	invalidDurationConfig.DefaultDuration = "abc"
	maxCapConfig := validConfig
	maxCapConfig.DefaultMaxCap = 1000

	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
		bootstrapContainer.LoggingClientInterfaceName: func(get di.Get) interface{} {
			return mockLogger
		},
		container.ConfigurationName: func(get di.Get) interface{} {
			return &config.ConfigurationStruct{
				Writable: config.WritableInfo{
					PersistData: true,
				},
				Service: bootstrapConfig.ServiceInfo{
					MaxResultCount: 20,
				},
			}
		},
	})

	tests := []struct {
		name                string
		nodeInfos           []dbModels.NodeInfo
		retentionConfig     config.EventRetention
		expectParseError    bool
		expectLogError      bool
		expectLogErrorCount int
	}{
		{"Valid - successfully purge events by node info", []dbModels.NodeInfo{testNodeInfo}, validConfig, false, false, 0},
		{"Valid - no node info", []dbModels.NodeInfo{}, validConfig, false, false, 0},
		{"Invalid - invalid duration", []dbModels.NodeInfo{failedNodeInfo}, invalidDurationConfig, false, true, 1},
		{"Invalid - failed to delete events with maxCap", []dbModels.NodeInfo{failedNodeInfo}, maxCapConfig, false, true, 2},
		{"Invalid - failed to delete events with node name", []dbModels.NodeInfo{failedNodeInfo}, validConfig, false, true, 3},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			nodeInfoCache := cache.NewNodeInfoCache(dic, testCase.nodeInfos)
			coreDataConfig := container.ConfigurationFrom(dic.Get)
			coreDataConfig.Retention = testCase.retentionConfig
			dic.Update(di.ServiceConstructorMap{
				container.NodeInfoCacheInterfaceName: func(get di.Get) interface{} {
					return nodeInfoCache
				},
				container.ConfigurationName: func(get di.Get) interface{} { return coreDataConfig },
			})

			ep := newEventPurgeExecutor(nodeInfoCache.CloneNodeInfoMapWithSourceName())
			err := ep.purgeEventsByNodeInfo(dic)

			if testCase.expectParseError {
				assert.Error(t, err, "expected error when purge events by node info")
			} else {
				assert.NoError(t, err, "expected no error when purge events by node info")
			}
			if testCase.expectLogError {
				mockLogger.AssertNumberOfCalls(t, "Errorf", testCase.expectLogErrorCount)
			}
		})
	}
}

func TestHandlePurgeEvents(t *testing.T) {
	testResourceName := "test-resource"
	testNodeInfo := dbModels.NodeInfo{
		Id:           1,
		NodeName:     testNodeName,
		SourceName:   testSourceName,
		ResourceName: testResourceName,
		ValueType:    common.ValueTypeString,
	}

	testDuration := "3d10h"

	isValid, duration := common.ParseDurationWithDay(testDuration)
	assert.True(t, isValid)

	dic := di.NewContainer(di.ServiceConstructorMap{})

	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("DeleteEventsByAgeAndNodeNameAndSourceName", duration.Nanoseconds(), testNodeName, testSourceName).Return(nil)
	dbClientMock.On("EventCountByNodeNameAndSourceNameAndLimit", testNodeName, testSourceName, 1000).Return(int64(0), errors.NewCommonEdgeX(errors.KindServerError, "failed to get event count", nil))

	mockLogger := &lcMocks.LoggingClient{}
	mockLogger.On("Debugf", "Starting execute event retention for node name: '%s', source name: '%s', maxCap: %d, minCap: %d, duration: %s", testNodeName, testSourceName, int64(0), int64(0), testDuration).Return(nil)
	mockLogger.On("Debugf", "MinCap is disabled, purge events by duration '%d' and nodeName '%s', and sourceName '%s'", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	validConfig := config.EventRetention{
		Interval:        "10m",
		DefaultMaxCap:   -1,
		DefaultMinCap:   0,
		DefaultDuration: testDuration,
	}

	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
		bootstrapContainer.LoggingClientInterfaceName: func(get di.Get) interface{} {
			return mockLogger
		},
		container.ConfigurationName: func(get di.Get) interface{} {
			return &config.ConfigurationStruct{
				Writable: config.WritableInfo{
					PersistData: true,
				},
				Service: bootstrapConfig.ServiceInfo{
					MaxResultCount: 20,
				},
			}
		},
	})

	nodeInfoCache := cache.NewNodeInfoCache(dic, []dbModels.NodeInfo{testNodeInfo})
	coreDataConfig := container.ConfigurationFrom(dic.Get)
	coreDataConfig.Retention = validConfig
	dic.Update(di.ServiceConstructorMap{
		container.NodeInfoCacheInterfaceName: func(get di.Get) interface{} {
			return nodeInfoCache
		},
		container.ConfigurationName: func(get di.Get) interface{} { return coreDataConfig },
	})

	err := handleEventRetention(testNodeName, testSourceName, 0, 0, testDuration, dic)
	assert.NoError(t, err, "expected no error when purge events by node info")
}
