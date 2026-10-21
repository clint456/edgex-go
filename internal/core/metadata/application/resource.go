//
// Copyright (C) 2021-2022 IOTech Ltd
// Copyright (C) 2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"fmt"

	"github.com/clint456/edgex-go/internal/core/metadata/container"
	"github.com/clint456/edgex-go/internal/pkg/correlation"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/requests"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
)

// ResourceByProfileNameAndResourceName query the node resource by profileName and resourceName
func ResourceByProfileNameAndResourceName(profileName string, resourceName string, dic *di.Container) (resource dtos.Resource, err errors.EdgeX) {
	if profileName == "" {
		return resource, errors.NewCommonEdgeX(errors.KindContractInvalid, "profile name is empty", nil)
	}
	if resourceName == "" {
		return resource, errors.NewCommonEdgeX(errors.KindContractInvalid, "resource name is empty", nil)
	}
	dbClient := container.DBClientFrom(dic.Get)
	profile, err := dbClient.ProfileByName(profileName)
	if err != nil {
		return resource, errors.NewCommonEdgeXWrapper(err)
	}
	r, err := resourceByName(profile.Resources, resourceName)
	if err != nil {
		return resource, errors.NewCommonEdgeXWrapper(err)
	}

	resource = dtos.FromResourceModelToDTO(r)
	return resource, nil
}

func resourceByName(resources []models.Resource, resourceName string) (models.Resource, errors.EdgeX) {
	for _, r := range resources {
		if r.Name == resourceName {
			return r, nil
		}
	}
	return models.Resource{}, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("resource %s not exists", resourceName), nil)
}

func AddProfileResource(profileName string, resource models.Resource, ctx context.Context, dic *di.Container) errors.EdgeX {
	dbClient := container.DBClientFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	config := container.ConfigurationFrom(dic.Get)

	profile, err := dbClient.ProfileByName(profileName)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	err = resourceUoMValidation(resource, dic)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	if config.Writable.MaxResources > 0 {
		if err = checkResourceCapacityByNewResource(profileName, resource, dic); err != nil {
			return errors.NewCommonEdgeXWrapper(err)
		}
	}

	profile.Resources = append(profile.Resources, resource)

	profileDTO := dtos.FromProfileModelToDTO(profile)
	validateErr := profileDTO.Validate()
	if validateErr != nil {
		return errors.NewCommonEdgeXWrapper(validateErr)
	}

	err = dbClient.UpdateProfile(profile)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	lc.Debugf("Profile resources added on DB successfully. Correlation-id: %s ", correlation.FromContext(ctx))
	go publishUpdateProfileSystemEvent(profileDTO, ctx, dic)

	return nil
}

func PatchProfileResource(profileName string, dto dtos.UpdateResource, ctx context.Context, dic *di.Container) errors.EdgeX {
	dbClient := container.DBClientFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)

	profile, err := dbClient.ProfileByName(profileName)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	// Find matched resource
	index := -1
	for i := range profile.Resources {
		if profile.Resources[i].Name == *dto.Name {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "node resource not found", nil)
	}

	requests.ReplaceResourceModelFieldsWithDTO(&profile.Resources[index], dto)

	err = dbClient.UpdateProfile(profile)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	lc.Debugf("Profile resources patched on DB successfully. Correlation-id: %s ", correlation.FromContext(ctx))
	profileDTO := dtos.FromProfileModelToDTO(profile)
	go publishUpdateProfileSystemEvent(profileDTO, ctx, dic)

	return nil
}

func DeleteResourceByName(profileName string, resourceName string, ctx context.Context, dic *di.Container) errors.EdgeX {
	if profileName == "" {
		return errors.NewCommonEdgeX(errors.KindContractInvalid, "profile name is empty", nil)
	}
	if resourceName == "" {
		return errors.NewCommonEdgeX(errors.KindContractInvalid, "resource name is empty", nil)
	}

	strictProfileChanges := container.ConfigurationFrom(dic.Get).Writable.ProfileChange.StrictProfileChanges
	if strictProfileChanges {
		return errors.NewCommonEdgeX(errors.KindServiceLocked, "profile change is not allowed when StrictProfileChanges config is enabled", nil)
	}

	dbClient := container.DBClientFrom(dic.Get)
	// Check the associated Node existence
	nodes, err := dbClient.NodesByProfileName(0, 1, profileName)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	} else if len(nodes) > 0 {
		return errors.NewCommonEdgeX(errors.KindStatusConflict, "fail to update the node profile when associated node exists", nil)
	}

	profile, err := dbClient.ProfileByName(profileName)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	index := -1
	for i := range profile.Resources {
		if profile.Resources[i].Name == resourceName {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "node resource not found", nil)
	}

	profile.Resources = append(profile.Resources[:index], profile.Resources[index+1:]...)
	profileDTO := dtos.FromProfileModelToDTO(profile)
	e := (&profileDTO).Validate()
	if e != nil {
		return errors.NewCommonEdgeXWrapper(e)
	}

	err = dbClient.UpdateProfile(profile)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	go publishUpdateProfileSystemEvent(profileDTO, ctx, dic)
	return nil
}

func resourceUoMValidation(r models.Resource, dic *di.Container) errors.EdgeX {
	if container.ConfigurationFrom(dic.Get).Writable.UoM.Validation {
		uom := container.UnitsOfMeasureFrom(dic.Get)
		if ok := uom.Validate(r.Properties.Units); !ok {
			return errors.NewCommonEdgeX(errors.KindContractInvalid, fmt.Sprintf("Resource %s units %s is invalid", r.Name, r.Properties.Units), nil)
		}
	}

	return nil
}
