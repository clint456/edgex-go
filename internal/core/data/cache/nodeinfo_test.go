//
// Copyright (C) 2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"testing"

	"github.com/clint456/edgex-go/internal/pkg/infrastructure/models"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/interfaces/mocks"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	testNodeName     = "test-node"
	testResourceName = "test-resource"
	testSourceName   = "testSource"
	testNodeInfo     = models.NodeInfo{
		Id:           1,
		NodeName:     testNodeName,
		SourceName:   testSourceName,
		ResourceName: testResourceName,
		ValueType:    common.ValueTypeString,
	}
)

func mockDic() *di.Container {
	mockMetricsManager := &mocks.MetricsManager{}
	mockMetricsManager.On("Register", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockMetricsManager.On("Unregister", mock.Anything)
	return di.NewContainer(di.ServiceConstructorMap{
		bootstrapContainer.MetricsManagerInterfaceName: func(get di.Get) interface{} {
			return mockMetricsManager
		},
		bootstrapContainer.LoggingClientInterfaceName: func(get di.Get) interface{} {
			return logger.NewMockClient()
		},
	})
}

func TestNodeInfoCache_GetNodeInfoId(t *testing.T) {
	dic := mockDic()
	cache := NewNodeInfoCache(dic, []models.NodeInfo{testNodeInfo})

	id, exist := cache.GetNodeInfoId(testNodeInfo)
	assert.Equal(t, testNodeInfo.Id, id)
	assert.True(t, exist)

	invalidNodeInfo := testNodeInfo
	invalidNodeInfo.Id = 2
	invalidNodeInfo.SourceName = "invalid"
	_, exist = cache.GetNodeInfoId(invalidNodeInfo)
	assert.False(t, exist)
}

func TestNodeInfoCache_GetNodeInfoMap(t *testing.T) {
	dic := mockDic()
	cache := NewNodeInfoCache(dic, []models.NodeInfo{testNodeInfo})

	result := cache.CloneNodeInfoMapWithSourceName()
	assert.Equal(t, 1, len(result))
	assert.Equal(t, testNodeInfo, result[1])
}

func TestNodeInfoCache_Add(t *testing.T) {
	dic := mockDic()
	cache := NewNodeInfoCache(dic, []models.NodeInfo{})

	cache.Add(testNodeInfo)
	result := cache.CloneNodeInfoMapWithSourceName()
	assert.Equal(t, 1, len(result))
	assert.Equal(t, testNodeInfo, result[1])
}

func TestNodeInfoCache_Remove(t *testing.T) {
	dic := mockDic()

	testNodeInfo2 := testNodeInfo
	testNodeInfo2.Id = 2
	testNodeInfo2.NodeName = "testNode2"
	cache := NewNodeInfoCache(dic, []models.NodeInfo{testNodeInfo, testNodeInfo2})

	cache.Remove(testNodeInfo)
	result := cache.CloneNodeInfoMapWithSourceName()
	assert.Equal(t, 1, len(result))
	_, exists := cache.GetNodeInfoId(testNodeInfo)
	assert.False(t, exists)

	cache.Remove(testNodeInfo2)
	result = cache.CloneNodeInfoMapWithSourceName()
	assert.Equal(t, 0, len(result))
	_, exists = cache.GetNodeInfoId(testNodeInfo)
	assert.False(t, exists)
}
