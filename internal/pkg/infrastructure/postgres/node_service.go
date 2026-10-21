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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	pkgCommon "github.com/clint456/edgex-go/internal/pkg/common"
	pgClient "github.com/clint456/edgex-go/internal/pkg/db/postgres"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	model "github.com/edgexfoundry/go-mod-core-contracts/v4/models"
)

// AddNodeService adds a new node service
func (c *Client) AddNodeService(ds model.NodeService) (model.NodeService, errors.EdgeX) {
	ctx := context.Background()

	if len(ds.Id) == 0 {
		ds.Id = uuid.New().String()
	}

	// verify if node service name is unique or not
	var exists bool
	exists, _ = nodeServiceNameExists(ctx, c.ConnPool, ds.Name)
	if exists {
		return model.NodeService{}, errors.NewCommonEdgeX(errors.KindDuplicateName, fmt.Sprintf("node service name %s already exists", ds.Name), nil)
	}

	timestamp := pkgCommon.MakeTimestamp()
	ds.Created = timestamp
	ds.Modified = timestamp
	// Marshal the node service to store it in the database
	nodeServiceJSONBytes, err := json.Marshal(ds)
	if err != nil {
		return model.NodeService{}, errors.NewCommonEdgeX(errors.KindContractInvalid, "unable to JSON marshal node service for Postgres persistence", err)
	}

	_, err = c.ConnPool.Exec(ctx, sqlInsert(nodeServiceTableName, idCol, contentCol), ds.Id, nodeServiceJSONBytes)
	if err != nil {
		return model.NodeService{}, pgClient.WrapDBError("failed to insert node service", err)
	}

	return ds, nil
}

// NodeServiceById gets a node service by id
func (c *Client) NodeServiceById(id string) (nodeService model.NodeService, edgeXerr errors.EdgeX) {
	ctx := context.Background()

	nodeService, err := queryOneNodeService(ctx, c.ConnPool, sqlQueryContentById(nodeServiceTableName), id)
	if err != nil {
		if stdErrs.Is(err, pgx.ErrNoRows) {
			return nodeService, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("no node service with id '%s' found", id), err)
		}
		return nodeService, pgClient.WrapDBError("failed to scan row to node service model", err)
	}
	return nodeService, nil
}

// NodeServiceByName gets a node service by name
func (c *Client) NodeServiceByName(name string) (nodeService model.NodeService, edgeXerr errors.EdgeX) {
	ctx := context.Background()

	queryObj := map[string]any{nameField: name}
	nodeService, err := queryOneNodeService(ctx, c.ConnPool, sqlQueryContentByJSONField(nodeServiceTableName), queryObj)
	if err != nil {
		if stdErrs.Is(err, pgx.ErrNoRows) {
			return nodeService, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("no node service with name '%s' found", name), err)
		}
		return nodeService, pgClient.WrapDBError("failed to scan row to node service model", err)
	}
	return nodeService, nil
}

// DeleteNodeServiceById deletes a node service by id
func (c *Client) DeleteNodeServiceById(id string) errors.EdgeX {
	ctx := context.Background()

	_, err := c.ConnPool.Exec(ctx, sqlDeleteById(nodeServiceTableName), id)
	if err != nil {
		return pgClient.WrapDBError(fmt.Sprintf("failed to delete node service by id %s", id), err)
	}
	return nil
}

// DeleteNodeServiceByName deletes a node service by name
func (c *Client) DeleteNodeServiceByName(name string) errors.EdgeX {
	ctx := context.Background()

	queryObj := map[string]any{nameField: name}
	_, err := c.ConnPool.Exec(ctx, sqlDeleteByJSONField(nodeServiceTableName), queryObj)
	if err != nil {
		return pgClient.WrapDBError(fmt.Sprintf("failed to delete node service by name %s", name), err)
	}
	return nil
}

// NodeServiceNameExists checks the node service exists by name
func (c *Client) NodeServiceNameExists(name string) (bool, errors.EdgeX) {
	ctx := context.Background()
	return nodeServiceNameExists(ctx, c.ConnPool, name)
}

// AllNodeServices returns multiple node services per query criteria, including
// offset: the number of items to skip before starting to collect the result set
// limit: The numbers of items to return
// labels: allows for querying a given object by associated user-defined labels
func (c *Client) AllNodeServices(offset int, limit int, labels []string) (nodeServices []model.NodeService, err errors.EdgeX) {
	ctx := context.Background()

	offset, validLimit := getValidOffsetAndLimit(offset, limit)
	if len(labels) > 0 {
		c.loggingClient.Debugf("Querying node services by labels: %v", labels)
		queryObj := map[string]any{labelsField: labels}
		nodeServices, err = queryNodeServices(ctx, c.ConnPool, sqlQueryContentByJSONFieldWithPaginationAsNamedArgs(nodeServiceTableName),
			pgx.NamedArgs{jsonContentCondition: queryObj, offsetCondition: offset, limitCondition: validLimit})
		if err != nil {
			return nodeServices, errors.NewCommonEdgeX(errors.Kind(err), "failed to query all node services by labels", err)
		}
	} else {
		nodeServices, err = queryNodeServices(ctx, c.ConnPool, sqlQueryContentWithPagination(nodeServiceTableName), offset, validLimit)
		if err != nil {
			return nodeServices, errors.NewCommonEdgeX(errors.Kind(err), "failed to query all node services", err)
		}
	}

	return nodeServices, nil
}

// UpdateNodeService updates a node service
func (c *Client) UpdateNodeService(ds model.NodeService) errors.EdgeX {
	ctx := context.Background()

	// Check if the node service exists
	exists, edgeXErr := nodeServiceNameExists(ctx, c.ConnPool, ds.Name)
	if edgeXErr != nil {
		return errors.NewCommonEdgeXWrapper(edgeXErr)
	} else if !exists {
		return errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("node service '%s' does not exist", ds.Name), nil)
	}

	ds.Modified = pkgCommon.MakeTimestamp()

	// Marshal the node service to store it in the database
	updatedNodeServiceJSONBytes, err := json.Marshal(ds)
	if err != nil {
		return errors.NewCommonEdgeX(errors.KindContractInvalid, "unable to JSON marshal node service for Postgres persistence", err)
	}

	queryObj := map[string]any{nameField: ds.Name}
	_, err = c.ConnPool.Exec(ctx, sqlUpdateColsByJSONCondCol(nodeServiceTableName, contentCol), updatedNodeServiceJSONBytes, queryObj)
	if err != nil {
		return pgClient.WrapDBError(fmt.Sprintf("failed to update node service by name '%s' from %s table", ds.Name, nodeServiceTableName), err)
	}

	return nil
}

// NodeServiceCountByLabels returns the total count of Node Services with labels specified.  If no label is specified, the total count of all node services will be returned.
func (c *Client) NodeServiceCountByLabels(labels []string) (int64, errors.EdgeX) {
	ctx := context.Background()

	if len(labels) > 0 {
		queryObj := map[string]any{labelsField: labels}
		return getTotalRowsCount(ctx, c.ConnPool, sqlQueryCountByJSONField(nodeServiceTableName), queryObj)
	}
	return getTotalRowsCount(ctx, c.ConnPool, sqlQueryCount(nodeServiceTableName))
}

func nodeServiceNameExists(ctx context.Context, connPool *pgxpool.Pool, name string) (bool, errors.EdgeX) {
	var exists bool
	queryObj := map[string]any{nameField: name}
	err := connPool.QueryRow(ctx, sqlCheckExistsByJSONField(nodeServiceTableName), queryObj).Scan(&exists)
	if err != nil {
		return false, pgClient.WrapDBError(fmt.Sprintf("failed to query node service by name '%s' from %s table", name, nodeServiceTableName), err)
	}
	return exists, nil
}

func queryOneNodeService(ctx context.Context, connPool *pgxpool.Pool, sql string, args ...any) (model.NodeService, errors.EdgeX) {
	var ds model.NodeService
	row := connPool.QueryRow(ctx, sql, args...)

	if err := row.Scan(&ds); err != nil {
		return ds, pgClient.WrapDBError("failed to query node service", err)
	}
	return ds, nil
}

func queryNodeServices(ctx context.Context, connPool *pgxpool.Pool, sql string, args ...any) ([]model.NodeService, errors.EdgeX) {
	rows, err := connPool.Query(ctx, sql, args...)
	if err != nil {
		return nil, pgClient.WrapDBError("failed to query node services", err)
	}

	nodeServices, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.NodeService, error) {
		var ds model.NodeService
		scanErr := row.Scan(&ds)
		return ds, scanErr
	})
	if err != nil {
		return nil, pgClient.WrapDBError("failed to collect rows to NodeService model", err)
	}

	return nodeServices, nil
}
