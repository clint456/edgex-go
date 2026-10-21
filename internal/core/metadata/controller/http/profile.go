//
// Copyright (C) 2021-2024 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"math"
	"net/http"

	"github.com/clint456/edgex-go/internal/core/metadata/application"
	metadataContainer "github.com/clint456/edgex-go/internal/core/metadata/container"
	"github.com/clint456/edgex-go/internal/io"
	"github.com/clint456/edgex-go/internal/pkg"
	"github.com/clint456/edgex-go/internal/pkg/correlation"
	"github.com/clint456/edgex-go/internal/pkg/utils"

	"github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	commonDTO "github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/common"
	requestDTO "github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/requests"
	responseDTO "github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/responses"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"

	"github.com/labstack/echo/v4"
)

const yamlFileName = "file"

type ProfileController struct {
	jsonDtoReader io.DtoReader
	yamlDtoReader io.DtoReader
	dic           *di.Container
}

// NewProfileController creates and initializes an ProfileController
func NewProfileController(dic *di.Container) *ProfileController {
	return &ProfileController{
		jsonDtoReader: io.NewJsonDtoReader(),
		yamlDtoReader: io.NewYamlDtoReader(),
		dic:           dic,
	}
}

func (dc *ProfileController) AddProfile(c echo.Context) error {
	r := c.Request()
	w := c.Response()
	if r.Body != nil {
		defer func() { _ = r.Body.Close() }()
	}

	lc := container.LoggingClientFrom(dc.dic.Get)

	ctx := r.Context()
	correlationId := correlation.FromContext(ctx)

	var reqDTOs []requestDTO.ProfileRequest
	err := dc.jsonDtoReader.Read(r.Body, &reqDTOs)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}
	profiles := requestDTO.ProfileReqToProfileModels(reqDTOs)

	var addResponses []interface{}
	for i, d := range profiles {
		var addProfileResponse interface{}
		reqId := reqDTOs[i].RequestId
		newId, err := application.AddProfile(d, ctx, dc.dic)
		if err != nil {
			lc.Error(err.Error(), common.CorrelationHeader, correlationId)
			lc.Debug(err.DebugMessages(), common.CorrelationHeader, correlationId)
			addProfileResponse = commonDTO.NewBaseResponse(
				reqId,
				err.Message(),
				err.Code())
		} else {
			addProfileResponse = commonDTO.NewBaseWithIdResponse(
				reqId,
				"",
				http.StatusCreated,
				newId)
		}
		addResponses = append(addResponses, addProfileResponse)
	}

	utils.WriteHttpHeader(w, ctx, http.StatusMultiStatus)
	return pkg.EncodeAndWriteResponse(addResponses, w, lc)
}

func (dc *ProfileController) UpdateProfile(c echo.Context) error {
	r := c.Request()
	w := c.Response()
	if r.Body != nil {
		defer func() { _ = r.Body.Close() }()
	}

	lc := container.LoggingClientFrom(dc.dic.Get)

	ctx := r.Context()
	correlationId := correlation.FromContext(ctx)

	strictProfileChanges := metadataContainer.ConfigurationFrom(dc.dic.Get).Writable.ProfileChange.StrictProfileChanges
	if strictProfileChanges {
		return utils.WriteErrorResponse(w, ctx, lc, errors.NewCommonEdgeX(errors.KindServiceLocked, "profile change is not allowed when StrictProfileChanges config is enabled", nil), "")
	}

	var reqDTOs []requestDTO.ProfileRequest
	err := dc.jsonDtoReader.Read(r.Body, &reqDTOs)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}
	profiles := requestDTO.ProfileReqToProfileModels(reqDTOs)

	var responses []interface{}
	for i, d := range profiles {
		var response interface{}
		reqId := reqDTOs[i].RequestId
		err := application.UpdateProfile(d, ctx, dc.dic)
		if err != nil {
			lc.Error(err.Error(), common.CorrelationHeader, correlationId)
			lc.Debug(err.DebugMessages(), common.CorrelationHeader, correlationId)
			response = commonDTO.NewBaseResponse(
				reqId,
				err.Message(),
				err.Code())
		} else {
			response = commonDTO.NewBaseResponse(
				reqId,
				"",
				http.StatusOK)
		}
		responses = append(responses, response)
	}

	utils.WriteHttpHeader(w, ctx, http.StatusMultiStatus)
	return pkg.EncodeAndWriteResponse(responses, w, lc)
}

func (dc *ProfileController) AddProfileByYaml(c echo.Context) error {
	r := c.Request()
	w := c.Response()
	if r.Body != nil {
		defer func() { _ = r.Body.Close() }()
	}

	lc := container.LoggingClientFrom(dc.dic.Get)
	ctx := r.Context()

	file, _, fileErr := r.FormFile(yamlFileName)
	if fileErr == http.ErrMissingFile {
		return utils.WriteErrorResponse(w, ctx, lc, errors.NewCommonEdgeX(errors.KindContractInvalid, "missing yaml file", nil), "")
	} else if fileErr != nil {
		return utils.WriteErrorResponse(w, ctx, lc, errors.NewCommonEdgeX(errors.KindServerError, fileErr.Error(), nil), "")
	}

	var profileDTO dtos.Profile
	err := dc.yamlDtoReader.Read(file, &profileDTO)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}
	profile := dtos.ToProfileModel(profileDTO)

	newId, err := application.AddProfile(profile, ctx, dc.dic)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	response := commonDTO.NewBaseWithIdResponse("", "", http.StatusCreated, newId)
	utils.WriteHttpHeader(w, ctx, http.StatusCreated)
	// EncodeAndWriteResponse and send the resp body as JSON format
	return pkg.EncodeAndWriteResponse(response, w, lc)
}

func (dc *ProfileController) UpdateProfileByYaml(c echo.Context) error {
	r := c.Request()
	w := c.Response()
	if r.Body != nil {
		defer func() { _ = r.Body.Close() }()
	}

	lc := container.LoggingClientFrom(dc.dic.Get)
	ctx := r.Context()

	file, _, fileErr := r.FormFile(yamlFileName)
	if fileErr == http.ErrMissingFile {
		return utils.WriteErrorResponse(w, ctx, lc, errors.NewCommonEdgeX(errors.KindContractInvalid, "missing yaml file", nil), "")
	} else if fileErr != nil {
		return utils.WriteErrorResponse(w, ctx, lc, errors.NewCommonEdgeX(errors.KindServerError, fileErr.Error(), nil), "")
	}

	strictProfileChanges := metadataContainer.ConfigurationFrom(dc.dic.Get).Writable.ProfileChange.StrictProfileChanges
	if strictProfileChanges {
		return utils.WriteErrorResponse(w, ctx, lc, errors.NewCommonEdgeX(errors.KindServiceLocked, "profile change is not allowed when StrictProfileChanges config is enabled", nil), "")
	}

	var profileDTO dtos.Profile
	err := dc.yamlDtoReader.Read(file, &profileDTO)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	profile := dtos.ToProfileModel(profileDTO)
	err = application.UpdateProfile(profile, ctx, dc.dic)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	response := commonDTO.NewBaseResponse("", "", http.StatusOK)
	utils.WriteHttpHeader(w, ctx, http.StatusOK)
	return pkg.EncodeAndWriteResponse(response, w, lc)
}

func (dc *ProfileController) ProfileByName(c echo.Context) error {
	lc := container.LoggingClientFrom(dc.dic.Get)
	r := c.Request()
	w := c.Response()
	ctx := r.Context()

	// URL parameters
	name := c.Param(common.Name)

	profile, err := application.ProfileByName(name, ctx, dc.dic)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	response := responseDTO.NewProfileResponse("", "", http.StatusOK, profile)
	utils.WriteHttpHeader(w, ctx, http.StatusOK)
	return pkg.EncodeAndWriteResponse(response, w, lc) // encode and send out the response
}

func (dc *ProfileController) DeleteProfileByName(c echo.Context) error {
	lc := container.LoggingClientFrom(dc.dic.Get)
	r := c.Request()
	w := c.Response()
	ctx := r.Context()

	// URL parameters
	name := c.Param(common.Name)

	err := application.DeleteProfileByName(name, ctx, dc.dic)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	response := commonDTO.NewBaseResponse("", "", http.StatusOK)
	utils.WriteHttpHeader(w, ctx, http.StatusOK)
	return pkg.EncodeAndWriteResponse(response, w, lc)
}

func (dc *ProfileController) AllProfiles(c echo.Context) error {
	lc := container.LoggingClientFrom(dc.dic.Get)
	r := c.Request()
	w := c.Response()
	ctx := r.Context()
	config := metadataContainer.ConfigurationFrom(dc.dic.Get)

	// parse URL query string for offset, limit, and labels
	offset, limit, labels, err := utils.ParseGetAllObjectsRequestQueryString(c, 0, math.MaxInt32, -1, config.Service.MaxResultCount)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}
	profiles, totalCount, err := application.AllProfiles(offset, limit, labels, dc.dic)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	response := responseDTO.NewMultiProfilesResponse("", "", http.StatusOK, totalCount, profiles)
	utils.WriteHttpHeader(w, ctx, http.StatusOK)
	return pkg.EncodeAndWriteResponse(response, w, lc)
}

func (dc *ProfileController) ProfilesByModel(c echo.Context) error {
	lc := container.LoggingClientFrom(dc.dic.Get)
	r := c.Request()
	w := c.Response()
	ctx := r.Context()
	config := metadataContainer.ConfigurationFrom(dc.dic.Get)

	model := c.Param(common.Model)

	// parse URL query string for offset, limit
	offset, limit, _, err := utils.ParseGetAllObjectsRequestQueryString(c, 0, math.MaxInt32, -1, config.Service.MaxResultCount)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}
	profiles, totalCount, err := application.ProfilesByModel(offset, limit, model, dc.dic)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	response := responseDTO.NewMultiProfilesResponse("", "", http.StatusOK, totalCount, profiles)
	utils.WriteHttpHeader(w, ctx, http.StatusOK)
	return pkg.EncodeAndWriteResponse(response, w, lc)
}

func (dc *ProfileController) ProfilesByManufacturer(c echo.Context) error {
	lc := container.LoggingClientFrom(dc.dic.Get)
	r := c.Request()
	w := c.Response()
	ctx := r.Context()
	config := metadataContainer.ConfigurationFrom(dc.dic.Get)

	manufacturer := c.Param(common.Manufacturer)

	// parse URL query string for offset, limit
	offset, limit, _, err := utils.ParseGetAllObjectsRequestQueryString(c, 0, math.MaxInt32, -1, config.Service.MaxResultCount)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}
	profiles, totalCount, err := application.ProfilesByManufacturer(offset, limit, manufacturer, dc.dic)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	response := responseDTO.NewMultiProfilesResponse("", "", http.StatusOK, totalCount, profiles)
	utils.WriteHttpHeader(w, ctx, http.StatusOK)
	return pkg.EncodeAndWriteResponse(response, w, lc)
}

func (dc *ProfileController) ProfilesByManufacturerAndModel(c echo.Context) error {
	lc := container.LoggingClientFrom(dc.dic.Get)
	r := c.Request()
	w := c.Response()
	ctx := r.Context()
	config := metadataContainer.ConfigurationFrom(dc.dic.Get)

	manufacturer := c.Param(common.Manufacturer)
	model := c.Param(common.Model)

	// parse URL query string for offset, limit
	offset, limit, _, err := utils.ParseGetAllObjectsRequestQueryString(c, 0, math.MaxInt32, -1, config.Service.MaxResultCount)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}
	profiles, totalCount, err := application.ProfilesByManufacturerAndModel(offset, limit, manufacturer, model, dc.dic)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	response := responseDTO.NewMultiProfilesResponse("", "", http.StatusOK, totalCount, profiles)
	utils.WriteHttpHeader(w, ctx, http.StatusOK)
	return pkg.EncodeAndWriteResponse(response, w, lc)
}

func (dc *ProfileController) PatchProfileBasicInfo(c echo.Context) error {
	r := c.Request()
	w := c.Response()
	if r.Body != nil {
		defer func() { _ = r.Body.Close() }()
	}

	ctx := r.Context()
	correlationId := correlation.FromContext(ctx)
	lc := container.LoggingClientFrom(dc.dic.Get)

	var reqDTOs []requestDTO.ProfileBasicInfoRequest
	err := dc.jsonDtoReader.Read(r.Body, &reqDTOs)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	var updateResponses []interface{}
	for _, dto := range reqDTOs {
		var response interface{}
		reqId := dto.RequestId
		err := application.PatchProfileBasicInfo(ctx, dto.BasicInfo, dc.dic)
		if err != nil {
			lc.Error(err.Error(), common.CorrelationHeader, correlationId)
			lc.Debug(err.DebugMessages(), common.CorrelationHeader, correlationId)
			response = commonDTO.NewBaseResponse(reqId, err.Message(), err.Code())
		} else {
			response = commonDTO.NewBaseResponse(reqId, "", http.StatusOK)
		}
		updateResponses = append(updateResponses, response)
	}

	utils.WriteHttpHeader(w, ctx, http.StatusMultiStatus)
	return pkg.EncodeAndWriteResponse(updateResponses, w, lc)
}

func (dc *ProfileController) AllProfileBasicInfos(c echo.Context) error {
	lc := container.LoggingClientFrom(dc.dic.Get)
	r := c.Request()
	w := c.Response()
	ctx := r.Context()
	config := metadataContainer.ConfigurationFrom(dc.dic.Get)

	// parse URL query string for offset, limit, and labels
	offset, limit, labels, err := utils.ParseGetAllObjectsRequestQueryString(c, 0, math.MaxInt32, -1, config.Service.MaxResultCount)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}
	profileBasicInfos, totalCount, err := application.AllProfileBasicInfos(offset, limit, labels, dc.dic)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	response := responseDTO.NewMultiProfileBasicInfosResponse("", "", http.StatusOK, totalCount, profileBasicInfos)
	utils.WriteHttpHeader(w, ctx, http.StatusOK)
	return pkg.EncodeAndWriteResponse(response, w, lc)
}

func (dc *ProfileController) PatchProfileTags(c echo.Context) error {
	r := c.Request()
	w := c.Response()
	if r.Body != nil {
		defer func() { _ = r.Body.Close() }()
	}

	lc := container.LoggingClientFrom(dc.dic.Get)

	ctx := r.Context()

	// URL parameters
	name := c.Param(common.Name)

	var reqDTO requestDTO.ProfileTagsRequest
	err := dc.jsonDtoReader.Read(r.Body, &reqDTO)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	reqId := reqDTO.RequestId
	err = application.PatchProfileTags(name, reqDTO.UpdateProfileTags, ctx, dc.dic)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, reqId)
	}

	response := commonDTO.NewBaseResponse(reqId, "", http.StatusOK)
	utils.WriteHttpHeader(w, ctx, http.StatusOK)
	return pkg.EncodeAndWriteResponse(response, w, lc)
}
