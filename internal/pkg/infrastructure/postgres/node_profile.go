//
// Copyright (C) 2024-2025 IOTech Ltd
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

// AddProfile adds a new node profile
func (c *Client) AddProfile(dp model.Profile) (model.Profile, errors.EdgeX) {
	ctx := context.Background()

	if len(dp.Id) == 0 {
		dp.Id = uuid.New().String()
	}

	// verify if node profile name is unique or not
	var exists bool
	exists, _ = profileNameExists(ctx, c.ConnPool, dp.Name)
	if exists {
		return model.Profile{}, errors.NewCommonEdgeX(errors.KindDuplicateName, fmt.Sprintf("node profile name %s already exists", dp.Name), nil)
	}

	timestamp := pkgCommon.MakeTimestamp()
	dp.Created = timestamp
	dp.Modified = timestamp
	// Marshal the node profile to store it in the database
	profileJSONBytes, err := json.Marshal(dp)
	if err != nil {
		return model.Profile{}, errors.NewCommonEdgeX(errors.KindContractInvalid, "unable to JSON marshal node profile for Postgres persistence", err)
	}

	_, err = c.ConnPool.Exec(ctx, sqlInsert(profileTableName, idCol, contentCol), dp.Id, profileJSONBytes)
	if err != nil {
		return model.Profile{}, pgClient.WrapDBError("failed to insert node profile", err)
	}

	return dp, nil
}

// UpdateProfile updates a new node profile
func (c *Client) UpdateProfile(dp model.Profile) errors.EdgeX {
	ctx := context.Background()

	// Check if the node profile exists
	exists, edgeXErr := profileNameExists(ctx, c.ConnPool, dp.Name)
	if edgeXErr != nil {
		return errors.NewCommonEdgeXWrapper(edgeXErr)
	} else if !exists {
		return errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("node profile '%s' does not exist", dp.Name), nil)
	}

	dp.Modified = pkgCommon.MakeTimestamp()

	// Marshal the node profile to store it in the database
	updatedProfileJSONBytes, err := json.Marshal(dp)
	if err != nil {
		return errors.NewCommonEdgeX(errors.KindContractInvalid, "unable to JSON marshal node profile for Postgres persistence", err)
	}

	queryObj := map[string]any{nameField: dp.Name}
	_, err = c.ConnPool.Exec(ctx, sqlUpdateColsByJSONCondCol(profileTableName, contentCol), updatedProfileJSONBytes, queryObj)
	if err != nil {
		return pgClient.WrapDBError(fmt.Sprintf("failed to update node profile by name '%s' from %s table", dp.Name, profileTableName), err)
	}

	return nil
}

// ProfileById gets a node profile by id
func (c *Client) ProfileById(id string) (model.Profile, errors.EdgeX) {
	ctx := context.Background()

	dp, err := queryOneProfile(ctx, c.ConnPool, sqlQueryContentById(profileTableName), id)
	if err != nil {
		if stdErrs.Is(err, pgx.ErrNoRows) {
			return dp, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("no node profile with id '%s' found", id), err)
		}
		return dp, pgClient.WrapDBError("failed to scan row to node profile model", err)
	}
	return dp, nil
}

// ProfileByName gets a node profile by name
func (c *Client) ProfileByName(name string) (model.Profile, errors.EdgeX) {
	ctx := context.Background()

	queryObj := map[string]any{nameField: name}
	dp, err := queryOneProfile(ctx, c.ConnPool, sqlQueryContentByJSONField(profileTableName), queryObj)
	if err != nil {
		if stdErrs.Is(err, pgx.ErrNoRows) {
			return dp, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("no node profile with name '%s' found", name), err)
		}
		return dp, pgClient.WrapDBError("failed to scan row to node profile model", err)
	}
	return dp, nil
}

// DeleteProfileById deletes a node profile by id
func (c *Client) DeleteProfileById(id string) errors.EdgeX {
	ctx := context.Background()

	_, err := c.ConnPool.Exec(ctx, sqlDeleteById(profileTableName), id)
	if err != nil {
		return pgClient.WrapDBError(fmt.Sprintf("failed to delete node profile by id %s", id), err)
	}
	return nil
}

// DeleteProfileByName deletes a node profile by name
func (c *Client) DeleteProfileByName(name string) errors.EdgeX {
	ctx := context.Background()

	queryObj := map[string]any{nameField: name}
	_, err := c.ConnPool.Exec(ctx, sqlDeleteByJSONField(profileTableName), queryObj)
	if err != nil {
		return pgClient.WrapDBError(fmt.Sprintf("failed to delete node profile by name %s", name), err)
	}
	return nil
}

// ProfileNameExists checks the node profile exists by name
func (c *Client) ProfileNameExists(name string) (bool, errors.EdgeX) {
	ctx := context.Background()
	return profileNameExists(ctx, c.ConnPool, name)
}

// AllProfiles query node profiles with offset, limit and labels
func (c *Client) AllProfiles(offset int, limit int, labels []string) (profiles []model.Profile, err errors.EdgeX) {
	ctx := context.Background()

	offset, validLimit := getValidOffsetAndLimit(offset, limit)
	if len(labels) > 0 {
		c.loggingClient.Debugf("Querying node profiles by labels: %v", labels)
		queryObj := map[string]any{labelsField: labels}
		profiles, err = queryProfiles(ctx, c.ConnPool, sqlQueryContentByJSONFieldWithPaginationAsNamedArgs(profileTableName),
			pgx.NamedArgs{jsonContentCondition: queryObj, offsetCondition: offset, limitCondition: validLimit})
		if err != nil {
			return profiles, errors.NewCommonEdgeX(errors.Kind(err), "failed to query all node profiles by labels", err)
		}
	} else {
		profiles, err = queryProfiles(ctx, c.ConnPool, sqlQueryContentWithPaginationAsNamedArgs(profileTableName),
			pgx.NamedArgs{offsetCondition: offset, limitCondition: validLimit})
		if err != nil {
			return profiles, errors.NewCommonEdgeX(errors.Kind(err), "failed to query all node profiles", err)
		}
	}

	return profiles, nil
}

// ProfilesByModel query node profiles with offset, limit and model
func (c *Client) ProfilesByModel(offset int, limit int, model string) ([]model.Profile, errors.EdgeX) {
	ctx := context.Background()
	offset, validLimit := getValidOffsetAndLimit(offset, limit)
	queryObj := map[string]any{modelField: model}
	return queryProfiles(ctx, c.ConnPool, sqlQueryContentByJSONFieldWithPaginationAsNamedArgs(profileTableName),
		pgx.NamedArgs{jsonContentCondition: queryObj, offsetCondition: offset, limitCondition: validLimit})
}

// ProfilesByManufacturer query node profiles with offset, limit and manufacturer
func (c *Client) ProfilesByManufacturer(offset int, limit int, manufacturer string) ([]model.Profile, errors.EdgeX) {
	ctx := context.Background()
	offset, validLimit := getValidOffsetAndLimit(offset, limit)
	queryObj := map[string]any{manufacturerField: manufacturer}
	return queryProfiles(ctx, c.ConnPool, sqlQueryContentByJSONFieldWithPaginationAsNamedArgs(profileTableName),
		pgx.NamedArgs{jsonContentCondition: queryObj, offsetCondition: offset, limitCondition: validLimit})
}

// ProfilesByManufacturerAndModel query node profiles with offset, limit, manufacturer and model
func (c *Client) ProfilesByManufacturerAndModel(offset int, limit int, manufacturer string, model string) (profiles []model.Profile, err errors.EdgeX) {
	ctx := context.Background()
	offset, validLimit := getValidOffsetAndLimit(offset, limit)
	queryObj := map[string]any{modelField: model, manufacturerField: manufacturer}
	return queryProfiles(ctx, c.ConnPool, sqlQueryContentByJSONFieldWithPaginationAsNamedArgs(profileTableName),
		pgx.NamedArgs{jsonContentCondition: queryObj, offsetCondition: offset, limitCondition: validLimit})
}

// ProfileCountByLabels returns the total count of Node Profiles with labels specified.  If no label is specified, the total count of all node profiles will be returned.
func (c *Client) ProfileCountByLabels(labels []string) (int64, errors.EdgeX) {
	ctx := context.Background()

	if len(labels) > 0 {
		queryObj := map[string]any{labelsField: labels}
		return getTotalRowsCount(ctx, c.ConnPool, sqlQueryCountByJSONField(profileTableName), queryObj)
	}
	return getTotalRowsCount(ctx, c.ConnPool, sqlQueryCount(profileTableName))
}

// ProfileCountByManufacturer returns the count of Node Profiles associated with specified manufacturer
func (c *Client) ProfileCountByManufacturer(manufacturer string) (int64, errors.EdgeX) {
	ctx := context.Background()
	queryObj := map[string]any{manufacturerField: manufacturer}
	return getTotalRowsCount(ctx, c.ConnPool, sqlQueryCountByJSONField(profileTableName), queryObj)
}

// ProfileCountByModel returns the count of Node Profiles associated with specified model
func (c *Client) ProfileCountByModel(model string) (int64, errors.EdgeX) {
	ctx := context.Background()
	queryObj := map[string]any{modelField: model}
	return getTotalRowsCount(ctx, c.ConnPool, sqlQueryCountByJSONField(profileTableName), queryObj)
}

// ProfileCountByManufacturerAndModel returns the count of Node Profiles associated with specified manufacturer and model
func (c *Client) ProfileCountByManufacturerAndModel(manufacturer, model string) (int64, errors.EdgeX) {
	ctx := context.Background()
	queryObj := map[string]any{manufacturerField: manufacturer, modelField: model}
	return getTotalRowsCount(ctx, c.ConnPool, sqlQueryCountByJSONField(profileTableName), queryObj)
}

// ResourceCount returns the total count of Resources
func (c *Client) InUseResourceCount() (int64, errors.EdgeX) {
	ctx := context.Background()
	return getTotalRowsCount(ctx, c.ConnPool, sqlQueryCountInUseResource())
}

func profileNameExists(ctx context.Context, connPool *pgxpool.Pool, name string) (bool, errors.EdgeX) {
	var exists bool
	queryObj := map[string]any{nameField: name}
	err := connPool.QueryRow(ctx, sqlCheckExistsByJSONField(profileTableName), queryObj).Scan(&exists)
	if err != nil {
		return false, pgClient.WrapDBError(fmt.Sprintf("failed to query node profile by name '%s' from %s table", name, profileTableName), err)
	}
	return exists, nil
}

func queryOneProfile(ctx context.Context, connPool *pgxpool.Pool, sql string, args ...any) (model.Profile, errors.EdgeX) {
	var dp model.Profile
	row := connPool.QueryRow(ctx, sql, args...)

	if err := row.Scan(&dp); err != nil {
		return dp, pgClient.WrapDBError("failed to query node profile", err)
	}
	return dp, nil
}

func queryProfiles(ctx context.Context, connPool *pgxpool.Pool, sql string, args ...any) ([]model.Profile, errors.EdgeX) {
	rows, err := connPool.Query(ctx, sql, args...)
	if err != nil {
		return nil, pgClient.WrapDBError("failed to query node profiles", err)
	}

	profiles, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Profile, error) {
		var dp model.Profile
		scanErr := row.Scan(&dp)
		return dp, scanErr
	})
	if err != nil {
		return nil, pgClient.WrapDBError("failed to collect rows to Profile model", err)
	}

	return profiles, nil
}
