//
// Copyright (C) 2021-2023 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clint456/edgex-go/internal/core/command/application"
	"github.com/clint456/edgex-go/internal/core/command/config"
	commandContainer "github.com/clint456/edgex-go/internal/core/command/container"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	bootstrapConfig "github.com/edgexfoundry/go-mod-bootstrap/v4/config"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/interfaces/mocks"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	commonDTO "github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/common"
	responseDTO "github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/responses"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	mockHost = "127.0.0.1"
	mockPort = 66666

	testProfileName     = "testProfile"
	testResourceName    = "testResource"
	testNodeName        = "testNode"
	testSourceName      = "testSourceName"
	testNodeServiceName = "testNodeService"
	testCommandName     = "testCommand"
	testBaseAddress     = "http://localhost:49990"
	testQueryStrings    = "a=1&b=2&ds-pushevent=false"
)

// NewMockDIC function returns a mock bootstrap di Container
func NewMockDIC() *di.Container {
	return di.NewContainer(di.ServiceConstructorMap{
		commandContainer.ConfigurationName: func(get di.Get) interface{} {
			return &config.ConfigurationStruct{
				Service: bootstrapConfig.ServiceInfo{
					Host:           mockHost,
					Port:           mockPort,
					MaxResultCount: 20,
				},
			}
		},
		bootstrapContainer.LoggingClientInterfaceName: func(get di.Get) interface{} {
			return logger.NewMockClient()
		},
	})
}

func buildTestSettings() map[string]interface{} {
	var settings = make(map[string]interface{})
	settings["AHU-TargetTemperature"] = "28.5"
	settings["AHU-TargetBand"] = "4.0"
	settings["AHU-TargetHumidity"] = map[string]interface{}{
		"Accuracy": "0.2-0.3% RH",
		"Value":    float64(59),
	}
	return settings
}

func buildNodeCoreCommands(t *testing.T, node dtos.Node, profile dtos.Profile) dtos.NodeCoreCommand {
	dcMock := &mocks.NodeClient{}
	dpcMock := &mocks.ProfileClient{}
	dcMock.On("NodeByName", context.Background(), node.Name).
		Return(responseDTO.NewNodeResponse("", "", http.StatusOK, node), nil)
	dpcMock.On("ProfileByName", context.Background(), profile.Name).
		Return(responseDTO.NewProfileResponse("", "", http.StatusOK, profile), nil)
	dic := NewMockDIC()
	dic.Update(di.ServiceConstructorMap{
		bootstrapContainer.NodeClientName: func(get di.Get) interface{} {
			return dcMock
		},
		bootstrapContainer.ProfileClientName: func(get di.Get) interface{} {
			return dpcMock
		},
	})
	commands, err := application.CommandsByNodeName(node.Name, dic)
	require.NoError(t, err)
	return commands
}

func buildNodeResponse() responseDTO.NodeResponse {
	node := dtos.Node{
		Name:        testNodeName,
		ProfileName: testProfileName,
		ServiceName: testNodeServiceName,
	}
	nodeResponse := responseDTO.NodeResponse{
		Node: node,
	}
	return nodeResponse
}

func buildMultiNodesResponse() responseDTO.MultiNodesResponse {
	nodes := []dtos.Node{
		{Name: testNodeName + "1", ProfileName: testProfileName, ServiceName: testNodeServiceName},
		{Name: testNodeName + "2", ProfileName: testProfileName, ServiceName: testNodeServiceName},
	}
	return responseDTO.MultiNodesResponse{
		BaseWithTotalCountResponse: commonDTO.NewBaseWithTotalCountResponse("", "", http.StatusOK, int64(2)),
		Nodes:                      nodes,
	}
}

func buildCommands() []dtos.Command {
	c1 := dtos.Command{
		Name:      "command1",
		ReadWrite: common.ReadWrite_R,
	}
	c2 := dtos.Command{
		Name:      "command2",
		ReadWrite: common.ReadWrite_R,
	}
	var commands []dtos.Command
	commands = append(commands, c1, c2)
	return commands
}

func buildProfileResponse() responseDTO.ProfileResponse {
	commands := buildCommands()
	profile := dtos.Profile{
		ProfileBasicInfo: dtos.ProfileBasicInfo{Name: testProfileName},
		Commands:         commands,
	}
	nodeResponse := responseDTO.ProfileResponse{
		Profile: profile,
	}
	return nodeResponse
}

func buildNodeServiceResponse() responseDTO.NodeServiceResponse {
	service := dtos.NodeService{
		Name:        testNodeServiceName,
		BaseAddress: testBaseAddress,
	}
	return responseDTO.NodeServiceResponse{
		Service: service,
	}
}

func buildEvent() dtos.Event {
	event := dtos.NewEvent(testProfileName, testNodeName, testSourceName)
	_ = event.AddSimpleReading(testResourceName, common.ValueTypeUint16, uint16(45))
	id, _ := uuid.NewUUID()
	event.Id = id.String()
	event.Readings[0].Id = id.String()
	return event
}

func buildEventResponse() responseDTO.EventResponse {
	return responseDTO.NewEventResponse("", "", http.StatusOK, buildEvent())
}

func TestAllCommands(t *testing.T) {
	expectedMultiNodesResponse := buildMultiNodesResponse()
	expectedProfileResponse := buildProfileResponse()
	nodeCoreCommand1 := buildNodeCoreCommands(t, expectedMultiNodesResponse.Nodes[0], expectedProfileResponse.Profile)
	nodeCoreCommand2 := buildNodeCoreCommands(t, expectedMultiNodesResponse.Nodes[1], expectedProfileResponse.Profile)
	expectedMultiNodeCoreCommandsResponse := responseDTO.MultiNodeCoreCommandsResponse{
		NodeCoreCommands: []dtos.NodeCoreCommand{nodeCoreCommand1, nodeCoreCommand2},
	}

	dcMock := &mocks.NodeClient{}
	dcMock.On("AllNodes", context.Background(), []string(nil), 0, 20).Return(expectedMultiNodesResponse, nil)
	dcMock.On("AllNodes", context.Background(), []string(nil), 0, 2).Return(expectedMultiNodesResponse, nil)
	dcMock.On("AllNodes", context.Background(), []string(nil), 3, 10).Return(responseDTO.MultiNodesResponse{}, errors.NewCommonEdgeX(errors.KindRangeNotSatisfiable, "bounds out of range", nil))

	dpcMock := &mocks.ProfileClient{}
	dpcMock.On("ProfileByName", context.Background(), testProfileName).Return(expectedProfileResponse, nil)

	dic := NewMockDIC()
	dic.Update(di.ServiceConstructorMap{
		bootstrapContainer.NodeClientName: func(get di.Get) interface{} {
			return dcMock
		},
		bootstrapContainer.ProfileClientName: func(get di.Get) interface{} {
			return dpcMock
		},
	})
	cc := NewCommandController(dic)
	assert.NotNil(t, cc)

	tests := []struct {
		name               string
		offset             string
		limit              string
		errorExpected      bool
		expectedCount      int
		expectedTotalCount int64
		expectedStatusCode int
	}{
		{"Valid - get commands without offset and limit", "", "", false, len(expectedMultiNodeCoreCommandsResponse.NodeCoreCommands), expectedMultiNodesResponse.TotalCount, http.StatusOK},
		{"Valid - get commands with offset and limit", "0", "2", false, len(expectedMultiNodeCoreCommandsResponse.NodeCoreCommands), expectedMultiNodesResponse.TotalCount, http.StatusOK},
		{"Invalid - bounds out of range", "3", "10", true, 0, 0, http.StatusRequestedRangeNotSatisfiable},
		{"Invalid - invalid offset format", "aaa", "10", true, 0, 0, http.StatusBadRequest},
		{"Invalid - invalid limit format", "0", "aaa", true, 0, 0, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiAllNodeRoute, http.NoBody)
			query := req.URL.Query()
			if testCase.offset != "" {
				query.Add(common.Offset, testCase.offset)
			}
			if testCase.limit != "" {
				query.Add(common.Limit, testCase.limit)
			}
			req.URL.RawQuery = query.Encode()
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			handler := echo.HandlerFunc(cc.AllCommands)
			c := e.NewContext(req, recorder)
			err = handler(c)
			assert.NoError(t, err)

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
				var res responseDTO.MultiNodeCoreCommandsResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.expectedCount, len(res.NodeCoreCommands), "Node count not as expected")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestCommandsByNodeName(t *testing.T) {
	var nonExistNodeName = "nonExistNode"

	expectedNodeResponse := buildNodeResponse()
	expectedProfileResponse := buildProfileResponse()
	expectedNodeCoreCommand := buildNodeCoreCommands(t, expectedNodeResponse.Node, expectedProfileResponse.Profile)

	dcMock := &mocks.NodeClient{}
	dcMock.On("NodeByName", context.Background(), testNodeName).Return(expectedNodeResponse, nil)
	dcMock.On("NodeByName", context.Background(), nonExistNodeName).Return(responseDTO.NodeResponse{}, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "fail to query node by name", nil))

	dpcMock := &mocks.ProfileClient{}
	dpcMock.On("ProfileByName", context.Background(), testProfileName).Return(expectedProfileResponse, nil)

	dic := NewMockDIC()
	dic.Update(di.ServiceConstructorMap{
		bootstrapContainer.NodeClientName: func(get di.Get) interface{} {
			return dcMock
		},
		bootstrapContainer.ProfileClientName: func(get di.Get) interface{} {
			return dpcMock
		},
	})
	cc := NewCommandController(dic)
	assert.NotNil(t, cc)

	tests := []struct {
		name               string
		nodeName           string
		errorExpected      bool
		expectedCount      int
		expectedStatusCode int
	}{
		{"Valid - get coreCommands with nodeName", testNodeName, false, len(expectedNodeCoreCommand.CoreCommands), http.StatusOK},
		{"Invalid - get coreCommands with empty nodeName", "", true, 0, http.StatusBadRequest},
		{"Invalid - get coreCommands with non exist nodeName", nonExistNodeName, true, 0, http.StatusNotFound},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest("GET", "/api/v3/node/name/:name", nil)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)

			// Set url path params
			c.SetParamNames(common.Name)
			c.SetParamValues(testCase.nodeName)
			handler := echo.HandlerFunc(cc.CommandsByNodeName)
			err := handler(c)
			assert.NoError(t, err)

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
				var res responseDTO.NodeCoreCommandResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.expectedCount, len(res.NodeCoreCommand.CoreCommands), "Node count not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestIssueGetCommand(t *testing.T) {
	var nonExistName = "nonExist"

	expectedEventResponse := buildEventResponse()
	expectedNodeResponse := buildNodeResponse()
	expectedNodeServiceResponse := buildNodeServiceResponse()

	dcMock := &mocks.NodeClient{}
	dcMock.On("NodeByName", context.Background(), testNodeName).Return(expectedNodeResponse, nil)
	dcMock.On("NodeByName", context.Background(), nonExistName).Return(responseDTO.NodeResponse{}, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "fail to query node by name", nil))

	dscMock := &mocks.NodeServiceClient{}
	dscMock.On("NodeServiceByName", context.Background(), testNodeServiceName).Return(expectedNodeServiceResponse, nil)

	dsccMock := &mocks.NodeServiceCommandClient{}
	dsccMock.On("GetCommand", context.Background(), testBaseAddress, testNodeName, testCommandName, testQueryStrings).Return(&expectedEventResponse, nil)
	dsccMock.On("GetCommand", context.Background(), testBaseAddress, testNodeName, testCommandName, "").Return(&expectedEventResponse, nil)
	dsccMock.On("GetCommand", context.Background(), testBaseAddress, testNodeName, nonExistName, testQueryStrings).Return(&responseDTO.EventResponse{}, errors.NewCommonEdgeX(errors.KindContractInvalid, "fail to query node service by name", nil))

	dic := NewMockDIC()
	dic.Update(di.ServiceConstructorMap{
		bootstrapContainer.NodeClientName: func(get di.Get) interface{} {
			return dcMock
		},
		bootstrapContainer.NodeServiceClientName: func(get di.Get) interface{} {
			return dscMock
		},
		bootstrapContainer.NodeServiceCommandClientName: func(get di.Get) interface{} {
			return dsccMock
		},
	})
	cc := NewCommandController(dic)
	assert.NotNil(t, cc)

	tests := []struct {
		name               string
		nodeName           string
		commandName        string
		queryStrings       string
		errorExpected      bool
		expectedStatusCode int
	}{
		{"Valid - execute read command with valid nodeName, commandName, and query strings", testNodeName, testCommandName, testQueryStrings, false, http.StatusOK},
		{"Valid - empty query strings", testNodeName, testCommandName, "", false, http.StatusOK},
		{"Invalid - execute read command with invalid nodeName", nonExistName, testCommandName, testQueryStrings, true, http.StatusNotFound},
		{"Invalid - execute read command with invalid commandName", testNodeName, nonExistName, testQueryStrings, true, http.StatusBadRequest},
		{"Invalid - empty node name", "", nonExistName, testQueryStrings, true, http.StatusBadRequest},
		{"Invalid - empty command name", testNodeName, "", testQueryStrings, true, http.StatusBadRequest},
		{"Invalid - invalid ds-pushevent paramter", testNodeName, "", "ds-pushevent=123", true, http.StatusBadRequest},
		{"Invalid - invalid ds-returnevent paramter", testNodeName, "", "ds-returnevent=123", true, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v3/node/name/:name/:command", http.NoBody)
			req.URL.RawQuery = testCase.queryStrings

			// Act
			recorder := httptest.NewRecorder()
			handler := echo.HandlerFunc(cc.IssueGetCommandByName)
			c := e.NewContext(req, recorder)

			// Set url path params
			c.SetParamNames(common.Name, common.Command)
			c.SetParamValues(testCase.nodeName, testCase.commandName)
			err := handler(c)
			assert.NoError(t, err)

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
				var res responseDTO.EventResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestIssueSetCommand(t *testing.T) {
	var nonExistName = "nonExist"

	expectedBaseResponse := commonDTO.NewBaseResponse("", "", http.StatusOK)
	expectedNodeResponse := buildNodeResponse()
	expectedNodeServiceResponse := buildNodeServiceResponse()

	dcMock := &mocks.NodeClient{}
	dcMock.On("NodeByName", context.Background(), testNodeName).Return(expectedNodeResponse, nil)
	dcMock.On("NodeByName", context.Background(), nonExistName).Return(responseDTO.NodeResponse{}, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "fail to query node by name", nil))

	dscMock := &mocks.NodeServiceClient{}
	dscMock.On("NodeServiceByName", context.Background(), testNodeServiceName).Return(expectedNodeServiceResponse, nil)

	testSettings := buildTestSettings()
	testSettingsJsonStr, _ := json.Marshal(testSettings)
	dsccMock := &mocks.NodeServiceCommandClient{}
	dsccMock.On("SetCommandWithObject", context.Background(), testBaseAddress, testNodeName, testCommandName, testQueryStrings, testSettings).Return(expectedBaseResponse, nil)
	dsccMock.On("SetCommandWithObject", context.Background(), testBaseAddress, testNodeName, testCommandName, "", testSettings).Return(expectedBaseResponse, nil)
	dsccMock.On("SetCommandWithObject", context.Background(), testBaseAddress, testNodeName, testCommandName, testQueryStrings, "").Return(commonDTO.BaseResponse{}, errors.NewCommonEdgeX(errors.KindServerError, "no request body provided for PUT command", nil))
	dsccMock.On("SetCommandWithObject", context.Background(), testBaseAddress, testNodeName, nonExistName, testQueryStrings, testSettings).Return(commonDTO.BaseResponse{}, errors.NewCommonEdgeX(errors.KindContractInvalid, "no corresponding PUT command", nil))

	dic := NewMockDIC()
	dic.Update(di.ServiceConstructorMap{
		bootstrapContainer.NodeClientName: func(get di.Get) interface{} {
			return dcMock
		},
		bootstrapContainer.NodeServiceClientName: func(get di.Get) interface{} {
			return dscMock
		},
		bootstrapContainer.NodeServiceCommandClientName: func(get di.Get) interface{} {
			return dsccMock
		},
	})
	cc := NewCommandController(dic)
	assert.NotNil(t, cc)

	tests := []struct {
		name               string
		nodeName           string
		commandName        string
		queryStrings       string
		settings           []byte
		errorExpected      bool
		expectedStatusCode int
	}{
		{"Valid - execute set command with valid nodeName, commandName, query strings, and settings", testNodeName, testCommandName, testQueryStrings, testSettingsJsonStr, false, http.StatusOK},
		{"Valid - empty query strings", testNodeName, testCommandName, "", testSettingsJsonStr, false, http.StatusOK},
		{"Invalid - execute set command with invalid nodeName", nonExistName, testCommandName, testQueryStrings, testSettingsJsonStr, true, http.StatusNotFound},
		{"Invalid - execute set command with invalid commandName", testNodeName, nonExistName, testQueryStrings, testSettingsJsonStr, true, http.StatusBadRequest},
		{"Invalid - empty node name", "", testCommandName, testQueryStrings, testSettingsJsonStr, true, http.StatusBadRequest},
		{"Invalid - empty command name", testNodeName, "", testQueryStrings, testSettingsJsonStr, true, http.StatusBadRequest},
		{"Invalid - empty settings", testNodeName, testCommandName, testQueryStrings, []byte{}, true, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPut, "/api/v3/node/name/:name/:command", bytes.NewBuffer(testCase.settings))
			req.URL.RawQuery = testCase.queryStrings

			// Act
			recorder := httptest.NewRecorder()
			handler := echo.HandlerFunc(cc.IssueSetCommandByName)
			c := e.NewContext(req, recorder)

			// Set url path params
			c.SetParamNames(common.Name, common.Command)
			c.SetParamValues(testCase.nodeName, testCase.commandName)

			err := handler(c)
			assert.NoError(t, err)

			// Assert
			var res commonDTO.BaseResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &res)
			require.NoError(t, err)
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
