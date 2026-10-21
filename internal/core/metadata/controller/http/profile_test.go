//
// Copyright (C) 2020-2026 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"gopkg.in/yaml.v3"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	bootstrapConfig "github.com/edgexfoundry/go-mod-bootstrap/v4/config"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	commonDTO "github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/requests"
	responseDTO "github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/responses"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"

	"github.com/clint456/edgex-go/internal/core/metadata/config"
	"github.com/clint456/edgex-go/internal/core/metadata/container"
	"github.com/clint456/edgex-go/internal/core/metadata/infrastructure/interfaces/mocks"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testProfileLabels = []string{"MODBUS", "TEMP"}
var testAttributes = map[string]interface{}{
	"TestAttribute": "TestAttributeValue",
}
var testMappings = map[string]string{"0": "off", "1": "on"}
var testTags = map[string]any{
	"TestTagKey": "TestTagValue",
}
var testOptional = map[string]any{"TestOptionalKey": "TestOptionalValue"}

func buildTestProfileRequest() requests.ProfileRequest {
	var testResources = []dtos.Resource{{
		Name:        TestResourceName,
		Description: TestDescription,
		Attributes:  testAttributes,
		Properties: dtos.ResourceProperties{
			ValueType: common.ValueTypeInt16,
			ReadWrite: common.ReadWrite_RW,
			Units:     TestUnits,
			Optional:  testOptional,
		},
		Tags: testTags,
	}, {
		Name:        TestResourceName + "-dup",
		Description: TestDescription,
		Attributes:  testAttributes,
		Properties: dtos.ResourceProperties{
			ValueType: common.ValueTypeInt16,
			ReadWrite: common.ReadWrite_RW,
			Units:     TestUnits,
			Optional:  testOptional,
		},
		Tags: testTags,
	}}
	var testCommands = []dtos.Command{{
		Name:      TestCommandName,
		ReadWrite: common.ReadWrite_RW,
		ResourceOperations: []dtos.ResourceOperation{{
			Resource: TestResourceName,
			Mappings: testMappings,
		}},
	}}

	var testProfileReq = requests.ProfileRequest{
		BaseRequest: commonDTO.BaseRequest{
			RequestId:   ExampleUUID,
			Versionable: commonDTO.NewVersionable(),
		},
		Profile: dtos.Profile{
			ProfileBasicInfo: dtos.ProfileBasicInfo{
				Id:           ExampleUUID,
				Name:         TestProfileName,
				Manufacturer: TestManufacturer,
				Description:  TestDescription,
				Model:        TestModel,
				Labels:       testProfileLabels,
			},
			Resources: testResources,
			Commands:  testCommands,
		},
	}

	return testProfileReq
}

func buildTestProfileBasicInfoRequest() requests.ProfileBasicInfoRequest {
	testUUID := ExampleUUID
	testName := TestProfileName
	testDescription := TestDescription
	testManufacturer := TestManufacturer
	testModel := TestModel

	var testBasicInfoReq = requests.ProfileBasicInfoRequest{
		BaseRequest: commonDTO.BaseRequest{
			Versionable: commonDTO.NewVersionable(),
			RequestId:   ExampleUUID,
		},
		BasicInfo: dtos.UpdateProfileBasicInfo{
			Id:           &testUUID,
			Name:         &testName,
			Manufacturer: &testManufacturer,
			Description:  &testDescription,
			Model:        &testModel,
			Labels:       testProfileLabels,
		},
	}

	return testBasicInfoReq
}

func mockDic() *di.Container {
	return di.NewContainer(di.ServiceConstructorMap{
		container.ConfigurationName: func(get di.Get) interface{} {
			return &config.ConfigurationStruct{
				Writable: config.WritableInfo{
					LogLevel: "DEBUG",
				},
				Service: bootstrapConfig.ServiceInfo{
					RequestTimeout: "30s",
					MaxResultCount: 30,
				},
			}
		},
		bootstrapContainer.LoggingClientInterfaceName: func(get di.Get) interface{} {
			return logger.NewMockClient()
		},
	})
}

func createProfileRequestWithFile(fileContents []byte) (*http.Request, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "profile.yaml")
	if err != nil {
		return nil, err
	}
	_, err = part.Write(fileContents)
	if err != nil {
		return nil, err
	}
	boundary := writer.Boundary()

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequest(http.MethodPost, common.ApiProfileRoute+"/uploadfile", body)
	req.Header.Set(common.ContentType, "multipart/form-data; boundary="+boundary)
	return req, nil
}

func TestAddProfile_Created(t *testing.T) {
	profileRequest := buildTestProfileRequest()
	profileModel := requests.ProfileReqToProfileModel(profileRequest)
	expectedRequestId := ExampleUUID

	dic := mockDic()
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("AddProfile", profileModel).Return(profileModel, nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	controller := NewProfileController(dic)
	assert.NotNil(t, controller)

	valid := profileRequest
	noRequestId := profileRequest
	noRequestId.RequestId = ""

	tests := []struct {
		name    string
		Request []requests.ProfileRequest
	}{
		{"Valid - AddProfileRequest", []requests.ProfileRequest{valid}},
		{"Valid - No requestId", []requests.ProfileRequest{noRequestId}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			jsonData, err := json.Marshal(testCase.Request)
			require.NoError(t, err)

			reader := strings.NewReader(string(jsonData))
			req, err := http.NewRequest(http.MethodPost, common.ApiProfileRoute, reader)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.AddProfile(c)
			require.NoError(t, err)

			var res []commonDTO.BaseWithIdResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &res)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, http.StatusMultiStatus, recorder.Result().StatusCode, "HTTP status code not as expected")
			assert.Equal(t, common.ApiVersion, res[0].ApiVersion, "API Version not as expected")
			if res[0].RequestId != "" {
				assert.Equal(t, expectedRequestId, res[0].RequestId, "RequestID not as expected")
			}
			assert.Equal(t, http.StatusCreated, res[0].StatusCode, "BaseResponse status code not as expected")
			assert.Empty(t, res[0].Message, "Message should be empty when it is successful")
		})
	}
}

func TestAddProfile_BadRequest(t *testing.T) {
	dic := mockDic()

	controller := NewProfileController(dic)
	assert.NotNil(t, controller)

	profile := buildTestProfileRequest()
	badRequestId := profile
	badRequestId.RequestId = "niv3sl"
	noName := profile
	noName.Profile.Name = ""
	noResource := profile
	noResource.Profile.Resources = []dtos.Resource{}
	noResourceName := profile
	noResourceName.Profile.Resources = []dtos.Resource{{
		Description: TestDescription,
		Attributes:  testAttributes,
		Properties: dtos.ResourceProperties{
			ValueType: "INT16",
			ReadWrite: common.ReadWrite_RW,
		},
		Tags: testTags,
	}}
	noResourcePropertyType := profile
	noResourcePropertyType.Profile.Resources = []dtos.Resource{{
		Name:        TestResourceName,
		Description: TestDescription,
		Attributes:  testAttributes,
		Properties: dtos.ResourceProperties{
			ReadWrite: common.ReadWrite_RW,
		},
		Tags: testTags,
	}}
	noCommandName := profile
	noCommandName.Profile.Commands = []dtos.Command{{
		ReadWrite: common.ReadWrite_RW,
	}}
	noCommandReadWrite := profile
	noCommandReadWrite.Profile.Commands = []dtos.Command{{
		Name: TestCommandName,
	}}

	tests := []struct {
		name    string
		Request []requests.ProfileRequest
	}{
		{"Invalid - Bad requestId", []requests.ProfileRequest{badRequestId}},
		{"Invalid - Bad name", []requests.ProfileRequest{noName}},
		{"Invalid - No resource", []requests.ProfileRequest{noResource}},
		{"Invalid - No resource name", []requests.ProfileRequest{noResourceName}},
		{"Invalid - No resource property type", []requests.ProfileRequest{noResourcePropertyType}},
		{"Invalid - No command name", []requests.ProfileRequest{noCommandName}},
		{"Invalid - No command readWrite", []requests.ProfileRequest{noCommandReadWrite}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			jsonData, err := json.Marshal(testCase.Request)
			require.NoError(t, err)

			reader := strings.NewReader(string(jsonData))
			req, err := http.NewRequest(http.MethodPost, common.ApiProfileRoute, reader)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.AddProfile(c)
			require.NoError(t, err)

			// Assert
			assert.Equal(t, http.StatusBadRequest, recorder.Result().StatusCode, "HTTP status code not as expected")
			assert.NotEmpty(t, recorder.Body.String(), "Message is empty")
		})
	}
}

func TestAddProfile_Duplicated(t *testing.T) {
	expectedRequestId := ExampleUUID

	duplicateIdRequest := buildTestProfileRequest()
	duplicateIdModel := requests.ProfileReqToProfileModel(duplicateIdRequest)
	duplicateIdDBError := errors.NewCommonEdgeX(errors.KindDuplicateName, fmt.Sprintf("node profile id %s exists", duplicateIdModel.Id), nil)

	duplicateNameRequest := buildTestProfileRequest()
	duplicateNameRequest.Profile.Id = "" // The infrastructure layer will generate id when the id field is empty
	duplicateNameModel := requests.ProfileReqToProfileModel(duplicateNameRequest)
	duplicateNameDBError := errors.NewCommonEdgeX(errors.KindDuplicateName, fmt.Sprintf("node profile name %s exists", duplicateNameModel.Name), nil)

	dic := mockDic()
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("AddProfile", duplicateNameModel).Return(duplicateNameModel, duplicateNameDBError)
	dbClientMock.On("AddProfile", duplicateIdModel).Return(duplicateIdModel, duplicateIdDBError)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	controller := NewProfileController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name          string
		request       []requests.ProfileRequest
		expectedError errors.CommonEdgeX
	}{
		{"duplicate id", []requests.ProfileRequest{duplicateIdRequest}, duplicateIdDBError},
		{"duplicate name", []requests.ProfileRequest{duplicateNameRequest}, duplicateNameDBError},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			jsonData, err := json.Marshal(testCase.request)
			require.NoError(t, err)

			reader := strings.NewReader(string(jsonData))
			req, err := http.NewRequest(http.MethodPost, common.ApiProfileRoute, reader)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.AddProfile(c)
			require.NoError(t, err)

			var res []commonDTO.BaseWithIdResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &res)
			require.NoError(t, err)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, http.StatusMultiStatus, recorder.Result().StatusCode, "HTTP status code not as expected")
			assert.Equal(t, common.ApiVersion, res[0].ApiVersion, "API Version not as expected")
			assert.Equal(t, expectedRequestId, res[0].RequestId, "RequestID not as expected")
			assert.Equal(t, http.StatusConflict, res[0].StatusCode, "BaseResponse status code not as expected")
			assert.Contains(t, res[0].Message, testCase.expectedError.Message(), "Message not as expected")
		})
	}
}

func TestAddProfile_UnitsOfMeasure_Validation(t *testing.T) {
	profileRequest := buildTestProfileRequest()
	profileModel := requests.ProfileReqToProfileModel(profileRequest)
	expectedRequestId := ExampleUUID

	emptyUnits := ""
	invalidUnits := "invalid"
	validReq := profileRequest
	emptyUnitsReq := buildTestProfileRequest()
	for i := range emptyUnitsReq.Profile.Resources {
		emptyUnitsReq.Profile.Resources[i].Properties.Units = emptyUnits
	}
	noUnitsModel := requests.ProfileReqToProfileModel(emptyUnitsReq)
	invalidUnitsReq := buildTestProfileRequest()
	for i := range invalidUnitsReq.Profile.Resources {
		invalidUnitsReq.Profile.Resources[i].Properties.Units = invalidUnits
	}

	dic := mockDic()
	container.ConfigurationFrom(dic.Get).Writable.UoM.Validation = true
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("AddProfile", profileModel).Return(profileModel, nil)
	dbClientMock.On("AddProfile", noUnitsModel).Return(noUnitsModel, nil)
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

	controller := NewProfileController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name               string
		Request            []requests.ProfileRequest
		expectedStatusCode int
	}{
		{"valid - expected units", []requests.ProfileRequest{validReq}, http.StatusCreated},
		{"valid - units not provided", []requests.ProfileRequest{emptyUnitsReq}, http.StatusCreated},
		{"invalid - unexpected units", []requests.ProfileRequest{invalidUnitsReq}, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			jsonData, err := json.Marshal(testCase.Request)
			require.NoError(t, err)

			reader := strings.NewReader(string(jsonData))
			req, err := http.NewRequest(http.MethodPost, common.ApiProfileRoute, reader)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.AddProfile(c)
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

func TestUpdateProfile(t *testing.T) {
	profileRequest := buildTestProfileRequest()
	profileModel := requests.ProfileReqToProfileModel(profileRequest)
	expectedRequestId := ExampleUUID

	valid := profileRequest
	noRequestId := profileRequest
	noRequestId.RequestId = ""
	noName := profileRequest
	noName.Profile.Name = ""
	noResource := profileRequest
	noResource.Profile.Resources = []dtos.Resource{}
	noResourceName := profileRequest
	noResourceName.Profile.Resources = []dtos.Resource{{
		Description: TestDescription,
		Attributes:  testAttributes,
		Properties: dtos.ResourceProperties{
			ValueType: "INT16",
			ReadWrite: common.ReadWrite_RW,
		},
		Tags: testTags,
	}}
	noResourcePropertyType := profileRequest
	noResourcePropertyType.Profile.Resources = []dtos.Resource{{
		Name:        TestResourceName,
		Description: TestDescription,
		Attributes:  testAttributes,
		Properties: dtos.ResourceProperties{
			ReadWrite: common.ReadWrite_RW,
		},
		Tags: testTags,
	}}
	noCommandName := profileRequest
	noCommandName.Profile.Commands = []dtos.Command{{
		ReadWrite: common.ReadWrite_RW,
	}}
	noCommandReadWrite := profileRequest
	noCommandReadWrite.Profile.Commands = []dtos.Command{{
		Name: TestCommandName,
	}}
	notFound := profileRequest
	notFound.Profile.Name = "testNode"
	notFoundProfileModel := dtos.ToProfileModel(notFound.Profile)
	notFoundDBError := errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("node profile %s does not exists", notFound.Profile.Name), nil)

	dic := mockDic()
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("UpdateProfile", profileModel).Return(nil)
	dbClientMock.On("UpdateProfile", notFoundProfileModel).Return(notFoundDBError)
	dbClientMock.On("NodeCountByProfileName", profileModel.Name).Return(int64(1), nil)
	dbClientMock.On("NodesByProfileName", 0, -1, profileModel.Name).Return([]models.Node{{ServiceName: testNodeServiceName}}, nil)
	dbClientMock.On("NodeServiceByName", testNodeServiceName).Return(models.NodeService{}, nil)
	dbClientMock.On("ProfileByName", profileModel.Name).Return(profileModel, nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	controller := NewProfileController(dic)
	require.NotNil(t, controller)

	tests := []struct {
		name               string
		request            []requests.ProfileRequest
		expectedStatusCode int
	}{
		{"Valid - ProfileRequest", []requests.ProfileRequest{valid}, http.StatusOK},
		{"Invalid - No name", []requests.ProfileRequest{noName}, http.StatusBadRequest},
		{"Invalid - No resource", []requests.ProfileRequest{noResource}, http.StatusBadRequest},
		{"Invalid - No resource name", []requests.ProfileRequest{noResourceName}, http.StatusBadRequest},
		{"Invalid - No resource property type", []requests.ProfileRequest{noResourcePropertyType}, http.StatusBadRequest},
		{"Invalid - No command name", []requests.ProfileRequest{noCommandName}, http.StatusBadRequest},
		{"Invalid - No command readWrite", []requests.ProfileRequest{noCommandReadWrite}, http.StatusBadRequest},
		{"Valid - No requestId", []requests.ProfileRequest{noRequestId}, http.StatusOK},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			jsonData, err := json.Marshal(testCase.request)
			require.NoError(t, err)

			reader := strings.NewReader(string(jsonData))
			req, err := http.NewRequest(http.MethodPost, common.ApiProfileRoute, reader)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.UpdateProfile(c)
			require.NoError(t, err)

			if testCase.expectedStatusCode == http.StatusBadRequest {
				var res commonDTO.BaseResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)

				// Assert
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, res.StatusCode, "BaseResponse status code not as expected")
				assert.NotEmpty(t, recorder.Body.String(), "Message is empty")
			} else {
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
			}
		})
	}
}

func TestUpdateProfile_StrictProfileChanges(t *testing.T) {
	valid := buildTestProfileRequest()

	dic := mockDic()
	configuration := container.ConfigurationFrom(dic.Get)
	configuration.Writable.ProfileChange.StrictProfileChanges = true
	dic.Update(di.ServiceConstructorMap{
		container.ConfigurationName: func(get di.Get) interface{} {
			return configuration
		},
	})

	controller := NewProfileController(dic)
	require.NotNil(t, controller)

	jsonData, err := json.Marshal(valid)
	require.NoError(t, err)

	reader := strings.NewReader(string(jsonData))
	req, err := http.NewRequest(http.MethodPost, common.ApiProfileRoute, reader)
	require.NoError(t, err)

	e := echo.New()
	// Act
	recorder := httptest.NewRecorder()
	c := e.NewContext(req, recorder)
	err = controller.UpdateProfile(c)
	require.NoError(t, err)

	var res commonDTO.BaseWithIdResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &res)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, http.StatusLocked, recorder.Result().StatusCode, "HTTP status code not as expected")
	assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
	assert.Equal(t, http.StatusLocked, res.StatusCode, "HTTP status code not as expected")
	assert.NotEmpty(t, recorder.Body.String(), "Message is empty")

}

func TestPatchProfileBasicInfo(t *testing.T) {
	expectedRequestId := ExampleUUID
	testReq := buildTestProfileBasicInfoRequest()
	dpModel := requests.ProfileReqToProfileModel(buildTestProfileRequest())

	valid := testReq
	noRequestId := testReq
	noRequestId.RequestId = ""
	noName := testReq
	noName.BasicInfo.Name = nil
	noId := testReq
	noId.BasicInfo.Id = nil

	noIdAndName := testReq
	noIdAndName.BasicInfo.Id = nil
	noIdAndName.BasicInfo.Name = nil
	emptyName := testReq
	emptyString := ""
	emptyName.BasicInfo.Name = &emptyString
	notFound := testReq
	notFound.BasicInfo.Id = nil
	notFoundName := "testNode"
	notFound.BasicInfo.Name = &notFoundName

	dic := mockDic()
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("ProfileById", *valid.BasicInfo.Id).Return(dpModel, nil)
	dbClientMock.On("ProfileByName", *valid.BasicInfo.Name).Return(dpModel, nil)
	dbClientMock.On("ProfileByName", notFoundName).Return(dpModel, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "not found", nil))
	dbClientMock.On("UpdateProfile", mock.Anything).Return(nil)
	dbClientMock.On("NodeCountByProfileName", *valid.BasicInfo.Name).Return(int64(1), nil)
	dbClientMock.On("NodesByProfileName", 0, -1, *valid.BasicInfo.Name).Return([]models.Node{{ServiceName: testNodeServiceName}}, nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	controller := NewProfileController(dic)
	require.NotNil(t, controller)

	tests := []struct {
		name               string
		request            []requests.ProfileBasicInfoRequest
		expectedStatusCode int
	}{
		{"valid", []requests.ProfileBasicInfoRequest{valid}, http.StatusOK},
		{"valid - no request id", []requests.ProfileBasicInfoRequest{noRequestId}, http.StatusOK},
		{"valid - no name", []requests.ProfileBasicInfoRequest{noName}, http.StatusOK},
		{"invalid - no id and name", []requests.ProfileBasicInfoRequest{noIdAndName}, http.StatusBadRequest},
		{"invalid - empty name", []requests.ProfileBasicInfoRequest{emptyName}, http.StatusBadRequest},
		{"invalid - node profile not found", []requests.ProfileBasicInfoRequest{notFound}, http.StatusNotFound},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			jsonData, err := json.Marshal(testCase.request)
			require.NoError(t, err)

			reader := strings.NewReader(string(jsonData))
			req, err := http.NewRequest(http.MethodPatch, common.ApiProfileBasicInfoRoute, reader)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.PatchProfileBasicInfo(c)
			require.NoError(t, err)

			if testCase.expectedStatusCode == http.StatusBadRequest {
				var res commonDTO.BaseResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)

				// Assert
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, res.StatusCode, "BaseResponse status code not as expected")
				assert.NotEmpty(t, recorder.Body.String(), "Message is empty")
			} else {
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
			}
		})
	}
}

func TestAddProfileByYaml_Created(t *testing.T) {
	profileDTO := buildTestProfileRequest().Profile
	profileModel := dtos.ToProfileModel(profileDTO)

	dic := mockDic()
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("AddProfile", profileModel).Return(profileModel, nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	controller := NewProfileController(dic)
	assert.NotNil(t, controller)

	valid, err := yaml.Marshal(profileDTO)
	require.NoError(t, err)
	req, err := createProfileRequestWithFile(valid)
	require.NoError(t, err)

	e := echo.New()
	// Act
	recorder := httptest.NewRecorder()
	c := e.NewContext(req, recorder)
	err = controller.AddProfileByYaml(c)
	require.NoError(t, err)

	var res commonDTO.BaseWithIdResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &res)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, recorder.Result().StatusCode, "HTTP status code not as expected")
	assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
	assert.Equal(t, http.StatusCreated, res.StatusCode, "BaseResponse status code not as expected")
	assert.Empty(t, res.Message, "Message should be empty when it is successful")
}

func TestAddProfileByYaml_BadRequest(t *testing.T) {
	dic := mockDic()

	controller := NewProfileController(dic)
	assert.NotNil(t, controller)

	profile := buildTestProfileRequest().Profile
	noName := profile
	noName.Name = ""
	noResource := profile
	noResource.Resources = []dtos.Resource{}
	noResourceName := profile
	noResourceName.Resources = []dtos.Resource{{
		Description: TestDescription,
		Attributes:  testAttributes,
		Properties: dtos.ResourceProperties{
			ValueType: "INT16",
			ReadWrite: common.ReadWrite_RW,
		},
		Tags: testTags,
	}}
	noResourcePropertyType := profile
	noResourcePropertyType.Resources = []dtos.Resource{{
		Name:        TestResourceName,
		Description: TestDescription,
		Attributes:  testAttributes,
		Properties: dtos.ResourceProperties{
			ReadWrite: common.ReadWrite_RW,
		},
		Tags: testTags,
	}}
	noCommandName := profile
	noCommandName.Commands = []dtos.Command{{
		ReadWrite: common.ReadWrite_RW,
	}}
	noCommandReadWrite := profile
	noCommandReadWrite.Commands = []dtos.Command{{
		Name: TestCommandName,
	}}
	tests := []struct {
		name    string
		Request dtos.Profile
	}{
		{"Invalid - No name", noName},
		{"Invalid - No resource", noResource},
		{"Invalid - No resource name", noResourceName},
		{"Invalid - No resource property type", noResourcePropertyType},
		{"Invalid - No command name", noCommandName},
		{"Invalid - No command readWrite", noCommandReadWrite},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			valid, err := yaml.Marshal(testCase.Request)
			require.NoError(t, err)
			req, err := createProfileRequestWithFile(valid)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.AddProfileByYaml(c)
			require.NoError(t, err)

			var res commonDTO.BaseWithIdResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &res)
			assert.NoError(t, err)

			// Assert
			assert.Equal(t, http.StatusBadRequest, recorder.Result().StatusCode, "HTTP status code not as expected")
			assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
			assert.Equal(t, http.StatusBadRequest, res.StatusCode, "HTTP status code not as expected")
			assert.NotEmpty(t, recorder.Body.String(), "Message is empty")
		})
	}
}

func TestAddProfileByYaml_Duplicated(t *testing.T) {
	profileDTO := buildTestProfileRequest().Profile
	profileModel := dtos.ToProfileModel(profileDTO)
	dbError := errors.NewCommonEdgeX(errors.KindDuplicateName, fmt.Sprintf("node profile %s already exists", TestProfileName), nil)

	dic := mockDic()
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("AddProfile", profileModel).Return(profileModel, dbError)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	controller := NewProfileController(dic)
	assert.NotNil(t, controller)

	valid, err := yaml.Marshal(profileDTO)
	require.NoError(t, err)
	req, err := createProfileRequestWithFile(valid)
	require.NoError(t, err)

	e := echo.New()
	// Act
	recorder := httptest.NewRecorder()
	c := e.NewContext(req, recorder)
	err = controller.AddProfileByYaml(c)
	require.NoError(t, err)

	var res commonDTO.BaseWithIdResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &res)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, http.StatusConflict, recorder.Result().StatusCode, "HTTP status code not as expected")
	assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
	assert.Equal(t, http.StatusConflict, res.StatusCode, "BaseResponse status code not as expected")
	assert.Contains(t, res.Message, dbError.Message(), "Message not as expected")
}

func TestAddProfileByYaml_MissingFile(t *testing.T) {
	profileDTO := buildTestProfileRequest().Profile
	dic := mockDic()

	controller := NewProfileController(dic)
	assert.NotNil(t, controller)

	valid, err := yaml.Marshal(profileDTO)
	require.NoError(t, err)
	req, err := createProfileRequestWithFile(valid)
	require.NoError(t, err)

	req.MultipartForm = new(multipart.Form)
	req.MultipartForm.File = nil

	e := echo.New()
	// Act
	recorder := httptest.NewRecorder()
	c := e.NewContext(req, recorder)
	err = controller.AddProfileByYaml(c)
	require.NoError(t, err)

	var res commonDTO.BaseWithIdResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &res)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, http.StatusBadRequest, recorder.Result().StatusCode, "HTTP status code not as expected")
	assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
	assert.Equal(t, http.StatusBadRequest, res.StatusCode, "BaseResponse status code not as expected")
	assert.Contains(t, res.Message, "missing yaml file")
}

func TestUpdateProfileByYaml(t *testing.T) {
	profile := buildTestProfileRequest().Profile

	valid := profile
	validProfileModel := dtos.ToProfileModel(valid)
	noName := profile
	noName.Name = ""
	noResource := profile
	noResource.Resources = []dtos.Resource{}
	noResourceName := profile
	noResourceName.Resources = []dtos.Resource{{
		Description: TestDescription,
		Attributes:  testAttributes,
		Properties: dtos.ResourceProperties{
			ValueType: "INT16",
			ReadWrite: common.ReadWrite_RW,
		},
		Tags: testTags,
	}}
	noResourcePropertyType := profile
	noResourcePropertyType.Resources = []dtos.Resource{{
		Name:        TestResourceName,
		Description: TestDescription,
		Attributes:  testAttributes,
		Properties: dtos.ResourceProperties{
			ReadWrite: common.ReadWrite_RW,
		},
		Tags: testTags,
	}}
	noCommandName := profile
	noCommandName.Commands = []dtos.Command{{
		ReadWrite: common.ReadWrite_RW,
	}}
	noCommandReadWrite := profile
	noCommandReadWrite.Commands = []dtos.Command{{
		Name: TestCommandName,
	}}
	notFound := profile
	notFound.Name = "testNode"
	notFoundProfileModel := dtos.ToProfileModel(notFound)
	notFoundDBError := errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("node profile %s does not exists", notFoundProfileModel.Name), nil)

	dic := mockDic()
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("UpdateProfile", validProfileModel).Return(nil)
	dbClientMock.On("UpdateProfile", notFoundProfileModel).Return(notFoundDBError)
	dbClientMock.On("NodeCountByProfileName", validProfileModel.Name).Return(int64(1), nil)
	dbClientMock.On("NodesByProfileName", 0, -1, validProfileModel.Name).Return([]models.Node{{ServiceName: testNodeServiceName}}, nil)
	dbClientMock.On("NodeServiceByName", testNodeServiceName).Return(models.NodeService{}, nil)
	dbClientMock.On("ProfileByName", validProfileModel.Name).Return(validProfileModel, nil)
	dbClientMock.On("NodeCountByProfileName", validProfileModel.Name).Return(int64(0), nil)
	dbClientMock.On("NodeCountByProfileName", notFoundProfileModel.Name).Return(int64(0), nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	controller := NewProfileController(dic)
	require.NotNil(t, controller)

	tests := []struct {
		name               string
		request            dtos.Profile
		expectedStatusCode int
	}{
		{"Valid", valid, http.StatusOK},
		{"Invalid - No name", noName, http.StatusBadRequest},
		{"Invalid - No resource", noResource, http.StatusBadRequest},
		{"Invalid - No resource name", noResourceName, http.StatusBadRequest},
		{"Invalid - No resource property type", noResourcePropertyType, http.StatusBadRequest},
		{"Invalid - No command name", noCommandName, http.StatusBadRequest},
		{"Invalid - No command readWrite", noCommandReadWrite, http.StatusBadRequest},
		{"Not found", notFound, http.StatusNotFound},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			valid, err := yaml.Marshal(testCase.request)
			require.NoError(t, err)
			req, err := createProfileRequestWithFile(valid)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			err = controller.UpdateProfileByYaml(c)
			require.NoError(t, err)

			var res commonDTO.BaseWithIdResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &res)
			assert.NoError(t, err)

			// Assert
			assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
			assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
			assert.Equal(t, testCase.expectedStatusCode, res.StatusCode, "HTTP status code not as expected")
			assert.NotEmpty(t, recorder.Body.String(), "Message is empty")
		})
	}
}

func TestUpdateProfileByYaml_StrictProfileChanges(t *testing.T) {
	valid := buildTestProfileRequest().Profile

	dic := mockDic()
	configuration := container.ConfigurationFrom(dic.Get)
	configuration.Writable.ProfileChange.StrictProfileChanges = true
	dic.Update(di.ServiceConstructorMap{
		container.ConfigurationName: func(get di.Get) interface{} {
			return configuration
		},
	})

	controller := NewProfileController(dic)
	require.NotNil(t, controller)

	validBytes, err := yaml.Marshal(valid)
	require.NoError(t, err)
	req, err := createProfileRequestWithFile(validBytes)
	require.NoError(t, err)

	e := echo.New()
	// Act
	recorder := httptest.NewRecorder()
	c := e.NewContext(req, recorder)
	err = controller.UpdateProfileByYaml(c)
	require.NoError(t, err)

	var res commonDTO.BaseWithIdResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &res)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, http.StatusLocked, recorder.Result().StatusCode, "HTTP status code not as expected")
	assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
	assert.Equal(t, http.StatusLocked, res.StatusCode, "HTTP status code not as expected")
	assert.NotEmpty(t, recorder.Body.String(), "Message is empty")
}

func TestProfileByName(t *testing.T) {
	profile := dtos.ToProfileModel(buildTestProfileRequest().Profile)
	emptyName := ""
	notFoundName := "notFoundName"

	dic := mockDic()
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("ProfileByName", profile.Name).Return(profile, nil)
	dbClientMock.On("ProfileByName", notFoundName).Return(models.Profile{}, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "node profile doesn't exist in the database", nil))
	dbClientMock.On("NodeCountByProfileName", profile.Name).Return(int64(1), nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	controller := NewProfileController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name               string
		profileName        string
		errorExpected      bool
		expectedStatusCode int
	}{
		{"Valid - find node profile by name", profile.Name, false, http.StatusOK},
		{"Invalid - name parameter is empty", emptyName, true, http.StatusBadRequest},
		{"Invalid - node profile not found by name", notFoundName, true, http.StatusNotFound},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			reqPath := fmt.Sprintf("%s/%s/%s", common.ApiProfileRoute, common.Name, testCase.profileName)
			req, err := http.NewRequest(http.MethodGet, reqPath, http.NoBody)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Name)
			c.SetParamValues(testCase.profileName)
			err = controller.ProfileByName(c)
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
				var res responseDTO.ProfileResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.profileName, res.Profile.Name, "Event Id not as expected")
				assert.Equal(t, int64(1), res.Profile.LinkedNodeCount, "Linked node count not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestDeleteProfileByName(t *testing.T) {
	profile := dtos.ToProfileModel(buildTestProfileRequest().Profile)
	noName := ""
	notFoundName := "notFoundName"
	nodeExists := "nodeExists"
	provisionWatcherExists := "provisionWatcherExists"

	dic := mockDic()
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("ProfileByName", profile.Name).Return(models.Profile{}, nil)
	dbClientMock.On("NodesByProfileName", 0, 1, profile.Name).Return([]models.Node{}, nil)
	dbClientMock.On("ProvisionWatchersByProfileName", 0, 1, profile.Name).Return([]models.ProvisionWatcher{}, nil)
	dbClientMock.On("DeleteProfileByName", profile.Name).Return(nil)

	dbClientMock.On("ProfileByName", notFoundName).Return(models.Profile{}, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "not found", nil))

	dbClientMock.On("ProfileByName", nodeExists).Return(models.Profile{}, nil)
	dbClientMock.On("NodesByProfileName", 0, 1, nodeExists).Return([]models.Node{{ServiceName: testNodeServiceName}}, nil)

	dbClientMock.On("ProfileByName", provisionWatcherExists).Return(models.Profile{}, nil)
	dbClientMock.On("NodesByProfileName", 0, 1, provisionWatcherExists).Return([]models.Node{}, nil)
	dbClientMock.On("ProvisionWatchersByProfileName", 0, 1, provisionWatcherExists).Return([]models.ProvisionWatcher{models.ProvisionWatcher{}}, nil)

	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	controller := NewProfileController(dic)
	require.NotNil(t, controller)

	tests := []struct {
		name               string
		profileName        string
		errorExpected      bool
		expectedStatusCode int
	}{
		{"Valid - delete node profile by name", profile.Name, false, http.StatusOK},
		{"Invalid - name parameter is empty", noName, true, http.StatusBadRequest},
		{"Invalid - node profile not found by name", notFoundName, true, http.StatusNotFound},
		{"Invalid - associated node exists", nodeExists, true, http.StatusConflict},
		{"Invalid - associated provisionWatcher Exists", provisionWatcherExists, true, http.StatusConflict},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			reqPath := fmt.Sprintf("%s/%s/%s", common.ApiProfileRoute, common.Name, testCase.profileName)
			req, err := http.NewRequest(http.MethodDelete, reqPath, http.NoBody)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Name)
			c.SetParamValues(testCase.profileName)
			err = controller.DeleteProfileByName(c)
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

func TestDeleteProfileByName_StrictProfileChanges(t *testing.T) {
	dic := mockDic()
	configuration := container.ConfigurationFrom(dic.Get)
	configuration.Writable.ProfileChange.StrictProfileDeletes = true
	dic.Update(di.ServiceConstructorMap{
		container.ConfigurationName: func(get di.Get) interface{} {
			return configuration
		},
	})

	controller := NewProfileController(dic)
	require.NotNil(t, controller)

	e := echo.New()
	req, err := http.NewRequest(http.MethodDelete, common.ApiProfileByNameRoute, http.NoBody)
	require.NoError(t, err)

	// Act
	recorder := httptest.NewRecorder()
	c := e.NewContext(req, recorder)
	c.SetParamNames(common.Name)
	c.SetParamValues(TestProfileName)
	err = controller.DeleteProfileByName(c)
	require.NoError(t, err)

	var res commonDTO.BaseResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &res)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, http.StatusLocked, recorder.Result().StatusCode, "HTTP status code not as expected")
	assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
	assert.Equal(t, http.StatusLocked, res.StatusCode, "BaseResponse status code not as expected")
	assert.NotEmpty(t, res.Message, "Message is empty")
}

func TestAllProfiles(t *testing.T) {
	profile := dtos.ToProfileModel(buildTestProfileRequest().Profile)
	profiles := []models.Profile{profile, profile, profile}
	expectedTotalProfileCount := int64(3)
	expectedLinkedNodeCount := int64(1)

	dic := mockDic()
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("ProfileCountByLabels", []string(nil)).Return(expectedTotalProfileCount, nil)
	dbClientMock.On("ProfileCountByLabels", testProfileLabels).Return(expectedTotalProfileCount, nil)
	dbClientMock.On("AllProfiles", 0, 10, []string(nil)).Return(profiles, nil)
	dbClientMock.On("AllProfiles", 0, 5, testProfileLabels).Return([]models.Profile{profiles[0], profiles[1]}, nil)
	dbClientMock.On("AllProfiles", 1, 2, []string(nil)).Return([]models.Profile{profiles[1], profiles[2]}, nil)
	dbClientMock.On("AllProfiles", 4, 1, testProfileLabels).Return([]models.Profile{}, errors.NewCommonEdgeX(errors.KindRangeNotSatisfiable, "query objects bounds out of range.", nil))
	dbClientMock.On("NodeCountByProfileName", profile.Name).Return(expectedLinkedNodeCount, nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})
	controller := NewProfileController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name                    string
		offset                  string
		limit                   string
		labels                  string
		errorExpected           bool
		expectedCount           int
		expectedTotalCount      int64
		expectedLinkedNodeCount int64
		expectedStatusCode      int
	}{
		{"Valid - get node profiles without labels", "0", "10", "", false, 3, expectedTotalProfileCount, expectedLinkedNodeCount, http.StatusOK},
		{"Valid - get node profiles with labels", "0", "5", strings.Join(testProfileLabels, ","), false, 2, expectedTotalProfileCount, expectedLinkedNodeCount, http.StatusOK},
		{"Valid - get node profiles with offset and no labels", "1", "2", "", false, 2, expectedTotalProfileCount, expectedLinkedNodeCount, http.StatusOK},
		{"Invalid - offset out of range", "4", "1", strings.Join(testProfileLabels, ","), true, 0, expectedTotalProfileCount, expectedLinkedNodeCount, http.StatusRequestedRangeNotSatisfiable},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiAllProfileRoute, http.NoBody)
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
			err = controller.AllProfiles(c)
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
				var res responseDTO.MultiProfilesResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.expectedCount, len(res.Profiles), "Profile count not as expected")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
				for _, profile := range res.Profiles {
					assert.Equal(t, testCase.expectedLinkedNodeCount, profile.LinkedNodeCount, "Linked node count not as expected")
				}
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestProfilesByModel(t *testing.T) {
	profile := dtos.ToProfileModel(buildTestProfileRequest().Profile)
	profiles := []models.Profile{profile, profile, profile}
	expectedTotalProfileCount := int64(3)

	dic := mockDic()
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("ProfileCountByModel", TestModel).Return(expectedTotalProfileCount, nil)
	dbClientMock.On("ProfilesByModel", 0, 10, TestModel).Return(profiles, nil)
	dbClientMock.On("ProfilesByModel", 1, 2, TestModel).Return([]models.Profile{profiles[1], profiles[2]}, nil)
	dbClientMock.On("ProfilesByModel", 4, 1, TestModel).Return([]models.Profile{}, errors.NewCommonEdgeX(errors.KindRangeNotSatisfiable, "query objects bounds out of range.", nil))
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})
	controller := NewProfileController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name               string
		offset             string
		limit              string
		model              string
		errorExpected      bool
		expectedCount      int
		expectedTotalCount int64
		expectedStatusCode int
	}{
		{"Valid - get node profiles by model", "0", "10", TestModel, false, 3, expectedTotalProfileCount, http.StatusOK},
		{"Valid - get node profiles by model with offset and limit", "1", "2", TestModel, false, 2, expectedTotalProfileCount, http.StatusOK},
		{"Invalid - offset out of range", "4", "1", TestModel, true, 0, expectedTotalProfileCount, http.StatusRequestedRangeNotSatisfiable},
		{"Invalid - model is empty", "0", "10", "", true, 0, expectedTotalProfileCount, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiProfileByModelRoute, http.NoBody)
			query := req.URL.Query()
			query.Add(common.Offset, testCase.offset)
			query.Add(common.Limit, testCase.limit)
			req.URL.RawQuery = query.Encode()
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Model)
			c.SetParamValues(testCase.model)
			err = controller.ProfilesByModel(c)
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
				var res responseDTO.MultiProfilesResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.expectedCount, len(res.Profiles), "Profile count not as expected")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestProfilesByManufacturer(t *testing.T) {
	profile := dtos.ToProfileModel(buildTestProfileRequest().Profile)
	profiles := []models.Profile{profile, profile, profile}
	expectedTotalProfileCount := int64(3)

	dic := mockDic()
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("ProfileCountByManufacturer", TestManufacturer).Return(expectedTotalProfileCount, nil)
	dbClientMock.On("ProfilesByManufacturer", 0, 10, TestManufacturer).Return(profiles, nil)
	dbClientMock.On("ProfilesByManufacturer", 1, 2, TestManufacturer).Return([]models.Profile{profiles[1], profiles[2]}, nil)
	dbClientMock.On("ProfilesB yManufacturer", 4, 1, TestManufacturer).Return([]models.Profile{}, errors.NewCommonEdgeX(errors.KindRangeNotSatisfiable, "query objects bounds out of range.", nil))
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})
	controller := NewProfileController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name               string
		offset             string
		limit              string
		manufacturer       string
		errorExpected      bool
		expectedCount      int
		expectedTotalCount int64
		expectedStatusCode int
	}{
		{"Valid - get node profiles by manufacturer", "0", "10", TestManufacturer, false, 3, expectedTotalProfileCount, http.StatusOK},
		{"Valid - get node profiles by manufacturer with offset and limit", "1", "2", TestManufacturer, false, 2, expectedTotalProfileCount, http.StatusOK},
		{"Invalid - offset out of range", "4", "1", TestManufacturer, true, 0, expectedTotalProfileCount, http.StatusRequestedRangeNotSatisfiable},
		{"Invalid - manufacturer is empty", "0", "10", "", true, 0, expectedTotalProfileCount, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiProfileByManufacturerRoute, http.NoBody)
			query := req.URL.Query()
			query.Add(common.Offset, testCase.offset)
			query.Add(common.Limit, testCase.limit)
			req.URL.RawQuery = query.Encode()
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Manufacturer)
			c.SetParamValues(testCase.manufacturer)
			err = controller.ProfilesByManufacturer(c)
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
				var res responseDTO.MultiProfilesResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.expectedCount, len(res.Profiles), "Profile count not as expected")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestProfilesByManufacturerAndModel(t *testing.T) {
	profile := dtos.ToProfileModel(buildTestProfileRequest().Profile)
	profiles := []models.Profile{profile, profile, profile}
	expectedTotalProfileCount := int64(3)

	dic := mockDic()
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("ProfileCountByManufacturerAndModel", TestManufacturer, TestModel).Return(expectedTotalProfileCount, nil)
	dbClientMock.On("ProfilesByManufacturerAndModel", 0, 10, TestManufacturer, TestModel).Return(profiles, nil)
	dbClientMock.On("ProfilesByManufacturerAndModel", 1, 2, TestManufacturer, TestModel).Return([]models.Profile{profiles[1], profiles[2]}, nil)
	dbClientMock.On("ProfilesByManufacturerAndModel", 4, 1, TestManufacturer, TestModel).Return([]models.Profile{}, expectedTotalProfileCount, errors.NewCommonEdgeX(errors.KindRangeNotSatisfiable, "query objects bounds out of range.", nil))
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})
	controller := NewProfileController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name               string
		offset             string
		limit              string
		manufacturer       string
		model              string
		errorExpected      bool
		expectedCount      int
		expectedTotalCount int64
		expectedStatusCode int
	}{
		{"Valid - get node profiles by manufacturer and model", "0", "10", TestManufacturer, TestModel, false, 3, expectedTotalProfileCount, http.StatusOK},
		{"Valid - get node profiles by manufacturer with offset and limit", "1", "2", TestManufacturer, TestModel, false, 2, expectedTotalProfileCount, http.StatusOK},
		{"Invalid - offset out of range", "4", "1", TestManufacturer, TestModel, true, 0, expectedTotalProfileCount, http.StatusRequestedRangeNotSatisfiable},
		{"Invalid - manufacturer is empty", "0", "10", "", TestModel, true, 0, expectedTotalProfileCount, http.StatusBadRequest},
		{"Invalid - model is empty", "0", "10", TestManufacturer, "", true, 0, expectedTotalProfileCount, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiProfileByManufacturerAndModelRoute, http.NoBody)
			query := req.URL.Query()
			query.Add(common.Offset, testCase.offset)
			query.Add(common.Limit, testCase.limit)
			req.URL.RawQuery = query.Encode()
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Manufacturer, common.Model)
			c.SetParamValues(testCase.manufacturer, testCase.model)
			err = controller.ProfilesByManufacturerAndModel(c)
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
				var res responseDTO.MultiProfilesResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.expectedCount, len(res.Profiles), "Profile count not as expected")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestAllProfileBasicInfos(t *testing.T) {
	profile := dtos.ToProfileModel(buildTestProfileRequest().Profile)
	profiles := []models.Profile{profile, profile, profile}
	expectedTotalProfileCount := int64(3)
	expectedLinkedNodeCount := int64(1)

	dic := mockDic()
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("ProfileCountByLabels", []string(nil)).Return(expectedTotalProfileCount, nil)
	dbClientMock.On("ProfileCountByLabels", testProfileLabels).Return(expectedTotalProfileCount, nil)
	dbClientMock.On("AllProfiles", 0, 10, []string(nil)).Return(profiles, nil)
	dbClientMock.On("AllProfiles", 0, 5, testProfileLabels).Return([]models.Profile{profiles[0], profiles[1]}, nil)
	dbClientMock.On("AllProfiles", 1, 2, []string(nil)).Return([]models.Profile{profiles[1], profiles[2]}, nil)
	dbClientMock.On("AllProfiles", 4, 1, testProfileLabels).Return([]models.Profile{}, errors.NewCommonEdgeX(errors.KindRangeNotSatisfiable, "query objects bounds out of range.", nil))
	dbClientMock.On("NodeCountByProfileName", profile.Name).Return(expectedLinkedNodeCount, nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})
	controller := NewProfileController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name                    string
		offset                  string
		limit                   string
		labels                  string
		errorExpected           bool
		expectedCount           int
		expectedTotalCount      int64
		expectedLinkedNodeCount int64
		expectedStatusCode      int
	}{
		{"Valid - get node profile basic infos without labels", "0", "10", "", false, 3, expectedTotalProfileCount, expectedLinkedNodeCount, http.StatusOK},
		{"Valid - get node profile basic infos with labels", "0", "5", strings.Join(testProfileLabels, ","), false, 2, expectedTotalProfileCount, expectedLinkedNodeCount, http.StatusOK},
		{"Valid - get node profile basic infos with offset and no labels", "1", "2", "", false, 2, expectedTotalProfileCount, expectedLinkedNodeCount, http.StatusOK},
		{"Invalid - offset out of range", "4", "1", strings.Join(testProfileLabels, ","), true, 0, expectedTotalProfileCount, expectedLinkedNodeCount, http.StatusRequestedRangeNotSatisfiable},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiAllProfileBasicInfoRoute, http.NoBody)
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
			err = controller.AllProfileBasicInfos(c)
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
				var res responseDTO.MultiProfileBasicInfoResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.expectedCount, len(res.Profiles), "Profile count not as expected")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
				for _, profile := range res.Profiles {
					assert.Equal(t, testCase.expectedLinkedNodeCount, profile.LinkedNodeCount, "Linked node count not as expected")
				}
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestPatchProfileTags(t *testing.T) {
	profile := dtos.ToProfileModel(buildTestProfileRequest().Profile)
	notFoundName := "notFoundName"
	expectedRequestId := ExampleUUID
	updateTags := map[string]any{"TestTagKey": "NewTestTagValue", "TestTagKey2": "TestTagValue2"}
	testReq := requests.ProfileTagsRequest{
		BaseRequest: commonDTO.BaseRequest{
			Versionable: commonDTO.NewVersionable(),
			RequestId:   ExampleUUID,
		},
		UpdateProfileTags: dtos.UpdateProfileTags{
			Resources: []dtos.UpdateTags{{Name: TestResourceName, Tags: updateTags}},
			Commands:  []dtos.UpdateTags{{Name: TestCommandName, Tags: updateTags}},
		},
	}

	valid := testReq
	noRequestId := valid
	noRequestId.RequestId = ""

	noDRName := testReq
	noDRName.Resources = []dtos.UpdateTags{{Tags: updateTags}}
	emptyDRName := testReq
	emptyDRName.Resources = []dtos.UpdateTags{{Name: " ", Tags: updateTags}}
	noDRTags := testReq
	noDRTags.Resources = []dtos.UpdateTags{{Name: TestResourceName}}
	emptyDRTags := testReq
	emptyDRTags.Resources = []dtos.UpdateTags{{Name: TestCommandName, Tags: map[string]any{}}}

	noDCName := testReq
	noDCName.Commands = []dtos.UpdateTags{{Tags: updateTags}}
	emptyDCName := testReq
	emptyDCName.Commands = []dtos.UpdateTags{{Name: " ", Tags: updateTags}}
	noDCTags := testReq
	noDCTags.Commands = []dtos.UpdateTags{{Name: TestCommandName}}

	emptyDCTags := testReq
	emptyDCTags.Commands = []dtos.UpdateTags{{Name: TestCommandName, Tags: map[string]any{}}}

	dic := mockDic()
	dbClientMock := &mocks.DBClient{}
	dbClientMock.On("ProfileByName", profile.Name).Return(profile, nil)
	dbClientMock.On("ProfileByName", notFoundName).Return(profile, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "not found", nil))
	dbClientMock.On("NodesByProfileName", 0, -1, profile.Name).Return([]models.Node{{ServiceName: testNodeServiceName}}, nil)
	dbClientMock.On("NodeCountByProfileName", profile.Name).Return(int64(1), nil)
	dbClientMock.On("UpdateProfile", mock.Anything).Return(nil)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
	})

	controller := NewProfileController(dic)
	require.NotNil(t, controller)

	tests := []struct {
		name               string
		profileName        string
		request            requests.ProfileTagsRequest
		expectedStatusCode int
	}{
		{"valid", profile.Name, valid, http.StatusOK},
		{"valid - no request id", profile.Name, noRequestId, http.StatusOK},
		{"invalid - node profile not found", notFoundName, valid, http.StatusNotFound},
		{"invalid - no node resource name", profile.Name, noDRName, http.StatusBadRequest},
		{"invalid - empty node resource name", profile.Name, emptyDRName, http.StatusBadRequest},
		{"invalid - no node resource tags", profile.Name, noDRTags, http.StatusBadRequest},
		{"invalid - empty node resource tags", profile.Name, emptyDRTags, http.StatusBadRequest},
		{"invalid - no node command name", profile.Name, noDCName, http.StatusBadRequest},
		{"invalid - empty node command name", profile.Name, emptyDCName, http.StatusBadRequest},
		{"invalid - no node command tags", profile.Name, noDRTags, http.StatusBadRequest},
		{"invalid - empty node command tags", profile.Name, emptyDCTags, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			jsonData, err := json.Marshal(testCase.request)
			require.NoError(t, err)

			reader := strings.NewReader(string(jsonData))
			req, err := http.NewRequest(http.MethodPatch, common.ApiProfileTagsByNameRoute, reader)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Name)
			c.SetParamValues(testCase.profileName)
			err = controller.PatchProfileTags(c)
			require.NoError(t, err)

			var res commonDTO.BaseResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &res)
			require.NoError(t, err)

			assert.NotEmpty(t, recorder.Body.String(), "Message is empty")
			assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
			assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
			if res.RequestId != "" {
				assert.Equal(t, expectedRequestId, res.RequestId, "RequestID not as expected")
			}
		})
	}
}
