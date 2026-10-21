//
// Copyright (C) 2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/clint456/edgex-go/internal/core/data/container"
	pgClient "github.com/clint456/edgex-go/internal/pkg/db/postgres"
	dbModels "github.com/clint456/edgex-go/internal/pkg/infrastructure/models"
	"github.com/clint456/edgex-go/internal/pkg/infrastructure/postgres/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	model "github.com/edgexfoundry/go-mod-core-contracts/v4/models"
)

// AllNodeInfos query all nodeInfos
func (c *Client) AllNodeInfos(offset int, limit int) ([]dbModels.NodeInfo, errors.EdgeX) {
	sqlStmt := sqlQueryAllWithPaginationAsNamedArgs(nodeInfoTableName)
	offset, validLimit := getValidOffsetAndLimit(offset, limit)
	rows, err := c.ConnPool.Query(context.Background(), sqlStmt, pgx.NamedArgs{offsetCondition: offset, limitCondition: validLimit})
	if err != nil {
		return nil, pgClient.WrapDBError("failed to query nodeInfos", err)
	}
	nodeInfos, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (dbModels.NodeInfo, error) {
		d, scanErr := pgx.RowToStructByNameLax[dbModels.NodeInfo](row)
		if err != nil {
			return dbModels.NodeInfo{}, err
		}
		return d, scanErr
	})
	if err != nil {
		return nil, pgClient.WrapDBError("failed to collect rows to NodeInfo model", err)
	}
	return nodeInfos, nil
}

func (c *Client) nodeInfoIdByEvent(e model.Event) (int, errors.EdgeX) {
	nodeInfo := dbModels.NodeInfo{
		NodeName:    e.NodeName,
		ProfileName: e.ProfileName,
		SourceName:  e.SourceName,
		Tags:        e.Tags,
	}
	return c.nodeInfoIdFromCache(nodeInfo)
}

func (c *Client) nodeInfoIdByReading(r models.Reading) (int, errors.EdgeX) {
	nodeInfo := dbModels.NodeInfo{
		NodeName:     r.NodeName,
		ProfileName:  r.ProfileName,
		ResourceName: r.ResourceName,
		ValueType:    r.ValueType,
		Units:        r.Units,
		Tags:         r.Tags,
	}
	if r.MediaType != nil {
		nodeInfo.MediaType = *r.MediaType
	}
	return c.nodeInfoIdFromCache(nodeInfo)
}

func (c *Client) nodeInfoIdFromCache(nodeInfo dbModels.NodeInfo) (int, errors.EdgeX) {
	nodeInfoCache := container.NodeInfoCacheFrom(c.dic.Get)
	var exists bool
	var err error
	// get nodeInfo id from the cache
	id, exists := nodeInfoCache.GetNodeInfoId(nodeInfo)
	if !exists {
		// get nodeInfo id from the DB
		id, err = c.nodeInfoId(nodeInfo)
		if err != nil && errors.Kind(err) == errors.KindEntityDoesNotExist {
			// add new nodeInfo if not exist in the DB
			id, err = c.addNodeInfo(nodeInfo)
			if err != nil {
				return 0, errors.NewCommonEdgeXWrapper(err)
			}
			nodeInfo.Id = id

		} else if err != nil {
			return 0, errors.NewCommonEdgeXWrapper(err)
		}
		nodeInfo.Id = id
		// keep in the cache
		nodeInfoCache.Add(nodeInfo)
	}
	return id, nil
}

func (c *Client) nodeInfoId(nodeInfo dbModels.NodeInfo) (int, errors.EdgeX) {
	var id int
	tagsBytes, err := json.Marshal(nodeInfo.Tags)
	if err != nil {
		return 0, errors.NewCommonEdgeX(errors.KindServerError, "unable to JSON marshal nodeInfo tags", err)
	}
	row := c.ConnPool.QueryRow(
		context.Background(),
		sqlQueryFieldsByCol(
			nodeInfoTableName,
			[]string{idCol},
			nodeNameCol, profileNameCol, sourceNameCol, tagsCol,
			resourceNameCol, valueTypeCol, unitsCol, mediaTypeCol),
		nodeInfo.NodeName, nodeInfo.ProfileName, nodeInfo.SourceName, tagsBytes,
		nodeInfo.ResourceName, nodeInfo.ValueType, nodeInfo.Units, nodeInfo.MediaType,
	)

	if err := row.Scan(&id); err != nil {
		return 0, pgClient.WrapDBError("failed to query nodeInfo", err)
	}
	return id, nil
}

// addNodeInfo adds a new nodeInfo
func (c *Client) addNodeInfo(nodeInfo dbModels.NodeInfo) (int, errors.EdgeX) {
	tagsBytes, err := json.Marshal(nodeInfo.Tags)
	if err != nil {
		return 0, errors.NewCommonEdgeX(errors.KindServerError, "unable to JSON marshal nodeInfo tags", err)
	}

	var id int
	err = c.ConnPool.QueryRow(context.Background(),
		sqlInsert(
			nodeInfoTableName, nodeNameCol, profileNameCol, sourceNameCol, tagsCol,
			resourceNameCol, valueTypeCol, unitsCol, mediaTypeCol,
		)+" RETURNING id",
		nodeInfo.NodeName, nodeInfo.ProfileName, nodeInfo.SourceName, tagsBytes,
		nodeInfo.ResourceName, nodeInfo.ValueType, nodeInfo.Units, nodeInfo.MediaType,
	).Scan(&id)
	if err != nil {
		return 0, pgClient.WrapDBError("failed to insert nodeInfo", err)
	}
	return id, nil
}

// nodeInfosByConds query nodeInfos by specified conditions
func (c *Client) nodeInfosByConds(cols []string, values pgx.NamedArgs) ([]dbModels.NodeInfo, errors.EdgeX) {
	sqlStmt := sqlQueryAllWithNamedArgConds(nodeInfoTableName, cols...)
	rows, err := c.ConnPool.Query(context.Background(), sqlStmt, values)
	if err != nil {
		return nil, pgClient.WrapDBError("failed to query nodeInfos", err)
	}
	nodeInfos, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (dbModels.NodeInfo, error) {
		d, scanErr := pgx.RowToStructByNameLax[dbModels.NodeInfo](row)
		if err != nil {
			return dbModels.NodeInfo{}, err
		}
		return d, scanErr
	})
	if err != nil {
		return nil, pgClient.WrapDBError("failed to collect rows to NodeInfo model", err)
	}
	return nodeInfos, nil
}

// deleteNodeInfoById deletes a nodeInfo by id
// only calling this func when corresponding events/readings have been removed, otherwise may
// potentially break referential integrity between events/readings/node_info records
func deleteNodeInfoById(ctx context.Context, tx pgx.Tx, id int) errors.EdgeX {
	_, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s = @%s", nodeInfoTableName, idCol, idCol), pgx.NamedArgs{idCol: id})
	if err != nil {
		return pgClient.WrapDBError(fmt.Sprintf("failed to delete nodeInfo by id %d", id), err)
	}
	return nil
}

// updateNodeInfosDeletableByNodeName updates nodeInfos by nodeName as deletable
func (c *Client) updateNodeInfosDeletableByNodeName(nodeName string) errors.EdgeX {
	_, err := c.ConnPool.Exec(context.Background(), fmt.Sprintf("UPDATE %s SET %s = true WHERE %s=$1", nodeInfoTableName, markDeletedCol, nodeNameCol), nodeName)
	if err != nil {
		return pgClient.WrapDBError(fmt.Sprintf("update %s to deletable by nodename '%s'", nodeInfoTableName, nodeName), err)
	}
	return nil
}
