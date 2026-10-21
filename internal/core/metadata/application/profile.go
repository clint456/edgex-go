//
// Copyright (C) 2020-2026 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"fmt"

	"github.com/clint456/edgex-go/internal/core/metadata/container"
	"github.com/clint456/edgex-go/internal/core/metadata/infrastructure/interfaces"
	"github.com/clint456/edgex-go/internal/pkg/correlation"
	"github.com/clint456/edgex-go/internal/pkg/utils"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/requests"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
)

// The AddProfile function accepts the new node profile model from the controller functions
// and invokes addProfile function in the infrastructure layer
func AddProfile(d models.Profile, ctx context.Context, dic *di.Container) (id string, err errors.EdgeX) {
	dbClient := container.DBClientFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)

	err = profileUoMValidation(d, dic)
	if err != nil {
		return "", errors.NewCommonEdgeXWrapper(err)
	}

	correlationId := correlation.FromContext(ctx)
	addedProfile, err := dbClient.AddProfile(d)
	if err != nil {
		return "", errors.NewCommonEdgeXWrapper(err)
	}

	lc.Debugf(
		"Profile created on DB successfully. Profile-id: %s, Correlation-id: %s ",
		addedProfile.Id,
		correlationId,
	)

	profileDTO := dtos.FromProfileModelToDTO(addedProfile)
	go publishSystemEvent(common.ProfileSystemEventType, common.SystemEventActionAdd, common.CoreMetaDataServiceKey, profileDTO, ctx, dic)

	return addedProfile.Id, nil
}

// The UpdateProfile function accepts the node profile model from the controller functions
// and invokes updateProfile function in the infrastructure layer
func UpdateProfile(d models.Profile, ctx context.Context, dic *di.Container) (err errors.EdgeX) {
	dbClient := container.DBClientFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	config := container.ConfigurationFrom(dic.Get)

	err = profileUoMValidation(d, dic)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	if config.Writable.MaxResources > 0 {
		if err = checkResourceCapacityByUpdateProfile(d, dic); err != nil {
			return errors.NewCommonEdgeXWrapper(err)
		}
	}

	err = dbClient.UpdateProfile(d)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	lc.Debugf(
		"Profile updated on DB successfully. Correlation-id: %s ",
		correlation.FromContext(ctx),
	)

	profile, err := dbClient.ProfileByName(d.Name)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	profileDTO := dtos.FromProfileModelToDTO(profile)
	go publishUpdateProfileSystemEvent(profileDTO, ctx, dic)

	return nil
}

func isProfileInUse(profileName string, dic *di.Container) (bool, errors.EdgeX) {
	dbClient := container.DBClientFrom(dic.Get)
	count, err := dbClient.NodeCountByProfileName(profileName)
	if err != nil {
		return false, errors.NewCommonEdgeXWrapper(err)
	}
	return count > 0, nil
}

// ProfileByName query the node profile by name
func ProfileByName(name string, ctx context.Context, dic *di.Container) (profile dtos.Profile, err errors.EdgeX) {
	if name == "" {
		return profile, errors.NewCommonEdgeX(errors.KindContractInvalid, "name is empty", nil)
	}
	dbClient := container.DBClientFrom(dic.Get)
	dp, err := dbClient.ProfileByName(name)
	if err != nil {
		return profile, errors.NewCommonEdgeXWrapper(err)
	}
	profile = dtos.FromProfileModelToDTO(dp)

	// get the linked node count
	nodeCount, err := dbClient.NodeCountByProfileName(profile.Name)
	if err != nil {
		return profile, errors.NewCommonEdgeXWrapper(err)
	}
	profile.LinkedNodeCount = nodeCount

	return profile, nil
}

// DeleteProfileByName delete the node profile by name
func DeleteProfileByName(name string, ctx context.Context, dic *di.Container) errors.EdgeX {
	strictProfileDeletes := container.ConfigurationFrom(dic.Get).Writable.ProfileChange.StrictProfileDeletes
	if strictProfileDeletes {
		return errors.NewCommonEdgeX(errors.KindServiceLocked, "profile deletion is not allowed when StrictProfileDeletes config is enabled", nil)
	}
	if name == "" {
		return errors.NewCommonEdgeX(errors.KindContractInvalid, "name is empty", nil)
	}
	dbClient := container.DBClientFrom(dic.Get)
	profile, err := dbClient.ProfileByName(name)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	// Check the associated Node and ProvisionWatcher existence
	nodes, edgeXErr := dbClient.NodesByProfileName(0, 1, name)
	if edgeXErr != nil {
		return errors.NewCommonEdgeXWrapper(edgeXErr)
	}
	if len(nodes) > 0 {
		return errors.NewCommonEdgeX(errors.KindStatusConflict, "fail to delete the node profile when associated node exists", nil)
	}
	provisionWatchers, edgeXErr := dbClient.ProvisionWatchersByProfileName(0, 1, name)
	if edgeXErr != nil {
		return errors.NewCommonEdgeXWrapper(edgeXErr)
	}
	if len(provisionWatchers) > 0 {
		return errors.NewCommonEdgeX(errors.KindStatusConflict, "fail to delete the node profile when associated provisionWatcher exists", nil)
	}

	err = dbClient.DeleteProfileByName(name)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	profileDTO := dtos.FromProfileModelToDTO(profile)
	go publishSystemEvent(common.ProfileSystemEventType, common.SystemEventActionDelete, common.CoreMetaDataServiceKey, profileDTO, ctx, dic)

	return nil
}

// AllProfiles query the node profiles with offset, limit, and labels.
func AllProfiles(offset int, limit int, labels []string, dic *di.Container) (profiles []dtos.Profile, totalCount int64, err errors.EdgeX) {
	dbClient := container.DBClientFrom(dic.Get)

	totalCount, err = dbClient.ProfileCountByLabels(labels)
	if err != nil {
		return profiles, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	cont, err := utils.CheckCountRange(totalCount, offset, limit)
	if !cont {
		return []dtos.Profile{}, totalCount, err
	}

	dps, err := dbClient.AllProfiles(offset, limit, labels)
	if err != nil {
		return profiles, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	profiles = make([]dtos.Profile, len(dps))
	for i, dp := range dps {
		profiles[i] = dtos.FromProfileModelToDTO(dp)

		// get the linked node count for each node profile DTO
		nodeCount, err := dbClient.NodeCountByProfileName(dp.Name)
		if err != nil {
			return []dtos.Profile{}, totalCount, errors.NewCommonEdgeXWrapper(err)
		}
		profiles[i].LinkedNodeCount = nodeCount
	}
	return profiles, totalCount, nil
}

// ProfilesByModel query the node profiles with offset, limit and model
func ProfilesByModel(offset int, limit int, model string, dic *di.Container) (profiles []dtos.Profile, totalCount int64, err errors.EdgeX) {
	if model == "" {
		return profiles, totalCount, errors.NewCommonEdgeX(errors.KindContractInvalid, "model is empty", nil)
	}

	dbClient := container.DBClientFrom(dic.Get)
	totalCount, err = dbClient.ProfileCountByModel(model)
	if err != nil {
		return profiles, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	cont, err := utils.CheckCountRange(totalCount, offset, limit)
	if !cont {
		return []dtos.Profile{}, totalCount, err
	}

	dps, err := dbClient.ProfilesByModel(offset, limit, model)
	if err != nil {
		return profiles, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	profiles = make([]dtos.Profile, len(dps))
	for i, dp := range dps {
		profiles[i] = dtos.FromProfileModelToDTO(dp)
	}
	return profiles, totalCount, nil
}

// ProfilesByManufacturer query the node profiles with offset, limit and manufacturer
func ProfilesByManufacturer(offset int, limit int, manufacturer string, dic *di.Container) (profiles []dtos.Profile, totalCount int64, err errors.EdgeX) {
	if manufacturer == "" {
		return profiles, totalCount, errors.NewCommonEdgeX(errors.KindContractInvalid, "manufacturer is empty", nil)
	}

	dbClient := container.DBClientFrom(dic.Get)
	totalCount, err = dbClient.ProfileCountByManufacturer(manufacturer)
	if err != nil {
		return profiles, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	cont, err := utils.CheckCountRange(totalCount, offset, limit)
	if !cont {
		return []dtos.Profile{}, totalCount, err
	}

	dps, err := dbClient.ProfilesByManufacturer(offset, limit, manufacturer)
	if err != nil {
		return profiles, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	profiles = make([]dtos.Profile, len(dps))
	for i, dp := range dps {
		profiles[i] = dtos.FromProfileModelToDTO(dp)
	}
	return profiles, totalCount, nil
}

// ProfilesByManufacturerAndModel query the node profiles with offset, limit, manufacturer and model
func ProfilesByManufacturerAndModel(offset int, limit int, manufacturer string, model string, dic *di.Container) (profiles []dtos.Profile, totalCount int64, err errors.EdgeX) {
	if manufacturer == "" {
		return profiles, totalCount, errors.NewCommonEdgeX(errors.KindContractInvalid, "manufacturer is empty", nil)
	}
	if model == "" {
		return profiles, totalCount, errors.NewCommonEdgeX(errors.KindContractInvalid, "model is empty", nil)
	}
	dbClient := container.DBClientFrom(dic.Get)
	totalCount, err = dbClient.ProfileCountByManufacturerAndModel(manufacturer, model)
	if err != nil {
		return profiles, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	cont, err := utils.CheckCountRange(totalCount, offset, limit)
	if !cont {
		return []dtos.Profile{}, totalCount, err
	}

	dps, err := dbClient.ProfilesByManufacturerAndModel(offset, limit, manufacturer, model)
	if err != nil {
		return profiles, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	profiles = make([]dtos.Profile, len(dps))
	for i, dp := range dps {
		profiles[i] = dtos.FromProfileModelToDTO(dp)
	}
	return profiles, totalCount, nil
}

func PatchProfileBasicInfo(ctx context.Context, dto dtos.UpdateProfileBasicInfo, dic *di.Container) errors.EdgeX {
	dbClient := container.DBClientFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)

	profile, err := profileByDTO(dbClient, dto)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	requests.ReplaceProfileModelBasicInfoFieldsWithDTO(&profile, dto)
	err = dbClient.UpdateProfile(profile)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	lc.Debugf(
		"Profile basic info patched on DB successfully. Correlation-ID: %s ",
		correlation.FromContext(ctx),
	)

	profileDTO := dtos.FromProfileModelToDTO(profile)
	go publishUpdateProfileSystemEvent(profileDTO, ctx, dic)

	return nil
}

// AllProfileBasicInfos query the node profile basic infos with offset, limit, and labels.
func AllProfileBasicInfos(offset int, limit int, labels []string, dic *di.Container) (profileBasicInfos []dtos.ProfileBasicInfo, totalCount int64, err errors.EdgeX) {
	dbClient := container.DBClientFrom(dic.Get)

	totalCount, err = dbClient.ProfileCountByLabels(labels)
	if err != nil {
		return profileBasicInfos, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	cont, err := utils.CheckCountRange(totalCount, offset, limit)
	if !cont {
		return profileBasicInfos, totalCount, err
	}

	dps, err := dbClient.AllProfiles(offset, limit, labels)
	if err != nil {
		return profileBasicInfos, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	profileBasicInfos = make([]dtos.ProfileBasicInfo, len(dps))
	for i, dp := range dps {
		profileBasicInfos[i] = dtos.FromProfileModelToBasicInfoDTO(dp)

		// get the linked node count for each node profile DTO
		nodeCount, err := dbClient.NodeCountByProfileName(dp.Name)
		if err != nil {
			return []dtos.ProfileBasicInfo{}, totalCount, errors.NewCommonEdgeXWrapper(err)
		}
		profileBasicInfos[i].LinkedNodeCount = nodeCount
	}
	return profileBasicInfos, totalCount, nil
}

func PatchProfileTags(profileName string, dto dtos.UpdateProfileTags, ctx context.Context, dic *di.Container) errors.EdgeX {
	dbClient := container.DBClientFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)

	profile, err := dbClient.ProfileByName(profileName)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	requests.ReplaceProfileModelTagsWithDTO(&profile, dto)

	err = dbClient.UpdateProfile(profile)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	lc.Debugf(
		"Profile node resources/commands tags patched on DB successfully. Correlation-ID: %s ",
		correlation.FromContext(ctx),
	)

	profileDTO := dtos.FromProfileModelToDTO(profile)
	go publishUpdateProfileSystemEvent(profileDTO, ctx, dic)

	return nil
}

func profileByDTO(dbClient interfaces.DBClient, dto dtos.UpdateProfileBasicInfo) (profile models.Profile, err errors.EdgeX) {
	// The ID or Name is required by DTO and the DTO also accepts empty string ID if the Name is provided
	if dto.Id != nil && *dto.Id != "" {
		profile, err = dbClient.ProfileById(*dto.Id)
		if err != nil {
			return profile, errors.NewCommonEdgeXWrapper(err)
		}
	} else {
		profile, err = dbClient.ProfileByName(*dto.Name)
		if err != nil {
			return profile, errors.NewCommonEdgeXWrapper(err)
		}
	}
	if dto.Name != nil && *dto.Name != profile.Name {
		return profile, errors.NewCommonEdgeX(errors.KindContractInvalid, fmt.Sprintf("node profile name '%s' not match the exsting '%s' ", *dto.Name, profile.Name), nil)
	}
	return profile, nil
}

func profileUoMValidation(p models.Profile, dic *di.Container) errors.EdgeX {
	if container.ConfigurationFrom(dic.Get).Writable.UoM.Validation {
		uom := container.UnitsOfMeasureFrom(dic.Get)
		for _, dr := range p.Resources {
			if ok := uom.Validate(dr.Properties.Units); !ok {
				return errors.NewCommonEdgeX(errors.KindContractInvalid, fmt.Sprintf("Resource %s units %s is invalid", dr.Name, dr.Properties.Units), nil)
			}
		}
	}

	return nil
}
