package http

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clint456/edgex-go/internal/core/data/application"
	"github.com/clint456/edgex-go/internal/core/data/container"
	dbMock "github.com/clint456/edgex-go/internal/core/data/infrastructure/interfaces/mocks"
	"github.com/clint456/edgex-go/internal/core/data/mocks"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	commonDTO "github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/common"
	responseDTO "github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/responses"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"

	"github.com/labstack/echo/v4"
)

var (
	validAggFunc   = common.AvgFunc
	invalidAggFunc = "invalid"
)

func TestReadingTotalCount(t *testing.T) {
	expectedReadingCount := int64(656672)
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("ReadingTotalCount").Return(expectedReadingCount, nil)

	dic := mocks.NewMockDIC()
	app := application.NewCoreDataApp(dic)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
		application.CoreDataAppName: func(get di.Get) interface{} {
			return app
		},
	})
	rc := NewReadingController(dic)

	e := echo.New()
	req, err := http.NewRequest(http.MethodGet, common.ApiReadingCountRoute, http.NoBody)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	c := e.NewContext(req, recorder)
	err = rc.ReadingTotalCount(c)
	require.NoError(t, err)

	var actualResponse commonDTO.CountResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &actualResponse)
	require.NoError(t, err)
	assert.Equal(t, common.ApiVersion, actualResponse.ApiVersion, "API Version not as expected")
	assert.Equal(t, http.StatusOK, recorder.Result().StatusCode, "HTTP status code not as expected")
	assert.Equal(t, http.StatusOK, int(actualResponse.StatusCode), "Response status code not as expected")
	assert.Empty(t, actualResponse.Message, "Message should be empty when it is successful")
	assert.Equal(t, expectedReadingCount, actualResponse.Count, "Event count in the response body is not expected")
}

func TestAllReadings(t *testing.T) {
	totalCount := int64(0)
	dic := mocks.NewMockDIC()
	app := application.NewCoreDataApp(dic)

	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("ReadingTotalCount").Return(totalCount, nil)
	dbClientMock.On("AllReadings", 0, 20).Return([]models.Reading{}, nil)
	dbClientMock.On("AllReadings", 0, 1).Return([]models.Reading{}, nil)
	dbClientMock.On("AllReadingsAggregation", validAggFunc, 0, 10).Return([]models.Reading{}, nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
		application.CoreDataAppName: func(get di.Get) interface{} {
			return app
		},
	})
	controller := NewReadingController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name               string
		offset             string
		limit              string
		aggFunc            string
		errorExpected      bool
		expectedTotalCount int64
		expectedStatusCode int
	}{
		{"Valid - get readings without offset, and limit", "", "", "", false, totalCount, http.StatusOK},
		{"Valid - get readings with offset, and limit", "0", "1", "", false, totalCount, http.StatusOK},
		{"Invalid - invalid offset format", "aaa", "1", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - invalid limit format", "1", "aaa", "", true, totalCount, http.StatusBadRequest},
		{"Valid - get readings with aggregateFunc", "0", "10", validAggFunc, false, totalCount, http.StatusOK},
		{"Invalid - get readings with invalid aggregateFunc", "0", "1", invalidAggFunc, true, 0, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiAllReadingRoute, http.NoBody)
			query := req.URL.Query()
			if testCase.offset != "" {
				query.Add(common.Offset, testCase.offset)
			}
			if testCase.limit != "" {
				query.Add(common.Limit, testCase.limit)
			}
			if testCase.aggFunc != "" {
				query.Add(common.AggregateFunc, testCase.aggFunc)
			}
			req.URL.RawQuery = query.Encode()
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.AllReadings(c)
			require.NoError(t, err)

			// Assert
			if testCase.errorExpected {
				var res commonDTO.BaseResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, res.StatusCode, "Response status code not as expected")
				assert.NotEmpty(t, res.Message, "Response message doesn't contain the error message")
			} else {
				var res responseDTO.MultiReadingsResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, res.StatusCode, "Response status code not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
			}
		})
	}
}

func TestReadingsByTimeRange(t *testing.T) {
	totalCount := int64(0)
	dic := mocks.NewMockDIC()
	app := application.NewCoreDataApp(dic)

	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("ReadingCountByTimeRange", int64(0), int64(100)).Return(totalCount, nil)
	dbClientMock.On("ReadingsByTimeRange", int64(0), int64(100), 0, 10).Return([]models.Reading{}, nil)
	dbClientMock.On("AllReadingsAggregationByTimeRange", validAggFunc, int64(0), int64(100), 0, 10).Return([]models.Reading{}, nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
		application.CoreDataAppName: func(get di.Get) interface{} {
			return app
		},
	})
	rc := NewReadingController(dic)
	assert.NotNil(t, rc)

	tests := []struct {
		name               string
		start              string
		end                string
		offset             string
		limit              string
		aggFunc            string
		errorExpected      bool
		expectedCount      int
		expectedTotalCount int64
		expectedStatusCode int
	}{
		{"Valid - with proper start/end/offset/limit", "0", "100", "0", "10", "", false, 0, totalCount, http.StatusOK},
		{"Invalid - invalid start format", "aaa", "100", "0", "10", "", true, 0, totalCount, http.StatusBadRequest},
		{"Invalid - invalid end format", "0", "bbb", "0", "10", "", true, 0, totalCount, http.StatusBadRequest},
		{"Invalid - empty start", "", "100", "0", "10", "", true, 0, totalCount, http.StatusBadRequest},
		{"Invalid - empty end", "0", "", "0", "10", "", true, 0, totalCount, http.StatusBadRequest},
		{"Invalid - end before start", "10", "0", "0", "10", "", true, 0, totalCount, http.StatusBadRequest},
		{"Invalid - invalid offset format", "0", "100", "aaa", "10", "", true, 0, totalCount, http.StatusBadRequest},
		{"Invalid - invalid limit format", "0", "100", "0", "aaa", "", true, 0, totalCount, http.StatusBadRequest},
		{"Valid - get readings by time range with aggregateFunc", "0", "100", "0", "10", validAggFunc, false, 0, totalCount, http.StatusOK},
		{"Invalid - get readings by time range with invalid aggregateFunc", "0", "100", "", "", invalidAggFunc, true, 0, totalCount, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiReadingByTimeRangeRoute, http.NoBody)
			query := req.URL.Query()
			query.Add(common.Offset, testCase.offset)
			query.Add(common.Limit, testCase.limit)
			query.Add(common.AggregateFunc, testCase.aggFunc)
			req.URL.RawQuery = query.Encode()
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Start, common.End)
			c.SetParamValues(testCase.start, testCase.end)
			err = rc.ReadingsByTimeRange(c)
			require.NoError(t, err)

			// Assert
			if testCase.errorExpected {
				var res commonDTO.BaseResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.NotEmpty(t, res.Message, "Response message doesn't contain the error message")
			} else {
				var res responseDTO.MultiReadingsResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.expectedCount, len(res.Readings), "Node count not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
			}
		})
	}
}

func TestReadingsByResourceName(t *testing.T) {
	totalCount := int64(0)
	dic := mocks.NewMockDIC()
	app := application.NewCoreDataApp(dic)

	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("ReadingCountByResourceName", TestResourceName).Return(totalCount, nil)
	dbClientMock.On("ReadingsByResourceName", 0, 20, TestResourceName).Return([]models.Reading{}, nil)
	dbClientMock.On("ReadingsByResourceName", 0, 1, TestResourceName).Return([]models.Reading{}, nil)
	dbClientMock.On("ReadingsAggregationByResourceName", TestResourceName, validAggFunc, 0, 10).Return([]models.Reading{}, nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
		application.CoreDataAppName: func(get di.Get) interface{} {
			return app
		},
	})
	controller := NewReadingController(dic)
	require.NotNil(t, controller)

	tests := []struct {
		name               string
		offset             string
		limit              string
		resourceName       string
		aggFunc            string
		errorExpected      bool
		expectedTotalCount int64
		expectedStatusCode int
	}{
		{"Valid - get readings without offset, and limit", "", "", TestResourceName, "", false, totalCount, http.StatusOK},
		{"Valid - get readings with offset, and limit", "0", "1", TestResourceName, "", false, totalCount, http.StatusOK},
		{"Invalid - invalid offset format", "aaa", "1", TestResourceName, "", true, totalCount, http.StatusBadRequest},
		{"Invalid - invalid limit format", "1", "aaa", TestResourceName, "", true, totalCount, http.StatusBadRequest},
		{"Valid - get readings by resource name with aggregateFunc", "0", "10", TestResourceName, validAggFunc, false, totalCount, http.StatusOK},
		{"Invalid - get readings by resource name with invalid aggregateFunc", "", "", TestResourceName, invalidAggFunc, true, totalCount, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiReadingByResourceNameRoute, http.NoBody)
			query := req.URL.Query()
			if testCase.offset != "" {
				query.Add(common.Offset, testCase.offset)
			}
			if testCase.limit != "" {
				query.Add(common.Limit, testCase.limit)
			}
			if testCase.aggFunc != "" {
				query.Add(common.AggregateFunc, testCase.aggFunc)
			}
			req.URL.RawQuery = query.Encode()
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.ResourceName)
			c.SetParamValues(testCase.resourceName)
			err = controller.ReadingsByResourceName(c)
			require.NoError(t, err)

			// Assert
			if testCase.errorExpected {
				var res commonDTO.BaseResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, res.StatusCode, "Response status code not as expected")
				assert.NotEmpty(t, res.Message, "Response message doesn't contain the error message")
			} else {
				var res responseDTO.MultiReadingsResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, res.StatusCode, "Response status code not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
			}
		})
	}
}

func TestReadingsByNodeName(t *testing.T) {
	totalCount := int64(0)
	dic := mocks.NewMockDIC()
	app := application.NewCoreDataApp(dic)

	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("ReadingCountByNodeName", TestNodeName).Return(totalCount, nil)
	dbClientMock.On("ReadingsByNodeName", 0, 20, TestNodeName).Return([]models.Reading{}, nil)
	dbClientMock.On("ReadingsByNodeName", 0, 1, TestNodeName).Return([]models.Reading{}, nil)
	dbClientMock.On("ReadingsAggregationByNodeName", TestNodeName, validAggFunc, 0, 10).Return([]models.Reading{}, nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
		application.CoreDataAppName: func(get di.Get) interface{} {
			return app
		},
	})
	controller := NewReadingController(dic)
	require.NotNil(t, controller)

	tests := []struct {
		name               string
		offset             string
		limit              string
		nodeName           string
		aggFunc            string
		errorExpected      bool
		expectedTotalCount int64
		expectedStatusCode int
	}{
		{"Valid - get readings without offset, and limit", "", "", TestNodeName, "", false, totalCount, http.StatusOK},
		{"Valid - get readings with offset, and limit", "0", "1", TestNodeName, "", false, totalCount, http.StatusOK},
		{"Invalid - invalid offset format", "aaa", "1", TestNodeName, "", true, totalCount, http.StatusBadRequest},
		{"Invalid - invalid limit format", "1", "aaa", TestNodeName, "", true, totalCount, http.StatusBadRequest},
		{"Valid - get readings by node name with aggregateFunc", "0", "10", TestNodeName, validAggFunc, false, totalCount, http.StatusOK},
		{"Invalid - get readings by node name with invalid aggregateFunc", "", "", TestNodeName, invalidAggFunc, true, totalCount, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiReadingByNodeNameRoute, http.NoBody)
			query := req.URL.Query()
			if testCase.offset != "" {
				query.Add(common.Offset, testCase.offset)
			}
			if testCase.limit != "" {
				query.Add(common.Limit, testCase.limit)
			}
			if testCase.aggFunc != "" {
				query.Add(common.AggregateFunc, testCase.aggFunc)
			}
			req.URL.RawQuery = query.Encode()
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Name)
			c.SetParamValues(testCase.nodeName)
			err = controller.ReadingsByNodeName(c)
			require.NoError(t, err)

			// Assert
			if testCase.errorExpected {
				var res commonDTO.BaseResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, res.StatusCode, "Response status code not as expected")
				assert.NotEmpty(t, res.Message, "Response message doesn't contain the error message")
			} else {
				var res responseDTO.MultiReadingsResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, res.StatusCode, "Response status code not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
			}
		})
	}
}

func TestReadingCountByNodeName(t *testing.T) {
	expectedReadingCount := int64(656672)
	nodeName := "nodeA"
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("ReadingCountByNodeName", nodeName).Return(expectedReadingCount, nil)

	dic := mocks.NewMockDIC()
	app := application.NewCoreDataApp(dic)

	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
		application.CoreDataAppName: func(get di.Get) interface{} {
			return app
		},
	})
	rc := NewReadingController(dic)

	e := echo.New()
	req, err := http.NewRequest(http.MethodGet, common.ApiReadingCountByNodeNameRoute, http.NoBody)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	c := e.NewContext(req, recorder)
	c.SetParamNames(common.Name)
	c.SetParamValues(nodeName)
	err = rc.ReadingCountByNodeName(c)
	require.NoError(t, err)

	var actualResponse commonDTO.CountResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &actualResponse)
	require.NoError(t, err)
	assert.Equal(t, common.ApiVersion, actualResponse.ApiVersion, "API Version not as expected")
	assert.Equal(t, http.StatusOK, recorder.Result().StatusCode, "HTTP status code not as expected")
	assert.Equal(t, http.StatusOK, int(actualResponse.StatusCode), "Response status code not as expected")
	assert.Empty(t, actualResponse.Message, "Message should be empty when it is successful")
	assert.Equal(t, expectedReadingCount, actualResponse.Count, "Reading count in the response body is not expected")
}

func TestReadingsByResourceNameAndTimeRange(t *testing.T) {
	totalCount := int64(0)
	dic := mocks.NewMockDIC()
	app := application.NewCoreDataApp(dic)

	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("ReadingCountByResourceNameAndTimeRange", TestResourceName, int64(0), int64(100)).Return(totalCount, nil)
	dbClientMock.On("ReadingsByResourceNameAndTimeRange", TestResourceName, int64(0), int64(100), 0, 10).Return([]models.Reading{}, nil)
	dbClientMock.On("ReadingsAggregationByResourceNameAndTimeRange", TestResourceName, validAggFunc, int64(0), int64(100), 0, 10).Return([]models.Reading{}, nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
		application.CoreDataAppName: func(get di.Get) interface{} {
			return app
		},
	})
	rc := NewReadingController(dic)
	assert.NotNil(t, rc)

	tests := []struct {
		name               string
		resourceName       string
		start              string
		end                string
		offset             string
		limit              string
		aggFunc            string
		errorExpected      bool
		expectedTotalCount int64
		expectedStatusCode int
	}{
		{"Valid ", TestResourceName, "0", "100", "0", "10", "", false, totalCount, http.StatusOK},
		{"Invalid - empty resourceName", "", "0", "100", "0", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - invalid start format", TestResourceName, "aaa", "100", "0", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - invalid end format", TestResourceName, "0", "bbb", "0", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - empty start", TestResourceName, "", "100", "0", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - empty end", TestResourceName, "0", "", "0", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - end before start", TestResourceName, "10", "0", "0", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - invalid offset format", TestResourceName, "0", "100", "aaa", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - invalid limit format", TestResourceName, "0", "100", "0", "aaa", "", true, totalCount, http.StatusBadRequest},
		{"Valid - get readings by resource name with aggregateFunc", TestResourceName, "0", "100", "0", "10", validAggFunc, false, totalCount, http.StatusOK},
		{"Invalid - get readings by resource name with invalid aggregateFunc", TestResourceName, "0", "100", "", TestNodeName, invalidAggFunc, true, totalCount, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiReadingByResourceNameAndTimeRangeRoute, http.NoBody)
			query := req.URL.Query()
			query.Add(common.Offset, testCase.offset)
			query.Add(common.Limit, testCase.limit)
			query.Add(common.AggregateFunc, testCase.aggFunc)
			req.URL.RawQuery = query.Encode()
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.ResourceName, common.Start, common.End)
			c.SetParamValues(testCase.resourceName, testCase.start, testCase.end)
			err = rc.ReadingsByResourceNameAndTimeRange(c)
			require.NoError(t, err)

			// Assert
			if testCase.errorExpected {
				var res commonDTO.BaseResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, res.StatusCode, "Response status code not as expected")
				assert.NotEmpty(t, res.Message, "Response message doesn't contain the error message")
			} else {
				var res responseDTO.MultiReadingsResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, res.StatusCode, "Response status code not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
			}
		})
	}
}

func TestReadingsByNodeNameAndResourceName(t *testing.T) {
	totalCount := int64(0)
	dic := mocks.NewMockDIC()
	app := application.NewCoreDataApp(dic)

	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("ReadingCountByNodeNameAndResourceName", TestNodeName, TestResourceName).Return(totalCount, nil)
	dbClientMock.On("ReadingsByNodeNameAndResourceName", TestNodeName, TestResourceName, 0, 20).Return([]models.Reading{}, nil)
	dbClientMock.On("ReadingsByNodeNameAndResourceName", TestNodeName, TestResourceName, 0, 1).Return([]models.Reading{}, nil)
	dbClientMock.On("ReadingsAggregationByNodeNameAndResourceName", TestNodeName, TestResourceName, validAggFunc, 0, 10).Return([]models.Reading{}, nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
		application.CoreDataAppName: func(get di.Get) interface{} {
			return app
		},
	})
	controller := NewReadingController(dic)
	require.NotNil(t, controller)

	tests := []struct {
		name               string
		nodeName           string
		resourceName       string
		offset             string
		limit              string
		aggFunc            string
		errorExpected      bool
		expectedTotalCount int64
		expectedStatusCode int
	}{
		{"valid - get readings without offset, and limit", TestNodeName, TestResourceName, "", "", "", false, totalCount, http.StatusOK},
		{"valid - get readings with offset, and limit", TestNodeName, TestResourceName, "0", "1", "", false, totalCount, http.StatusOK},
		{"invalid - empty nodeName", "", TestResourceName, "0", "1", "", true, totalCount, http.StatusBadRequest},
		{"invalid - empty resourceName", TestNodeName, "", "0", "1", "", true, totalCount, http.StatusBadRequest},
		{"invalid - invalid offset format", TestNodeName, TestResourceName, "aaa", "1", "", true, totalCount, http.StatusBadRequest},
		{"invalid - invalid limit format", TestNodeName, TestResourceName, "1", "aaa", "", true, totalCount, http.StatusBadRequest},
		{"Valid - get readings by node and resource with aggregateFunc", TestNodeName, TestResourceName, "0", "10", validAggFunc, false, totalCount, http.StatusOK},
		{"Invalid - get readings by node and resource with invalid aggregateFunc", TestNodeName, TestResourceName, "", "", invalidAggFunc, true, totalCount, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiReadingByNodeNameAndResourceNameRoute, http.NoBody)
			require.NoError(t, err)
			query := req.URL.Query()
			if testCase.offset != "" {
				query.Add(common.Offset, testCase.offset)
			}
			if testCase.limit != "" {
				query.Add(common.Limit, testCase.limit)
			}
			if testCase.aggFunc != "" {
				query.Add(common.AggregateFunc, testCase.aggFunc)
			}
			req.URL.RawQuery = query.Encode()
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Name, common.ResourceName)
			c.SetParamValues(testCase.nodeName, testCase.resourceName)
			err = controller.ReadingsByNodeNameAndResourceName(c)
			require.NoError(t, err)

			// Assert
			if testCase.errorExpected {
				var res commonDTO.BaseResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, res.StatusCode, "Response status code not as expected")
				assert.NotEmpty(t, res.Message, "Response message doesn't contain the error message")
			} else {
				var res responseDTO.MultiReadingsResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, res.StatusCode, "Response status code not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
			}
		})
	}
}

func TestReadingsByNodeNameAndResourceNameAndTimeRange(t *testing.T) {
	totalCount := int64(0)
	totalCountWithValue := int64(1)
	TestNodeSkipTotalCount := "TestNode_1"
	dic := mocks.NewMockDIC()
	app := application.NewCoreDataApp(dic)

	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("ReadingCountByNodeNameAndResourceNameAndTimeRange", TestNodeName, TestResourceName, int64(0), int64(100)).Return(totalCountWithValue, nil)
	dbClientMock.On("ReadingsByNodeNameAndResourceNameAndTimeRange", TestNodeName, TestResourceName, int64(0), int64(100), 0, 10).Return([]models.Reading{persistedReading}, nil)
	dbClientMock.On("ReadingsByNodeNameAndResourceNameAndTimeRange", TestNodeSkipTotalCount, TestResourceName, int64(0), int64(100), -1, 10).Return([]models.Reading{persistedReading}, nil)
	dbClientMock.On("ReadingsAggregationByNodeNameAndResourceNameAndTimeRange", TestNodeName, TestResourceName, validAggFunc, int64(0), int64(100), 0, 10).Return([]models.Reading{}, nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
		application.CoreDataAppName: func(get di.Get) interface{} {
			return app
		},
	})
	rc := NewReadingController(dic)
	assert.NotNil(t, rc)

	tests := []struct {
		name               string
		nodeName           string
		resourceName       string
		start              string
		end                string
		offset             string
		limit              string
		aggFunc            string
		errorExpected      bool
		expectedTotalCount int64
		expectedStatusCode int
	}{
		{"Valid ", TestNodeName, TestResourceName, "0", "100", "0", "10", "", false, totalCountWithValue, http.StatusOK},
		{"Valid with skip total count ", TestNodeSkipTotalCount, TestResourceName, "0", "100", "-1", "10", "", false, totalCount, http.StatusOK},
		{"Invalid - empty nodeName", "", TestResourceName, "0", "100", "0", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - empty resourceName", TestNodeName, "", "0", "100", "0", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - invalid start format", TestNodeName, TestResourceName, "aaa", "100", "0", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - invalid end format", TestNodeName, TestResourceName, "0", "bbb", "0", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - empty start", TestNodeName, TestResourceName, "", "100", "0", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - empty end", TestNodeName, TestResourceName, "0", "", "0", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - end before start", TestNodeName, TestResourceName, "10", "0", "0", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - invalid offset format", TestNodeName, TestResourceName, "0", "100", "aaa", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - invalid limit format", TestNodeName, TestResourceName, "0", "100", "0", "aaa", "", true, totalCount, http.StatusBadRequest},
		{"Valid - get readings by node, resource and time range with aggregateFunc", TestNodeName, TestResourceName, "0", "100", "0", "10", validAggFunc, false, totalCount, http.StatusOK},
		{"Invalid - get readings by node, resource and time range with invalid aggregateFunc", TestNodeName, TestResourceName, "0", "100", "", "", invalidAggFunc, true, totalCount, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiReadingByNodeNameAndResourceNameAndTimeRangeRoute, http.NoBody)
			require.NoError(t, err)
			query := req.URL.Query()
			query.Add(common.Offset, testCase.offset)
			query.Add(common.Limit, testCase.limit)
			query.Add(common.AggregateFunc, testCase.aggFunc)
			req.URL.RawQuery = query.Encode()
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Name, common.ResourceName, common.Start, common.End)
			c.SetParamValues(testCase.nodeName, testCase.resourceName, testCase.start, testCase.end)
			err = rc.ReadingsByNodeNameAndResourceNameAndTimeRange(c)
			require.NoError(t, err)

			// Assert
			if testCase.errorExpected {
				var res commonDTO.BaseResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, res.StatusCode, "Response status code not as expected")
				assert.NotEmpty(t, res.Message, "Response message doesn't contain the error message")
			} else {
				var res responseDTO.MultiReadingsResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, res.StatusCode, "Response status code not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
			}
		})
	}
}

func TestReadingsByNodeNameAndResourceNamesAndTimeRange(t *testing.T) {
	totalCount := int64(0)
	totalCountWithValue := int64(1)
	testResourceNames := []string{"resource01", "resource02"}
	emptyPayload := make(map[string]interface{})
	testResourceNamesPayload := emptyPayload
	testResourceNamesPayload[common.ResourceNames] = testResourceNames
	dic := mocks.NewMockDIC()
	app := application.NewCoreDataApp(dic)
	TestNodeSkipTotalCount := "TestNode_1"

	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("ReadingCountByNodeNameAndTimeRange", TestNodeName, int64(0), int64(100)).Return(totalCountWithValue, nil)
	dbClientMock.On("ReadingsByNodeNameAndTimeRange", TestNodeName, int64(0), int64(100), 0, 10).Return([]models.Reading{persistedReading}, nil)
	dbClientMock.On("ReadingsByNodeNameAndTimeRange", TestNodeSkipTotalCount, int64(0), int64(100), -1, 10).Return([]models.Reading{persistedReading}, nil)
	dbClientMock.On("ReadingCountByNodeNameAndResourceNamesAndTimeRange", TestNodeName, testResourceNames, int64(0), int64(100)).Return(totalCountWithValue, nil)
	dbClientMock.On("ReadingsByNodeNameAndResourceNamesAndTimeRange", TestNodeName, testResourceNames, int64(0), int64(100), 0, 10).Return([]models.Reading{persistedReading}, nil)
	dbClientMock.On("ReadingsByNodeNameAndResourceNamesAndTimeRange", TestNodeSkipTotalCount, testResourceNames, int64(0), int64(100), -1, 10).Return([]models.Reading{persistedReading}, nil)
	dbClientMock.On("ReadingsAggregationByNodeNameAndTimeRange", TestNodeName, validAggFunc, int64(0), int64(100), 0, 10).Return([]models.Reading{}, nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
		application.CoreDataAppName: func(get di.Get) interface{} {
			return app
		},
	})
	rc := NewReadingController(dic)
	assert.NotNil(t, rc)

	tests := []struct {
		name               string
		nodeName           string
		payload            map[string]interface{}
		start              string
		end                string
		offset             string
		limit              string
		aggFunc            string
		errorExpected      bool
		expectedTotalCount int64
		expectedStatusCode int
	}{
		{"Valid - provide nodeName and nil resourceNames", TestNodeName, nil, "0", "100", "0", "10", "", false, totalCountWithValue, http.StatusOK},
		{"Valid - provide nodeName and empty resourceNames", TestNodeName, emptyPayload, "0", "100", "0", "10", "", false, totalCountWithValue, http.StatusOK},
		{"Valid - provide nodeName and empty resourceNames and skip total count", TestNodeSkipTotalCount, emptyPayload, "0", "100", "-1", "10", "", false, totalCount, http.StatusOK},
		{"Valid - provide nodeName and resourceNames", TestNodeName, testResourceNamesPayload, "0", "100", "0", "10", "", false, totalCountWithValue, http.StatusOK},
		{"Valid - provide nodeName and resourceNames and skip total count", TestNodeSkipTotalCount, testResourceNamesPayload, "0", "100", "-1", "10", "", false, totalCount, http.StatusOK},
		{"Invalid - empty nodeName", "", testResourceNamesPayload, "0", "100", "0", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - invalid start format", TestNodeName, testResourceNamesPayload, "aaa", "100", "0", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - invalid end format", TestNodeName, testResourceNamesPayload, "0", "bbb", "0", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - empty start", TestNodeName, testResourceNamesPayload, "", "100", "0", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - empty end", TestNodeName, testResourceNamesPayload, "0", "", "0", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - end before start", TestNodeName, testResourceNamesPayload, "10", "0", "0", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - invalid offset format", TestNodeName, testResourceNamesPayload, "0", "100", "aaa", "10", "", true, totalCount, http.StatusBadRequest},
		{"Invalid - invalid limit format", TestNodeName, testResourceNamesPayload, "0", "100", "0", "aaa", "", true, totalCount, http.StatusBadRequest},
		{"Valid - get readings by node and time range with aggregateFunc", TestNodeName, nil, "0", "100", "0", "10", validAggFunc, false, totalCount, http.StatusOK},
		{"Invalid - get readings by node time range with invalid aggregateFunc", TestNodeName, nil, "0", "100", "", "", invalidAggFunc, true, totalCount, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			var reader io.Reader
			if testCase.payload != nil {
				byteData, err := toByteArray(common.ContentTypeJSON, testCase.payload)
				require.NoError(t, err)
				reader = strings.NewReader(string(byteData))
			} else {
				reader = http.NoBody
			}
			req, err := http.NewRequest(http.MethodGet, common.ApiReadingByNodeNameAndTimeRangeRoute, reader)
			req.Header.Set(common.ContentType, common.ContentTypeJSON)
			require.NoError(t, err)
			query := req.URL.Query()
			query.Add(common.Offset, testCase.offset)
			query.Add(common.Limit, testCase.limit)
			query.Add(common.AggregateFunc, testCase.aggFunc)
			req.URL.RawQuery = query.Encode()
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Name, common.Start, common.End)
			c.SetParamValues(testCase.nodeName, testCase.start, testCase.end)
			err = rc.ReadingsByNodeNameAndResourceNamesAndTimeRange(c)
			require.NoError(t, err)

			// Assert
			if testCase.errorExpected {
				var res commonDTO.BaseResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, res.StatusCode, "Response status code not as expected")
				assert.NotEmpty(t, res.Message, "Response message doesn't contain the error message")
			} else {
				var res responseDTO.MultiReadingsResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, res.StatusCode, "Response status code not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
			}
		})
	}
}
