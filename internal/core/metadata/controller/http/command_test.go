//
// Copyright (C) 2022-2023 IOTech Ltd
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
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var testCommandRequest = requests.AddCommandRequest{
	BaseRequest: commonDTO.BaseRequest{
		RequestId:   ExampleUUID,
		Versionable: commonDTO.NewVersionable(),
	},
	ProfileName: TestProfileName,
	Command: dtos.Command{
		Name:      "TestCommandNewName",
		IsHidden:  false,
		ReadWrite: common.ReadWrite_RW,
		ResourceOperations: []dtos.ResourceOperation{{
			Resource: TestResourceName,
			Mappings: testMappings,
		}},
	},
}

func buildTestUpdateCommandRequest() requests.UpdateCommandRequest {
	testCommandName := TestCommandName
	testIsHidden := false
	var testUpdateCommandReq = requests.UpdateCommandRequest{
		BaseRequest: commonDTO.BaseRequest{
			RequestId:   ExampleUUID,
			Versionable: commonDTO.NewVersionable(),
		},
		ProfileName: TestProfileName,
		Command: dtos.UpdateCommand{
			Name:     &testCommandName,
			IsHidden: &testIsHidden,
		},
	}
	return testUpdateCommandReq
}

func TestAddProfileCommand(t *testing.T) {
	profile := dtos.ToProfileModel(buildTestProfileRequest().Profile)
	expectedRequestId := ExampleUUID

	valid := testCommandRequest
	noRequestId := testCommandRequest
	noRequestId.RequestId = ""
	duplicateName := testCommandRequest
	duplicateName.Command.Name = TestCommandName
	notFoundResource := testCommandRequest
	notFoundResource.Command.ResourceOperations = []dtos.ResourceOperation{{
		Resource: "notFoundName",
		Mappings: testMappings,
	}}

	dic := mockDic()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("ProfileByName", valid.ProfileName).Return(profile, nil)
	dbClientMock.On("UpdateProfile", mock.Anything).Return(nil)
	dbClientMock.On("NodesByProfileName", 0, -1, TestProfileName).Return([]models.Node{}, nil)
	dbClientMock.On("NodeCountByProfileName", TestProfileName).Return(int64(1), nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})
	controller := NewCommandController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name               string
		Request            []requests.AddCommandRequest
		expectedStatusCode int
	}{
		{"Valid - AddCommandRequest", []requests.AddCommandRequest{valid}, http.StatusCreated},
		{"Valid - No requestId", []requests.AddCommandRequest{noRequestId}, http.StatusCreated},
		{"invalid - Duplicate command name", []requests.AddCommandRequest{duplicateName}, http.StatusBadRequest},
		{"invalid - Not Exist resource", []requests.AddCommandRequest{notFoundResource}, http.StatusBadRequest},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			jsonData, err := json.Marshal(testCase.Request)
			require.NoError(t, err)

			reader := strings.NewReader(string(jsonData))
			req, err := http.NewRequest(http.MethodPost, common.ApiProfileCommandRoute, reader)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.AddProfileCommand(c)
			require.NoError(t, err)

			var res []commonDTO.BaseResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &res)
			require.NoError(t, err)

			assert.Equal(t, http.StatusMultiStatus, recorder.Result().StatusCode, "HTTP status code not as expected")
			assert.Equal(t, common.ApiVersion, res[0].ApiVersion, "API Version not as expected")
			if res[0].RequestId != "" {
				assert.Equal(t, expectedRequestId, res[0].RequestId, "RequestID not as expected")
			}
			assert.Equal(t, testCase.expectedStatusCode, res[0].StatusCode, "BaseResponse status code not as expected")
		})
	}
}

func TestAddProfileCommand_BadRequest(t *testing.T) {
	noCommandName := testCommandRequest
	noCommandName.Command.Name = ""
	invalidReadWrite := testCommandRequest
	invalidReadWrite.Command.ReadWrite = "invalid"
	noResourceOperations := testCommandRequest
	noResourceOperations.Command.ResourceOperations = []dtos.ResourceOperation{{}}

	dic := mockDic()
	controller := NewCommandController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name    string
		Request []requests.AddCommandRequest
	}{
		{"invalid - No command name", []requests.AddCommandRequest{noCommandName}},
		{"invalid - Invalid ReadWrite", []requests.AddCommandRequest{invalidReadWrite}},
		{"invalid - No ResourceOperations", []requests.AddCommandRequest{noResourceOperations}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			jsonData, err := json.Marshal(testCase.Request)
			require.NoError(t, err)

			reader := strings.NewReader(string(jsonData))
			req, err := http.NewRequest(http.MethodPost, common.ApiProfileCommandRoute, reader)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.AddProfileCommand(c)
			require.NoError(t, err)

			var res commonDTO.BaseResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &res)
			require.NoError(t, err)

			assert.Equal(t, http.StatusBadRequest, recorder.Result().StatusCode, "HTTP status code not as expected")
			assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
			assert.Equal(t, http.StatusBadRequest, res.StatusCode, "BaseResponse status code not as expected")
			assert.NotEmpty(t, recorder.Body.String(), "Message is empty")
		})
	}
}

func TestPatchProfileCommand(t *testing.T) {
	profile := dtos.ToProfileModel(buildTestProfileRequest().Profile)
	testReq := buildTestUpdateCommandRequest()
	expectedRequestId := ExampleUUID
	emptyString := ""
	notFound := "notFoundName"

	valid := testReq
	validWithNoReqID := testReq
	validWithNoReqID.RequestId = emptyString
	notFoundCommand := testReq
	notFoundCommand.Command.Name = &notFound
	notFoundProfile := testReq
	notFoundProfile.ProfileName = notFound
	notFoundDBError := errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("node profile %s does not exists", notFoundProfile.ProfileName), nil)

	noProfileName := testReq
	noProfileName.ProfileName = emptyString

	dic := mockDic()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("ProfileByName", valid.ProfileName).Return(profile, nil)
	dbClientMock.On("UpdateProfile", mock.Anything).Return(nil)
	dbClientMock.On("ProfileByName", notFound).Return(profile, notFoundDBError)
	dbClientMock.On("NodesByProfileName", 0, -1, TestProfileName).Return([]models.Node{}, nil)
	dbClientMock.On("NodeCountByProfileName", TestProfileName).Return(int64(1), nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})
	controller := NewCommandController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name                 string
		Request              []requests.UpdateCommandRequest
		expectedStatusCode   int
		expectedResponseCode int
	}{
		{"Valid - PatchProfileCommand", []requests.UpdateCommandRequest{valid}, http.StatusMultiStatus, http.StatusOK},
		{"Valid - PatchProfileCommand no requestId", []requests.UpdateCommandRequest{validWithNoReqID}, http.StatusMultiStatus, http.StatusOK},
		{"Invalid - Not found node command", []requests.UpdateCommandRequest{notFoundCommand}, http.StatusMultiStatus, http.StatusNotFound},
		{"Invalid - Not found node profile", []requests.UpdateCommandRequest{notFoundProfile}, http.StatusMultiStatus, http.StatusNotFound},
		{"Invalid - No profile name", []requests.UpdateCommandRequest{noProfileName}, http.StatusBadRequest, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			jsonData, err := json.Marshal(testCase.Request)
			require.NoError(t, err)

			reader := strings.NewReader(string(jsonData))

			req, err := http.NewRequest(http.MethodPatch, common.ApiProfileCommandRoute, reader)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.PatchProfileCommand(c)
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

func TestDeleteCommandByName(t *testing.T) {
	dpModel := dtos.ToProfileModel(buildTestProfileRequest().Profile)
	notFoundName := "notFoundName"
	nodeExists := "nodeExists"

	dic := mockDic()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("NodesByProfileName", 0, mock.Anything, TestProfileName).Return([]models.Node{}, nil)
	dbClientMock.On("NodeCountByProfileName", TestProfileName).Return(int64(1), nil)
	dbClientMock.On("ProfileByName", TestProfileName).Return(dpModel, nil)
	dbClientMock.On("UpdateProfile", mock.Anything).Return(nil)

	dbClientMock.On("NodesByProfileName", 0, 1, nodeExists).Return([]models.Node{models.Node{}}, nil)

	dbClientMock.On("NodesByProfileName", 0, 1, notFoundName).Return([]models.Node{}, nil)
	dbClientMock.On("ProfileByName", notFoundName).Return(models.Profile{}, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "not found", nil))
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	controller := NewCommandController(dic)
	require.NotNil(t, controller)

	tests := []struct {
		name               string
		profileName        string
		commandName        string
		expectedStatusCode int
	}{
		{"valid", TestProfileName, TestCommandName, http.StatusOK},
		{"invalid - empty profile name", "", TestCommandName, http.StatusBadRequest},
		{"invalid - empty command name", TestProfileName, "", http.StatusBadRequest},
		{"invalid - profile is in use by other node", nodeExists, TestCommandName, http.StatusConflict},
		{"invalid - profile not found", notFoundName, TestCommandName, http.StatusNotFound},
		{"invalid - command not found in profile", TestProfileName, notFoundName, http.StatusNotFound},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodDelete, common.ApiProfileCommandByNameRoute, http.NoBody)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Name, common.CommandName)
			c.SetParamValues(testCase.profileName, testCase.commandName)

			err = controller.DeleteCommandByName(c)
			require.NoError(t, err)

			var res commonDTO.BaseResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &res)
			require.NoError(t, err)

			// Assert
			assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
			assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
			assert.Equal(t, testCase.expectedStatusCode, res.StatusCode, "BaseResponse status code not as expected")
			if testCase.expectedStatusCode != http.StatusOK {
				assert.NotEmpty(t, recorder.Body.String(), "Message is empty")
			}
		})
	}

}

func TestDeleteCommandByName_StrictProfileChanges(t *testing.T) {
	dic := mockDic()
	configuration := container.ConfigurationFrom(dic.Get)
	configuration.Writable.ProfileChange.StrictProfileChanges = true
	dic.Update(di.ServiceConstructorMap{
		container.ConfigurationName: func(get di.Get) interface{} {
			return configuration
		},
	})

	controller := NewCommandController(dic)
	require.NotNil(t, controller)

	e := echo.New()
	req, err := http.NewRequest(http.MethodDelete, common.ApiProfileCommandByNameRoute, http.NoBody)
	require.NoError(t, err)

	// Act
	recorder := httptest.NewRecorder()
	c := e.NewContext(req, recorder)
	c.SetParamNames(common.Name, common.CommandName)
	c.SetParamValues(TestProfileName, TestResourceName)

	err = controller.DeleteCommandByName(c)
	require.NoError(t, err)

	var res commonDTO.BaseResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &res)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, http.StatusLocked, recorder.Result().StatusCode, "HTTP status code not as expected")
	assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
	assert.Equal(t, http.StatusLocked, res.StatusCode, "BaseResponse status code not as expected")
}
