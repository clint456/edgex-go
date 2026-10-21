//
// Copyright (C) 2020-2024 IOTech Ltd
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
	commonDTO "github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/common"
	requestDTO "github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/requests"
	responseDTO "github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/responses"

	"github.com/labstack/echo/v4"
)

const (
	bypassValidationQueryParam = "bypassValidation" // query param to specify whether to skip the Node Service Validation API call
	forceQueryParam            = "force"            // query param to specify whether to force add a node
)

type NodeController struct {
	reader io.DtoReader
	dic    *di.Container
}

// NewNodeController creates and initializes an NodeController
func NewNodeController(dic *di.Container) *NodeController {
	return &NodeController{
		reader: io.NewJsonDtoReader(),
		dic:    dic,
	}
}

func (dc *NodeController) AddNode(c echo.Context) error {
	r := c.Request()
	w := c.Response()
	if r.Body != nil {
		defer func() { _ = r.Body.Close() }()
	}

	lc := container.LoggingClientFrom(dc.dic.Get)

	ctx := r.Context()
	correlationId := correlation.FromContext(ctx)

	var bypassValidation, force bool
	// parse URL query string for bypassValidation
	bypassValidationParamStr := utils.ParseQueryStringToString(r, bypassValidationQueryParam, common.ValueFalse)
	if bypassValidationParamStr == common.ValueTrue {
		bypassValidation = true
	}
	// parse URL query string for force add node
	forceParamStr := utils.ParseQueryStringToString(r, forceQueryParam, common.ValueFalse)
	if forceParamStr == common.ValueTrue {
		force = true
	}

	var reqDTOs []requestDTO.AddNodeRequest
	err := dc.reader.Read(r.Body, &reqDTOs)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}
	nodes := requestDTO.AddNodeReqToNodeModels(reqDTOs)

	var addResponses []interface{}
	for i, d := range nodes {
		var response interface{}
		reqId := reqDTOs[i].RequestId
		newId, err := application.AddNode(d, ctx, dc.dic, bypassValidation, force)
		if err != nil {
			lc.Error(err.Error(), common.CorrelationHeader, correlationId)
			lc.Debug(err.DebugMessages(), common.CorrelationHeader, correlationId)
			response = commonDTO.NewBaseResponse(
				reqId,
				err.Message(),
				err.Code())
		} else {
			response = commonDTO.NewBaseWithIdResponse(
				reqId,
				"",
				http.StatusCreated,
				newId)
		}
		addResponses = append(addResponses, response)
	}

	utils.WriteHttpHeader(w, ctx, http.StatusMultiStatus)
	return pkg.EncodeAndWriteResponse(addResponses, w, lc)
}

func (dc *NodeController) DeleteNodeByName(c echo.Context) error {
	lc := container.LoggingClientFrom(dc.dic.Get)
	r := c.Request()
	w := c.Response()
	ctx := r.Context()

	// URL parameters
	name := c.Param(common.Name)

	err := application.DeleteNodeByName(name, ctx, dc.dic)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	response := commonDTO.NewBaseResponse("", "", http.StatusOK)
	utils.WriteHttpHeader(w, ctx, http.StatusOK)
	return pkg.EncodeAndWriteResponse(response, w, lc)
}

func (dc *NodeController) NodesByServiceName(c echo.Context) error {
	lc := container.LoggingClientFrom(dc.dic.Get)
	r := c.Request()
	w := c.Response()
	ctx := r.Context()
	config := metadataContainer.ConfigurationFrom(dc.dic.Get)

	name := c.Param(common.Name)

	// parse URL query string for offset, limit
	offset, limit, _, err := utils.ParseGetAllObjectsRequestQueryString(c, 0, math.MaxInt32, -1, config.Service.MaxResultCount)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}
	nodes, totalCount, err := application.NodesByServiceName(offset, limit, name, ctx, dc.dic)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	response := responseDTO.NewMultiNodesResponse("", "", http.StatusOK, totalCount, nodes)
	utils.WriteHttpHeader(w, ctx, http.StatusOK)
	return pkg.EncodeAndWriteResponse(response, w, lc)
}

func (dc *NodeController) NodeNameExists(c echo.Context) error {
	lc := container.LoggingClientFrom(dc.dic.Get)
	r := c.Request()
	w := c.Response()
	ctx := r.Context()

	// URL parameters
	name := c.Param(common.Name)

	var response interface{}
	var statusCode int

	exists, err := application.NodeNameExists(name, dc.dic)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	if exists {
		response = commonDTO.NewBaseResponse("", "", http.StatusOK)
		statusCode = http.StatusOK
	} else {
		response = commonDTO.NewBaseResponse("", "", http.StatusNotFound)
		statusCode = http.StatusNotFound
	}
	utils.WriteHttpHeader(w, ctx, statusCode)
	return pkg.EncodeAndWriteResponse(response, w, lc)
}

func (dc *NodeController) PatchNode(c echo.Context) error {
	r := c.Request()
	w := c.Response()
	if r.Body != nil {
		defer func() { _ = r.Body.Close() }()
	}

	lc := container.LoggingClientFrom(dc.dic.Get)

	ctx := r.Context()
	correlationId := correlation.FromContext(ctx)

	var bypassValidation bool
	// parse URL query string for bypassValidation
	bypassValidationParamStr := utils.ParseQueryStringToString(r, bypassValidationQueryParam, common.ValueFalse)
	if bypassValidationParamStr == common.ValueTrue {
		lc.Debugf("%s is true. %s=%s", bypassValidationQueryParam, common.CorrelationHeader, correlationId)
		bypassValidation = true
	}

	var reqDTOs []requestDTO.UpdateNodeRequest
	err := dc.reader.Read(r.Body, &reqDTOs)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	var updateResponses []interface{}
	for _, dto := range reqDTOs {
		var response interface{}
		reqId := dto.RequestId
		err := application.PatchNode(dto.Node, ctx, dc.dic, bypassValidation)
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
		updateResponses = append(updateResponses, response)
	}

	utils.WriteHttpHeader(w, ctx, http.StatusMultiStatus)
	return pkg.EncodeAndWriteResponse(updateResponses, w, lc)
}

func (dc *NodeController) AllNodes(c echo.Context) error {
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
	parent := utils.ParseQueryStringToString(r, common.DescendantsOf, "")
	levels, err := utils.ParseQueryStringToInt(c, common.MaxLevels, 0, -1, math.MaxInt32)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}
	if levels < 0 {
		levels = math.MaxInt32
	}
	nodes, totalCount, err := application.AllNodes(offset, limit, labels, parent, levels, dc.dic)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	response := responseDTO.NewMultiNodesResponse("", "", http.StatusOK, totalCount, nodes)
	utils.WriteHttpHeader(w, ctx, http.StatusOK)
	return pkg.EncodeAndWriteResponse(response, w, lc)
}

func (dc *NodeController) NodeByName(c echo.Context) error {
	lc := container.LoggingClientFrom(dc.dic.Get)
	r := c.Request()
	w := c.Response()
	ctx := r.Context()

	// URL parameters
	name := c.Param(common.Name)

	node, err := application.NodeByName(name, dc.dic)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	response := responseDTO.NewNodeResponse("", "", http.StatusOK, node)
	utils.WriteHttpHeader(w, ctx, http.StatusOK)
	return pkg.EncodeAndWriteResponse(response, w, lc)
}

func (dc *NodeController) NodeById(c echo.Context) error {
	lc := container.LoggingClientFrom(dc.dic.Get)
	r := c.Request()
	w := c.Response()
	ctx := r.Context()

	// URL parameters
	id := c.Param(common.Id)

	node, err := application.NodeById(id, dc.dic)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	response := responseDTO.NewNodeResponse("", "", http.StatusOK, node)
	utils.WriteHttpHeader(w, ctx, http.StatusOK)
	return pkg.EncodeAndWriteResponse(response, w, lc)
}

func (dc *NodeController) NodesByProfileName(c echo.Context) error {
	lc := container.LoggingClientFrom(dc.dic.Get)
	r := c.Request()
	w := c.Response()
	ctx := r.Context()
	config := metadataContainer.ConfigurationFrom(dc.dic.Get)

	name := c.Param(common.Name)

	// parse URL query string for offset, limit
	offset, limit, _, err := utils.ParseGetAllObjectsRequestQueryString(c, 0, math.MaxInt32, -1, config.Service.MaxResultCount)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}
	nodes, totalCount, err := application.NodesByProfileName(offset, limit, name, dc.dic)
	if err != nil {
		return utils.WriteErrorResponse(w, ctx, lc, err, "")
	}

	response := responseDTO.NewMultiNodesResponse("", "", http.StatusOK, totalCount, nodes)
	utils.WriteHttpHeader(w, ctx, http.StatusOK)
	return pkg.EncodeAndWriteResponse(response, w, lc)
}
