//
// Copyright (C) 2024-2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	stdErrs "errors"
	"fmt"
	"time"

	"github.com/clint456/edgex-go/internal/core/data/container"
	pgClient "github.com/clint456/edgex-go/internal/pkg/db/postgres"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	model "github.com/edgexfoundry/go-mod-core-contracts/v4/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AllEvents queries the events with the given range, offset, and limit
func (c *Client) AllEvents(offset, limit int) ([]model.Event, errors.EdgeX) {
	ctx := context.Background()
	offset, validLimit := getValidOffsetAndLimit(offset, limit)

	events, err := queryEvents(ctx, c.ConnPool, sqlQueryAllEventWithPaginationDescByCol(originCol), pgx.NamedArgs{offsetCondition: offset, limitCondition: validLimit})
	if err != nil {
		return nil, errors.NewCommonEdgeX(errors.KindDatabaseError, "failed to query all events", err)
	}

	return events, nil
}

// AddEvent adds a new event model to DB
func (c *Client) AddEvent(e model.Event) (model.Event, errors.EdgeX) {
	ctx := context.Background()

	if e.Id == "" {
		e.Id = uuid.NewString()
	}

	nodeInfoId, err := c.nodeInfoIdByEvent(e)
	if err != nil {
		return model.Event{}, errors.NewCommonEdgeXWrapper(err)
	}

	event := model.Event{
		Id:          e.Id,
		NodeName:    e.NodeName,
		ProfileName: e.ProfileName,
		SourceName:  e.SourceName,
		Origin:      e.Origin,
		Tags:        e.Tags,
	}

	pgxErr := pgx.BeginFunc(ctx, c.ConnPool, func(tx pgx.Tx) error {
		// insert event in a transaction
		_, pgxErr := tx.Exec(
			ctx,
			sqlInsert(eventTableName, idCol, nodeInfoIdFKCol, originCol),
			event.Id,
			nodeInfoId,
			event.Origin,
		)
		if pgxErr != nil {
			return pgClient.WrapDBError("failed to insert event", pgxErr)
		}

		// insert readings in a transaction
		pgxErr = c.addReadingsInTx(tx, e.Readings, e.Id)
		if pgxErr != nil {
			return errors.NewCommonEdgeXWrapper(pgxErr)
		}
		return nil
	})

	if pgxErr != nil {
		return model.Event{}, errors.NewCommonEdgeXWrapper(pgxErr)
	}

	// return the event with readings to ensure readingsPersistedCounter will be increased
	event.Readings = e.Readings
	return event, nil
}

// EventById gets an event by id
func (c *Client) EventById(id string) (model.Event, errors.EdgeX) {
	ctx := context.Background()
	var event model.Event

	rows, err := c.ConnPool.Query(ctx, sqlQueryAllEventById(), pgx.NamedArgs{idCol: id})
	if err != nil {
		return event, pgClient.WrapDBError(fmt.Sprintf("failed to query event with id '%s'", id), err)
	}

	event, err = pgx.CollectExactlyOneRow(rows, func(row pgx.CollectableRow) (model.Event, error) {
		e, err := pgx.RowToStructByNameLax[model.Event](row)
		if err != nil {
			return model.Event{}, err
		}

		// query reading by the specific even_id and origin descending
		readings, err := queryReadings(ctx, c.ConnPool, sqlQueryAllReadingAndDescWithConds(originCol, eventIdFKCol), pgx.NamedArgs{eventIdFKCol: e.Id})
		if err != nil {
			return model.Event{}, err
		}

		e.Readings = readings
		return e, err
	})
	if err != nil {
		if stdErrs.Is(err, pgx.ErrNoRows) {
			return event, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("no event with id '%s' found", id), err)
		}
		return event, pgClient.WrapDBError("failed to scan row to event model", err)
	}

	return event, nil
}

// EventTotalCount returns the total count of Event from db
func (c *Client) EventTotalCount() (int64, errors.EdgeX) {
	return getTotalRowsCount(context.Background(), c.ConnPool, sqlQueryCountEvent())
}

// EventCountByNodeName returns the count of Event associated a specific Node from db
func (c *Client) EventCountByNodeName(nodeName string) (int64, errors.EdgeX) {
	sqlStatement := sqlQueryCountEventByCol(nodeNameCol)

	return getTotalRowsCount(context.Background(), c.ConnPool, sqlStatement, pgx.NamedArgs{nodeNameCol: nodeName})
}

// EventCountByTimeRange returns the count of Event by time range from db
func (c *Client) EventCountByTimeRange(start int64, end int64) (int64, errors.EdgeX) {
	return getTotalRowsCount(context.Background(), c.ConnPool, sqlQueryCountEventByTimeRangeCol(originCol, nil), pgx.NamedArgs{startTimeCondition: start, endTimeCondition: end})
}

// EventCountByNodeNameAndSourceNameAndLimit returns the count of Event by nodeName, resourceName, and limit from db
// this is used to check whether the event number is reach the event retention maxCap
func (c *Client) EventCountByNodeNameAndSourceNameAndLimit(nodeName, sourceName string, limit int) (int64, errors.EdgeX) {
	sqlStatement := sqlCountEventByNodeNameAndSourceNameAndLimit()
	return getTotalRowsCount(context.Background(), c.ConnPool, sqlStatement, pgx.NamedArgs{nodeNameCol: nodeName, sourceNameCol: sourceName, limitCondition: limit})
}

// EventsByNodeName query events by offset, limit and node name
func (c *Client) EventsByNodeName(offset int, limit int, name string) ([]model.Event, errors.EdgeX) {
	offset, validLimit := getValidOffsetAndLimit(offset, limit)

	sqlStatement := sqlQueryAllEventAndDescWithCondsAndPage(originCol, nodeNameCol)

	events, err := queryEvents(context.Background(), c.ConnPool, sqlStatement, pgx.NamedArgs{nodeNameCol: name, offsetCondition: offset, limitCondition: validLimit})
	if err != nil {
		return nil, errors.NewCommonEdgeX(errors.KindDatabaseError, fmt.Sprintf("failed to query events by node '%s'", name), err)
	}
	return events, nil
}

// EventsByTimeRange query events by time range, offset, and limit
func (c *Client) EventsByTimeRange(start int64, end int64, offset int, limit int) ([]model.Event, errors.EdgeX) {
	ctx := context.Background()
	offset, validLimit := getValidOffsetAndLimit(offset, limit)
	sqlStatement := sqlQueryAllEventWithPaginationAndTimeRangeDescByCol(originCol, originCol)

	events, err := queryEvents(ctx, c.ConnPool, sqlStatement, pgx.NamedArgs{startTimeCondition: start, endTimeCondition: end, offsetCondition: offset, limitCondition: validLimit})
	if err != nil {
		return nil, errors.NewCommonEdgeXWrapper(err)
	}
	return events, nil
}

// DeleteEventById removes an event by id
func (c *Client) DeleteEventById(id string) errors.EdgeX {
	ctx := context.Background()

	event, err := c.EventById(id)
	if err != nil {
		return errors.NewCommonEdgeX(errors.Kind(err), "failed delete event", err)
	}

	// delete event and readings in a transaction
	pgxErr := pgx.BeginFunc(ctx, c.ConnPool, func(tx pgx.Tx) error {
		// use event.origin to filter readings to improve delete speed
		if err = deleteReadingsByOriginAndEventId(ctx, tx, event.Origin, id); err != nil {
			return err
		}

		sqlStatement := fmt.Sprintf("DELETE FROM %s WHERE %s = @%s", eventTableName, idCol, idCol)
		if err = deleteEvents(ctx, tx, sqlStatement, pgx.NamedArgs{idCol: id}); err != nil {
			return err
		}
		return nil
	})
	if pgxErr != nil {
		return errors.NewCommonEdgeX(errors.Kind(pgxErr), "failed delete event", pgxErr)
	}
	return nil
}

// DeleteEventsByNodeName deletes specific node's events and corresponding readings
// This function is implemented to starts up two goroutines to delete readings and events in the background to achieve better performance
func (c *Client) DeleteEventsByNodeName(nodeName string) errors.EdgeX {
	// update nodeInfo as deletable, then event and reading will not return when the user query event or reading by the specific node
	if err := c.updateNodeInfosDeletableByNodeName(nodeName); err != nil {
		return errors.NewCommonEdgeX(errors.Kind(err), "delete events by nodeName", err)
	}
	// delete events, readings, nodeInfos
	if err := c.deleteEventsByConditions([]string{nodeNameCol}, pgx.NamedArgs{nodeNameCol: nodeName}); err != nil {
		return errors.NewCommonEdgeX(errors.Kind(err), "delete events by nodeName", err)
	}
	return nil
}

// DeleteEventsByNodeNameAndSourceName deletes specific node's events and corresponding readings
func (c *Client) DeleteEventsByNodeNameAndSourceName(nodeName, sourceName string) errors.EdgeX {
	return c.deleteEventsByConditions([]string{nodeNameCol, sourceNameCol}, pgx.NamedArgs{nodeNameCol: nodeName, sourceNameCol: sourceName})
}

// deleteEventsByConditions deletes specific node's events and corresponding readings
func (c *Client) deleteEventsByConditions(cols []string, values pgx.NamedArgs) errors.EdgeX {
	ctx := context.Background()

	sqlStatement := sqlDeleteEventsByColumn(cols...)

	go func() {
		// nodeInfos are used to remove data by id and remove the id value from the cache after finishing the transaction
		nodeInfos, err := c.nodeInfosByConds(cols, values)
		if err != nil {
			return
		}
		// delete events and readings in a transaction
		pgxErr := pgx.BeginFunc(ctx, c.ConnPool, func(tx pgx.Tx) error {
			// select the event-ids of the specified node name from event table as the sub-query of deleting readings
			subSqlStatement := sqlQueryEventIdFieldsByCol(cols...)
			if err = deleteReadingsBySubQuery(ctx, tx, subSqlStatement, values); err != nil {
				if errors.Kind(err) == errors.KindEntityDoesNotExist {
					c.loggingClient.Debugf("no readings found for deletion: %s", err.Error())
				} else {
					c.loggingClient.Errorf("failed delete readings with conditions '%v' '%v': %v", cols, values, err)
					return err
				}
			}

			err = deleteEvents(ctx, tx, sqlStatement, values)
			if err != nil {
				if errors.Kind(err) == errors.KindEntityDoesNotExist {
					c.loggingClient.Debugf("no events found for deletion: %s", err.Error())
				} else {
					c.loggingClient.Errorf("failed delete event with conditions '%v' '%v': %v", cols, values, err)
					return err
				}
			}

			for _, nodeInfo := range nodeInfos {
				err = deleteNodeInfoById(ctx, tx, nodeInfo.Id)
				if err != nil {
					return errors.NewCommonEdgeXWrapper(err)
				}
			}
			return nil
		})
		if pgxErr != nil {
			c.loggingClient.Errorf("failed delete events with conditions '%v' '%v': %v", cols, values, pgxErr)
			return
		}

		nodeInfoCache := container.NodeInfoCacheFrom(c.dic.Get)
		for _, nodeInfo := range nodeInfos {
			nodeInfoCache.Remove(nodeInfo)
		}
	}()

	return nil
}

// DeleteEventsByAge deletes events and their corresponding readings that are older than age
// This function is implemented to starts up two goroutines to delete readings and events in the background to achieve better performance
func (c *Client) DeleteEventsByAge(age int64) errors.EdgeX {
	return c.deleteEventsByAgeAndConditions(age, nil, nil)
}

// DeleteEventsByAgeAndNodeNameAndSourceName deletes events and their corresponding readings that are older than age
// This function is implemented to starts up two goroutines to delete readings and events in the background to achieve better performance
func (c *Client) DeleteEventsByAgeAndNodeNameAndSourceName(age int64, nodeName, sourceName string) errors.EdgeX {
	return c.deleteEventsByAgeAndConditions(age, []string{nodeNameCol, sourceNameCol}, pgx.NamedArgs{nodeNameCol: nodeName, sourceNameCol: sourceName})
}

// deleteEventsByAgeAndConditions deletes events and their corresponding readings that are older than age
// This function is implemented to starts up two goroutines to delete readings and events in the background to achieve better performance
func (c *Client) deleteEventsByAgeAndConditions(age int64, cols []string, values pgx.NamedArgs) errors.EdgeX {
	ctx := context.Background()
	expireTimestamp := time.Now().UnixNano() - age
	if len(values) == 0 {
		values = pgx.NamedArgs{}
	}
	values[endTimeCondition] = expireTimestamp
	sqlStatement := sqlDeleteEventsByTimeRangeAndColumn(originCol, cols...)

	go func() {
		// delete events and readings in a transaction
		_ = pgx.BeginFunc(ctx, c.ConnPool, func(tx pgx.Tx) error {
			// select the event ids within the origin time range from event table as the sub-query of deleting readings
			subSqlStatement := sqlQueryEventIdFieldByTimeRangeAndConditions(originCol, cols...)
			if err := deleteReadingsBySubQuery(ctx, tx, subSqlStatement, values); err != nil {
				if errors.Kind(err) == errors.KindEntityDoesNotExist {
					c.loggingClient.Debugf("no readings found for deletion by age '%d' nanoseconds: %s", age, err.Error())
				} else {
					c.loggingClient.Errorf("failed delete readings by age '%d' nanoseconds: %v", age, err)
					return err
				}
			}

			err := deleteEvents(ctx, tx, sqlStatement, values)
			if err != nil {
				if errors.Kind(err) == errors.KindEntityDoesNotExist {
					c.loggingClient.Debugf("no events found for deletion by age '%d' nanoseconds: %s", age, err.Error())
				} else {
					c.loggingClient.Errorf("failed delete event by age '%d' nanoseconds: %v", age, err)
					return err
				}
			}
			return nil
		})
	}()

	return nil
}

// queryEvents queries the data rows with given sql statement and passed args, and unmarshal the data rows to the Event model slice
func queryEvents(ctx context.Context, connPool *pgxpool.Pool, sql string, args pgx.NamedArgs) ([]model.Event, errors.EdgeX) {
	rows, err := connPool.Query(ctx, sql, args)
	if err != nil {
		return nil, pgClient.WrapDBError("query failed", err)
	}

	var events []model.Event
	events, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Event, error) {
		event, err := pgx.RowToStructByNameLax[model.Event](row)
		if err != nil {
			return model.Event{}, err
		}

		// query reading by the specific even_id and origin descending
		readings, err := queryReadings(ctx, connPool, sqlQueryAllReadingAndDescWithConds(originCol, eventIdFKCol), pgx.NamedArgs{eventIdFKCol: event.Id})

		if err != nil {
			return model.Event{}, err
		}

		event.Readings = readings
		return event, nil
	})

	if err != nil {
		return nil, pgClient.WrapDBError("failed to scan events", err)
	}

	return events, nil
}

// deleteEvents delete the data rows with given sql statement and passed args in a db transaction
func deleteEvents(ctx context.Context, tx pgx.Tx, sqlStatement string, args pgx.NamedArgs) errors.EdgeX {
	commandTag, err := tx.Exec(
		ctx,
		sqlStatement,
		args,
	)
	if err != nil {
		return pgClient.WrapDBError("event(s) delete failed", err)
	}
	if commandTag.RowsAffected() == 0 {
		return errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("no event found; SQL statement: %s", sqlStatement), nil)
	}
	return nil
}

func (c *Client) LatestEventByNodeNameAndSourceNameAndOffset(nodeName, sourceName string, offset int64) (model.Event, errors.EdgeX) {
	ctx := context.Background()

	if err := validateDatabaseOffset(offset); err != nil {
		return model.Event{}, errors.NewCommonEdgeX(errors.KindContractInvalid, "invalid offset", err)
	}

	events, err := queryEvents(
		ctx, c.ConnPool,
		sqlQueryAllEventAndDescWithCondsAndPage(originCol, nodeNameCol, sourceNameCol),
		pgx.NamedArgs{nodeNameCol: nodeName, sourceNameCol: sourceName, offsetCondition: offset, limitCondition: 1},
	)
	if err != nil {
		return model.Event{}, errors.NewCommonEdgeX(errors.KindDatabaseError, "failed to query all events", err)
	}

	if len(events) == 0 {
		return model.Event{}, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("no event found with offset '%d'", offset), err)
	}
	return events[0], nil
}

// LatestEventByNodeNameAndSourceNameAndAgeAndOffset query an event with specified conditions and age and offset
func (c *Client) LatestEventByNodeNameAndSourceNameAndAgeAndOffset(nodeName, sourceName string, age int64, offset int64) (model.Event, errors.EdgeX) {
	ctx := context.Background()

	if err := validateDatabaseOffset(offset); err != nil {
		return model.Event{}, errors.NewCommonEdgeX(errors.KindContractInvalid, "invalid offset", err)
	}

	expireTimestamp := time.Now().UnixNano() - age

	sqlStmt := sqlQueryAllEventAndDescWithCondsAndPagAndUpperLimitTime(originCol, originCol, nodeNameCol, sourceNameCol)
	events, err := queryEvents(
		ctx, c.ConnPool, sqlStmt,
		pgx.NamedArgs{
			endTimeCondition: expireTimestamp,
			nodeNameCol:      nodeName, sourceNameCol: sourceName,
			offsetCondition: offset, limitCondition: 1},
	)
	if err != nil {
		return model.Event{}, errors.NewCommonEdgeX(errors.KindDatabaseError, "failed to query all events", err)
	}

	if len(events) == 0 {
		return model.Event{}, errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("no event found with offset '%d'", offset), err)
	}
	return events[0], nil
}

func validateDatabaseOffset(offset int64) error {
	// A database offset cannot be negative, so returning error when the offset is negative
	if offset < 0 {
		// Return an error or handle this case appropriately. Do not proceed.
		return fmt.Errorf("found negative database offset '%d'", offset)
	}
	return nil
}
