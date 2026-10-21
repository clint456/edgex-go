//
// Copyright (C) 2022 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"github.com/clint456/edgex-go/internal/core/metadata/container"
	"github.com/clint456/edgex-go/internal/pkg/correlation"
	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/requests"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
)

// The AddProfileCommand function accepts the node profile name and node command model from the controller functions
// and invokes updateProfile function in the infrastructure layer
func AddProfileCommand(profileName string, command models.Command, ctx context.Context, dic *di.Container) errors.EdgeX {
	dbClient := container.DBClientFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)

	profile, err := dbClient.ProfileByName(profileName)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	profile.Commands = append(profile.Commands, command)

	profileDTO := dtos.FromProfileModelToDTO(profile)
	validateErr := profileDTO.Validate()
	if validateErr != nil {
		return errors.NewCommonEdgeXWrapper(validateErr)
	}

	err = dbClient.UpdateProfile(profile)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	lc.Debugf("Profile commands added on DB successfully. Correlation-id: %s ", correlation.FromContext(ctx))
	go publishUpdateProfileSystemEvent(profileDTO, ctx, dic)

	return nil
}

func PatchProfileCommand(profileName string, dto dtos.UpdateCommand, ctx context.Context, dic *di.Container) errors.EdgeX {
	dbClient := container.DBClientFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)

	profile, err := dbClient.ProfileByName(profileName)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	// Find matched command
	index := -1
	for i := range profile.Commands {
		if profile.Commands[i].Name == *dto.Name {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "node command not found", nil)
	}

	requests.ReplaceCommandModelFieldsWithDTO(&profile.Commands[index], dto)

	err = dbClient.UpdateProfile(profile)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	lc.Debugf("Profile commands patched on DB successfully. Correlation-id: %s ", correlation.FromContext(ctx))
	profileDTO := dtos.FromProfileModelToDTO(profile)
	go publishUpdateProfileSystemEvent(profileDTO, ctx, dic)

	return nil
}

func DeleteCommandByName(profileName string, commandName string, ctx context.Context, dic *di.Container) errors.EdgeX {
	if profileName == "" {
		return errors.NewCommonEdgeX(errors.KindContractInvalid, "profile name is empty", nil)
	}
	if commandName == "" {
		return errors.NewCommonEdgeX(errors.KindContractInvalid, "command name is empty", nil)
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
	for i := range profile.Commands {
		if profile.Commands[i].Name == commandName {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, "node command not found", nil)
	}

	profile.Commands = append(profile.Commands[:index], profile.Commands[index+1:]...)
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
