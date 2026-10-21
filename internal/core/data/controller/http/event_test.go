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

	"github.com/fxamacker/cbor/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	commonDTO "github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/requests"
	responseDTO "github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/responses"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"

	"github.com/clint456/edgex-go/internal/core/data/application"
	"github.com/clint456/edgex-go/internal/core/data/config"
	"github.com/clint456/edgex-go/internal/core/data/container"
	dbMock "github.com/clint456/edgex-go/internal/core/data/infrastructure/interfaces/mocks"
	"github.com/clint456/edgex-go/internal/core/data/mocks"

	"github.com/labstack/echo/v4"
)

var expectedEventId = uuid.New().String()

var testReading = dtos.BaseReading{
	NodeName:     TestNodeName,
	ResourceName: TestResourceName,
	ProfileName:  TestProfileName,
	Origin:       TestOriginTime,
	ValueType:    common.ValueTypeUint8,
	SimpleReading: dtos.SimpleReading{
		Value: "45",
	},
}

var testAddEvent = requests.AddEventRequest{
	BaseRequest: commonDTO.BaseRequest{
		RequestId:   ExampleUUID,
		Versionable: commonDTO.NewVersionable(),
	},
	Event: dtos.Event{
		Versionable: commonDTO.NewVersionable(),
		Id:          expectedEventId,
		NodeName:    TestNodeName,
		ProfileName: TestProfileName,
		SourceName:  TestSourceName,
		Origin:      TestOriginTime,
		Readings:    []dtos.BaseReading{testReading},
	},
}

var persistedReading = models.SimpleReading{
	BaseReading: models.BaseReading{
		Id:           ExampleUUID,
		Origin:       TestOriginTime,
		NodeName:     TestNodeName,
		ResourceName: TestResourceName,
		ProfileName:  TestProfileName,
		ValueType:    common.ValueTypeUint8,
	},
	Value: TestReadingValue,
}

var persistedEvent = models.Event{
	Id:          expectedEventId,
	NodeName:    TestNodeName,
	ProfileName: TestProfileName,
	Origin:      TestOriginTime,
	Readings:    []models.Reading{persistedReading},
}

func toByteArray(contentType string, v interface{}) ([]byte, error) {
	switch strings.ToLower(contentType) {
	case common.ContentTypeCBOR:
		return cbor.Marshal(v)
	default:
		return json.Marshal(v)
	}
}

func TestAddEvent(t *testing.T) {
	expectedRequestId := "82eb2e26-0f24-48aa-ae4c-de9dac3fb9bc"

	dbClientMock := &dbMock.DBClient{}

	dic := mocks.NewMockDIC()
	app := application.NewCoreDataApp(dic)
	dic.Update(di.ServiceConstructorMap{
		container.ConfigurationName: func(get di.Get) interface{} {
			return &config.ConfigurationStruct{
				Writable: config.WritableInfo{
					PersistData: false,
				},
			}
		},
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
		application.CoreDataAppName: func(get di.Get) interface{} {
			return app
		},
	})
	ec := NewEventController(dic)

	validRequest := testAddEvent

	noRequestId := validRequest
	noRequestId.RequestId = ""
	badRequestId := validRequest
	badRequestId.RequestId = "niv3sl"
	noEvent := validRequest
	noEvent.Event = dtos.Event{}
	noEventID := validRequest
	noEventID.Event.Id = ""
	badEventID := validRequest
	badEventID.Event.Id = "DIWNI09320"
	noEventNode := validRequest
	noEventNode.Event.NodeName = ""
	noEventProfile := validRequest
	noEventProfile.Event.ProfileName = ""
	noEventOrigin := validRequest
	noEventOrigin.Event.Origin = 0
	noEventSourceName := validRequest
	noEventSourceName.Event.SourceName = ""

	noReading := validRequest
	noReading.Event.Readings = []dtos.BaseReading{}
	noReadingNode := validRequest
	noReadingNode.Event.Readings = []dtos.BaseReading{testReading}
	noReadingNode.Event.Readings[0].NodeName = ""
	noReadingResourceName := validRequest
	noReadingResourceName.Event.Readings = []dtos.BaseReading{testReading}
	noReadingResourceName.Event.Readings[0].ResourceName = ""
	noReadingProfileName := validRequest
	noReadingProfileName.Event.Readings = []dtos.BaseReading{testReading}
	noReadingProfileName.Event.Readings[0].ProfileName = ""
	noReadingOrigin := validRequest
	noReadingOrigin.Event.Readings = []dtos.BaseReading{testReading}
	noReadingOrigin.Event.Readings[0].Origin = 0
	noReadingValueType := validRequest
	noReadingValueType.Event.Readings = []dtos.BaseReading{testReading}
	noReadingValueType.Event.Readings[0].ValueType = ""
	invalidReadingInvalidValueType := validRequest
	invalidReadingInvalidValueType.Event.Readings = []dtos.BaseReading{testReading}
	invalidReadingInvalidValueType.Event.Readings[0].ValueType = "BadType"

	noSimpleValue := validRequest
	noSimpleValue.Event.Readings = []dtos.BaseReading{testReading}
	noSimpleValue.Event.Readings[0].Value = ""
	noBinaryValue := validRequest
	noBinaryValue.Event.Readings = []dtos.BaseReading{{
		NodeName:     TestNodeName,
		ResourceName: TestResourceName,
		ProfileName:  TestProfileName,
		Origin:       TestOriginTime,
		ValueType:    common.ValueTypeBinary,
		BinaryReading: dtos.BinaryReading{
			BinaryValue: []byte{},
			MediaType:   TestBinaryReadingMediaType,
		},
	}}
	noBinaryMediaType := validRequest
	noBinaryMediaType.Event.Readings = []dtos.BaseReading{{
		NodeName:     TestNodeName,
		ResourceName: TestResourceName,
		ProfileName:  TestProfileName,
		Origin:       TestOriginTime,
		ValueType:    common.ValueTypeBinary,
		BinaryReading: dtos.BinaryReading{
			BinaryValue: []byte(TestReadingBinaryValue),
			MediaType:   "",
		},
	}}

	tests := []struct {
		Name               string
		Request            requests.AddEventRequest
		RequestContentType string
		ProfileName        string
		NodeName           string
		ErrorExpected      bool
		ExpectedStatusCode int
	}{
		{"Valid - AddEventRequest JSON", validRequest, common.ContentTypeJSON, validRequest.Event.ProfileName, validRequest.Event.NodeName, false, http.StatusCreated},
		{"Valid - No RequestId JSON", noRequestId, common.ContentTypeJSON, noRequestId.Event.ProfileName, noRequestId.Event.NodeName, false, http.StatusCreated},
		{"Invalid - Bad RequestId JSON", badRequestId, common.ContentTypeJSON, badRequestId.Event.ProfileName, badRequestId.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Event JSON", noEvent, common.ContentTypeJSON, "", "", true, http.StatusBadRequest},
		{"Invalid - No Event Id JSON", noEventID, common.ContentTypeJSON, noEventID.Event.ProfileName, noEventID.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - Bad Event Id JSON", badEventID, common.ContentTypeJSON, badEventID.Event.ProfileName, badEventID.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Event NodeName JSON", noEventNode, common.ContentTypeJSON, noEventNode.Event.ProfileName, noEventNode.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Event ProfileName JSON", noEventProfile, common.ContentTypeJSON, noEventProfile.Event.ProfileName, noEventProfile.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Event SourceName JSON", noEventSourceName, common.ContentTypeJSON, noEventSourceName.Event.ProfileName, noEventProfile.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Event Origin JSON", noEventOrigin, common.ContentTypeJSON, noEventOrigin.Event.ProfileName, noEventOrigin.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Reading JSON", noReading, common.ContentTypeJSON, noReading.Event.ProfileName, noReading.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Reading NodeName JSON", noReadingNode, common.ContentTypeJSON, noReadingNode.Event.ProfileName, noReadingNode.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Reading ResourceName JSON", noReadingResourceName, common.ContentTypeJSON, noReadingResourceName.Event.ProfileName, noReadingResourceName.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Reading ProfileName JSON", noReadingProfileName, common.ContentTypeJSON, noReadingProfileName.Event.ProfileName, noReadingProfileName.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Reading Origin JSON", noReadingOrigin, common.ContentTypeJSON, noReadingOrigin.Event.ProfileName, noReadingOrigin.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Reading ValueType JSON", noReadingValueType, common.ContentTypeJSON, noReadingValueType.Event.ProfileName, noReadingValueType.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - Invalid Reading ValueType JSON", invalidReadingInvalidValueType, common.ContentTypeJSON, invalidReadingInvalidValueType.Event.ProfileName, invalidReadingInvalidValueType.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No SimpleReading Value JSON", noSimpleValue, common.ContentTypeJSON, noSimpleValue.Event.ProfileName, noSimpleValue.Event.NodeName, true, http.StatusBadRequest},
		{"Valid - No BinaryReading BinaryValue JSON", noBinaryValue, common.ContentTypeJSON, noBinaryValue.Event.ProfileName, noBinaryValue.Event.NodeName, false, http.StatusCreated},
		{"Invalid - No BinaryReading MediaType JSON", noBinaryMediaType, common.ContentTypeJSON, noBinaryMediaType.Event.ProfileName, noBinaryMediaType.Event.NodeName, true, http.StatusBadRequest},
		{"Valid - AddEventRequest CBOR", validRequest, common.ContentTypeCBOR, validRequest.Event.ProfileName, validRequest.Event.NodeName, false, http.StatusCreated},
		{"Valid - No RequestId CBOR", noRequestId, common.ContentTypeCBOR, noRequestId.Event.ProfileName, noRequestId.Event.NodeName, false, http.StatusCreated},
		{"Invalid - Bad RequestId CBOR", badRequestId, common.ContentTypeCBOR, badRequestId.Event.ProfileName, badRequestId.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Event CBOR", noEvent, common.ContentTypeCBOR, "", "", true, http.StatusBadRequest},
		{"Invalid - No Event Id CBOR", noEventID, common.ContentTypeCBOR, noEventID.Event.ProfileName, noEventID.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - Bad Event Id CBOR", badEventID, common.ContentTypeCBOR, badEventID.Event.ProfileName, badEventID.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Event NodeName CBOR", noEventNode, common.ContentTypeCBOR, noEventNode.Event.ProfileName, noEventNode.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Event ProfileName CBOR", noEventProfile, common.ContentTypeCBOR, noEventProfile.Event.ProfileName, noEventProfile.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Event SourceName CBOR", noEventSourceName, common.ContentTypeCBOR, noEventSourceName.Event.ProfileName, noEventProfile.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Event Origin CBOR", noEventOrigin, common.ContentTypeCBOR, noEventOrigin.Event.ProfileName, noEventOrigin.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Reading CBOR", noReading, common.ContentTypeCBOR, noReading.Event.ProfileName, noReading.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Reading NodeName CBOR", noReadingNode, common.ContentTypeCBOR, noReadingNode.Event.ProfileName, noReadingNode.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Reading ResourceName CBOR", noReadingResourceName, common.ContentTypeCBOR, noReadingResourceName.Event.ProfileName, noReadingResourceName.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Reading ProfileName CBOR", noReadingProfileName, common.ContentTypeCBOR, noReadingProfileName.Event.ProfileName, noReadingProfileName.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Reading Origin CBOR", noReadingOrigin, common.ContentTypeCBOR, noReadingOrigin.Event.ProfileName, noReadingOrigin.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No Reading ValueType CBOR", noReadingValueType, common.ContentTypeCBOR, noReadingValueType.Event.ProfileName, noReadingValueType.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - Invalid Reading ValueType CBOR", invalidReadingInvalidValueType, common.ContentTypeCBOR, invalidReadingInvalidValueType.Event.ProfileName, invalidReadingInvalidValueType.Event.NodeName, true, http.StatusBadRequest},
		{"Invalid - No SimpleReading Value CBOR", noSimpleValue, common.ContentTypeCBOR, noSimpleValue.Event.ProfileName, noSimpleValue.Event.NodeName, true, http.StatusBadRequest},
		{"Valid - No BinaryReading BinaryValue CBOR", noBinaryValue, common.ContentTypeCBOR, noBinaryValue.Event.ProfileName, noBinaryValue.Event.NodeName, false, http.StatusCreated},
		{"Invalid - No BinaryReading MediaType CBOR", noBinaryMediaType, common.ContentTypeCBOR, noBinaryMediaType.Event.ProfileName, noBinaryMediaType.Event.NodeName, true, http.StatusBadRequest},
	}

	for _, testCase := range tests {
		t.Run(testCase.Name, func(t *testing.T) {
			e := echo.New()
			byteData, err := toByteArray(testCase.RequestContentType, testCase.Request)
			require.NoError(t, err)

			reader := strings.NewReader(string(byteData))
			req, err := http.NewRequest(http.MethodPost, common.ApiEventServiceNameProfileNameNodeNameSourceNameRoute, reader)
			req.Header.Set(common.ContentType, testCase.RequestContentType)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.ServiceName, common.ProfileName, common.NodeName, common.SourceName)
			c.SetParamValues(TestServiceName, testCase.ProfileName, testCase.NodeName, testCase.Request.Event.SourceName)
			err = ec.AddEvent(c)
			require.NoError(t, err)

			var actualResponse commonDTO.BaseWithIdResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &actualResponse)

			if testCase.ErrorExpected {
				assert.Equal(t, testCase.ExpectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				return // Test complete for error cases
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.ExpectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
			assert.Equal(t, common.ApiVersion, actualResponse.ApiVersion, "API Version not as expected")
			assert.Equal(t, testCase.ExpectedStatusCode, int(actualResponse.StatusCode), "BaseResponse status code not as expected")
			if actualResponse.RequestId != "" {
				assert.Equal(t, expectedRequestId, actualResponse.RequestId, "RequestID not as expected")
			}
			assert.Empty(t, actualResponse.Message, "Message should be empty when it is successful")
		})
	}
}

func TestAddEventSize(t *testing.T) {

	dbClientMock := &dbMock.DBClient{}

	validRequest := testAddEvent
	TestReadingLargeBinaryValue := make([]byte, 26000000)
	largeBinaryRequest := validRequest
	largeBinaryRequest.Event.Readings = []dtos.BaseReading{{
		NodeName:     TestNodeName,
		ResourceName: TestResourceName,
		ProfileName:  TestProfileName,
		Origin:       TestOriginTime,
		ValueType:    common.ValueTypeBinary,
		BinaryReading: dtos.BinaryReading{
			BinaryValue: []byte(TestReadingLargeBinaryValue),
			MediaType:   TestBinaryReadingMediaType,
		},
	}}

	tests := []struct {
		Name               string
		Request            requests.AddEventRequest
		MaxEventSize       int64
		RequestContentType string
		ErrorExpected      bool
		ExpectedStatusCode int
	}{
		{"Valid - AddEventRequest CBOR with default MaxEventSize", validRequest, 25000, common.ContentTypeCBOR, false, http.StatusCreated},
		{"Valid - AddEventRequest CBOR with unlimit MaxEventSize", validRequest, 0, common.ContentTypeCBOR, false, http.StatusCreated},
		{"Valid - AddEventRequest CBOR with higher MaxEventSize", largeBinaryRequest, 50000, common.ContentTypeCBOR, false, http.StatusCreated},
		{"Invalid - AddEventRequest CBOR with invalid event size", largeBinaryRequest, 25000, common.ContentTypeCBOR, true, http.StatusRequestEntityTooLarge},
	}

	for _, testCase := range tests {
		t.Run(testCase.Name, func(t *testing.T) {
			e := echo.New()
			dic := mocks.NewMockDIC()
			app := application.NewCoreDataApp(dic)
			dic.Update(di.ServiceConstructorMap{
				container.ConfigurationName: func(get di.Get) interface{} {
					return &config.ConfigurationStruct{
						MaxEventSize: testCase.MaxEventSize,
						Writable: config.WritableInfo{
							PersistData: false,
						},
					}
				},
				container.DBClientInterfaceName: func(get di.Get) interface{} {
					return dbClientMock
				},
				application.CoreDataAppName: func(get di.Get) interface{} {
					return app
				},
			})
			ec := NewEventController(dic)
			byteData, err := toByteArray(testCase.RequestContentType, testCase.Request)
			require.NoError(t, err)

			reader := strings.NewReader(string(byteData))
			req, err := http.NewRequest(http.MethodPost, common.ApiEventServiceNameProfileNameNodeNameSourceNameRoute, reader)
			req.Header.Set(common.ContentType, testCase.RequestContentType)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.ServiceName, common.ProfileName, common.NodeName, common.SourceName)
			c.SetParamValues(TestServiceName, validRequest.Event.ProfileName, validRequest.Event.NodeName, validRequest.Event.SourceName)
			err = ec.AddEvent(c)
			require.NoError(t, err)

			var actualResponse commonDTO.BaseWithIdResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &actualResponse)

			if testCase.ErrorExpected {
				assert.Equal(t, testCase.ExpectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, common.ApiVersion, actualResponse.ApiVersion, "API Version not as expected")
			assert.Equal(t, testCase.ExpectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
			assert.Equal(t, testCase.ExpectedStatusCode, int(actualResponse.StatusCode), "BaseResponse status code not as expected")
			assert.Empty(t, actualResponse.Message, "Message should be empty when it is successful")
		})
	}
}

func TestEventById(t *testing.T) {
	validEventId := expectedEventId
	emptyEventId := ""
	invalidEventId := "bad"
	notFoundEventId := NonexistentEventID

	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("EventById", notFoundEventId).Return(models.Event{}, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "event doesn't exist in the database", nil))
	dbClientMock.On("EventById", validEventId).Return(persistedEvent, nil)

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
	ec := NewEventController(dic)

	tests := []struct {
		Name               string
		EventId            string
		ErrorExpected      bool
		ExpectedStatusCode int
	}{
		{"Valid - Find Event by Id", validEventId, false, http.StatusOK},
		{"Invalid - Empty EventId", emptyEventId, true, http.StatusBadRequest},
		{"Invalid - EventId is not an UUID", invalidEventId, true, http.StatusBadRequest},
		{"Invalid - Event doesn't exist", notFoundEventId, true, http.StatusNotFound},
	}

	for _, testCase := range tests {
		t.Run(testCase.Name, func(t *testing.T) {
			e := echo.New()
			reqPath := fmt.Sprintf("%s/%s/%s", common.ApiEventRoute, common.Id, testCase.EventId)
			req, err := http.NewRequest(http.MethodGet, reqPath, http.NoBody)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Id)
			c.SetParamValues(testCase.EventId)
			err = ec.EventById(c)
			require.NoError(t, err)

			if testCase.ErrorExpected {
				var actualResponse commonDTO.BaseResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &actualResponse)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, actualResponse.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.ExpectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.ExpectedStatusCode, int(actualResponse.StatusCode), "Response status code not as expected")
				assert.NotEmpty(t, actualResponse.Message, "Response message doesn't contain the error message")
			} else {
				var actualResponse responseDTO.EventResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &actualResponse)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, actualResponse.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.ExpectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.ExpectedStatusCode, int(actualResponse.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.EventId, actualResponse.Event.Id, "Event Id not as expected")
				assert.Empty(t, actualResponse.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestDeleteEventById(t *testing.T) {
	validEventId := expectedEventId
	emptyEventId := ""
	invalidEventId := "bad"
	notFoundEventId := NonexistentEventID

	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("EventById", notFoundEventId).Return(models.Event{}, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "event doesn't exist in the database", nil))
	dbClientMock.On("EventById", validEventId).Return(persistedEvent, nil)
	dbClientMock.On("DeleteEventById", validEventId).Return(nil)

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
	ec := NewEventController(dic)

	tests := []struct {
		Name               string
		EventId            string
		ErrorExpected      bool
		ExpectedStatusCode int
	}{
		{"Valid - Delete Event by Id", validEventId, false, http.StatusOK},
		{"Invalid - Empty EventId", emptyEventId, true, http.StatusBadRequest},
		{"Invalid - EventId is not an UUID", invalidEventId, true, http.StatusBadRequest},
		{"Invalid - Event doesn't exist", notFoundEventId, true, http.StatusNotFound},
	}

	for _, testCase := range tests {
		t.Run(testCase.Name, func(t *testing.T) {
			e := echo.New()
			reqPath := fmt.Sprintf("%s/%s/%s", common.ApiEventRoute, common.Id, testCase.EventId)
			req, err := http.NewRequest(http.MethodDelete, reqPath, http.NoBody)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Id)
			c.SetParamValues(testCase.EventId)
			err = ec.DeleteEventById(c)
			require.NoError(t, err)

			var actualResponse commonDTO.BaseResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &actualResponse)
			require.NoError(t, err)
			assert.Equal(t, common.ApiVersion, actualResponse.ApiVersion, "API Version not as expected")
			assert.Equal(t, testCase.ExpectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
			assert.Equal(t, testCase.ExpectedStatusCode, int(actualResponse.StatusCode), "Response status code not as expected")
			if testCase.ErrorExpected {
				assert.NotEmpty(t, actualResponse.Message, "Response message doesn't contain the error message")
			} else {
				assert.Empty(t, actualResponse.Message, "Response message should be empty when it is successful")
			}
		})
	}
}

func TestEventTotalCount(t *testing.T) {
	expectedEventCount := int64(656672)
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("EventTotalCount").Return(expectedEventCount, nil)

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
	ec := NewEventController(dic)

	e := echo.New()
	req, err := http.NewRequest(http.MethodGet, common.ApiEventCountRoute, http.NoBody)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	c := e.NewContext(req, recorder)
	err = ec.EventTotalCount(c)
	require.NoError(t, err)

	var actualResponse commonDTO.CountResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &actualResponse)
	require.NoError(t, err)
	assert.Equal(t, common.ApiVersion, actualResponse.ApiVersion, "API Version not as expected")
	assert.Equal(t, http.StatusOK, recorder.Result().StatusCode, "HTTP status code not as expected")
	assert.Equal(t, http.StatusOK, int(actualResponse.StatusCode), "Response status code not as expected")
	assert.Empty(t, actualResponse.Message, "Message should be empty when it is successful")
	assert.Equal(t, expectedEventCount, actualResponse.Count, "Event count in the response body is not expected")
}

func TestEventCountByNodeName(t *testing.T) {
	expectedEventCount := int64(656672)
	nodeName := "nodeA"
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("EventCountByNodeName", nodeName).Return(expectedEventCount, nil)

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
	ec := NewEventController(dic)

	e := echo.New()
	req, err := http.NewRequest(http.MethodGet, common.ApiEventCountByNodeNameRoute, http.NoBody)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	c := e.NewContext(req, recorder)
	c.SetParamNames(common.Name)
	c.SetParamValues(nodeName)
	err = ec.EventCountByNodeName(c)
	require.NoError(t, err)

	var actualResponse commonDTO.CountResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &actualResponse)
	require.NoError(t, err)
	assert.Equal(t, common.ApiVersion, actualResponse.ApiVersion, "API Version not as expected")
	assert.Equal(t, http.StatusOK, recorder.Result().StatusCode, "HTTP status code not as expected")
	assert.Equal(t, http.StatusOK, int(actualResponse.StatusCode), "Response status code not as expected")
	assert.Empty(t, actualResponse.Message, "Message should be empty when it is successful")
	assert.Equal(t, expectedEventCount, actualResponse.Count, "Event count in the response body is not expected")
}

func TestDeleteEventsByNodeName(t *testing.T) {
	nodeName := "nodeA"
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("DeleteEventsByNodeName", nodeName).Return(nil)

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
	ec := NewEventController(dic)

	e := echo.New()
	req, err := http.NewRequest(http.MethodDelete, common.ApiEventByNodeNameRoute, http.NoBody)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	c := e.NewContext(req, recorder)
	c.SetParamNames(common.Name)
	c.SetParamValues(nodeName)
	err = ec.DeleteEventsByNodeName(c)
	require.NoError(t, err)

	var actualResponse commonDTO.BaseResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &actualResponse)
	assert.NoError(t, err)

	assert.Equal(t, common.ApiVersion, actualResponse.ApiVersion, "API Version not as expected")
	assert.Equal(t, http.StatusAccepted, recorder.Result().StatusCode, "HTTP status code not as expected")
	assert.Empty(t, actualResponse.Message, "Message should be empty when it is successful")
}

func TestAllEvents(t *testing.T) {
	events := []models.Event{persistedEvent, persistedEvent, persistedEvent}
	totalCount := int64(len(events))

	dic := mocks.NewMockDIC()
	dbClientMock := &dbMock.DBClient{}

	dbClientMock.On("EventTotalCount").Return(totalCount, nil)
	dbClientMock.On("AllEvents", 0, 20).Return(events, nil)
	dbClientMock.On("AllEvents", 1, 1).Return([]models.Event{events[1]}, nil)
	dbClientMock.On("AllEvents", -1, 1).Return([]models.Event{events[1]}, nil)
	dbClientMock.On("AllEvents", 4, 1).Return([]models.Event{}, errors.NewCommonEdgeX(errors.KindRangeNotSatisfiable, "query objects bounds out of range.", nil))
	app := application.NewCoreDataApp(dic)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
		application.CoreDataAppName: func(get di.Get) interface{} {
			return app
		},
	})
	controller := NewEventController(dic)
	assert.NotNil(t, controller)

	tests := []struct {
		name               string
		offset             string
		limit              string
		errorExpected      bool
		expectedCount      int
		expectedTotalCount int64
		expectedStatusCode int
	}{
		{"Valid - get events without offset and limit", "", "", false, 3, totalCount, http.StatusOK},
		{"Valid - get events with offset and limit", "1", "1", false, 1, totalCount, http.StatusOK},
		{"Valid - set offset -1 to skip total count", "-1", "1", false, 1, 0, http.StatusOK},
		{"Invalid - offset out of range", "4", "1", true, 0, 0, http.StatusRequestedRangeNotSatisfiable},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiAllEventRoute, http.NoBody)
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
			c := e.NewContext(req, recorder)
			err = controller.AllEvents(c)
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
				var res responseDTO.MultiEventsResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, res.StatusCode, "Response status code not as expected")
				assert.Equal(t, testCase.expectedCount, len(res.Events), "Event count not as expected")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestAllEventsByNodeName(t *testing.T) {
	testNodeA := "testNodeA"
	testNodeB := "testNodeB"
	event1WithNodeA := persistedEvent
	event1WithNodeA.NodeName = testNodeA
	event2WithNodeA := persistedEvent
	event2WithNodeA.NodeName = testNodeA
	event3WithNodeB := persistedEvent
	event3WithNodeB.NodeName = testNodeB

	events := []models.Event{event1WithNodeA, event2WithNodeA, event3WithNodeB}
	totalCountNodeA := int64(2)
	totalCountNodeB := int64(1)

	dic := mocks.NewMockDIC()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("EventCountByNodeName", testNodeA).Return(totalCountNodeA, nil)
	dbClientMock.On("EventCountByNodeName", testNodeB).Return(totalCountNodeB, nil)
	dbClientMock.On("EventsByNodeName", 0, 5, testNodeA).Return([]models.Event{events[0], events[1]}, nil)
	dbClientMock.On("EventsByNodeName", 0, 5, testNodeB).Return([]models.Event{events[2]}, nil)
	dbClientMock.On("EventsByNodeName", 1, 1, testNodeA).Return([]models.Event{events[1]}, nil)
	dbClientMock.On("EventsByNodeName", -1, 1, testNodeA).Return([]models.Event{events[1]}, nil)
	dbClientMock.On("EventsByNodeName", 4, 1, testNodeB).Return([]models.Event{}, errors.NewCommonEdgeX(errors.KindRangeNotSatisfiable, "query objects bounds out of range.", nil))
	app := application.NewCoreDataApp(dic)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
		application.CoreDataAppName: func(get di.Get) interface{} {
			return app
		},
	})
	ec := NewEventController(dic)
	assert.NotNil(t, ec)

	tests := []struct {
		name               string
		offset             string
		limit              string
		nodeName           string
		errorExpected      bool
		expectedCount      int
		expectedTotalCount int64
		expectedStatusCode int
	}{
		{"Valid - get events with nodeName - nodeA", "0", "5", testNodeA, false, 2, totalCountNodeA, http.StatusOK},
		{"Valid - get events with nodeName - nodeB", "0", "5", testNodeB, false, 1, totalCountNodeB, http.StatusOK},
		{"Valid - get events with offset and no labels", "1", "1", testNodeA, false, 1, totalCountNodeA, http.StatusOK},
		{"Valid - set offset -1 to skip total count", "-1", "1", testNodeA, false, 1, 0, http.StatusOK},
		{"Invalid - offset out of range", "4", "1", testNodeB, true, 0, 0, http.StatusRequestedRangeNotSatisfiable},
		{"Invalid - get events without nodeName", "0", "10", "", true, 0, 0, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiEventByNodeNameRoute, http.NoBody)
			query := req.URL.Query()
			query.Add(common.Offset, testCase.offset)
			query.Add(common.Limit, testCase.limit)
			req.URL.RawQuery = query.Encode()
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Name)
			c.SetParamValues(testCase.nodeName)
			err = ec.EventsByNodeName(c)
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
				var res responseDTO.MultiEventsResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.expectedCount, len(res.Events), "Node count not as expected")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestAllEventsByTimeRange(t *testing.T) {
	totalCount := int64(0)
	dic := mocks.NewMockDIC()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("EventCountByTimeRange", int64(0), int64(100)).Return(totalCount, nil)
	dbClientMock.On("EventsByTimeRange", int64(0), int64(100), 0, 10).Return([]models.Event{}, nil)
	dbClientMock.On("EventsByTimeRange", int64(0), int64(100), -1, 10).Return([]models.Event{}, nil)
	app := application.NewCoreDataApp(dic)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
		application.CoreDataAppName: func(get di.Get) interface{} {
			return app
		},
	})
	ec := NewEventController(dic)
	assert.NotNil(t, ec)

	tests := []struct {
		name               string
		start              string
		end                string
		offset             string
		limit              string
		errorExpected      bool
		expectedCount      int
		expectedTotalCount int64
		expectedStatusCode int
	}{
		{"Valid - with proper start/end/offset/limit", "0", "100", "0", "10", false, 0, totalCount, http.StatusOK},
		{"Valid - set offset -1 to skip total count", "0", "100", "-1", "10", false, 0, 0, http.StatusOK},
		{"Invalid - invalid start format", "aaa", "100", "0", "10", true, 0, totalCount, http.StatusBadRequest},
		{"Invalid - invalid end format", "0", "bbb", "0", "10", true, 0, totalCount, http.StatusBadRequest},
		{"Invalid - empty start", "", "100", "0", "10", true, 0, totalCount, http.StatusBadRequest},
		{"Invalid - empty end", "0", "", "0", "10", true, 0, totalCount, http.StatusBadRequest},
		{"Invalid - end before start", "10", "0", "0", "10", true, 0, totalCount, http.StatusBadRequest},
		{"Invalid - invalid offset format", "0", "100", "aaa", "10", true, 0, totalCount, http.StatusBadRequest},
		{"Invalid - invalid limit format", "0", "100", "0", "aaa", true, 0, totalCount, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiEventByTimeRangeRoute, http.NoBody)
			query := req.URL.Query()
			query.Add(common.Offset, testCase.offset)
			query.Add(common.Limit, testCase.limit)
			req.URL.RawQuery = query.Encode()
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Start, common.End)
			c.SetParamValues(testCase.start, testCase.end)
			err = ec.EventsByTimeRange(c)
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
				var res responseDTO.MultiEventsResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.expectedCount, len(res.Events), "Node count not as expected")
				assert.Equal(t, testCase.expectedTotalCount, res.TotalCount, "Total count not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}

func TestDeleteEventsByAge(t *testing.T) {
	dic := mocks.NewMockDIC()
	dbClientMock := &dbMock.DBClient{}
	dbClientMock.On("DeleteEventsByAge", int64(0)).Return(nil)
	app := application.NewCoreDataApp(dic)
	dic.Update(di.ServiceConstructorMap{
		container.DBClientInterfaceName: func(get di.Get) interface{} {
			return dbClientMock
		},
		application.CoreDataAppName: func(get di.Get) interface{} {
			return app
		},
	})
	ec := NewEventController(dic)
	assert.NotNil(t, ec)

	tests := []struct {
		name               string
		age                string
		errorExpected      bool
		expectedCount      int
		expectedStatusCode int
	}{
		{"Valid - age with proper format", "0", false, 0, http.StatusAccepted},
		{"Invalid - age with unparsable format", "aaa", true, 0, http.StatusBadRequest},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, common.ApiEventByTimeRangeRoute, http.NoBody)
			require.NoError(t, err)

			// Act
			recorder := httptest.NewRecorder()
			c := e.NewContext(req, recorder)
			c.SetParamNames(common.Age)
			c.SetParamValues(testCase.age)
			err = ec.DeleteEventsByAge(c)
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
				var res responseDTO.MultiEventsResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &res)
				require.NoError(t, err)
				assert.Equal(t, common.ApiVersion, res.ApiVersion, "API Version not as expected")
				assert.Equal(t, testCase.expectedStatusCode, recorder.Result().StatusCode, "HTTP status code not as expected")
				assert.Equal(t, testCase.expectedStatusCode, int(res.StatusCode), "Response status code not as expected")
				assert.Equal(t, testCase.expectedCount, len(res.Events), "Node count not as expected")
				assert.Empty(t, res.Message, "Message should be empty when it is successful")
			}
		})
	}
}
