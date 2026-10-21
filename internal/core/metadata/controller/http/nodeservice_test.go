//
// Copyright (C) 2020-2023 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clint456/edgex-go/internal/core/metadata/container"
	dbMock "github.com/clint456/edgex-go/internal/core/metadata/infrastructure/interfaces/mocks"

	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	commonDTO "github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/requests"
	responseDTO "github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/responses"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var testNodeServiceName = "TestNodeService"
var testNodeServiceLabels = []string{"hvac", "thermostat"}
var testBaseAddress = "http://home-node-service:49990"

func buildTestNodeServiceRequest() requests.AddNodeServiceRequest {
	var testAddNodeServiceReq = requests.AddNodeServiceRequest{
		BaseRequest: commonDTO.BaseRequest{
			RequestId:   ExampleUUID,
			Versionable: commonDTO.NewVersionable(),
		},
		Service: dtos.NodeService{
			Id:          ExampleUUID,
			Name:        testNodeServiceName,
			Description: TestDescription,
			Labels:      testNodeServiceLabels,
			AdminState:  models.Unlocked,
			BaseAddress: testBaseAddress,
		},
	}

	return testAddNodeServiceReq
}

func buildTestUpdateNodeServiceRequest() requests.UpdateNodeServiceRequest {
	testUUID := ExampleUUID
	testAdminState := models.Unlocked
	var testUpdateNodeServiceReq = requests.UpdateNodeServiceRequest{
		BaseRequest: commonDTO.BaseRequest{
			RequestId:   ExampleUUID,
			Versionable: commonDTO.NewVersionable(),
		},
		Service: dtos.UpdateNodeService{
			Id:          &testUUID,
			Name:        &testNodeServiceName,
			Labels:      testNodeServiceLabels,
			AdminState:  &testAdminState,
			BaseAddress: &testBaseAddress,
		},
	}

	return testUpdateNodeServiceReq
}

func buildTestDBClient(dsModel models.NodeService, errKind errors.ErrKind, errorMessage string) *dbMock.DBClient {
	dbClientMock := &dbMock.DBClient{}
	if len(errKind) > 0 {
		err := errors.NewCommonEdgeX(errKind, errorMessage, nil)
		dbClientMock.On("AddNodeService", dsModel).Return(dsModel, err)
	} else {
		dbClientMock.On("AddNodeService", dsModel).Return(dsModel, nil)
	}
	return dbClientMock
}

func TestAddNodeService(t *testing.T) {
	validReq := buildTestNodeServiceRequest()
	dsModels := requests.AddNodeServiceReqToNodeServiceModels([]requests.AddNodeServiceRequest{validReq})
	expectedRequestId := ExampleUUID

	reqWithNoID := validReq
	reqWithNoID.RequestId = ""
	reqWithInvalidId := validReq
	reqWithInvalidId.RequestId = "InvalidUUID"
	reqWithNoName := validReq
	reqWithNoName.Service.Name = ""

	tests := []struct {
		name                   string
		isValidRequest         bool
		dbClientMock           *dbMock.DBClient
		Request                []requests.AddNodeServiceRequest
		expectedHttpStatusCode int
	}{
		{
			"Request Normal",
			true,
			buildTestDBClient(dsModels[0], "", ""),
			[]requests.AddNodeServiceRequest{validReq},
			http.StatusCreated,
		},
		{
			"Request without requestId",
			true,
			buildTestDBClient(dsModels[0], "", ""),
			[]requests.AddNodeServiceRequest{reqWithNoID},
			http.StatusCreated,
		},
		{
			"Request with duplicate service name",
			true,
			buildTestDBClient(dsModels[0], errors.KindDuplicateName, "duplicate service name error"),
			[]requests.AddNodeServiceRequest{validReq},
			http.StatusConflict,
		},
		{
			"Request with invalid requestId",
			false,
			buildTestDBClient(dsModels[0], "", ""),
			[]requests.AddNodeServiceRequest{reqWithInvalidId},
			http.StatusBadRequest,
		},
		{
			"Request without service name",
			false,
			buildTestDBClient(dsModels[0], "", ""),
			[]requests.AddNodeServiceRequest{reqWithNoName},
			http.StatusBadRequest,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {

			e := echo.New()
			dic := mockDic()
			dic.Update(di.ServiceConstructorMap{
				container.DBClientInterfaceName: func(get di.Get) interface{} {
					return testCase.dbClientMock
				},
			})

			controller := NewNodeServiceController(dic)
			require.NotNil(t, controller)

			jsonData, err := json.Marshal(testCase.Request)
			require.NoError(t, err)

			reader := strings.NewReader(string(jsonData))
			req, err := http.NewRequest(http.MethodPost, common.ApiNodeServiceRoute, reader)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.AddNodeService(c)
			require.NoError(t, err)

			if testCase.isValidRequest {
				var res []commonDTO.BaseWithIdResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)

				// Assert
				require.NoError(t, err)
				assert.Equal(t, http.StatusMultiStatus, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, common.ApiVersion, res[0].ApiVersion, "API Version not as expected")
				if res[0].RequestId != "" {
					assert.Equal(t, expectedRequestId, res[0].RequestId, "RequestID not as expected")
				}
				assert.Equal(t, testCase.expectedHttpStatusCode, res[0].StatusCode, "BaseResponse status code not as expected")
				if testCase.expectedHttpStatusCode == http.StatusCreated {
					assert.Empty(t, res[0].Message, "Message should be empty when it is successful")
				} else {
					assert.NotEmpty(t, res[0].Message, "Response message doesn't contain the error message")
				}
			} else {
				var res commonDTO.BaseResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				assert.NoError(t, err)
				assert.Equal(t, testCase.expectedHttpStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedHttpStatusCode, res.StatusCode, "BaseResponse status code not as expected")
				assert.NotEmpty(t, res.Message, "Response message doesn't contain the error message")
			}
		})
	}
}

func TestNodeServiceByName(t *testing.T) {
	nodeService := dtos.ToNodeServiceModel(buildTestNodeServiceRequest().Service)
	emptyName := ""
	notFoundName := "notFoundName"

	dic := mockDic()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("NodeServiceByName", nodeService.Name).Return(nodeService, nil)
	dbClientMock.On("NodeServiceByName", notFoundName).Return(models.NodeService{}, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "node service doesn't exist in the database", nil))
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	controller := NewNodeServiceController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name               string
		nodeServiceName    string
		errorExpected      bool
		expectedStatusCode int
	}{
		{"Valid - find node service by name", nodeService.Name, false, http.StatusOK},
		{"Invalid - name parameter is empty", emptyName, true, http.StatusBadRequest},
		{"Invalid - node service not found by name", notFoundName, true, http.StatusNotFound},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			reqPath := fmt.Sprintf("%s/%s", common.ApiNodeServiceByNameRoute, testCase.nodeServiceName)
			req, err := http.NewRequest(http.MethodGet, reqPath, http.NoBody)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Name)
			c.SetParamValues(testCase.nodeServiceName)
			err = controller.NodeServiceByName(c)
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
				var res responseDTO.NodeServiceResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.nodeServiceName, res.Service.Name, "Name not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestPatchNodeService(t *testing.T) {
	expectedRequestId := ExampleUUID
	dic := mockDic()
	dbClientMock := &dbMock.DBClient{}
	testReq := buildTestUpdateNodeServiceRequest()
	dsModels := models.NodeService{
		Id:          *testReq.Service.Id,
		Name:        *testReq.Service.Name,
		Labels:      testReq.Service.Labels,
		AdminState:  models.AdminState(*testReq.Service.AdminState),
		BaseAddress: *testReq.Service.BaseAddress,
	}

	valid := testReq
	dbClientMock.On("NodeServiceById", *valid.Service.Id).Return(dsModels, nil)
	dbClientMock.On("UpdateNodeService", mock.Anything).Return(nil)
	validWithNoReqID := testReq
	validWithNoReqID.RequestId = ""
	validWithNoId := testReq
	validWithNoId.Service.Id = nil
	dbClientMock.On("NodeServiceByName", *validWithNoId.Service.Name).Return(dsModels, nil)
	validWithNoName := testReq
	validWithNoName.Service.Name = nil

	invalidId := testReq
	invalidUUID := "invalidUUID"
	invalidId.Service.Id = &invalidUUID

	emptyString := ""
	emptyId := testReq
	emptyId.Service.Id = &emptyString
	emptyId.Service.Name = nil
	emptyName := testReq
	emptyName.Service.Id = nil
	emptyName.Service.Name = &emptyString

	invalidNoIdAndName := testReq
	invalidNoIdAndName.Service.Id = nil
	invalidNoIdAndName.Service.Name = nil

	invalidNotFoundId := testReq
	invalidNotFoundId.Service.Name = nil
	notFoundId := "12345678-1111-1234-5678-de9dac3fb9bc"
	invalidNotFoundId.Service.Id = &notFoundId
	notFoundIdError := errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("%s doesn't exist in the database", notFoundId), nil)
	dbClientMock.On("NodeServiceById", *invalidNotFoundId.Service.Id).Return(dsModels, notFoundIdError)

	invalidNotFoundName := testReq
	invalidNotFoundName.Service.Id = nil
	notFoundName := "notFoundName"
	invalidNotFoundName.Service.Name = &notFoundName
	notFoundNameError := errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("%s doesn't exist in the database", notFoundName), nil)
	dbClientMock.On("NodeServiceByName", *invalidNotFoundName.Service.Name).Return(dsModels, notFoundNameError)

	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})
	controller := NewNodeServiceController(dic)
	require.NotNil(t, controller)
	tests := []struct {
		name                 string
		request              []requests.UpdateNodeServiceRequest
		expectedStatusCode   int
		expectedResponseCode int
	}{
		{"Valid", []requests.UpdateNodeServiceRequest{valid}, http.StatusMultiStatus, http.StatusOK},
		{"Valid - no requestId", []requests.UpdateNodeServiceRequest{validWithNoReqID}, http.StatusMultiStatus, http.StatusOK},
		{"Valid - no id", []requests.UpdateNodeServiceRequest{validWithNoId}, http.StatusMultiStatus, http.StatusOK},
		{"Valid - no name", []requests.UpdateNodeServiceRequest{validWithNoName}, http.StatusMultiStatus, http.StatusOK},
		{"Invalid - invalid id", []requests.UpdateNodeServiceRequest{invalidId}, http.StatusBadRequest, http.StatusBadRequest},
		{"Invalid - empty id", []requests.UpdateNodeServiceRequest{emptyId}, http.StatusBadRequest, http.StatusBadRequest},
		{"Invalid - empty name", []requests.UpdateNodeServiceRequest{emptyName}, http.StatusBadRequest, http.StatusBadRequest},
		{"Invalid - no id and name", []requests.UpdateNodeServiceRequest{invalidNoIdAndName}, http.StatusBadRequest, http.StatusBadRequest},
		{"Invalid - not found id", []requests.UpdateNodeServiceRequest{invalidNotFoundId}, http.StatusMultiStatus, http.StatusNotFound},
		{"Invalid - not found name", []requests.UpdateNodeServiceRequest{invalidNotFoundName}, http.StatusMultiStatus, http.StatusNotFound},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			jsonData, err := json.Marshal(testCase.request)
			require.NoError(t, err)

			reader := strings.NewReader(string(jsonData))
			req, err := http.NewRequest(http.MethodPost, common.ApiNodeServiceRoute, reader)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.PatchNodeService(c)
			require.NoError(t, err)

			if testCase.expectedStatusCode == http.StatusMultiStatus {
				var res []commonDTO.BaseResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)

				// Assert
				assert.Equal(t, http.StatusMultiStatus, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, common.ApiVersion, res[0].ApiVersion, "API Version not as expected")
				if res[0].RequestId != "" {
					assert.Equal(t, expectedRequestId, res[0].RequestId, "RequestID not as expected")
				}
				assert.Equal(t, testCase.expectedResponseCode, res[0].StatusCode, "BaseResponse status code not as expected")
				if testCase.expectedResponseCode == http.StatusOK {
					assert.Empty(t, res[0].Message, "Message should be empty when it is successful")
				} else {
					assert.NotEmpty(t, res[0].Message, "Response message doesn't contain the error message")
				}
			} else {
				var res commonDTO.BaseResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)

				// Assert
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedResponseCode, res.StatusCode, "BaseResponse status code not as expected")
				assert.NotEmpty(t, res.Message, "Response message doesn't contain the error message")
			}

		})
	}

}

func TestAllNodeServices(t *testing.T) {
	nodeServices := []models.NodeService{
		{
			Name:   "ds1",
			Labels: testNodeServiceLabels,
		},
		{
			Name:   "ds2",
			Labels: testNodeServiceLabels,
		},
		{
			Name: "ds3",
		},
	}
	expectedTotalNodeServiceCount := int64(len(nodeServices))

	dic := mockDic()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("NodeServiceCountByLabels", []string(nil)).Return(expectedTotalNodeServiceCount, nil)
	dbClientMock.On("NodeServiceCountByLabels", testNodeServiceLabels).Return(expectedTotalNodeServiceCount, nil)
	dbClientMock.On("AllNodeServices", 0, 10, []string(nil)).Return(nodeServices, nil)
	dbClientMock.On("AllNodeServices", 0, 5, testNodeServiceLabels).Return([]models.NodeService{nodeServices[0], nodeServices[1]}, nil)
	dbClientMock.On("AllNodeServices", 1, 2, []string(nil)).Return([]models.NodeService{nodeServices[1], nodeServices[2]}, nil)
	dbClientMock.On("AllNodeServices", 4, 1, testNodeServiceLabels).Return([]models.NodeService{}, errors.NewCommonEdgeX(errors.KindRangeNotSatisfiable, "query objects bounds out of range.", nil))
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})
	controller := NewNodeServiceController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name               string
		offset             string
		limit              string
		labels             string
		errorExpected      bool
		expectedCount      int
		expectedTotalCount int64
		expectedStatusCode int
	}{
		{"Valid - get node services without labels", "0", "10", "", false, 3, expectedTotalNodeServiceCount, http.StatusOK},
		{"Valid - get node services with labels", "0", "5", strings.Join(testNodeServiceLabels, ","), false, 2, expectedTotalNodeServiceCount, http.StatusOK},
		{"Valid - get node services with offset and no labels", "1", "2", "", false, 2, expectedTotalNodeServiceCount, http.StatusOK},
		{"Invalid - offset out of range", "4", "1", strings.Join(testNodeServiceLabels, ","), true, 0, expectedTotalNodeServiceCount, http.StatusRequestedRangeNotSatisfiable},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiAllNodeServiceRoute, http.NoBody)
			query := req.URL.Query()
			query.Add(common.Offset, testCase.offset)
			query.Add(common.Limit, testCase.limit)
			if len(testCase.labels) > 0 {
				query.Add(common.Labels, testCase.labels)
			}
			req.URL.RawQuery = query.Encode()
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.AllNodeServices(c)
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
				var res responseDTO.MultiNodeServicesResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.expectedCount, len(res.Services), "Service count not as expected")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestDeleteNodeServiceByName(t *testing.T) {
	nodeService := dtos.ToNodeServiceModel(buildTestNodeServiceRequest().Service)
	noName := ""
	notFoundName := "notFoundName"
	nodeExists := "nodeExists"
	provisionWatcherExists := "provisionWatcherExists"

	dic := mockDic()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("NodeServiceByName", nodeService.Name).Return(models.NodeService{}, nil)
	dbClientMock.On("NodesByServiceName", 0, 1, nodeService.Name).Return([]models.Node{}, nil)
	dbClientMock.On("ProvisionWatchersByServiceName", 0, 1, nodeService.Name).Return([]models.ProvisionWatcher{}, nil)
	dbClientMock.On("DeleteNodeServiceByName", nodeService.Name).Return(nil)

	dbClientMock.On("NodeServiceByName", notFoundName).Return(models.NodeService{}, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "node service doesn't exist in the database", nil))

	dbClientMock.On("NodeServiceByName", nodeExists).Return(models.NodeService{}, nil)
	dbClientMock.On("NodesByServiceName", 0, 1, nodeExists).Return([]models.Node{{ServiceName: testNodeServiceName}}, nil)

	dbClientMock.On("NodeServiceByName", provisionWatcherExists).Return(models.NodeService{}, nil)
	dbClientMock.On("NodesByServiceName", 0, 1, provisionWatcherExists).Return([]models.Node{}, nil)
	dbClientMock.On("ProvisionWatchersByServiceName", 0, 1, provisionWatcherExists).Return([]models.ProvisionWatcher{models.ProvisionWatcher{}}, nil)

	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	controller := NewNodeServiceController(dic)
	require.NotNil(t, controller)

	tests := []struct {
		name               string
		nodeServiceName    string
		errorExpected      bool
		expectedStatusCode int
	}{
		{"Valid - delete node service by name", nodeService.Name, false, http.StatusOK},
		{"Invalid - name parameter is empty", noName, true, http.StatusBadRequest},
		{"Invalid - node service not found by name", notFoundName, true, http.StatusNotFound},
		{"Invalid - associated node exists", nodeExists, true, http.StatusConflict},
		{"Invalid - associated provisionWatcher Exists", provisionWatcherExists, true, http.StatusConflict},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			reqPath := fmt.Sprintf("%s/%s", common.ApiNodeServiceByNameRoute, testCase.nodeServiceName)
			req, err := http.NewRequest(http.MethodGet, reqPath, http.NoBody)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Name)
			c.SetParamValues(testCase.nodeServiceName)
			err = controller.DeleteNodeServiceByName(c)
			require.NoError(t, err)

			var res commonDTO.BaseResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &res)
			require.NoError(t, err)

			// Assert
			assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
			assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
			assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
			if testCase.errorExpected {
				assert.NotEmpty(t, res.Message, "Response message doesn't contain the error message")
			} else {
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}
