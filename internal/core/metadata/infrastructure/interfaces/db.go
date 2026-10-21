//
// Copyright (C) 2020-2022 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package interfaces

import (
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	model "github.com/edgexfoundry/go-mod-core-contracts/v4/models"
)

type DBClient interface {
	CloseSession()

	AddProfile(e model.Profile) (model.Profile, errors.EdgeX)
	UpdateProfile(e model.Profile) errors.EdgeX
	ProfileById(id string) (model.Profile, errors.EdgeX)
	ProfileByName(name string) (model.Profile, errors.EdgeX)
	DeleteProfileById(id string) errors.EdgeX
	DeleteProfileByName(name string) errors.EdgeX
	ProfileNameExists(name string) (bool, errors.EdgeX)
	AllProfiles(offset int, limit int, labels []string) ([]model.Profile, errors.EdgeX)
	ProfilesByModel(offset int, limit int, model string) ([]model.Profile, errors.EdgeX)
	ProfilesByManufacturer(offset int, limit int, manufacturer string) ([]model.Profile, errors.EdgeX)
	ProfilesByManufacturerAndModel(offset int, limit int, manufacturer string, model string) ([]model.Profile, errors.EdgeX)
	ProfileCountByLabels(labels []string) (int64, errors.EdgeX)
	ProfileCountByManufacturer(manufacturer string) (int64, errors.EdgeX)
	ProfileCountByModel(model string) (int64, errors.EdgeX)
	ProfileCountByManufacturerAndModel(manufacturer string, model string) (int64, errors.EdgeX)
	InUseResourceCount() (int64, errors.EdgeX)

	AddNodeService(ds model.NodeService) (model.NodeService, errors.EdgeX)
	NodeServiceById(id string) (model.NodeService, errors.EdgeX)
	NodeServiceByName(name string) (model.NodeService, errors.EdgeX)
	DeleteNodeServiceById(id string) errors.EdgeX
	DeleteNodeServiceByName(name string) errors.EdgeX
	NodeServiceNameExists(name string) (bool, errors.EdgeX)
	AllNodeServices(offset int, limit int, labels []string) ([]model.NodeService, errors.EdgeX)
	UpdateNodeService(ds model.NodeService) errors.EdgeX
	NodeServiceCountByLabels(labels []string) (int64, errors.EdgeX)

	AddNode(d model.Node) (model.Node, errors.EdgeX)
	DeleteNodeById(id string) errors.EdgeX
	DeleteNodeByName(name string) errors.EdgeX
	NodesByServiceName(offset int, limit int, name string) ([]model.Node, errors.EdgeX)
	NodeIdExists(id string) (bool, errors.EdgeX)
	NodeNameExists(name string) (bool, errors.EdgeX)
	NodeById(id string) (model.Node, errors.EdgeX)
	NodeByName(name string) (model.Node, errors.EdgeX)
	AllNodes(offset int, limit int, labels []string) ([]model.Node, errors.EdgeX)
	NodesByProfileName(offset int, limit int, profileName string) ([]model.Node, errors.EdgeX)
	UpdateNode(d model.Node) errors.EdgeX
	NodeCountByLabels(labels []string) (int64, errors.EdgeX)
	NodeCountByProfileName(profileName string) (int64, errors.EdgeX)
	NodeCountByServiceName(serviceName string) (int64, errors.EdgeX)
	NodeTree(parent string, levels int, offset int, limit int, labels []string) (int64, []model.Node, errors.EdgeX)
	AddProvisionWatcher(pw model.ProvisionWatcher) (model.ProvisionWatcher, errors.EdgeX)
	ProvisionWatcherById(id string) (model.ProvisionWatcher, errors.EdgeX)
	ProvisionWatcherByName(name string) (model.ProvisionWatcher, errors.EdgeX)
	ProvisionWatchersByServiceName(offset int, limit int, name string) ([]model.ProvisionWatcher, errors.EdgeX)
	ProvisionWatchersByProfileName(offset int, limit int, name string) ([]model.ProvisionWatcher, errors.EdgeX)
	AllProvisionWatchers(offset int, limit int, labels []string) ([]model.ProvisionWatcher, errors.EdgeX)
	DeleteProvisionWatcherByName(name string) errors.EdgeX
	UpdateProvisionWatcher(pw model.ProvisionWatcher) errors.EdgeX
	ProvisionWatcherCountByLabels(labels []string) (int64, errors.EdgeX)
	ProvisionWatcherCountByServiceName(name string) (int64, errors.EdgeX)
	ProvisionWatcherCountByProfileName(name string) (int64, errors.EdgeX)
}
