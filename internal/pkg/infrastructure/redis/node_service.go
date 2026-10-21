//
// Copyright (C) 2020-2021 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"encoding/json"
	"fmt"

	pkgCommon "github.com/clint456/edgex-go/internal/pkg/common"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"

	"github.com/gomodule/redigo/redis"
)

const (
	NodeServiceCollection      = "md|ds"
	NodeServiceCollectionName  = NodeServiceCollection + DBKeySeparator + common.Name
	NodeServiceCollectionLabel = NodeServiceCollection + DBKeySeparator + common.Label
)

// nodeServiceStoredKey return the node service's stored key which combines the collection name and object id
func nodeServiceStoredKey(id string) string {
	return CreateKey(NodeServiceCollection, id)
}

// nodeServiceNameExist whether the node service exists by name
func nodeServiceNameExist(conn redis.Conn, name string) (bool, errors.EdgeX) {
	exists, err := objectNameExists(conn, NodeServiceCollectionName, name)
	if err != nil {
		return false, errors.NewCommonEdgeXWrapper(err)
	}
	return exists, nil
}

// sendAddNodeServiceCmd send redis command for adding node service
func sendAddNodeServiceCmd(conn redis.Conn, storedKey string, ds models.NodeService) errors.EdgeX {
	m, err := json.Marshal(ds)
	if err != nil {
		return errors.NewCommonEdgeX(errors.KindContractInvalid, "unable to JSON marshal node service for Redis persistence", err)
	}
	// Set the storedKey to associate with object byte array for later retrieval
	_ = conn.Send(SET, storedKey, m)
	// Store the storedKey into a Sorted Set with Modified as the score for order
	_ = conn.Send(ZADD, NodeServiceCollection, ds.Modified, storedKey)
	// Store the ds.Name into a Hash for later Name existence check
	_ = conn.Send(HSET, NodeServiceCollectionName, ds.Name, storedKey)
	for _, label := range ds.Labels { // Store the redisKey into Sorted Set of labels with Modified as the score for order
		_ = conn.Send(ZADD, CreateKey(NodeServiceCollectionLabel, label), ds.Modified, storedKey)
	}
	return nil
}

// addNodeService adds a new node service into DB
func addNodeService(conn redis.Conn, ds models.NodeService) (addedNodeService models.NodeService, edgeXerr errors.EdgeX) {
	// retrieve Node Service by Id first to ensure there is no Id conflict; when Id exists, return duplicate error
	exists, edgeXerr := objectIdExists(conn, nodeServiceStoredKey(ds.Id))
	if edgeXerr != nil {
		return addedNodeService, errors.NewCommonEdgeXWrapper(edgeXerr)
	} else if exists {
		return addedNodeService, errors.NewCommonEdgeX(errors.KindDuplicateName, fmt.Sprintf("node service id %s already exists", ds.Id), edgeXerr)
	}

	// verify if node service name is unique or not
	exists, edgeXerr = objectNameExists(conn, NodeServiceCollectionName, ds.Name)
	if edgeXerr != nil {
		return addedNodeService, errors.NewCommonEdgeXWrapper(edgeXerr)
	} else if exists {
		return addedNodeService, errors.NewCommonEdgeX(errors.KindDuplicateName, fmt.Sprintf("node service name %s already exists", ds.Name), edgeXerr)
	}

	// For Redis DB, the PUT or PATCH operation will removes the old object and add the modified one,
	// so the Created is not zero value and we shouldn't set the timestamp again.
	if ds.Created == 0 {
		ds.Created = pkgCommon.MakeTimestamp()
	}
	// query API will sort the result based on Modified, so even newly created node service shall specify Modified as Created
	ds.Modified = ds.Created

	// storedKey represents the key stored in the redis, use the format of #{NodeServiceCollection}:#{ds.Id}
	// as the redisKey to avoid data being accidentally deleted when other objects, e.g. node profiles, also
	// coincidentally have the same Id.
	storedKey := nodeServiceStoredKey(ds.Id)
	_ = conn.Send(MULTI)
	edgeXerr = sendAddNodeServiceCmd(conn, storedKey, ds)
	if edgeXerr != nil {
		return ds, errors.NewCommonEdgeXWrapper(edgeXerr)
	}
	_, err := conn.Do(EXEC)
	if err != nil {
		edgeXerr = errors.NewCommonEdgeX(errors.KindDatabaseError, "node service creation failed", err)
	}

	return ds, edgeXerr
}

// nodeServiceById query node service by id from DB
func nodeServiceById(conn redis.Conn, id string) (nodeService models.NodeService, edgeXerr errors.EdgeX) {
	edgeXerr = getObjectById(conn, nodeServiceStoredKey(id), &nodeService)
	if edgeXerr != nil {
		return nodeService, errors.NewCommonEdgeX(errors.Kind(edgeXerr), fmt.Sprintf("fail to query node service by id %s", id), edgeXerr)
	}
	return
}

// nodeServiceByName query node service by name from DB
func nodeServiceByName(conn redis.Conn, name string) (nodeService models.NodeService, edgeXerr errors.EdgeX) {
	edgeXerr = getObjectByHash(conn, NodeServiceCollectionName, name, &nodeService)
	if edgeXerr != nil {
		return nodeService, errors.NewCommonEdgeX(errors.Kind(edgeXerr), fmt.Sprintf("fail to query node service by name %s", name), edgeXerr)
	}
	return
}

// sendDeleteNodeServiceCmd send redis command for deleting node service
func sendDeleteNodeServiceCmd(conn redis.Conn, storedKey string, ds models.NodeService) {
	_ = conn.Send(DEL, storedKey)
	_ = conn.Send(ZREM, NodeServiceCollection, storedKey)
	_ = conn.Send(HDEL, NodeServiceCollectionName, ds.Name)
	for _, label := range ds.Labels {
		_ = conn.Send(ZREM, CreateKey(NodeServiceCollectionLabel, label), storedKey)
	}
}

func deleteNodeService(conn redis.Conn, ds models.NodeService) errors.EdgeX {
	storedKey := nodeServiceStoredKey(ds.Id)
	_ = conn.Send(MULTI)
	sendDeleteNodeServiceCmd(conn, storedKey, ds)
	_, err := conn.Do(EXEC)
	if err != nil {
		return errors.NewCommonEdgeX(errors.KindDatabaseError, "node service deletion failed", err)
	}
	return nil
}

// deleteNodeServiceById deletes the node service by id
func deleteNodeServiceById(conn redis.Conn, id string) errors.EdgeX {
	nodeService, err := nodeServiceById(conn, id)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	err = deleteNodeService(conn, nodeService)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	return nil
}

// deleteNodeServiceByName deletes the node service by name
func deleteNodeServiceByName(conn redis.Conn, name string) errors.EdgeX {
	nodeService, err := nodeServiceByName(conn, name)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	// Check the associated Node and ProvisionWatcher existence
	nodes, err := nodesByServiceName(conn, 0, 1, name)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	if len(nodes) > 0 {
		return errors.NewCommonEdgeX(errors.KindStatusConflict, "fail to delete the node service when associated node exists", nil)
	}
	provisionWatchers, err := provisionWatchersByServiceName(conn, 0, 1, name)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	if len(provisionWatchers) > 0 {
		return errors.NewCommonEdgeX(errors.KindStatusConflict, "fail to delete the node service when associated provisionWatcher exists", nil)
	}

	err = deleteNodeService(conn, nodeService)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	return nil
}

// nodeServicesByLabels query multiple node services from DB per labels
func nodeServicesByLabels(conn redis.Conn, offset int, limit int, labels []string) (nodeServices []models.NodeService, edgeXerr errors.EdgeX) {
	objects, err := getObjectsByLabelsAndSomeRange(conn, ZREVRANGE, NodeServiceCollection, labels, offset, limit)
	if err != nil {
		return nodeServices, errors.NewCommonEdgeXWrapper(err)
	}

	nodeServices = make([]models.NodeService, len(objects))
	for i, in := range objects {
		s := models.NodeService{}
		err := json.Unmarshal(in, &s)
		if err != nil {
			return []models.NodeService{}, errors.NewCommonEdgeX(errors.KindDatabaseError, "node service format parsing failed from the database", err)
		}
		nodeServices[i] = s
	}
	return nodeServices, nil
}

func updateNodeService(conn redis.Conn, ds models.NodeService) errors.EdgeX {
	oldNodeService, edgeXerr := nodeServiceByName(conn, ds.Name)
	if edgeXerr != nil {
		return errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	ds.Modified = pkgCommon.MakeTimestamp()
	storedKey := nodeServiceStoredKey(ds.Id)
	_ = conn.Send(MULTI)
	sendDeleteNodeServiceCmd(conn, storedKey, oldNodeService)
	edgeXerr = sendAddNodeServiceCmd(conn, storedKey, ds)
	if edgeXerr != nil {
		return errors.NewCommonEdgeXWrapper(edgeXerr)
	}
	_, err := conn.Do(EXEC)
	if err != nil {
		return errors.NewCommonEdgeX(errors.KindDatabaseError, "node service update failed", err)
	}

	return nil
}
