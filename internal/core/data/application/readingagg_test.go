//
// Copyright (C) 2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"net/http"
	"testing"

	"github.com/clint456/edgex-go/internal/core/data/container"
	"github.com/clint456/edgex-go/internal/core/data/infrastructure/interfaces"
	dbMock "github.com/clint456/edgex-go/internal/core/data/infrastructure/interfaces/mocks"
	"github.com/clint456/edgex-go/internal/core/data/mocks"
	"github.com/clint456/edgex-go/internal/core/data/query"

	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"

	"github.com/stretchr/testify/assert"
)

var (
	node1       = "node1"
	node2       = "node2"
	resource1   = "resource1"
	resource2   = "resource2"
	invalidFunc = "invalid"
	validStart  = int64(1755244649710610498)
	validEnd    = int64(1765244649710610498)
	aggReading  = models.NumericReading{
		BaseReading: models.BaseReading{
			NodeName:     node1,
			ResourceName: resource1,
			ProfileName:  "TempProfile",
			ValueType:    common.ValueTypeUint16,
		},
		NumericValue: 9,
	}
	aggReading2 = models.NumericReading{
		BaseReading: models.BaseReading{
			NodeName:     node2,
			ResourceName: resource2,
			ProfileName:  "TempProfile",
			ValueType:    common.ValueTypeUint16,
		},
		NumericValue: 5,
	}
	aggReading3 = models.NumericReading{
		BaseReading: models.BaseReading{
			NodeName:     node2,
			ResourceName: resource1,
			ProfileName:  "TempProfile",
			ValueType:    common.ValueTypeInt8,
		},
		NumericValue: 2,
	}
	aggReading4 = models.NumericReading{
		BaseReading: models.BaseReading{
			NodeName:     node1,
			ResourceName: resource2,
			ProfileName:  "TempProfile",
			ValueType:    common.ValueTypeUint16,
		},
		NumericValue: 7,
	}
)

func TestAllReadingsAggregation(t *testing.T) {
	dic := mocks.NewMockDIC()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("AllReadingsAggregation", "MAX", 0, 0).Return([]models.Reading{aggReading, aggReading2}, nil)
	dbClientMock.On("AllReadingsAggregation", invalidFunc, 0, 0).Return(nil, errors.NewCommonEdgeX(errors.KindServerError, "unknown sql func", nil))

	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	app := NewCoreDataApp(dic)

	tests := []struct {
		name               string
		aggFunc            string
		errorExpected      bool
		expectedCount      int
		ExpectedErrKind    errors.ErrKind
		expectedStatusCode int
	}{
		{"Valid - all aggregated readings", common.MaxFunc, false, 2, "", http.StatusOK},
		{"InValid - invalid aggregate function", invalidFunc, true, 0, errors.KindServerError, http.StatusInternalServerError},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := app.AllAggregateReadings(testCase.aggFunc, dic, query.Parameters{})
			if testCase.errorExpected {
				assert.Error(t, err)
				assert.NotEmpty(t, err.Error(), "Error message is empty")
				assert.Equal(t, testCase.ExpectedErrKind, errors.Kind(err), "Error kind not as expected")
				assert.Equal(t, testCase.expectedStatusCode, err.Code(), "Status code not as expected")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expectedCount, len(result), "Result count not as expected")
			}
		})
	}
}

func TestAllAggregateReadingsByTimeRange(t *testing.T) {
	dic := mocks.NewMockDIC()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("AllReadingsAggregationByTimeRange", "MAX", validStart, validEnd, 0, 0).Return([]models.Reading{aggReading, aggReading2}, nil)
	dbClientMock.On("AllReadingsAggregationByTimeRange", invalidFunc, int64(0), int64(1), 0, 0).Return(nil, errors.NewCommonEdgeX(errors.KindServerError, "unknown sql func", nil))

	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	app := NewCoreDataApp(dic)

	tests := []struct {
		name               string
		aggFunc            string
		start              int64
		end                int64
		errorExpected      bool
		expectedCount      int
		ExpectedErrKind    errors.ErrKind
		expectedStatusCode int
	}{
		{"Valid - all aggregated readings by time range", common.MaxFunc, validStart, validEnd, false, 2, "", http.StatusOK},
		{"InValid - invalid aggregate function", invalidFunc, 0, 1, true, 0, errors.KindServerError, http.StatusInternalServerError},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := app.AllAggregateReadingsByTimeRange(testCase.aggFunc, query.Parameters{Start: testCase.start, End: testCase.end}, dic)
			if testCase.errorExpected {
				assert.Error(t, err)
				assert.NotEmpty(t, err.Error(), "Error message is empty")
				assert.Equal(t, testCase.ExpectedErrKind, errors.Kind(err), "Error kind not as expected")
				assert.Equal(t, testCase.expectedStatusCode, err.Code(), "Status code not as expected")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expectedCount, len(result), "Result count not as expected")
			}
		})
	}
}

func TestAggregateReadingsByResourceName(t *testing.T) {
	dic := mocks.NewMockDIC()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("ReadingsAggregationByResourceName", resource1, "MIN", 0, 0).Return([]models.Reading{aggReading, aggReading3}, nil)
	dbClientMock.On("ReadingsAggregationByResourceName", resource1, invalidFunc, 0, 0).Return(nil, errors.NewCommonEdgeX(errors.KindServerError, "unknown sql func", nil))

	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	app := NewCoreDataApp(dic)

	tests := []struct {
		name               string
		resourceName       string
		aggFunc            string
		errorExpected      bool
		expectedCount      int
		ExpectedErrKind    errors.ErrKind
		expectedStatusCode int
	}{
		{"Valid - all aggregated readings by resource", resource1, common.MinFunc, false, 2, "", http.StatusOK},
		{"InValid - invalid aggregate function", resource1, invalidFunc, true, 0, errors.KindServerError, http.StatusInternalServerError},
		{"InValid - empty resource name", "", "", true, 0, errors.KindContractInvalid, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := app.AggregateReadingsByResourceName(testCase.resourceName, testCase.aggFunc, dic, query.Parameters{})
			if testCase.errorExpected {
				assert.Error(t, err)
				assert.NotEmpty(t, err.Error(), "Error message is empty")
				assert.Equal(t, testCase.ExpectedErrKind, errors.Kind(err), "Error kind not as expected")
				assert.Equal(t, testCase.expectedStatusCode, err.Code(), "Status code not as expected")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expectedCount, len(result), "Result count not as expected")
			}
		})
	}
}

func TestAggregateReadingsByResourceNameAndTimeRange(t *testing.T) {
	dic := mocks.NewMockDIC()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("ReadingsAggregationByResourceNameAndTimeRange", resource1, "MIN", validStart, validEnd, 0, 0).Return([]models.Reading{aggReading, aggReading3}, nil)
	dbClientMock.On("ReadingsAggregationByResourceNameAndTimeRange", resource1, invalidFunc, int64(0), int64(1), 0, 0).Return(nil, errors.NewCommonEdgeX(errors.KindServerError, "unknown sql func", nil))

	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	app := NewCoreDataApp(dic)

	tests := []struct {
		name               string
		resourceName       string
		aggFunc            string
		start              int64
		end                int64
		errorExpected      bool
		expectedCount      int
		ExpectedErrKind    errors.ErrKind
		expectedStatusCode int
	}{
		{"Valid - all aggregated readings by resource and time range", resource1, common.MinFunc, validStart, validEnd, false, 2, "", http.StatusOK},
		{"InValid - invalid aggregate function", resource1, invalidFunc, int64(0), int64(1), true, 0, errors.KindServerError, http.StatusInternalServerError},
		{"InValid - empty resource name", "", "", int64(0), int64(1), true, 0, errors.KindContractInvalid, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := app.AggregateReadingsByResourceNameAndTimeRange(testCase.resourceName, testCase.aggFunc, query.Parameters{Start: testCase.start, End: testCase.end}, dic)
			if testCase.errorExpected {
				assert.Error(t, err)
				assert.NotEmpty(t, err.Error(), "Error message is empty")
				assert.Equal(t, testCase.ExpectedErrKind, errors.Kind(err), "Error kind not as expected")
				assert.Equal(t, testCase.expectedStatusCode, err.Code(), "Status code not as expected")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expectedCount, len(result), "Result count not as expected")
			}
		})
	}
}

func TestAggregateReadingsByNodeName(t *testing.T) {
	dic := mocks.NewMockDIC()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("ReadingsAggregationByNodeName", node1, "AVG", 0, 0).Return([]models.Reading{aggReading, aggReading4}, nil)
	dbClientMock.On("ReadingsAggregationByNodeName", node1, invalidFunc, 0, 0).Return(nil, errors.NewCommonEdgeX(errors.KindServerError, "unknown sql func", nil))

	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	app := NewCoreDataApp(dic)

	tests := []struct {
		name               string
		nodeName           string
		aggFunc            string
		errorExpected      bool
		expectedCount      int
		ExpectedErrKind    errors.ErrKind
		expectedStatusCode int
	}{
		{"Valid - all aggregated readings by node", node1, common.AvgFunc, false, 2, "", http.StatusOK},
		{"InValid - invalid aggregate function", node1, invalidFunc, true, 0, errors.KindServerError, http.StatusInternalServerError},
		{"InValid - empty node name", "", "", true, 0, errors.KindContractInvalid, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := app.AggregateReadingsByNodeName(testCase.nodeName, testCase.aggFunc, dic, query.Parameters{})
			if testCase.errorExpected {
				assert.Error(t, err)
				assert.NotEmpty(t, err.Error(), "Error message is empty")
				assert.Equal(t, testCase.ExpectedErrKind, errors.Kind(err), "Error kind not as expected")
				assert.Equal(t, testCase.expectedStatusCode, err.Code(), "Status code not as expected")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expectedCount, len(result), "Result count not as expected")
			}
		})
	}
}

func TestAggregateReadingsByNodeNameAndTimeRange(t *testing.T) {
	dic := mocks.NewMockDIC()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("ReadingsAggregationByNodeNameAndTimeRange", node1, "MIN", validStart, validEnd, 0, 0).Return([]models.Reading{aggReading, aggReading4}, nil)
	dbClientMock.On("ReadingsAggregationByNodeNameAndTimeRange", node1, invalidFunc, int64(0), int64(1), 0, 0).Return(nil, errors.NewCommonEdgeX(errors.KindServerError, "unknown sql func", nil))

	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	app := NewCoreDataApp(dic)

	tests := []struct {
		name               string
		nodeName           string
		aggFunc            string
		start              int64
		end                int64
		errorExpected      bool
		expectedCount      int
		ExpectedErrKind    errors.ErrKind
		expectedStatusCode int
	}{
		{"Valid - all aggregated readings by node and time range", node1, common.MinFunc, validStart, validEnd, false, 2, "", http.StatusOK},
		{"InValid - invalid aggregate function", node1, invalidFunc, int64(0), int64(1), true, 0, errors.KindServerError, http.StatusInternalServerError},
		{"InValid - empty node name", "", "", int64(0), int64(1), true, 0, errors.KindContractInvalid, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := app.AggregateReadingsByNodeNameAndTimeRange(testCase.nodeName, testCase.aggFunc, query.Parameters{Start: testCase.start, End: testCase.end}, dic)
			if testCase.errorExpected {
				assert.Error(t, err)
				assert.NotEmpty(t, err.Error(), "Error message is empty")
				assert.Equal(t, testCase.ExpectedErrKind, errors.Kind(err), "Error kind not as expected")
				assert.Equal(t, testCase.expectedStatusCode, err.Code(), "Status code not as expected")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expectedCount, len(result), "Result count not as expected")
			}
		})
	}
}

func TestAggregateReadingsByNodeNameAndResourceName(t *testing.T) {
	dic := mocks.NewMockDIC()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("ReadingsAggregationByNodeNameAndResourceName", node1, resource1, "COUNT", 0, -1).Return([]models.Reading{aggReading}, nil)
	dbClientMock.On("ReadingsAggregationByNodeNameAndResourceName", node1, resource1, invalidFunc, 0, -1).Return(nil, errors.NewCommonEdgeX(errors.KindServerError, "unknown sql func", nil))

	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	app := NewCoreDataApp(dic)

	tests := []struct {
		name               string
		nodeName           string
		resourceName       string
		aggFunc            string
		errorExpected      bool
		expectedCount      int
		ExpectedErrKind    errors.ErrKind
		expectedStatusCode int
	}{
		{"Valid - all aggregated readings by node and resource", node1, resource1, common.CountFunc, false, 1, "", http.StatusOK},
		{"InValid - invalid aggregate function", node1, resource1, invalidFunc, true, 0, errors.KindServerError, http.StatusInternalServerError},
		{"InValid - empty node name", "", resource1, "", true, 0, errors.KindContractInvalid, http.StatusBadRequest},
		{"InValid - empty resource name", node1, "", "", true, 0, errors.KindContractInvalid, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := app.AggregateReadingsByNodeNameAndResourceName(testCase.nodeName, testCase.resourceName, testCase.aggFunc, dic, query.Parameters{Offset: 0, Limit: -1})
			if testCase.errorExpected {
				assert.Error(t, err)
				assert.NotEmpty(t, err.Error(), "Error message is empty")
				assert.Equal(t, testCase.ExpectedErrKind, errors.Kind(err), "Error kind not as expected")
				assert.Equal(t, testCase.expectedStatusCode, err.Code(), "Status code not as expected")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expectedCount, len(result), "Result count not as expected")
			}
		})
	}
}

func TestAggregateReadingsByNodeNameAndResourceNameAndTimeRange(t *testing.T) {
	dic := mocks.NewMockDIC()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("ReadingsAggregationByNodeNameAndResourceNameAndTimeRange", node1, resource1, "MIN", validStart, validEnd, 0, 0).Return([]models.Reading{aggReading, aggReading4}, nil)
	dbClientMock.On("ReadingsAggregationByNodeNameAndResourceNameAndTimeRange", node1, resource1, invalidFunc, int64(0), int64(1), 0, 0).Return(nil, errors.NewCommonEdgeX(errors.KindServerError, "unknown sql func", nil))

	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	app := NewCoreDataApp(dic)

	tests := []struct {
		name               string
		nodeName           string
		resourceName       string
		aggFunc            string
		start              int64
		end                int64
		errorExpected      bool
		expectedCount      int
		ExpectedErrKind    errors.ErrKind
		expectedStatusCode int
	}{
		{"Valid - all aggregated readings by node and time range", node1, resource1, common.MinFunc, validStart, validEnd, false, 2, "", http.StatusOK},
		{"InValid - invalid aggregate function", node1, resource1, invalidFunc, int64(0), int64(1), true, 0, errors.KindServerError, http.StatusInternalServerError},
		{"InValid - empty node name", "", resource1, "", int64(0), int64(1), true, 0, errors.KindContractInvalid, http.StatusBadRequest},
		{"InValid - empty resource name", node1, "", "", int64(0), int64(1), true, 0, errors.KindContractInvalid, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := app.AggregateReadingsByNodeNameAndResourceNameAndTimeRange(testCase.nodeName, testCase.resourceName, testCase.aggFunc, query.Parameters{Start: testCase.start, End: testCase.end}, dic)
			if testCase.errorExpected {
				assert.Error(t, err)
				assert.NotEmpty(t, err.Error(), "Error message is empty")
				assert.Equal(t, testCase.ExpectedErrKind, errors.Kind(err), "Error kind not as expected")
				assert.Equal(t, testCase.expectedStatusCode, err.Code(), "Status code not as expected")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expectedCount, len(result), "Result count not as expected")
			}
		})
	}
}

func TestGetReadingAggregation(t *testing.T) {
	dic := mocks.NewMockDIC()

	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("ReadingsAggregationByNodeNameAndResourceNameAndTimeRange", node1, resource1, "MIN", validStart, validEnd, 0, -1).Return([]models.Reading{aggReading}, nil)
	expectedDTO := dtos.FromReadingModelToDTO(aggReading)

	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	aggDBFunc := func(dbClient interfaces.DBClient) ([]models.Reading, errors.EdgeX) {
		return dbClientMock.ReadingsAggregationByNodeNameAndResourceNameAndTimeRange(node1, resource1, "MIN", validStart, validEnd, 0, -1)
	}
	result, err := getReadingAggregation(dic, aggDBFunc)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result), "Result count not as expected")
	assert.Equal(t, expectedDTO, result[0], "Reading aggregated readings not as expected")
}
