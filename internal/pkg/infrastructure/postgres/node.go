//
// Copyright (C) 2024 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"
	stdErrs "errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	pkgCommon "github.com/clint456/edgex-go/internal/pkg/common"
	pgClient "github.com/clint456/edgex-go/internal/pkg/db/postgres"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	model "github.com/edgexfoundry/go-mod-core-contracts/v4/models"
)

// AddNode adds a new node
func (c *Client) AddNode(d model.Node) (model.Node, errors.EdgeX) {
	ctx := context.Background()

	if len(d.Id) == 0 {
		d.Id = uuid.New().String()
	}

	exists, _ := nodeNameExists(ctx, c.ConnPool, d.Name)
	if exists {
		return model.Node{}, errors.NewCommonEdgeX(errors.KindDuplicateName, fmt.Sprintf("node name %s already exists", d.Name), nil)
	}

	timestamp := pkgCommon.MakeTimestamp()
	d.Created = timestamp
	d.Modified = timestamp
	// Marshal the node to store it in the database
	nodeJSONBytes, err := json.Marshal(d)
	if err != nil {
		return model.Node{}, errors.NewCommonEdgeX(errors.KindContractInvalid, "unable to JSON marshal node for Postgres persistence", err)
	}

	_, err = c.ConnPool.Exec(ctx, sqlInsert(nodeTableName, idCol, contentCol), d.Id, nodeJSONBytes)
	if err != nil {
		return model.Node{}, pgClient.WrapDBError("failed to insert node", err)
	}

	return d, nil
}

// DeleteNodeById deletes a node by id
func (c *Client) DeleteNodeById(id string) errors.EdgeX {
	ctx := context.Background()

	_, err := c.ConnPool.Exec(ctx, sqlDeleteById(nodeTableName), id)
	if err != nil {
		return pgClient.WrapDBError(fmt.Sprintf("failed to delete node by id %s", id), err)
	}
	return nil
}

// DeleteNodeByName deletes a node by name
func (c *Client) DeleteNodeByName(name string) errors.EdgeX {
	ctx := context.Background()

	queryObj := map[string]any{nameField: name}
	_, err := c.ConnPool.Exec(ctx, sqlDeleteByJSONField(nodeTableName), queryObj)
	if err != nil {
		return pgClient.WrapDBError(fmt.Sprintf("failed to delete node by name %s", name), err)
	}
	return nil
}

// NodesByServiceName query nodes by offset, limit and name
func (c *Client) NodesByServiceName(offset int, limit int, name string) ([]model.Node, errors.EdgeX) {
	ctx := context.Background()
	offset, validLimit := getValidOffsetAndLimit(offset, limit)
	queryObj := map[string]any{serviceNameField: name}
	return queryNodes(ctx, c.ConnPool, sqlQueryContentByJSONFieldWithPaginationAsNamedArgs(nodeTableName),
		pgx.NamedArgs{jsonContentCondition: queryObj, offsetCondition: offset, limitCondition: validLimit})
}

// NodeIdExists checks the node existence by id
func (c *Client) NodeIdExists(id string) (bool, errors.EdgeX) {
	ctx := context.Background()
	return nodeIdExists(ctx, c.ConnPool, id)
}

// NodeNameExists checks the node existence by name
func (c *Client) NodeNameExists(name string) (bool, errors.EdgeX) {
	ctx := context.Background()
	return nodeNameExists(ctx, c.ConnPool, name)
}

// NodeById gets a node by id
func (c *Client) NodeById(id string) (model.Node, errors.EdgeX) {
	ctx := context.Background()

	d, err := queryOneNode(ctx, c.ConnPool, sqlQueryContentById(nodeTableName), id)
	if err != nil {
		if stdErrs.Is(err, pgx.ErrNoRows) {
			return d, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("no node with id '%s' found", id), err)
		}
		return d, pgClient.WrapDBError("failed to scan row to node model", err)
	}
	return d, nil
}

// NodeByName gets a node by name
func (c *Client) NodeByName(name string) (model.Node, errors.EdgeX) {
	ctx := context.Background()

	queryObj := map[string]any{nameField: name}
	d, err := queryOneNode(ctx, c.ConnPool, sqlQueryContentByJSONField(nodeTableName), queryObj)
	if err != nil {
		if stdErrs.Is(err, pgx.ErrNoRows) {
			return d, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("no node with name '%s' found", name), err)
		}
		return d, pgClient.WrapDBError("failed to scan row to node model", err)
	}
	return d, nil
}

// AllNodes query the nodes with offset, limit, and labels
func (c *Client) AllNodes(offset int, limit int, labels []string) (nodes []model.Node, err errors.EdgeX) {
	ctx := context.Background()

	offset, validLimit := getValidOffsetAndLimit(offset, limit)
	if len(labels) > 0 {
		c.loggingClient.Debugf("Querying nodes by labels: %v", labels)
		queryObj := map[string]any{labelsField: labels}
		nodes, err = queryNodes(ctx, c.ConnPool, sqlQueryContentByJSONFieldWithPaginationAsNamedArgs(nodeTableName),
			pgx.NamedArgs{jsonContentCondition: queryObj, offsetCondition: offset, limitCondition: validLimit})
		if err != nil {
			return nodes, errors.NewCommonEdgeX(errors.Kind(err), "failed to query all nodes by labels", err)
		}
	} else {
		nodes, err = queryNodes(ctx, c.ConnPool, sqlQueryContentWithPaginationAsNamedArgs(nodeTableName),
			pgx.NamedArgs{offsetCondition: offset, limitCondition: validLimit})
		if err != nil {
			return nodes, errors.NewCommonEdgeX(errors.Kind(err), "failed to query all nodes", err)
		}
	}

	return nodes, nil
}

// NodesByProfileName query nodes by offset, limit and profile name
func (c *Client) NodesByProfileName(offset int, limit int, profileName string) ([]model.Node, errors.EdgeX) {
	ctx := context.Background()
	offset, validLimit := getValidOffsetAndLimit(offset, limit)
	queryObj := map[string]any{profileNameField: profileName}
	return queryNodes(ctx, c.ConnPool, sqlQueryContentByJSONFieldWithPaginationAsNamedArgs(nodeTableName),
		pgx.NamedArgs{jsonContentCondition: queryObj, offsetCondition: offset, limitCondition: validLimit})
}

// UpdateNode updates a node
func (c *Client) UpdateNode(d model.Node) errors.EdgeX {
	ctx := context.Background()

	// Check if the node exists
	exists, edgeXErr := nodeNameExists(ctx, c.ConnPool, d.Name)
	if edgeXErr != nil {
		return errors.NewCommonEdgeXWrapper(edgeXErr)
	} else if !exists {
		return errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("node '%s' does not exist", d.Name), nil)
	}

	d.Modified = pkgCommon.MakeTimestamp()

	// Marshal the node to store it in the database
	updatedNodeJSONBytes, err := json.Marshal(d)
	if err != nil {
		return errors.NewCommonEdgeX(errors.KindContractInvalid, "unable to JSON marshal node for Postgres persistence", err)
	}

	queryObj := map[string]any{nameField: d.Name}
	_, err = c.ConnPool.Exec(ctx, sqlUpdateColsByJSONCondCol(nodeTableName, contentCol), updatedNodeJSONBytes, queryObj)
	if err != nil {
		return pgClient.WrapDBError(fmt.Sprintf("failed to update node by name '%s' from %s table", d.Name, nodeTableName), err)
	}

	return nil
}

// NodeCountByLabels returns the total count of Nodes with labels specified.  If no label is specified, the total count of all nodes will be returned.
func (c *Client) NodeCountByLabels(labels []string) (int64, errors.EdgeX) {
	ctx := context.Background()

	if len(labels) > 0 {
		queryObj := map[string]any{labelsField: labels}
		return getTotalRowsCount(ctx, c.ConnPool, sqlQueryCountByJSONField(nodeTableName), queryObj)
	}
	return getTotalRowsCount(ctx, c.ConnPool, sqlQueryCount(nodeTableName))
}

// NodeCountByProfileName returns the count of Nodes associated with specified profile
func (c *Client) NodeCountByProfileName(profileName string) (int64, errors.EdgeX) {
	ctx := context.Background()
	queryObj := map[string]any{profileNameField: profileName}
	return getTotalRowsCount(ctx, c.ConnPool, sqlQueryCountByJSONField(nodeTableName), queryObj)
}

// NodeCountByServiceName returns the count of Nodes associated with specified service
func (c *Client) NodeCountByServiceName(serviceName string) (int64, errors.EdgeX) {
	ctx := context.Background()
	queryObj := map[string]any{serviceNameField: serviceName}
	return getTotalRowsCount(ctx, c.ConnPool, sqlQueryCountByJSONField(nodeTableName), queryObj)
}

// Get node objects with matching parent and labels (one level of the tree).
func nodeTreeLevel(ctx context.Context, connPool *pgxpool.Pool, parent string, labels []string) ([]model.Node, errors.EdgeX) {
	queryObj := map[string]any{parentField: parent}
	if len(labels) != 0 {
		queryObj[labelsField] = labels
	}
	return queryNodes(ctx, connPool, sqlQueryContentByJSONField(nodeTableName), queryObj)
}

// Get the entire subtree starting with the given parent, descending at most the given number of levels.
func nodeSubTree(ctx context.Context, connPool *pgxpool.Pool, parent string, levels int, labels []string) ([]model.Node, errors.EdgeX) {
	var emptyList = []model.Node{}
	if levels <= 0 {
		return emptyList, nil
	}
	topLevelList, err := nodeTreeLevel(ctx, connPool, parent, labels)
	if err != nil {
		return emptyList, err
	}
	if levels == 1 {
		return topLevelList, nil
	}
	var subtreesAtThisLevel []model.Node
	for _, node := range topLevelList {
		if node.Name == node.Parent {
			message := "Node " + node.Name + " is its own parent, stopping tree query"
			return emptyList, errors.NewCommonEdgeX(errors.KindDatabaseError, message, nil)
		}
		subtree, err := nodeSubTree(ctx, connPool, node.Name, levels-1, labels)
		if err != nil {
			return emptyList, err
		}
		subtreesAtThisLevel = append(subtreesAtThisLevel, subtree...)
	}
	return append(topLevelList, subtreesAtThisLevel...), nil
}

// Get the full result-set since that's the only way to correctly get totalCount.
// Then return the subset of the result-set that corresponds to the requested offset and limit.
func (c *Client) NodeTree(parent string, levels int, offset int, limit int, labels []string) (int64, []model.Node, errors.EdgeX) {
	var maxLevels int
	var emptyList = []model.Node{}
	if levels <= 0 {
		maxLevels = math.MaxInt
	} else {
		maxLevels = levels
	}
	allNodes, err := nodeSubTree(context.Background(), c.ConnPool, parent, maxLevels, labels)
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

func nodeNameExists(ctx context.Context, connPool *pgxpool.Pool, name string) (bool, errors.EdgeX) {
	var exists bool
	queryObj := map[string]any{nameField: name}
	err := connPool.QueryRow(ctx, sqlCheckExistsByJSONField(nodeTableName), queryObj).Scan(&exists)
	if err != nil {
		return false, pgClient.WrapDBError(fmt.Sprintf("failed to query node by name '%s' from %s table", name, nodeTableName), err)
	}
	return exists, nil
}

func nodeIdExists(ctx context.Context, connPool *pgxpool.Pool, id string) (bool, errors.EdgeX) {
	var exists bool
	err := connPool.QueryRow(ctx, sqlCheckExistsByCol(nodeTableName, idCol), id).Scan(&exists)
	if err != nil {
		return false, pgClient.WrapDBError(fmt.Sprintf("failed to query node by id '%s' from %s table", id, nodeTableName), err)
	}
	return exists, nil
}

func queryOneNode(ctx context.Context, connPool *pgxpool.Pool, sql string, args ...any) (model.Node, errors.EdgeX) {
	var d model.Node
	row := connPool.QueryRow(ctx, sql, args...)

	if err := row.Scan(&d); err != nil {
		return d, pgClient.WrapDBError("failed to query node", err)
	}
	return d, nil
}

func queryNodes(ctx context.Context, connPool *pgxpool.Pool, sql string, args ...any) ([]model.Node, errors.EdgeX) {
	rows, err := connPool.Query(ctx, sql, args...)
	if err != nil {
		return nil, pgClient.WrapDBError("failed to query nodes", err)
	}

	nodes, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Node, error) {
		var d model.Node
		scanErr := row.Scan(&d)
		return d, scanErr
	})
	if err != nil {
		return nil, pgClient.WrapDBError("failed to collect rows to Node model", err)
	}

	return nodes, nil
}
