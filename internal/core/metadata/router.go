//
// Copyright (C) 2021-2025 IOTech Ltd
// Copyright (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"github.com/clint456/edgex-go"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/controller"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/handlers"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"

	metadataController "github.com/clint456/edgex-go/internal/core/metadata/controller/http"

	"github.com/labstack/echo/v4"
)

func LoadRestRoutes(r *echo.Echo, dic *di.Container, serviceName string) {
	authenticationHook := handlers.AutoConfigAuthenticationFunc(dic)

	// Common
	_ = controller.NewCommonController(dic, r, serviceName, edgex.Version)

	// Units of Measure
	uc := metadataController.NewUnitOfMeasureController(dic)
	r.GET(common.ApiUnitsOfMeasureRoute, uc.UnitsOfMeasure, authenticationHook)

	// Node Profile
	dc := metadataController.NewProfileController(dic)
	r.POST(common.ApiProfileRoute, dc.AddProfile, authenticationHook)
	r.PUT(common.ApiProfileRoute, dc.UpdateProfile, authenticationHook)
	r.POST(common.ApiProfileUploadFileRoute, dc.AddProfileByYaml, authenticationHook)
	r.PUT(common.ApiProfileUploadFileRoute, dc.UpdateProfileByYaml, authenticationHook)
	r.GET(common.ApiProfileByNameRoute, dc.ProfileByName, authenticationHook)
	r.DELETE(common.ApiProfileByNameRoute, dc.DeleteProfileByName, authenticationHook)
	r.GET(common.ApiAllProfileRoute, dc.AllProfiles, authenticationHook)
	r.GET(common.ApiProfileByModelRoute, dc.ProfilesByModel, authenticationHook)
	r.GET(common.ApiProfileByManufacturerRoute, dc.ProfilesByManufacturer, authenticationHook)
	r.GET(common.ApiProfileByManufacturerAndModelRoute, dc.ProfilesByManufacturerAndModel, authenticationHook)
	r.PATCH(common.ApiProfileBasicInfoRoute, dc.PatchProfileBasicInfo, authenticationHook)
	r.GET(common.ApiAllProfileBasicInfoRoute, dc.AllProfileBasicInfos, authenticationHook)
	r.PATCH(common.ApiProfileTagsByNameRoute, dc.PatchProfileTags, authenticationHook)

	// Node Resource
	dr := metadataController.NewResourceController(dic)
	r.GET(common.ApiResourceByProfileAndResourceRoute, dr.ResourceByProfileNameAndResourceName, authenticationHook)
	r.POST(common.ApiProfileResourceRoute, dr.AddProfileResource, authenticationHook)
	r.PATCH(common.ApiProfileResourceRoute, dr.PatchProfileResource, authenticationHook)
	r.DELETE(common.ApiProfileResourceByNameRoute, dr.DeleteResourceByName, authenticationHook)

	// Deivce Command
	dcm := metadataController.NewCommandController(dic)
	r.POST(common.ApiProfileCommandRoute, dcm.AddProfileCommand, authenticationHook)
	r.PATCH(common.ApiProfileCommandRoute, dcm.PatchProfileCommand, authenticationHook)
	r.DELETE(common.ApiProfileCommandByNameRoute, dcm.DeleteCommandByName, authenticationHook)

	// Node Service
	ds := metadataController.NewNodeServiceController(dic)
	r.POST(common.ApiNodeServiceRoute, ds.AddNodeService, authenticationHook)
	r.PATCH(common.ApiNodeServiceRoute, ds.PatchNodeService, authenticationHook)
	r.GET(common.ApiNodeServiceByNameRoute, ds.NodeServiceByName, authenticationHook)
	r.DELETE(common.ApiNodeServiceByNameRoute, ds.DeleteNodeServiceByName, authenticationHook)
	r.GET(common.ApiAllNodeServiceRoute, ds.AllNodeServices, authenticationHook)

	// Node
	d := metadataController.NewNodeController(dic)
	r.POST(common.ApiNodeRoute, d.AddNode, authenticationHook)
	r.DELETE(common.ApiNodeByNameRoute, d.DeleteNodeByName, authenticationHook)
	r.GET(common.ApiNodeByServiceNameRoute, d.NodesByServiceName, authenticationHook)
	r.GET(common.ApiNodeNameExistsRoute, d.NodeNameExists, authenticationHook)
	r.PATCH(common.ApiNodeRoute, d.PatchNode, authenticationHook)
	r.GET(common.ApiAllNodeRoute, d.AllNodes, authenticationHook)
	r.GET(common.ApiNodeByNameRoute, d.NodeByName, authenticationHook)
	r.GET(common.ApiNodeRoute+"/"+common.Id+"/:"+common.Id, d.NodeById, authenticationHook)
	r.GET(common.ApiNodeByProfileNameRoute, d.NodesByProfileName, authenticationHook)

	// ProvisionWatcher
	pwc := metadataController.NewProvisionWatcherController(dic)
	r.POST(common.ApiProvisionWatcherRoute, pwc.AddProvisionWatcher, authenticationHook)
	r.GET(common.ApiProvisionWatcherByNameRoute, pwc.ProvisionWatcherByName, authenticationHook)
	r.GET(common.ApiProvisionWatcherByServiceNameRoute, pwc.ProvisionWatchersByServiceName, authenticationHook)
	r.GET(common.ApiProvisionWatcherByProfileNameRoute, pwc.ProvisionWatchersByProfileName, authenticationHook)
	r.GET(common.ApiAllProvisionWatcherRoute, pwc.AllProvisionWatchers, authenticationHook)
	r.DELETE(common.ApiProvisionWatcherByNameRoute, pwc.DeleteProvisionWatcherByName, authenticationHook)
	r.PATCH(common.ApiProvisionWatcherRoute, pwc.PatchProvisionWatcher, authenticationHook)
}
