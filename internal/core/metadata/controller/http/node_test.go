//
// Copyright (C) 2020-2024 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	messagingMocks "github.com/edgexfoundry/go-mod-messaging/v4/messaging/mocks"
	"github.com/edgexfoundry/go-mod-messaging/v4/pkg/types"
	"github.com/stretchr/testify/mock"

	"github.com/clint456/edgex-go/internal/core/metadata/container"
	dbMock "github.com/clint456/edgex-go/internal/core/metadata/infrastructure/interfaces/mocks"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	commonDTO "github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/requests"
	responseDTO "github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/responses"
	edgexErr "github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testNodeLabels = []string{"MODBUS", "TEMP"}

var testProperties = map[string]any{
	"TestProperty1": "property1",
	"TestProperty2": true,
	"TestProperty3": 123.45,
}

func buildTestNodeRequest() requests.AddNodeRequest {
	var testAutoEvents = []dtos.AutoEvent{
		{SourceName: "TestResource", Interval: "300ms", OnChange: true},
	}
	var testProtocols = map[string]dtos.ProtocolProperties{
		"modbus-ip": {
			"Address": "localhost",
			"Port":    "1502",
			"UnitID":  "1",
		},
	}
	var testAddNodeReq = requests.AddNodeRequest{
		BaseRequest: commonDTO.BaseRequest{
			RequestId:   ExampleUUID,
			Versionable: commonDTO.NewVersionable(),
		},
		Node: dtos.Node{
			Id:             ExampleUUID,
			Name:           TestNodeName,
			ServiceName:    TestNodeServiceName,
			ProfileName:    TestProfileName,
			AdminState:     models.Locked,
			OperatingState: models.Up,
			Labels:         testNodeLabels,
			Location:       "{40lat;45long}",
			AutoEvents:     testAutoEvents,
			Protocols:      testProtocols,
			Properties:     testProperties,
		},
	}

	return testAddNodeReq
}

func buildTestUpdateNodeRequest() requests.UpdateNodeRequest {
	testUUID := ExampleUUID
	testName := TestNodeName
	testDescription := TestDescription
	testServiceName := TestNodeServiceName
	testProfileName := TestProfileName
	testAdminState := models.Unlocked
	testOperatingState := models.Up
	var testAutoEvents = []dtos.AutoEvent{
		{SourceName: "TestResource", Interval: "300ms", OnChange: true},
	}
	var testProtocols = map[string]dtos.ProtocolProperties{
		"modbus-ip": {
			"Address": "localhost",
			"Port":    "1502",
			"UnitID":  "1",
		},
	}
	var testUpdateNodeReq = requests.UpdateNodeRequest{
		BaseRequest: commonDTO.BaseRequest{
			RequestId:   ExampleUUID,
			Versionable: commonDTO.NewVersionable(),
		},
		Node: dtos.UpdateNode{
			Id:             &testUUID,
			Name:           &testName,
			Description:    &testDescription,
			ServiceName:    &testServiceName,
			ProfileName:    &testProfileName,
			AdminState:     &testAdminState,
			OperatingState: &testOperatingState,
			Labels:         []string{"MODBUS", "TEMP"},
			Location:       "{40lat;45long}",
			AutoEvents:     testAutoEvents,
			Protocols:      testProtocols,
			Properties:     testProperties,
		},
	}

	return testUpdateNodeReq
}

func TestAddNode(t *testing.T) {
	testNode := buildTestNodeRequest()
	nodeModel := dtos.ToNodeModel(testNode.Node)
	expectedRequestId := ExampleUUID
	dic := mockDic()
	dbClientMock := &dbMock.DBClient{}

	valid := testNode
	dbClientMock.On("NodeServiceNameExists", nodeModel.ServiceName).Return(true, nil)
	dbClientMock.On("NodeNameExists", nodeModel.Name).Return(false, nil)
	dbClientMock.On("AddNode", nodeModel).Return(nodeModel, nil)
	dbClientMock.On("ProfileByName", mock.Anything).Return(models.Profile{Name: "test-profile", Resources: []models.Resource{{Name: "TestResource"}}}, nil)

	validForceAdd := testNode
	validForceAdd.Node.Name = "forceAdd"
	forceAddDm := dtos.ToNodeModel(validForceAdd.Node)
	dbClientMock.On("NodeNameExists", validForceAdd.Node.Name).Return(true, nil)
	dbClientMock.On("NodeByName", validForceAdd.Node.Name).Return(forceAddDm, nil)
	dbClientMock.On("UpdateNode", forceAddDm).Return(nil)

	notFoundProfile := testNode
	notFoundProfile.Node.ProfileName = "notFoundProfile"
	notFoundProfileNodeModel := requests.AddNodeReqToNodeModels([]requests.AddNodeRequest{notFoundProfile})[0]
	dbClientMock.On("AddNode", notFoundProfileNodeModel).Return(notFoundProfileNodeModel,
		edgexErr.NewCommonEdgeX(edgexErr.KindEntityDoesNotExist, fmt.Sprintf("node profile '%s' does not exists",
			notFoundProfile.Node.ProfileName), nil))

	notFoundService := testNode
	notFoundService.Node.ServiceName = "notFoundService"
	notFoundServiceNodeModel := requests.AddNodeReqToNodeModels([]requests.AddNodeRequest{notFoundService})[0]
	dbClientMock.On("NodeServiceNameExists", notFoundService.Node.ServiceName).Return(false, nil)
	dbClientMock.On("AddNode", notFoundServiceNodeModel).Return(notFoundServiceNodeModel,
		edgexErr.NewCommonEdgeX(edgexErr.KindContractInvalid, fmt.Sprintf("node service '%s' does not exists", notFoundService.Node.ServiceName), nil))

	noName := testNode
	noName.Node.Name = ""
	noAdminState := testNode
	noAdminState.Node.AdminState = ""
	noOperatingState := testNode
	noOperatingState.Node.OperatingState = ""
	invalidAdminState := testNode
	invalidAdminState.Node.AdminState = "invalidAdminState"
	invalidOperatingState := testNode
	invalidOperatingState.Node.OperatingState = "invalidOperatingState"
	noServiceName := testNode
	noServiceName.Node.ServiceName = ""
	noProfileName := testNode
	noProfileName.Node.ProfileName = ""
	np := dtos.ToNodeModel(noProfileName.Node)
	dbClientMock.On("AddNode", np).Return(np, nil)
	noProfileAndAutoEvents := noProfileName
	noProfileAndAutoEvents.Node.AutoEvents = []dtos.AutoEvent{}
	dm := dtos.ToNodeModel(noProfileAndAutoEvents.Node)
	dbClientMock.On("AddNode", dm).Return(dm, nil)
	noProtocols := testNode
	noProtocols.Node.Protocols = nil
	emptyProtocols := testNode
	emptyProtocols.Node.Protocols = map[string]dtos.ProtocolProperties{}
	emptyProtocolsModel := dtos.ToNodeModel(emptyProtocols.Node)
	dbClientMock.On("AddNode", emptyProtocolsModel).Return(emptyProtocolsModel, nil)
	invalidProtocols := testNode
	invalidProtocols.Node.Protocols = map[string]dtos.ProtocolProperties{"others": {}}
	ownParent := testNode
	ownParent.Node.Parent = ownParent.Node.Name

	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})
	controller := NewNodeController(dic)
	assert.NotNil(t, controller)
	tests := []struct {
		name                 string
		request              []requests.AddNodeRequest
		expectedStatusCode   int
		expectedResponseCode int
		expectedValidation   bool
		expectedSystemEvent  bool
		forceAdd             bool
	}{
		{"Valid", []requests.AddNodeRequest{valid}, http.StatusMultiStatus, http.StatusCreated, true, true, false},
		{"Valid - bypassValidation", []requests.AddNodeRequest{valid}, http.StatusMultiStatus, http.StatusCreated, false, true, false},
		{"Valid - no profile name and no auto events", []requests.AddNodeRequest{noProfileAndAutoEvents}, http.StatusMultiStatus, http.StatusCreated, true, true, false},
		{"Invalid - not found profile", []requests.AddNodeRequest{notFoundProfile}, http.StatusMultiStatus, http.StatusNotFound, true, false, false},
		{"Invalid - no name", []requests.AddNodeRequest{noName}, http.StatusBadRequest, http.StatusBadRequest, false, false, false},
		{"Invalid - no adminState", []requests.AddNodeRequest{noAdminState}, http.StatusBadRequest, http.StatusBadRequest, false, false, false},
		{"Invalid - no operatingState", []requests.AddNodeRequest{noOperatingState}, http.StatusBadRequest, http.StatusBadRequest, false, false, false},
		{"Invalid - invalid adminState", []requests.AddNodeRequest{invalidAdminState}, http.StatusBadRequest, http.StatusBadRequest, false, false, false},
		{"Invalid - invalid operatingState", []requests.AddNodeRequest{invalidOperatingState}, http.StatusBadRequest, http.StatusBadRequest, false, false, false},
		{"Invalid - no service name", []requests.AddNodeRequest{noServiceName}, http.StatusBadRequest, http.StatusBadRequest, false, false, false},
		{"Valid - no profile name", []requests.AddNodeRequest{noProfileName}, http.StatusMultiStatus, http.StatusCreated, true, true, false},
		{"Invalid - no protocols", []requests.AddNodeRequest{noProtocols}, http.StatusBadRequest, http.StatusBadRequest, false, false, false},
		{"Valid - empty protocols", []requests.AddNodeRequest{emptyProtocols}, http.StatusMultiStatus, http.StatusCreated, true, true, false},
		{"Invalid - invalid protocols", []requests.AddNodeRequest{invalidProtocols}, http.StatusMultiStatus, http.StatusInternalServerError, true, false, false},
		{"Invalid - not found node service", []requests.AddNodeRequest{notFoundService}, http.StatusMultiStatus, http.StatusBadRequest, false, false, false},
		{"Invalid - node service unavailable", []requests.AddNodeRequest{valid}, http.StatusMultiStatus, http.StatusServiceUnavailable, true, false, false},
		{"Valid - force add node", []requests.AddNodeRequest{validForceAdd}, http.StatusMultiStatus, http.StatusCreated, false, true, true},
		{"Invalid - own parent", []requests.AddNodeRequest{ownParent}, http.StatusMultiStatus, http.StatusBadRequest, false, false, false},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			jsonData, err := json.Marshal(testCase.request)
			require.NoError(t, err)

			var responseEnvelope types.MessageEnvelope
			mockMessaging := &messagingMocks.MessageClient{}
			if testCase.expectedValidation {
				switch testCase.expectedResponseCode {
				case http.StatusInternalServerError:
					mockMessaging.On("Request", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
						requestEnvelope, ok := args.Get(0).(types.MessageEnvelope)
						require.True(t, ok)
						responseEnvelope = types.NewMessageEnvelopeWithError(requestEnvelope.RequestID, "validation failed")
					}).Return(&responseEnvelope, nil)
				case http.StatusServiceUnavailable:
					mockMessaging.On("Request", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&responseEnvelope, errors.New("timed out"))
				default:
					mockMessaging.On("Request", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
						requestEnvelope, ok := args.Get(0).(types.MessageEnvelope)
						require.True(t, ok)
						responseEnvelope, err = types.NewMessageEnvelopeForResponse(nil, requestEnvelope.RequestID, requestEnvelope.CorrelationID, common.ContentTypeJSON)
						require.NoError(t, err)
					}).Return(&responseEnvelope, nil)
				}
			}

			var wg sync.WaitGroup
			if testCase.expectedSystemEvent {
				wg.Add(1)
				mockMessaging.On("Publish", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
					wg.Done()
				}).Return(nil)
			}

			dic.Update(di.ServiceConstructorMap{
				bootstrapContainer.MessagingClientName: func(get di.Get) interface{} {
					return mockMessaging
				},
			})

			reader := strings.NewReader(string(jsonData))
			req, err := http.NewRequest(http.MethodPost, common.ApiNodeRoute, reader)
			require.NoError(t, err)

			if !testCase.expectedValidation {
				query := req.URL.Query()
				query.Add(bypassValidationQueryParam, common.ValueTrue)
				req.URL.RawQuery = query.Encode()
			}

			if testCase.forceAdd {
				query := req.URL.Query()
				query.Add(forceQueryParam, common.ValueTrue)
				req.URL.RawQuery = query.Encode()
			}

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			handler := controller.AddNode
			err = handler(c)
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
				if testCase.expectedResponseCode == http.StatusCreated {
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

			wg.Wait()
			mockMessaging.AssertExpectations(t)
		})
	}
}

func TestDeleteNodeByName(t *testing.T) {
	node := dtos.ToNodeModel(buildTestNodeRequest().Node)
	noName := ""
	notFoundName := "notFoundName"
	nodeParent := node
	nodeParent.Name = "someOtherName"

	dic := mockDic()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("NodeTree", node.Name, 1, 0, 1, []string(nil)).Return(int64(0), nil, nil)
	dbClientMock.On("DeleteNodeByName", node.Name).Return(nil)
	dbClientMock.On("DeleteNodeByName", notFoundName).Return(edgexErr.NewCommonEdgeX(edgexErr.KindEntityDoesNotExist, "node doesn't exist in the database", nil))
	dbClientMock.On("NodeByName", notFoundName).Return(node, edgexErr.NewCommonEdgeX(edgexErr.KindEntityDoesNotExist, "node doesn't exist in the database", nil))
	dbClientMock.On("NodeByName", node.Name).Return(node, nil)
	dbClientMock.On("NodeByName", nodeParent.Name).Return(node, nil)
	dbClientMock.On("NodeTree", nodeParent.Name, 1, 0, 1, []string(nil)).Return(int64(1), []models.Node{node}, nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	controller := NewNodeController(dic)
	require.NotNil(t, controller)

	tests := []struct {
		name               string
		nodeName           string
		expectedStatusCode int
	}{
		{"Valid - delete node by name", node.Name, http.StatusOK},
		{"Invalid - name parameter is empty", noName, http.StatusBadRequest},
		{"Invalid - node not found by name", notFoundName, http.StatusNotFound},
		{"Invalid - node has children", nodeParent.Name, http.StatusConflict},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			reqPath := fmt.Sprintf("%s/%s", common.ApiNodeByNameRoute, testCase.nodeName)
			req, err := http.NewRequest(http.MethodGet, reqPath, http.NoBody)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Name)
			c.SetParamValues(testCase.nodeName)

			err = controller.DeleteNodeByName(c)
			require.NoError(t, err)
			var res commonDTO.BaseResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &res)
			require.NoError(t, err)

			// Assert
			assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
			assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
			assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
			if testCase.expectedStatusCode == http.StatusOK {
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			} else {
				assert.NotEmpty(t, res.Message, "Response message doesn't contain the error message")
			}
		})
	}
}

func TestAllNodeByServiceName(t *testing.T) {
	node := dtos.ToNodeModel(buildTestNodeRequest().Node)
	testServiceA := "testServiceA"
	testServiceB := "testServiceB"
	node1WithServiceA := node
	node1WithServiceA.ServiceName = testServiceA
	node2WithServiceA := node
	node2WithServiceA.ServiceName = testServiceA
	node3WithServiceB := node
	node3WithServiceB.ServiceName = testServiceB

	nodes := []models.Node{node1WithServiceA, node2WithServiceA, node3WithServiceB}
	expectedTotalCountServiceA := int64(2)
	expectedTotalCountServiceB := int64(1)

	dic := mockDic()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("NodeCountByServiceName", testServiceA).Return(expectedTotalCountServiceA, nil)
	dbClientMock.On("NodeCountByServiceName", testServiceB).Return(expectedTotalCountServiceB, nil)
	dbClientMock.On("NodesByServiceName", 0, 5, testServiceA).Return([]models.Node{nodes[0], nodes[1]}, nil)
	dbClientMock.On("NodesByServiceName", 1, 1, testServiceA).Return([]models.Node{nodes[1]}, nil)
	dbClientMock.On("NodesByServiceName", 4, 1, testServiceB).Return([]models.Node{}, edgexErr.NewCommonEdgeX(edgexErr.KindRangeNotSatisfiable, "query objects bounds out of range.", nil))
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})
	controller := NewNodeController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name               string
		offset             string
		limit              string
		serviceName        string
		errorExpected      bool
		expectedCount      int
		expectedTotalCount int64
		expectedStatusCode int
	}{
		{"Valid - get nodes with serviceName", "0", "5", testServiceA, false, 2, expectedTotalCountServiceA, http.StatusOK},
		{"Valid - get nodes with offset and no labels", "1", "1", testServiceA, false, 1, expectedTotalCountServiceA, http.StatusOK},
		{"Invalid - offset out of range", "4", "1", testServiceB, true, 0, 0, http.StatusRequestedRangeNotSatisfiable},
		{"Invalid - get nodes without serviceName", "0", "10", "", true, 0, 0, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiNodeByServiceNameRoute, http.NoBody)
			query := req.URL.Query()
			query.Add(common.Offset, testCase.offset)
			query.Add(common.Limit, testCase.limit)
			req.URL.RawQuery = query.Encode()
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Name)
			c.SetParamValues(testCase.serviceName)

			err = controller.NodesByServiceName(c)
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
				var res responseDTO.MultiNodesResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.expectedCount, len(res.Nodes), "Node count not as expected")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestNodeNameExists(t *testing.T) {
	testName := TestNodeName
	notFoundName := "notFoundName"
	emptyName := ""

	dic := mockDic()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("NodeNameExists", testName).Return(true, nil)
	dbClientMock.On("NodeNameExists", notFoundName).Return(false, nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	controller := NewNodeController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name               string
		nodeName           string
		errorExpected      bool
		expectedStatusCode int
	}{
		{"Valid - check node by name", testName, false, http.StatusOK},
		{"Invalid - name parameter is empty", emptyName, true, http.StatusBadRequest},
		{"Invalid - node not found by name", notFoundName, false, http.StatusNotFound},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			reqPath := fmt.Sprintf("%s/%s", common.ApiNodeNameExistsRoute, testCase.nodeName)
			req, err := http.NewRequest(http.MethodGet, reqPath, http.NoBody)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Name)
			c.SetParamValues(testCase.nodeName)

			err = controller.NodeNameExists(c)
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

func TestPatchNode(t *testing.T) {
	expectedRequestId := ExampleUUID
	dic := mockDic()
	dbClientMock := &dbMock.DBClient{}
	testReq := buildTestUpdateNodeRequest()
	dsModels := models.Node{
		Id:             *testReq.Node.Id,
		Name:           *testReq.Node.Name,
		Description:    *testReq.Node.Description,
		Labels:         testReq.Node.Labels,
		AdminState:     models.AdminState(*testReq.Node.AdminState),
		OperatingState: models.OperatingState(*testReq.Node.OperatingState),
		Location:       testReq.Node.Location,
		ServiceName:    *testReq.Node.ServiceName,
		ProfileName:    *testReq.Node.ProfileName,
		AutoEvents:     dtos.ToAutoEventModels(testReq.Node.AutoEvents),
		Protocols:      dtos.ToProtocolModels(testReq.Node.Protocols),
		Properties:     testProperties,
	}

	valid := testReq
	dbClientMock.On("NodeServiceNameExists", *valid.Node.ServiceName).Return(true, nil)
	dbClientMock.On("NodeById", *valid.Node.Id).Return(dsModels, nil)
	dbClientMock.On("UpdateNode", dsModels).Return(nil)
	dbClientMock.On("ProfileByName", mock.Anything).Return(models.Profile{Name: "test-profile", Resources: []models.Resource{{Name: "TestResource"}}}, nil)

	validWithNoReqID := testReq
	validWithNoReqID.RequestId = ""
	validWithNoId := testReq
	validWithNoId.Node.Id = nil
	dbClientMock.On("NodeByName", *validWithNoId.Node.Name).Return(dsModels, nil)
	validWithNoName := testReq
	validWithNoName.Node.Name = nil

	invalidId := testReq
	invalidUUID := "invalidUUID"
	invalidId.Node.Id = &invalidUUID

	emptyString := ""
	emptyId := testReq
	emptyId.Node.Id = &emptyString
	emptyId.Node.Name = nil
	emptyName := testReq
	emptyName.Node.Id = nil
	emptyName.Node.Name = &emptyString

	emptyProfile := testReq
	emptyProfile.Node.ProfileName = &emptyString
	dm := dsModels
	dm.ProfileName = *emptyProfile.Node.ProfileName
	dbClientMock.On("UpdateNode", dm).Return(nil)

	invalidProtocols := testReq
	invalidProtocols.Node.Protocols = map[string]dtos.ProtocolProperties{"others": {}}

	invalidNoIdAndName := testReq
	invalidNoIdAndName.Node.Id = nil
	invalidNoIdAndName.Node.Name = nil

	invalidNotFoundId := testReq
	invalidNotFoundId.Node.Name = nil
	notFoundId := "12345678-1111-1234-5678-de9dac3fb9bc"
	invalidNotFoundId.Node.Id = &notFoundId
	notFoundIdError := edgexErr.NewCommonEdgeX(edgexErr.KindEntityDoesNotExist, fmt.Sprintf("%s doesn't exist in the database", notFoundId), nil)
	dbClientMock.On("NodeById", *invalidNotFoundId.Node.Id).Return(dsModels, notFoundIdError)

	invalidNotFoundName := testReq
	invalidNotFoundName.Node.Id = nil
	notFoundName := "notFoundName"
	invalidNotFoundName.Node.Name = &notFoundName
	notFoundNameError := edgexErr.NewCommonEdgeX(edgexErr.KindEntityDoesNotExist, fmt.Sprintf("%s doesn't exist in the database", notFoundName), nil)
	dbClientMock.On("NodeByName", *invalidNotFoundName.Node.Name).Return(dsModels, notFoundNameError)

	notFoundServiceName := "notFoundService"
	notFoundService := testReq
	notFoundService.Node.ServiceName = &notFoundServiceName
	dbClientMock.On("NodeServiceNameExists", *notFoundService.Node.ServiceName).Return(false, nil)

	notFoundProfileName := "notFoundProfile"
	notFoundProfile := testReq
	notFoundProfile.Node.ProfileName = &notFoundProfileName
	notFoundProfileNodeModel := dsModels
	notFoundProfileNodeModel.ProfileName = notFoundProfileName
	dbClientMock.On("UpdateNode", notFoundProfileNodeModel).Return(
		edgexErr.NewCommonEdgeX(edgexErr.KindEntityDoesNotExist,
			fmt.Sprintf("node profile '%s' does not exists", notFoundProfileName), nil))

	ownParent := testReq
	ownParent.Node.Parent = ownParent.Node.Name

	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})
	controller := NewNodeController(dic)
	require.NotNil(t, controller)
	tests := []struct {
		name                 string
		request              []requests.UpdateNodeRequest
		expectedStatusCode   int
		expectedResponseCode int
		expectedValidation   bool
		expectedSystemEvent  bool
	}{
		{"Valid", []requests.UpdateNodeRequest{valid}, http.StatusMultiStatus, http.StatusOK, true, true},
		{"Valid - no requestId", []requests.UpdateNodeRequest{validWithNoReqID}, http.StatusMultiStatus, http.StatusOK, true, true},
		{"Valid - no id", []requests.UpdateNodeRequest{validWithNoId}, http.StatusMultiStatus, http.StatusOK, true, true},
		{"Valid - no name", []requests.UpdateNodeRequest{validWithNoName}, http.StatusMultiStatus, http.StatusOK, true, true},
		{"Valid - bypassValidation", []requests.UpdateNodeRequest{valid}, http.StatusMultiStatus, http.StatusOK, false, true},
		{"Invalid - invalid id", []requests.UpdateNodeRequest{invalidId}, http.StatusBadRequest, http.StatusBadRequest, false, false},
		{"Invalid - empty id", []requests.UpdateNodeRequest{emptyId}, http.StatusBadRequest, http.StatusBadRequest, false, false},
		{"Invalid - empty name", []requests.UpdateNodeRequest{emptyName}, http.StatusBadRequest, http.StatusBadRequest, false, false},
		{"Invalid - no id and name", []requests.UpdateNodeRequest{invalidNoIdAndName}, http.StatusBadRequest, http.StatusBadRequest, false, false},
		{"Invalid - not found id", []requests.UpdateNodeRequest{invalidNotFoundId}, http.StatusMultiStatus, http.StatusNotFound, false, false},
		{"Invalid - not found name", []requests.UpdateNodeRequest{invalidNotFoundName}, http.StatusMultiStatus, http.StatusNotFound, false, false},
		{"Invalid - not found profile", []requests.UpdateNodeRequest{notFoundProfile}, http.StatusMultiStatus, http.StatusNotFound, true, false},
		{"Invalid - invalid protocols", []requests.UpdateNodeRequest{invalidProtocols}, http.StatusMultiStatus, http.StatusInternalServerError, true, false},
		{"Invalid - not found node service", []requests.UpdateNodeRequest{notFoundService}, http.StatusMultiStatus, http.StatusBadRequest, false, false},
		{"Invalid - node service unavailable", []requests.UpdateNodeRequest{valid}, http.StatusMultiStatus, http.StatusServiceUnavailable, true, false},
		{"Valid - empty profile", []requests.UpdateNodeRequest{emptyProfile}, http.StatusMultiStatus, http.StatusOK, true, true},
		{"Invalid - own parent", []requests.UpdateNodeRequest{ownParent}, http.StatusMultiStatus, http.StatusBadRequest, false, false}}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			jsonData, err := json.Marshal(testCase.request)
			require.NoError(t, err)

			var responseEnvelope types.MessageEnvelope
			mockMessaging := &messagingMocks.MessageClient{}
			if testCase.expectedValidation {
				switch testCase.expectedResponseCode {
				case http.StatusInternalServerError:
					mockMessaging.On("Request", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
						requestEnvelope, ok := args.Get(0).(types.MessageEnvelope)
						require.True(t, ok)
						responseEnvelope = types.NewMessageEnvelopeWithError(requestEnvelope.RequestID, "validation failed")
					}).Return(&responseEnvelope, nil)
				case http.StatusServiceUnavailable:
					mockMessaging.On("Request", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&responseEnvelope, errors.New("timed out"))
				default:
					mockMessaging.On("Request", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
						requestEnvelope, ok := args.Get(0).(types.MessageEnvelope)
						require.True(t, ok)
						responseEnvelope, err = types.NewMessageEnvelopeForResponse(nil, requestEnvelope.RequestID, requestEnvelope.CorrelationID, common.ContentTypeJSON)
						require.NoError(t, err)
					}).Return(&responseEnvelope, nil)
				}
			}

			var wg sync.WaitGroup
			if testCase.expectedSystemEvent {
				wg.Add(1)
				mockMessaging.On("Publish", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
					wg.Done()
				}).Return(nil)
			}

			dic.Update(di.ServiceConstructorMap{
				bootstrapContainer.MessagingClientName: func(get di.Get) interface{} {
					return mockMessaging
				},
			})

			reader := strings.NewReader(string(jsonData))
			req, err := http.NewRequest(http.MethodPatch, common.ApiNodeRoute, reader)
			require.NoError(t, err)

			if !testCase.expectedValidation {
				query := req.URL.Query()
				query.Add(bypassValidationQueryParam, common.ValueTrue)
				req.URL.RawQuery = query.Encode()
			}

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.PatchNode(c)
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

			wg.Wait()
			mockMessaging.AssertExpectations(t)
		})
	}

}

func TestAllNodes(t *testing.T) {
	node := dtos.ToNodeModel(buildTestNodeRequest().Node)
	nodes := []models.Node{node, node, node}
	expectedNodeTotalCount := int64(len(nodes))

	dic := mockDic()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("NodeCountByLabels", []string(nil)).Return(expectedNodeTotalCount, nil)
	dbClientMock.On("NodeCountByLabels", testNodeLabels).Return(expectedNodeTotalCount, nil)
	dbClientMock.On("AllNodes", 0, 10, []string(nil)).Return(nodes, nil)
	dbClientMock.On("AllNodes", 0, 5, testNodeLabels).Return([]models.Node{nodes[0], nodes[1]}, nil)
	dbClientMock.On("AllNodes", 1, 2, []string(nil)).Return([]models.Node{nodes[1], nodes[2]}, nil)
	dbClientMock.On("AllNodes", 4, 1, testNodeLabels).Return([]models.Node{}, edgexErr.NewCommonEdgeX(edgexErr.KindRangeNotSatisfiable, "query objects bounds out of range.", nil))
	dbClientMock.On("NodeTree", "foo", 4, 0, 10, []string(nil)).Return(expectedNodeTotalCount, nodes, nil)
	dbClientMock.On("NodeTree", "foo", math.MaxInt32, 0, 10, testNodeLabels).Return(expectedNodeTotalCount, nodes, nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})
	controller := NewNodeController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name               string
		offset             string
		limit              string
		labels             string
		descendantsOf      string
		maxLevels          string
		errorExpected      bool
		expectedCount      int
		expectedTotalCount int64
		expectedStatusCode int
	}{
		{"Valid - get nodes without labels", "0", "10", "", "", "", false, 3, expectedNodeTotalCount, http.StatusOK},
		{"Valid - get nodes with labels", "0", "5", strings.Join(testNodeLabels, ","), "", "", false, 2, expectedNodeTotalCount, http.StatusOK},
		{"Valid - get nodes with offset and no labels", "1", "2", "", "", "", false, 2, expectedNodeTotalCount, http.StatusOK},
		{"Invalid - offset out of range", "4", "1", strings.Join(testNodeLabels, ","), "", "", true, 0, expectedNodeTotalCount, http.StatusRequestedRangeNotSatisfiable},
		{"Valid - get tree without labels", "0", "10", "", "foo", "4", false, 3, expectedNodeTotalCount, http.StatusOK},
		{"Valid - get tree with labels", "0", "10", strings.Join(testNodeLabels, ","), "foo", "-1", false, 3, expectedNodeTotalCount, http.StatusOK},
		{"Invalid - maxLevels bad integer", "4", "1", strings.Join(testNodeLabels, ","), "foo", "bar", true, 0, 0, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiAllNodeRoute, http.NoBody)
			query := req.URL.Query()
			query.Add(common.Offset, testCase.offset)
			query.Add(common.Limit, testCase.limit)
			if len(testCase.labels) > 0 {
				query.Add(common.Labels, testCase.labels)
			}
			if testCase.descendantsOf != "" {
				query.Add(common.DescendantsOf, testCase.descendantsOf)
			}
			if testCase.maxLevels != "" {
				query.Add(common.MaxLevels, testCase.maxLevels)
			}
			req.URL.RawQuery = query.Encode()
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.AllNodes(c)
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
				var res responseDTO.MultiNodesResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.expectedCount, len(res.Nodes), "Node count not as expected")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestNodeByName(t *testing.T) {
	node := dtos.ToNodeModel(buildTestNodeRequest().Node)
	emptyName := ""
	notFoundName := "notFoundName"

	dic := mockDic()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("NodeByName", node.Name).Return(node, nil)
	dbClientMock.On("NodeByName", notFoundName).Return(models.Node{}, edgexErr.NewCommonEdgeX(edgexErr.KindEntityDoesNotExist, "node doesn't exist in the database", nil))
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	controller := NewNodeController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name               string
		nodeName           string
		errorExpected      bool
		expectedStatusCode int
	}{
		{"Valid - find node by name", node.Name, false, http.StatusOK},
		{"Invalid - name parameter is empty", emptyName, true, http.StatusBadRequest},
		{"Invalid - node not found by name", notFoundName, true, http.StatusNotFound},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			reqPath := fmt.Sprintf("%s/%s", common.ApiNodeByNameRoute, testCase.nodeName)
			req, err := http.NewRequest(http.MethodGet, reqPath, http.NoBody)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Name)
			c.SetParamValues(testCase.nodeName)

			err = controller.NodeByName(c)
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
				var res responseDTO.NodeResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.nodeName, res.Node.Name, "Name not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestNodeById(t *testing.T) {
	node := dtos.ToNodeModel(buildTestNodeRequest().Node)
	emptyId := ""
	notFoundId := "00000000-0000-0000-0000-000000000000"

	dic := mockDic()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("NodeById", node.Id).Return(node, nil)
	dbClientMock.On("NodeById", notFoundId).Return(models.Node{}, edgexErr.NewCommonEdgeX(edgexErr.KindEntityDoesNotExist, "node doesn't exist in the database", nil))
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	controller := NewNodeController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name               string
		nodeId             string
		errorExpected      bool
		expectedStatusCode int
	}{
		{"Valid - find node by id", node.Id, false, http.StatusOK},
		{"Invalid - id parameter is empty", emptyId, true, http.StatusBadRequest},
		{"Invalid - node not found by id", notFoundId, true, http.StatusNotFound},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			reqPath := fmt.Sprintf("%s/%s/%s", common.ApiNodeRoute, common.Id, testCase.nodeId)
			req, err := http.NewRequest(http.MethodGet, reqPath, http.NoBody)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Id)
			c.SetParamValues(testCase.nodeId)

			err = controller.NodeById(c)
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
				var res responseDTO.NodeResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.nodeId, res.Node.Id, "Id not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestNodesByProfileName(t *testing.T) {
	node := dtos.ToNodeModel(buildTestNodeRequest().Node)
	testProfileA := "testProfileA"
	testProfileB := "testServiceB"
	node1WithProfileA := node
	node1WithProfileA.ProfileName = testProfileA
	node2WithProfileA := node
	node2WithProfileA.ProfileName = testProfileA
	node3WithProfileB := node
	node3WithProfileB.ProfileName = testProfileB

	nodes := []models.Node{node1WithProfileA, node2WithProfileA, node3WithProfileB}
	expectedTotalCountProfileA := int64(2)
	expectedTotalCountProfileB := int64(1)

	dic := mockDic()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("NodeCountByProfileName", testProfileA).Return(expectedTotalCountProfileA, nil)
	dbClientMock.On("NodeCountByProfileName", testProfileB).Return(expectedTotalCountProfileB, nil)
	dbClientMock.On("NodesByProfileName", 0, 5, testProfileA).Return([]models.Node{nodes[0], nodes[1]}, nil)
	dbClientMock.On("NodesByProfileName", 1, 1, testProfileA).Return([]models.Node{nodes[1]}, nil)
	dbClientMock.On("NodesByProfileName", 4, 1, testProfileB).Return([]models.Node{}, edgexErr.NewCommonEdgeX(edgexErr.KindRangeNotSatisfiable, "query objects bounds out of range.", nil))
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})
	controller := NewNodeController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name               string
		offset             string
		limit              string
		profileName        string
		errorExpected      bool
		expectedCount      int
		expectedTotalCount int64
		expectedStatusCode int
	}{
		{"Valid - get nodes with profileName", "0", "5", testProfileA, false, 2, expectedTotalCountProfileA, http.StatusOK},
		{"Valid - get nodes with offset and limit", "1", "1", testProfileA, false, 1, expectedTotalCountProfileA, http.StatusOK},
		{"Invalid - offset out of range", "4", "1", testProfileB, true, 0, 0, http.StatusRequestedRangeNotSatisfiable},
		{"Invalid - get nodes without profileName", "0", "10", "", true, 0, 0, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiNodeByProfileNameRoute, http.NoBody)
			query := req.URL.Query()
			query.Add(common.Offset, testCase.offset)
			query.Add(common.Limit, testCase.limit)
			req.URL.RawQuery = query.Encode()
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Name)
			c.SetParamValues(testCase.profileName)
			err = controller.NodesByProfileName(c)
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
				var res responseDTO.MultiNodesResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.expectedCount, len(res.Nodes), "Node count not as expected")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}
