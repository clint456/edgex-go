//
// Copyright (C) 2020-2024 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"encoding/json"
	"fmt"
	"math"

	pkgCommon "github.com/clint456/edgex-go/internal/pkg/common"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"

	"github.com/gomodule/redigo/redis"
)

const (
	NodeCollection            = "md|dv"
	NodeCollectionName        = NodeCollection + DBKeySeparator + common.Name
	NodeCollectionLabel       = NodeCollection + DBKeySeparator + common.Label
	NodeCollectionParent      = NodeCollection + DBKeySeparator + "parent"
	NodeCollectionServiceName = NodeCollection + DBKeySeparator + common.Service + DBKeySeparator + common.Name
	NodeCollectionProfileName = NodeCollection + DBKeySeparator + common.Profile + DBKeySeparator + common.Name
)

// nodeStoredKey return the node's stored key which combines the collection name and object id
func nodeStoredKey(id string) string {
	return CreateKey(NodeCollection, id)
}

// nodeNameExists whether the node exists by name
func nodeNameExists(conn redis.Conn, name string) (bool, errors.EdgeX) {
	exists, err := objectNameExists(conn, NodeCollectionName, name)
	if err != nil {
		return false, errors.NewCommonEdgeX(errors.KindDatabaseError, "node existence check by name failed", err)
	}
	return exists, nil
}

// nodeIdExists checks whether the node exists by id
func nodeIdExists(conn redis.Conn, id string) (bool, errors.EdgeX) {
	exists, err := objectIdExists(conn, nodeStoredKey(id))
	if err != nil {
		return false, errors.NewCommonEdgeX(errors.KindDatabaseError, "node existence check by id failed", err)
	}
	return exists, nil
}

// sendAddNodeCmd send redis command for adding node
func sendAddNodeCmd(conn redis.Conn, storedKey string, d models.Node) errors.EdgeX {
	dsJSONBytes, err := json.Marshal(d)
	if err != nil {
		return errors.NewCommonEdgeX(errors.KindContractInvalid, "unable to JSON marshal node for Redis persistence", err)
	}
	_ = conn.Send(SET, storedKey, dsJSONBytes)
	_ = conn.Send(ZADD, NodeCollection, 0, storedKey)
	_ = conn.Send(HSET, NodeCollectionName, d.Name, storedKey)
	_ = conn.Send(ZADD, CreateKey(NodeCollectionServiceName, d.ServiceName), d.Modified, storedKey)
	_ = conn.Send(ZADD, CreateKey(NodeCollectionProfileName, d.ProfileName), d.Modified, storedKey)
	for _, label := range d.Labels {
		_ = conn.Send(ZADD, CreateKey(NodeCollectionLabel, label), d.Modified, storedKey)
	}
	if d.Parent != "" {
		_ = conn.Send(ZADD, CreateKey(NodeCollectionParent, d.Parent), d.Modified, storedKey)
	}
	return nil
}

// addNode adds a new node into DB
func addNode(conn redis.Conn, d models.Node) (models.Node, errors.EdgeX) {
	var exists bool
	var edgeXerr errors.EdgeX
	if d.ProfileName != "" {
		exists, edgeXerr = profileNameExists(conn, d.ProfileName)
		if edgeXerr != nil {
			return d, errors.NewCommonEdgeXWrapper(edgeXerr)
		}
		if !exists {
			return d, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("node profile '%s' does not exists", d.ProfileName), nil)
		}
	}

	exists, edgeXerr = nodeIdExists(conn, d.Id)
	if edgeXerr != nil {
		return d, errors.NewCommonEdgeXWrapper(edgeXerr)
	} else if exists {
		return d, errors.NewCommonEdgeX(errors.KindDuplicateName, fmt.Sprintf("node id %s already exists", d.Id), edgeXerr)
	}

	exists, edgeXerr = nodeNameExists(conn, d.Name)
	if edgeXerr != nil {
		return d, errors.NewCommonEdgeXWrapper(edgeXerr)
	} else if exists {
		return d, errors.NewCommonEdgeX(errors.KindDuplicateName, fmt.Sprintf("node name %s already exists", d.Name), edgeXerr)
	}

	ts := pkgCommon.MakeTimestamp()
	if d.Created == 0 {
		d.Created = ts
	}
	d.Modified = ts

	storedKey := nodeStoredKey(d.Id)
	_ = conn.Send(MULTI)
	edgeXerr = sendAddNodeCmd(conn, storedKey, d)
	if edgeXerr != nil {
		return d, errors.NewCommonEdgeXWrapper(edgeXerr)
	}
	_, err := conn.Do(EXEC)
	if err != nil {
		edgeXerr = errors.NewCommonEdgeX(errors.KindDatabaseError, "node creation failed", err)
	}

	return d, edgeXerr
}

// nodeById query node by id from DB
func nodeById(conn redis.Conn, id string) (node models.Node, edgeXerr errors.EdgeX) {
	edgeXerr = getObjectById(conn, nodeStoredKey(id), &node)
	if edgeXerr != nil {
		return node, errors.NewCommonEdgeXWrapper(edgeXerr)
	}
	return
}

// nodeByName query node by name from DB
func nodeByName(conn redis.Conn, name string) (node models.Node, edgeXerr errors.EdgeX) {
	edgeXerr = getObjectByHash(conn, NodeCollectionName, name, &node)
	if edgeXerr != nil {
		return node, errors.NewCommonEdgeXWrapper(edgeXerr)
	}
	return
}

// deleteNodeById deletes the node by id
func deleteNodeById(conn redis.Conn, id string) errors.EdgeX {
	node, err := nodeById(conn, id)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	err = deleteNode(conn, node)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	return nil
}

// deleteNodeByName deletes the node by name
func deleteNodeByName(conn redis.Conn, name string) errors.EdgeX {
	node, err := nodeByName(conn, name)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	err = deleteNode(conn, node)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	return nil
}

// sendDeleteNodeCmd send redis command for deleting node
func sendDeleteNodeCmd(conn redis.Conn, storedKey string, node models.Node) {
	_ = conn.Send(DEL, storedKey)
	_ = conn.Send(ZREM, NodeCollection, storedKey)
	_ = conn.Send(HDEL, NodeCollectionName, node.Name)
	_ = conn.Send(ZREM, CreateKey(NodeCollectionServiceName, node.ServiceName), storedKey)
	_ = conn.Send(ZREM, CreateKey(NodeCollectionProfileName, node.ProfileName), storedKey)
	for _, label := range node.Labels {
		_ = conn.Send(ZREM, CreateKey(NodeCollectionLabel, label), storedKey)
	}
	if node.Parent != "" {
		_ = conn.Send(ZREM, CreateKey(NodeCollectionParent, node.Parent), storedKey)
	}
}

// deleteNode deletes a node
func deleteNode(conn redis.Conn, node models.Node) errors.EdgeX {
	numChildren, edgexErr := getMemberNumber(conn, ZCARD, CreateKey(NodeCollectionParent, node.Name))
	if edgexErr != nil {
		return errors.NewCommonEdgeX(errors.KindDatabaseError, "Could not determine if node had any children", edgexErr)
	}
	if numChildren > 0 {
		return errors.NewCommonEdgeX(errors.KindStatusConflict, "Cannot delete node, it has child nodes", nil)
	}
	storedKey := nodeStoredKey(node.Id)
	_ = conn.Send(MULTI)
	sendDeleteNodeCmd(conn, storedKey, node)
	_, err := conn.Do(EXEC)
	if err != nil {
		return errors.NewCommonEdgeX(errors.KindDatabaseError, "node deletion failed", err)
	}
	return nil
}

// nodesByServiceName query nodes by offset, limit and name
func nodesByServiceName(conn redis.Conn, offset int, limit int, name string) (nodes []models.Node, edgeXerr errors.EdgeX) {
	objects, err := getObjectsByRevRange(conn, CreateKey(NodeCollectionServiceName, name), offset, limit)
	if err != nil {
		return nodes, errors.NewCommonEdgeXWrapper(err)
	}

	nodes = make([]models.Node, len(objects))
	for i, in := range objects {
		s := models.Node{}
		err := json.Unmarshal(in, &s)
		if err != nil {
			return []models.Node{}, errors.NewCommonEdgeX(errors.KindDatabaseError, "node format parsing failed from the database", err)
		}
		nodes[i] = s
	}
	return nodes, nil
}

// nodesByLabels query nodes with offset, limit and labels
func nodesByLabels(conn redis.Conn, offset int, limit int, labels []string) (nodes []models.Node, edgeXerr errors.EdgeX) {
	objects, edgeXerr := getObjectsByLabelsAndSomeRange(conn, ZREVRANGE, NodeCollection, labels, offset, limit)
	if edgeXerr != nil {
		return nodes, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	nodes = make([]models.Node, len(objects))
	for i, in := range objects {
		dp := models.Node{}
		err := json.Unmarshal(in, &dp)
		if err != nil {
			return []models.Node{}, errors.NewCommonEdgeX(errors.KindDatabaseError, "node format parsing failed from the database", err)
		}
		nodes[i] = dp
	}
	return nodes, nil
}

// nodesByProfileName query nodes by offset, limit and profile name
func nodesByProfileName(conn redis.Conn, offset int, limit int, profileName string) (nodes []models.Node, edgeXerr errors.EdgeX) {
	objects, err := getObjectsByRevRange(conn, CreateKey(NodeCollectionProfileName, profileName), offset, limit)
	if err != nil {
		return nodes, errors.NewCommonEdgeXWrapper(err)
	}

	nodes = make([]models.Node, len(objects))
	for i, in := range objects {
		s := models.Node{}
		err := json.Unmarshal(in, &s)
		if err != nil {
			return []models.Node{}, errors.NewCommonEdgeX(errors.KindDatabaseError, "node format parsing failed from the database", err)
		}
		nodes[i] = s
	}
	return nodes, nil
}

func updateNode(conn redis.Conn, d models.Node) errors.EdgeX {
	if d.ProfileName != "" {
		exists, edgeXerr := profileNameExists(conn, d.ProfileName)
		if edgeXerr != nil {
			return errors.NewCommonEdgeX(errors.Kind(edgeXerr), fmt.Sprintf("node profile '%s' existence check failed", d.ProfileName), edgeXerr)
		} else if !exists {
			return errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("node profile '%s' does not exists", d.ProfileName), nil)
		}
	}

	oldNode, edgexErr := nodeByName(conn, d.Name)
	if edgexErr != nil {
		return errors.NewCommonEdgeXWrapper(edgexErr)
	}

	ts := pkgCommon.MakeTimestamp()
	d.Modified = ts

	storedKey := nodeStoredKey(d.Id)
	_ = conn.Send(MULTI)
	sendDeleteNodeCmd(conn, storedKey, oldNode)
	edgexErr = sendAddNodeCmd(conn, storedKey, d)
	if edgexErr != nil {
		return errors.NewCommonEdgeXWrapper(edgexErr)
	}
	_, err := conn.Do(EXEC)
	if err != nil {
		return errors.NewCommonEdgeX(errors.KindDatabaseError, "node update failed", err)
	}

	return nil
}

// Return all nodes with the given parent and labels (one level of the tree).
func nodeTreeLevel(conn redis.Conn, parent string, labels []string) ([]models.Node, errors.EdgeX) {
	queryList := []string{CreateKey(NodeCollectionParent, parent)}
	for l := range labels {
		queryList = append(queryList, CreateKey(NodeCollectionLabel, labels[l]))
	}
	objects, err := intersectionObjectsByKeys(conn, 0, -1, queryList...)
	if err != nil {
		return []models.Node{}, errors.NewCommonEdgeXWrapper(err)
	}
	nodes := make([]models.Node, len(objects))
	for i, in := range objects {
		s := models.Node{}
		err := json.Unmarshal(in, &s)
		if err != nil {
			return []models.Node{}, errors.NewCommonEdgeX(errors.KindDatabaseError, "node format parsing failed from the database", err)
		}
		if s.Name == s.Parent {
			message := "Node " + s.Name + " is its own parent, stopping this query"
			return []models.Node{}, errors.NewCommonEdgeX(errors.KindDatabaseError, message, nil)
		}
		nodes[i] = s
	}
	return nodes, nil
}

// Get the entire subtree starting with the given parent, descending at most the given number of levels.
func nodeSubTree(conn redis.Conn, parent string, levels int, labels []string) ([]models.Node, errors.EdgeX) {
	if levels == 0 {
		return []models.Node{}, nil
	}
	nodes, err := nodeTreeLevel(conn, parent, labels)
	if err != nil {
		return []models.Node{}, errors.NewCommonEdgeXWrapper(err)
	}
	if levels == 1 {
		return nodes, nil
	}
	for i := range nodes {
		subNodes, err := nodeSubTree(conn, nodes[i].Name, levels-1, labels)
		if err != nil {
			return []models.Node{}, errors.NewCommonEdgeXWrapper(err)
		}
		nodes = append(nodes, subNodes...)
	}
	return nodes, nil
}

// Get the full result-set since that's the only way to correctly get totalCount.
// Then return the subset of the result-set that corresponds to the requested offset and limit.
func nodeTree(conn redis.Conn, parent string, levels int, offset int, limit int, labels []string) (int64, []models.Node, errors.EdgeX) {
	var maxLevels int
	var emptyList = []models.Node{}
	if levels <= 0 {
		maxLevels = math.MaxInt
	} else {
		maxLevels = levels
	}
	allNodes, err := nodeSubTree(conn, parent, maxLevels, labels)
	if err != nil {
		return 0, emptyList, err
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(allNodes) {
		return int64(len(allNodes)), emptyList, nil
	}
	numToReturn := len(allNodes) - offset
	if limit > 0 && limit < numToReturn {
		numToReturn = limit
	}
	return int64(len(allNodes)), allNodes[offset : offset+numToReturn], nil
}
