//
// Copyright (C) 2021-2023 IOTech Ltd
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
	"github.com/clint456/edgex-go/internal/core/metadata/infrastructure/interfaces/mocks"

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

var testResourceRequest = requests.AddResourceRequest{
	BaseRequest: commonDTO.BaseRequest{
		RequestId:   ExampleUUID,
		Versionable: commonDTO.NewVersionable(),
	},
	ProfileName: TestProfileName,
	Resource: dtos.Resource{
		Name:        "TestResourceNewName",
		Description: TestDescription,
		Attributes:  testAttributes,
		Properties: dtos.ResourceProperties{
			ValueType: common.ValueTypeInt16,
			ReadWrite: common.ReadWrite_RW,
		},
		Tags: testTags,
	},
}

func buildTestUpdateResourceRequest() requests.UpdateResourceRequest {
	testResourceName := TestResourceName
	testDescription := TestDescription
	testIsHidden := false
	var testUpdateResourceReq = requests.UpdateResourceRequest{
		BaseRequest: commonDTO.BaseRequest{
			RequestId:   ExampleUUID,
			Versionable: commonDTO.NewVersionable(),
		},
		ProfileName: TestProfileName,
		Resource: dtos.UpdateResource{
			Name:        &testResourceName,
			Description: &testDescription,
			IsHidden:    &testIsHidden,
		},
	}
	return testUpdateResourceReq
}

func TestResourceByProfileNameAndResourceName(t *testing.T) {
	profile := dtos.ToProfileModel(buildTestProfileRequest().Profile)
	emptyName := ""
	profileNotFoundName := "profileNotFoundName"
	resourceNotFoundName := "resourceNotFoundName"

	dic := mockDic()
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("ProfileByName", profile.Name).Return(profile, nil)
	dbClientMock.On("ProfileByName", profileNotFoundName).Return(models.Profile{}, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "node profile doesn't exist in the database", nil))
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	controller := NewResourceController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name               string
		profileName        string
		resourceName       string
		errorExpected      bool
		expectedStatusCode int
	}{
		{"Valid - find node resource by profileName and resourceName", profile.Name, TestResourceName, false, http.StatusOK},
		{"Invalid - profile name is empty", emptyName, TestResourceName, true, http.StatusBadRequest},
		{"Invalid - profile name is empty", profile.Name, emptyName, true, http.StatusBadRequest},
		{"Invalid - node profile not", profileNotFoundName, TestResourceName, true, http.StatusNotFound},
		{"Invalid - resource not found", profile.Name, resourceNotFoundName, true, http.StatusNotFound},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiResourceByProfileAndResourceRoute, http.NoBody)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.ProfileName, common.ResourceName)
			c.SetParamValues(testCase.profileName, testCase.resourceName)
			err = controller.ResourceByProfileNameAndResourceName(c)
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
				var res responseDTO.ResourceResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.resourceName, res.Resource.Name, "Resource name not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestAddProfileResource(t *testing.T) {
	profile := dtos.ToProfileModel(buildTestProfileRequest().Profile)
	expectedRequestId := ExampleUUID

	valid := testResourceRequest
	noRequestId := testResourceRequest
	noRequestId.RequestId = ""
	duplicateName := testResourceRequest
	duplicateName.Resource.Name = TestResourceName
	noProfileName := testResourceRequest
	noProfileName.ProfileName = ""
	noResourcePropertyType := testResourceRequest
	noResourcePropertyType.Resource.Properties.ValueType = ""
	notFoundProfileName := testResourceRequest
	notFoundProfileName.ProfileName = "notFoundName"
	notFoundDBError := errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("node profile %s does not exists", notFoundProfileName.ProfileName), nil)

	dic := mockDic()
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("ProfileByName", valid.ProfileName).Return(profile, nil)
	dbClientMock.On("ProfileByName", notFoundProfileName.ProfileName).Return(profile, notFoundDBError)
	dbClientMock.On("UpdateProfile", mock.Anything).Return(nil)
	dbClientMock.On("NodesByProfileName", 0, -1, TestProfileName).Return([]models.Node{}, nil)
	dbClientMock.On("NodeCountByProfileName", TestProfileName).Return(int64(1), nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})
	controller := NewResourceController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name               string
		Request            []requests.AddResourceRequest
		expectedStatusCode int
	}{
		{"Valid - AddProfileResourceRequest", []requests.AddResourceRequest{valid}, http.StatusCreated},
		{"Valid - No requestId", []requests.AddResourceRequest{noRequestId}, http.StatusCreated},
		{"invalid - duplicate name", []requests.AddResourceRequest{duplicateName}, http.StatusBadRequest},
		{"invalid - Not Exist node profile", []requests.AddResourceRequest{notFoundProfileName}, http.StatusNotFound},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			jsonData, err := json.Marshal(testCase.Request)
			require.NoError(t, err)

			reader := strings.NewReader(string(jsonData))
			req, err := http.NewRequest(http.MethodPost, common.ApiProfileResourceRoute, reader)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.AddProfileResource(c)
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

func TestAddProfileResource_BadRequest(t *testing.T) {
	noProfileName := testResourceRequest
	noProfileName.ProfileName = ""
	noResourcePropertyType := testResourceRequest
	noResourcePropertyType.Resource.Properties.ValueType = ""

	dic := mockDic()

	controller := NewResourceController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name    string
		Request []requests.AddResourceRequest
	}{
		{"invalid - no node profile name", []requests.AddResourceRequest{noProfileName}},
		{"invalid - No resource property type", []requests.AddResourceRequest{noResourcePropertyType}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			jsonData, err := json.Marshal(testCase.Request)
			require.NoError(t, err)

			reader := strings.NewReader(string(jsonData))
			req, err := http.NewRequest(http.MethodPost, common.ApiProfileResourceRoute, reader)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.AddProfileResource(c)
			require.NoError(t, err)

			var res commonDTO.BaseResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &res)
			require.NoError(t, err)

			// Assert
			assert.Equal(t, http.StatusBadRequest, recorder.Result().StatusCode, "HTTP status code not as expected")
			assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
			assert.Equal(t, http.StatusBadRequest, res.StatusCode, "BaseResponse status code not as expected")
			assert.NotEmpty(t, recorder.Body.String(), "Message is empty")
		})
	}
}

func TestAddProfileResource_UnitsOfMeasure_Validation(t *testing.T) {
	profile := dtos.ToProfileModel(buildTestProfileRequest().Profile)
	expectedRequestId := ExampleUUID

	emptyUnitsReq := testResourceRequest
	validReq := emptyUnitsReq
	validReq.Resource.Properties.Units = TestUnits
	invalidUnitsReq := emptyUnitsReq
	invalidUnitsReq.Resource.Properties.Units = "invalid"

	dic := mockDic()
	container.ConfigurationFrom(dic.Get).Writable.UoM.Validation = true
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("ProfileByName", validReq.ProfileName).Return(profile, nil)
	dbClientMock.On("UpdateProfile", mock.Anything).Return(nil)
	dbClientMock.On("NodesByProfileName", 0, -1, validReq.ProfileName).Return([]models.Node{}, nil)
	dbClientMock.On("NodeCountByProfileName", validReq.ProfileName).Return(int64(1), nil)
	uomMock := &mocks.UnitsOfMeasure{}
	uomMock.On("Validate", TestUnits).Return(true)
	uomMock.On("Validate", "").Return(true)
	uomMock.On("Validate", "invalid").Return(false)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
		container.UnitsOfMeasureInterfaceName: func(get di.Get) interface{} {
			return uomMock
		},
	})

	controller := NewResourceController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name               string
		Request            []requests.AddResourceRequest
		expectedStatusCode int
	}{
		{"valid - units not provided", []requests.AddResourceRequest{emptyUnitsReq}, http.StatusCreated},
		{"valid - expected units", []requests.AddResourceRequest{validReq}, http.StatusCreated},
		{"invalid - unexpected units", []requests.AddResourceRequest{invalidUnitsReq}, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			jsonData, err := json.Marshal(testCase.Request)
			require.NoError(t, err)

			reader := strings.NewReader(string(jsonData))
			req, err := http.NewRequest(http.MethodPost, common.ApiProfileResourceRoute, reader)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.AddProfileResource(c)
			require.NoError(t, err)

			var res []commonDTO.BaseResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &res)
			require.NoError(t, err)
			// Assert
			assert.Equal(t, http.StatusMultiStatus, recorder.Result().StatusCode, "HTTP status code not as expected")
			assert.Equal(t, common.ApiVersion, res[0].ApiVersion, "API Version not as expected")
			if res[0].RequestId != "" {
				assert.Equal(t, expectedRequestId, res[0].RequestId, "RequestID not as expected")
			}
			assert.Equal(t, testCase.expectedStatusCode, res[0].StatusCode, "BaseResponse status code not as expected")
			assert.NotEmpty(t, recorder.Body.String(), "Message is empty")
		})
	}
}
func TestPatchProfileResource(t *testing.T) {
	profile := dtos.ToProfileModel(buildTestProfileRequest().Profile)
	testReq := buildTestUpdateResourceRequest()
	nodeExistsProfileName := "nodeExists"
	nodeExistsProfile := profile
	nodeExistsProfile.Name = nodeExistsProfileName
	expectedRequestId := ExampleUUID
	emptyString := ""
	notFound := "notFoundName"

	valid := testReq
	validWithNoReqID := testReq
	validWithNoReqID.RequestId = emptyString
	nodeExists := testReq
	nodeExists.ProfileName = nodeExistsProfileName
	notFoundResource := testReq
	notFoundResource.Resource.Name = &notFound
	notFoundProfile := testReq
	notFoundProfile.ProfileName = notFound
	notFoundDBError := errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("node profile %s does not exists", notFoundProfile.ProfileName), nil)

	noProfileName := testReq
	noProfileName.ProfileName = emptyString

	dic := mockDic()
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("ProfileByName", valid.ProfileName).Return(profile, nil)
	dbClientMock.On("NodesByProfileName", 0, mock.Anything, valid.ProfileName).Return([]models.Node{}, nil)
	dbClientMock.On("NodeCountByProfileName", TestProfileName).Return(int64(1), nil)
	dbClientMock.On("UpdateProfile", mock.Anything).Return(nil)

	dbClientMock.On("ProfileByName", nodeExistsProfileName).Return(nodeExistsProfile, nil)
	dbClientMock.On("NodesByProfileName", 0, -1, nodeExistsProfileName).Return([]models.Node{{}}, nil)
	dbClientMock.On("NodeCountByProfileName", nodeExistsProfileName).Return(int64(1), nil)

	dbClientMock.On("ProfileByName", notFound).Return(profile, notFoundDBError)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})
	controller := NewResourceController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name                 string
		Request              []requests.UpdateResourceRequest
		expectedStatusCode   int
		expectedResponseCode int
	}{
		{"Valid - PatchProfileResourceRequest", []requests.UpdateResourceRequest{valid}, http.StatusMultiStatus, http.StatusOK},
		{"Valid - PatchProfileResourceRequest no requestId", []requests.UpdateResourceRequest{validWithNoReqID}, http.StatusMultiStatus, http.StatusOK},
		{"Valid - Profile is in use by other node", []requests.UpdateResourceRequest{nodeExists}, http.StatusMultiStatus, http.StatusOK},
		{"Invalid - Not found node resource", []requests.UpdateResourceRequest{notFoundResource}, http.StatusMultiStatus, http.StatusNotFound},
		{"Invalid - Not found node profile", []requests.UpdateResourceRequest{notFoundProfile}, http.StatusMultiStatus, http.StatusNotFound},
		{"Invalid - No profile name", []requests.UpdateResourceRequest{noProfileName}, http.StatusBadRequest, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			jsonData, err := json.Marshal(testCase.Request)
			require.NoError(t, err)

			reader := strings.NewReader(string(jsonData))

			req, err := http.NewRequest(http.MethodPatch, common.ApiProfileResourceRoute, reader)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.PatchProfileResource(c)
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

func TestDeleteResourceByName(t *testing.T) {
	dpModel := dtos.ToProfileModel(buildTestProfileRequest().Profile)
	notFoundName := "notFoundName"
	nodeExists := "nodeExists"

	dic := mockDic()
	dbClientMock := &mocks.DBClient{}
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

	controller := NewResourceController(dic)
	require.NotNil(t, controller)

	tests := []struct {
		name               string
		profileName        string
		resourceName       string
		expectedStatusCode int
	}{
		{"valid", TestProfileName, TestResourceName + "-dup", http.StatusOK},
		{"invalid - empty profile name", "", TestResourceName, http.StatusBadRequest},
		{"invalid - empty resource name", "", TestResourceName, http.StatusBadRequest},
		{"invalid - profile is in use by other node", nodeExists, TestResourceName, http.StatusConflict},
		{"invalid - profile not found", notFoundName, TestResourceName, http.StatusNotFound},
		{"invalid - resource not found in profile", TestProfileName, notFoundName, http.StatusNotFound},
		{"invalid - node resource is referenced by node command", TestProfileName, TestResourceName, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodDelete, common.ApiProfileResourceByNameRoute, http.NoBody)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Name, common.ResourceName)
			c.SetParamValues(testCase.profileName, testCase.resourceName)
			err = controller.DeleteResourceByName(c)
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

func TestDeleteResourceByName_StrictProfileChanges(t *testing.T) {
	dic := mockDic()
	configuration := container.ConfigurationFrom(dic.Get)
	configuration.Writable.ProfileChange.StrictProfileChanges = true
	dic.Update(di.ServiceConstructorMap{
		container.ConfigurationName: func(get di.Get) interface{} {
			return configuration
		},
	})

	controller := NewResourceController(dic)
	require.NotNil(t, controller)

	e := echo.New()
	req, err := http.NewRequest(http.MethodDelete, common.ApiProfileResourceByNameRoute, http.NoBody)
	require.NoError(t, err)

	// Act
	recorder := httptest.NewRecorder()
	c := e.NewContext(req, recorder)
	c.SetParamNames(common.Name, common.ResourceName)
	c.SetParamValues(TestProfileName, TestResourceName)
	err = controller.DeleteResourceByName(c)
	require.NoError(t, err)

	var res commonDTO.BaseResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &res)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, http.StatusLocked, recorder.Result().StatusCode, "HTTP status code not as expected")
	assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
	assert.Equal(t, http.StatusLocked, res.StatusCode, "BaseResponse status code not as expected")
}
