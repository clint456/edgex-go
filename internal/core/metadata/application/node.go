/********************************************************************************
 *  Copyright (C) 2020-2026 IOTech Ltd
 *  Copyright 2023 Intel Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 *******************************************************************************/

package application

import (
	"context"
	goErrors "errors"
	"fmt"
	"regexp"
	"slices"
	"time"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/requests"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"

	"github.com/clint456/edgex-go/internal/core/metadata/container"
	"github.com/clint456/edgex-go/internal/core/metadata/infrastructure/interfaces"
	"github.com/clint456/edgex-go/internal/pkg/correlation"
	"github.com/clint456/edgex-go/internal/pkg/utils"
)

// the suggested minimum duration for auto event interval
const minAutoEventInterval = 1 * time.Millisecond

// The AddNode function accepts the new node model from the controller function
// and then invokes AddNode function of infrastructure layer to add new node
func AddNode(d models.Node, ctx context.Context, dic *di.Container, bypassValidation bool, force bool) (id string, edgeXerr errors.EdgeX) {
	dbClient := container.DBClientFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	config := container.ConfigurationFrom(dic.Get)

	// Check the existence of node service before node validation
	exists, edgeXerr := dbClient.NodeServiceNameExists(d.ServiceName)
	if edgeXerr != nil {
		return id, errors.NewCommonEdgeX(errors.Kind(edgeXerr), fmt.Sprintf("node service '%s' existence check failed", d.ServiceName), edgeXerr)
	} else if !exists {
		return id, errors.NewCommonEdgeX(errors.KindContractInvalid, fmt.Sprintf("node service '%s' does not exists", d.ServiceName), nil)
	}

	err := validateParentProfileAndAutoEvent(dic, d)
	if err != nil {
		return "", errors.NewCommonEdgeXWrapper(err)
	}

	// check if node name already exists
	exists, err = dbClient.NodeNameExists(d.Name)
	if err != nil {
		return "", errors.NewCommonEdgeXWrapper(err)
	}
	if exists {
		if force {
			// invoke updateNode if force flag is enabled
			return updateNode(d, ctx, dic)
		} else {
			return "", errors.NewCommonEdgeX(errors.KindDuplicateName, fmt.Sprintf("node name %s already exists", d.Name), nil)
		}
	}

	if config.Writable.MaxNodes > 0 || config.Writable.MaxResources > 0 {
		if err = checkCapacityWithNewNode(d, dic); err != nil {
			return "", errors.NewCommonEdgeXWrapper(err)
		}
	}

	// Execute the Node Service Validation when both bypassValidation/force values are false by default
	// Skip the Node Service Validation if either bypassValidation or force is true
	if !bypassValidation && !force { // Per De Morgan's law, !(A || B) is equivalent to !A && !B
		err = validateNodeCallback(dtos.FromNodeModelToDTO(d), dic)
		if err != nil {
			return "", errors.NewCommonEdgeXWrapper(err)
		}
	}

	addedNode, err := dbClient.AddNode(d)
	if err != nil {
		return "", errors.NewCommonEdgeXWrapper(err)
	}

	lc.Debugf(
		"Node created on DB successfully. Node ID: %s, Correlation-ID: %s ",
		addedNode.Id,
		correlation.FromContext(ctx),
	)

	// If node is successfully created, check each AutoEvent interval value and display a warning if it's smaller than the suggested 10ms value
	for _, autoEvent := range d.AutoEvents {
		utils.CheckMinInterval(autoEvent.Interval, minAutoEventInterval, lc)
	}

	nodeDTO := dtos.FromNodeModelToDTO(addedNode)
	go publishSystemEvent(common.NodeSystemEventType, common.SystemEventActionAdd, d.ServiceName, nodeDTO, ctx, dic)

	return addedNode.Id, nil
}

// updateNode accepts the updated node model from AddNode function if force flag is enabled
// and then invokes UpdateNode function of infrastructure layer to update the existing node
// the "update node" system events will be published to the msg bus at last
func updateNode(d models.Node, ctx context.Context, dic *di.Container) (id string, edgeXerr errors.EdgeX) {
	dbClient := container.DBClientFrom(dic.Get)

	oldNode, err := dbClient.NodeByName(d.Name)
	if err != nil {
		return "", errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	// set the id and created fields from the old node
	if d.Id == "" {
		d.Id = oldNode.Id
	}
	if d.Created == 0 {
		d.Created = oldNode.Created
	}

	// Old service name is used for invoking callback
	var oldServiceName string
	if d.ServiceName != "" && d.ServiceName != oldNode.ServiceName {
		oldServiceName = oldNode.ServiceName
	}

	if container.ConfigurationFrom(dic.Get).Writable.MaxResources > 0 {
		if err = checkResourceCapacityByExistingAndNewProfile(oldNode.ProfileName, d.ProfileName, dic); err != nil {
			return "", errors.NewCommonEdgeXWrapper(err)
		}
	}

	err = updateNodeInDB(d, oldServiceName, ctx, dic)
	if err != nil {
		return "", errors.NewCommonEdgeXWrapper(err)
	}

	return d.Id, nil
}

// DeleteNodeByName deletes the node by name
func DeleteNodeByName(name string, ctx context.Context, dic *di.Container) errors.EdgeX {
	if name == "" {
		return errors.NewCommonEdgeX(errors.KindContractInvalid, "name is empty", nil)
	}
	dbClient := container.DBClientFrom(dic.Get)
	node, err := dbClient.NodeByName(name)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	childcount, _, err := dbClient.NodeTree(name, 1, 0, 1, nil)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	if childcount != 0 {
		return errors.NewCommonEdgeX(errors.KindStatusConflict, "cannot delete node with children", nil)
	}
	err = dbClient.DeleteNodeByName(name)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	nodeDTO := dtos.FromNodeModelToDTO(node)
	go publishSystemEvent(common.NodeSystemEventType, common.SystemEventActionDelete, node.ServiceName, nodeDTO, ctx, dic)

	return nil
}

// NodesByServiceName query nodes with offset, limit and name
func NodesByServiceName(offset int, limit int, name string, ctx context.Context, dic *di.Container) (nodes []dtos.Node, totalCount int64, err errors.EdgeX) {
	if name == "" {
		return nodes, totalCount, errors.NewCommonEdgeX(errors.KindContractInvalid, "name is empty", nil)
	}
	dbClient := container.DBClientFrom(dic.Get)

	totalCount, err = dbClient.NodeCountByServiceName(name)
	if err != nil {
		return nodes, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	cont, err := utils.CheckCountRange(totalCount, offset, limit)
	if !cont {
		return []dtos.Node{}, totalCount, err
	}

	nodeModels, err := dbClient.NodesByServiceName(offset, limit, name)
	if err != nil {
		return nodes, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	nodes = make([]dtos.Node, len(nodeModels))
	for i, d := range nodeModels {
		nodes[i] = dtos.FromNodeModelToDTO(d)
	}
	return nodes, totalCount, nil
}

// NodeNameExists checks the node existence by name
func NodeNameExists(name string, dic *di.Container) (exists bool, err errors.EdgeX) {
	if name == "" {
		return exists, errors.NewCommonEdgeX(errors.KindContractInvalid, "name is empty", nil)
	}
	dbClient := container.DBClientFrom(dic.Get)
	exists, err = dbClient.NodeNameExists(name)
	if err != nil {
		return exists, errors.NewCommonEdgeXWrapper(err)
	}
	return exists, nil
}

// PatchNode executes the PATCH operation with the node DTO to replace the old data
func PatchNode(dto dtos.UpdateNode, ctx context.Context, dic *di.Container, bypassValidation bool) errors.EdgeX {
	dbClient := container.DBClientFrom(dic.Get)

	// Check the existence of node service before node validation
	if dto.ServiceName != nil {
		exists, edgeXerr := dbClient.NodeServiceNameExists(*dto.ServiceName)
		if edgeXerr != nil {
			return errors.NewCommonEdgeX(errors.Kind(edgeXerr), fmt.Sprintf("node service '%s' existence check failed", *dto.ServiceName), edgeXerr)
		} else if !exists {
			return errors.NewCommonEdgeX(errors.KindContractInvalid, fmt.Sprintf("node service '%s' does not exists", *dto.ServiceName), nil)
		}
	}

	node, err := nodeByDTO(dbClient, dto)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	// Old service name is used for invoking callback
	var oldServiceName string
	if dto.ServiceName != nil && *dto.ServiceName != node.ServiceName {
		oldServiceName = node.ServiceName
	}

	oldProfileName := node.ProfileName
	newProfileName := oldProfileName
	if dto.ProfileName != nil {
		newProfileName = *dto.ProfileName
	}
	if container.ConfigurationFrom(dic.Get).Writable.MaxResources > 0 {
		if err = checkResourceCapacityByExistingAndNewProfile(oldProfileName, newProfileName, dic); err != nil {
			return errors.NewCommonEdgeXWrapper(err)
		}
	}

	requests.ReplaceNodeModelFieldsWithDTO(&node, dto)
	if oldProfileName != newProfileName {
		node, err = validateParentProfileAndAutoEventDropInvalid(dic, node)
	} else {
		err = validateParentProfileAndAutoEvent(dic, node)
	}
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	// Execute the Node Service Validation when bypassValidation is false by default
	// Skip the Node Service Validation if bypassValidation is true
	if !bypassValidation {
		nodeDTO := dtos.FromNodeModelToDTO(node)
		err = validateNodeCallback(nodeDTO, dic)
		if err != nil {
			return errors.NewCommonEdgeXWrapper(err)
		}
	}

	return updateNodeInDB(node, oldServiceName, ctx, dic)
}

// updateNodeInDB calls the UpdateNode method from the infrastructure layer and validate the node auto events
// and publish the "update node" system event at last
func updateNodeInDB(node models.Node, oldServiceName string, ctx context.Context, dic *di.Container) errors.EdgeX {
	dbClient := container.DBClientFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)

	err := dbClient.UpdateNode(node)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	// If node is successfully updated, check each AutoEvent interval value and display a warning if it's smaller than the suggested 10ms value
	for _, autoEvent := range node.AutoEvents {
		utils.CheckMinInterval(autoEvent.Interval, minAutoEventInterval, lc)
	}

	lc.Debugf(
		"Node updated on DB successfully. Correlation-ID: %s ",
		correlation.FromContext(ctx),
	)

	nodeDTO := dtos.FromNodeModelToDTO(node)
	if oldServiceName != "" {
		go publishSystemEvent(common.NodeSystemEventType, common.SystemEventActionUpdate, oldServiceName, nodeDTO, ctx, dic)
	}

	go publishSystemEvent(common.NodeSystemEventType, common.SystemEventActionUpdate, node.ServiceName, nodeDTO, ctx, dic)

	return nil
}

func nodeByDTO(dbClient interfaces.DBClient, dto dtos.UpdateNode) (node models.Node, edgeXerr errors.EdgeX) {
	// The ID or Name is required by DTO and the DTO also accepts empty string ID if the Name is provided
	if dto.Id != nil && *dto.Id != "" {
		node, edgeXerr = dbClient.NodeById(*dto.Id)
		if edgeXerr != nil {
			return node, errors.NewCommonEdgeXWrapper(edgeXerr)
		}
	} else {
		node, edgeXerr = dbClient.NodeByName(*dto.Name)
		if edgeXerr != nil {
			return node, errors.NewCommonEdgeXWrapper(edgeXerr)
		}
	}
	if dto.Name != nil && *dto.Name != node.Name {
		return node, errors.NewCommonEdgeX(errors.KindContractInvalid, fmt.Sprintf("node name '%s' not match the exsting '%s' ", *dto.Name, node.Name), nil)
	}
	return node, nil
}

// AllNodes query the nodes with offset, limit, and labels
func AllNodes(offset int, limit int, labels []string, parent string, maxLevels int, dic *di.Container) (nodes []dtos.Node, totalCount int64, err errors.EdgeX) {
	dbClient := container.DBClientFrom(dic.Get)
	var nodeModels []models.Node
	if parent != "" {
		totalCount, nodeModels, err = dbClient.NodeTree(parent, maxLevels, offset, limit, labels)
		if err != nil {
			return nodes, totalCount, errors.NewCommonEdgeXWrapper(err)
		}
	} else {
		totalCount, err = dbClient.NodeCountByLabels(labels)
		if err != nil {
			return nodes, totalCount, errors.NewCommonEdgeXWrapper(err)
		}
		cont, err := utils.CheckCountRange(totalCount, offset, limit)
		if !cont {
			return []dtos.Node{}, totalCount, err
		}

		nodeModels, err = dbClient.AllNodes(offset, limit, labels)
		if err != nil {
			return nodes, totalCount, errors.NewCommonEdgeXWrapper(err)
		}
	}

	nodes = make([]dtos.Node, len(nodeModels))
	for i, d := range nodeModels {
		nodes[i] = dtos.FromNodeModelToDTO(d)
	}
	return nodes, totalCount, nil
}

// NodeByName query the node by name
func NodeByName(name string, dic *di.Container) (node dtos.Node, err errors.EdgeX) {
	if name == "" {
		return node, errors.NewCommonEdgeX(errors.KindContractInvalid, "name is empty", nil)
	}
	dbClient := container.DBClientFrom(dic.Get)
	d, err := dbClient.NodeByName(name)
	if err != nil {
		return node, errors.NewCommonEdgeXWrapper(err)
	}
	node = dtos.FromNodeModelToDTO(d)
	return node, nil
}

// NodeById query the node by id
func NodeById(id string, dic *di.Container) (node dtos.Node, err errors.EdgeX) {
	if id == "" {
		return node, errors.NewCommonEdgeX(errors.KindContractInvalid, "id is empty", nil)
	}
	dbClient := container.DBClientFrom(dic.Get)
	d, err := dbClient.NodeById(id)
	if err != nil {
		return node, errors.NewCommonEdgeXWrapper(err)
	}
	node = dtos.FromNodeModelToDTO(d)
	return node, nil
}

// NodesByProfileName query the nodes with offset, limit, and profile name
func NodesByProfileName(offset int, limit int, profileName string, dic *di.Container) (nodes []dtos.Node, totalCount int64, err errors.EdgeX) {
	if profileName == "" {
		return nodes, totalCount, errors.NewCommonEdgeX(errors.KindContractInvalid, "profileName is empty", nil)
	}
	dbClient := container.DBClientFrom(dic.Get)

	totalCount, err = dbClient.NodeCountByProfileName(profileName)
	if err != nil {
		return nodes, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	cont, err := utils.CheckCountRange(totalCount, offset, limit)
	if !cont {
		return []dtos.Node{}, totalCount, err
	}

	nodeModels, err := dbClient.NodesByProfileName(offset, limit, profileName)
	if err != nil {
		return nodes, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	nodes = make([]dtos.Node, len(nodeModels))
	for i, d := range nodeModels {
		nodes[i] = dtos.FromNodeModelToDTO(d)
	}
	return nodes, totalCount, nil
}

var errNoMessagingClient = goErrors.New("MessageBus Client not available. Please update RequireMessageBus and MessageBus configuration to enable sending System Events via the EdgeX MessageBus")

func validateParentProfileAndAutoEvent(dic *di.Container, d models.Node) errors.EdgeX {
	if d.ProfileName == "" {
		// if the profile is not set, skip the validation until we have the profile
		return nil
	}
	dp, err := validateParentAndLoadProfile(dic, d)
	if err != nil {
		return err
	}
	if len(d.AutoEvents) == 0 {
		return nil
	}
	for _, a := range d.AutoEvents {
		if err := validateAutoEvent(a, d, dp); err != nil {
			return err
		}
	}
	return nil
}

func validateParentProfileAndAutoEventDropInvalid(dic *di.Container, d models.Node) (models.Node, errors.EdgeX) {
	if d.ProfileName == "" {
		// if the profile is not set, skip the validation until we have the profile
		return d, nil
	}
	dp, err := validateParentAndLoadProfile(dic, d)
	if err != nil {
		return d, err
	}
	if len(d.AutoEvents) == 0 {
		return d, nil
	}

	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	validAutoEvents := make([]models.AutoEvent, 0, len(d.AutoEvents))
	for _, a := range d.AutoEvents {
		if err := validateAutoEvent(a, d, dp); err != nil {
			lc.Warnf("Remove auto event: %v, since it's not in Node Profile", err)
			continue
		}
		validAutoEvents = append(validAutoEvents, a)
	}
	d.AutoEvents = validAutoEvents

	return d, nil
}

func validateParentAndLoadProfile(dic *di.Container, d models.Node) (dp models.Profile, err errors.EdgeX) {
	if d.Name != "" && d.Name == d.Parent {
		return dp, errors.NewCommonEdgeX(errors.KindContractInvalid, "a node cannot be its own parent", nil)
	}

	dbClient := container.DBClientFrom(dic.Get)
	dp, err = dbClient.ProfileByName(d.ProfileName)
	if err != nil {
		return dp, errors.NewCommonEdgeX(
			errors.KindContractInvalid,
			fmt.Sprintf("node profile '%s' not found during validating node '%s'", d.ProfileName, d.Name),
			err,
		)
	}

	return dp, nil
}

func validateAutoEvent(a models.AutoEvent, d models.Node, dp models.Profile) errors.EdgeX {
	_, err := time.ParseDuration(a.Interval)
	if err != nil {
		return errors.NewCommonEdgeX(errors.KindContractInvalid, fmt.Sprintf("auto event interval '%s' not valid in the node '%s'", a.Interval, d.Name), err)
	}

	regex, regErr := regexp.CompilePOSIX(a.SourceName)
	if regErr != nil {
		return errors.NewCommonEdgeX(errors.KindContractInvalid, "failed to CompilePOSIX the auto event source name: "+a.SourceName, regErr)
	}
	hasResource := slices.ContainsFunc(dp.Resources, func(r models.Resource) bool {
		matchedString := regex.FindString(r.Name)
		return (r.Name == regex.String()) || (r.Name == matchedString)
	})

	hasCommand := slices.ContainsFunc(dp.Commands, func(c models.Command) bool {
		return c.Name == a.SourceName
	})
	if !hasResource && !hasCommand {
		return errors.NewCommonEdgeX(errors.KindContractInvalid, fmt.Sprintf("auto event source '%s' cannot be found in the node profile '%s'", a.SourceName, dp.Name), nil)
	}
	return nil
}
