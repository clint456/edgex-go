//
// Copyright (C) 2024-2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	pgClient "github.com/clint456/edgex-go/internal/pkg/db/postgres"
	dbModels "github.com/clint456/edgex-go/internal/pkg/infrastructure/postgres/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	model "github.com/edgexfoundry/go-mod-core-contracts/v4/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// insertReadingCols defines the reading table columns in slice used in inserting readings
	insertReadingCols = []string{eventIdFKCol, nodeInfoIdFKCol, originCol, valueCol, numericValueCol, binaryValueCol, objectValueCol}
)

func (c *Client) ReadingTotalCount() (int64, errors.EdgeX) {
	return getTotalRowsCount(context.Background(), c.ConnPool, sqlQueryCountReading())
}

func (c *Client) AllReadings(offset int, limit int) ([]model.Reading, errors.EdgeX) {
	ctx := context.Background()
	offset, validLimit := getValidOffsetAndLimit(offset, limit)

	// query reading by origin descending with offset & limit
	readings, err := queryReadings(ctx, c.ConnPool, sqlQueryAllReadingWithPaginationDescByCol(originCol), pgx.NamedArgs{offsetCondition: offset, limitCondition: validLimit})
	if err != nil {
		return nil, errors.NewCommonEdgeX(errors.KindDatabaseError, "failed to query all readings", err)
	}

	return readings, nil
}

// ReadingsByResourceName query readings by offset, limit and resource name
func (c *Client) ReadingsByResourceName(offset int, limit int, resourceName string) ([]model.Reading, errors.EdgeX) {
	offset, validLimit := getValidOffsetAndLimit(offset, limit)

	// query reading by the resourceName and origin descending
	sqlStatement := sqlQueryAllReadingAndDescWithCondsAndPag(originCol, resourceNameCol)

	readings, err := queryReadings(context.Background(), c.ConnPool, sqlStatement,
		pgx.NamedArgs{resourceNameCol: resourceName, offsetCondition: offset, limitCondition: validLimit})
	if err != nil {
		return nil, errors.NewCommonEdgeX(errors.KindDatabaseError, fmt.Sprintf("failed to query readings by resource '%s'", resourceName), err)
	}
	return readings, nil
}

// ReadingsByNodeName query readings by offset, limit and node name
func (c *Client) ReadingsByNodeName(offset int, limit int, name string) ([]model.Reading, errors.EdgeX) {
	offset, validLimit := getValidOffsetAndLimit(offset, limit)

	// query reading by the nodeName and origin descending
	sqlStatement := sqlQueryAllReadingAndDescWithCondsAndPag(originCol, nodeNameCol)

	readings, err := queryReadings(context.Background(), c.ConnPool, sqlStatement,
		pgx.NamedArgs{nodeNameCol: name, offsetCondition: offset, limitCondition: validLimit})
	if err != nil {
		return nil, errors.NewCommonEdgeX(errors.KindDatabaseError, fmt.Sprintf("failed to query readings by node '%s'", name), err)
	}
	return readings, nil
}

// ReadingsByNodeNameAndResourceName query readings by offset, limit, node name and resource name
func (c *Client) ReadingsByNodeNameAndResourceName(nodeName string, resourceName string, offset int, limit int) ([]model.Reading, errors.EdgeX) {
	offset, validLimit := getValidOffsetAndLimit(offset, limit)

	// query reading by the nodeName/resourceName and origin descending
	sqlStatement := sqlQueryAllReadingAndDescWithCondsAndPag(originCol, nodeNameCol, resourceNameCol)

	readings, err := queryReadings(context.Background(), c.ConnPool, sqlStatement,
		pgx.NamedArgs{nodeNameCol: nodeName, resourceNameCol: resourceName, offsetCondition: offset, limitCondition: validLimit})
	if err != nil {
		return nil, errors.NewCommonEdgeX(errors.KindDatabaseError,
			fmt.Sprintf("failed to query readings by node '%s' and resource '%s'", nodeName, resourceName), err)
	}
	return readings, nil
}

// ReadingsByTimeRange query readings by origin within the time range with offset and limit
func (c *Client) ReadingsByTimeRange(start int64, end int64, offset int, limit int) ([]model.Reading, errors.EdgeX) {
	ctx := context.Background()
	offset, validLimit := getValidOffsetAndLimit(offset, limit)
	sqlStatement := sqlQueryAllReadingWithPaginationAndTimeRangeDescByCol(originCol, originCol, nil)

	readings, err := queryReadings(ctx, c.ConnPool, sqlStatement,
		pgx.NamedArgs{startTimeCondition: start, endTimeCondition: end,
			offsetCondition: offset, limitCondition: validLimit})
	if err != nil {
		return nil, errors.NewCommonEdgeXWrapper(err)
	}
	return readings, nil
}

// ReadingsByNodeNameAndTimeRange query readings by the specified node, origin within the time range, offset, and limit
func (c *Client) ReadingsByNodeNameAndTimeRange(nodeName string, start int64, end int64, offset int, limit int) ([]model.Reading, errors.EdgeX) {
	ctx := context.Background()
	offset, validLimit := getValidOffsetAndLimit(offset, limit)
	sqlStatement := sqlQueryAllReadingWithPaginationAndTimeRangeDescByCol(originCol, originCol, nil, nodeNameCol)

	readings, err := queryReadings(ctx, c.ConnPool, sqlStatement,
		pgx.NamedArgs{startTimeCondition: start, endTimeCondition: end,
			nodeNameCol: nodeName, offsetCondition: offset, limitCondition: validLimit})
	if err != nil {
		return nil, errors.NewCommonEdgeXWrapper(err)
	}
	return readings, nil
}

// ReadingsByResourceNameAndTimeRange query readings by the specified resource, origin within the time range, offset, and limit
func (c *Client) ReadingsByResourceNameAndTimeRange(resourceName string, start int64, end int64, offset int, limit int) ([]model.Reading, errors.EdgeX) {
	ctx := context.Background()
	offset, validLimit := getValidOffsetAndLimit(offset, limit)
	sqlStatement := sqlQueryAllReadingWithPaginationAndTimeRangeDescByCol(originCol, originCol, nil, resourceNameCol)

	readings, err := queryReadings(ctx, c.ConnPool, sqlStatement,
		pgx.NamedArgs{startTimeCondition: start, endTimeCondition: end,
			resourceNameCol: resourceName, offsetCondition: offset, limitCondition: validLimit})
	if err != nil {
		return nil, errors.NewCommonEdgeXWrapper(err)
	}
	return readings, nil
}

// ReadingsByNodeNameAndResourceNameAndTimeRange query readings by the specified node and resource, origin within the time range, offset, and limit
func (c *Client) ReadingsByNodeNameAndResourceNameAndTimeRange(nodeName string, resourceName string, start int64, end int64, offset int, limit int) ([]model.Reading, errors.EdgeX) {
	ctx := context.Background()
	offset, validLimit := getValidOffsetAndLimit(offset, limit)
	sqlStatement := sqlQueryAllReadingWithPaginationAndTimeRangeDescByCol(originCol, originCol, nil, nodeNameCol, resourceNameCol)

	readings, err := queryReadings(ctx, c.ConnPool, sqlStatement,
		pgx.NamedArgs{startTimeCondition: start, endTimeCondition: end,
			nodeNameCol: nodeName, resourceNameCol: resourceName, offsetCondition: offset, limitCondition: validLimit})
	if err != nil {
		return nil, errors.NewCommonEdgeXWrapper(err)
	}
	return readings, nil
}

// ReadingsByNodeNameAndResourceNamesAndTimeRange query readings by the specified node and resourceName slice, origin within the time range, offset and limit
func (c *Client) ReadingsByNodeNameAndResourceNamesAndTimeRange(nodeName string, resourceNames []string, start int64, end int64, offset, limit int) ([]model.Reading, errors.EdgeX) {
	ctx := context.Background()
	offset, validLimit := getValidOffsetAndLimit(offset, limit)

	sqlStatement := sqlQueryAllReadingWithPaginationAndTimeRangeDescByCol(originCol, originCol,
		[]string{resourceNameCol}, nodeNameCol, resourceNameCol)

	// build the query args for the where condition using in querying readings
	queryArgs := pgx.NamedArgs{startTimeCondition: start, endTimeCondition: end,
		nodeNameCol: nodeName, resourceNameCol: resourceNames, offsetCondition: offset, limitCondition: validLimit}
	// query readings
	readings, err := queryReadings(ctx, c.ConnPool, sqlStatement, queryArgs)
	if err != nil {
		return nil, errors.NewCommonEdgeXWrapper(err)
	}

	return readings, nil
}

// ReadingCountByNodeName returns the count of Readings associated a specific Node from db
func (c *Client) ReadingCountByNodeName(nodeName string) (int64, errors.EdgeX) {
	sqlStatement := sqlQueryCountReadingByCol(nodeNameCol)

	return getTotalRowsCount(context.Background(), c.ConnPool, sqlStatement, pgx.NamedArgs{nodeNameCol: nodeName})
}

// ReadingCountByResourceName returns the count of Readings associated a specific resource from db
func (c *Client) ReadingCountByResourceName(resourceName string) (int64, errors.EdgeX) {
	sqlStatement := sqlQueryCountReadingByCol(resourceNameCol)

	return getTotalRowsCount(context.Background(), c.ConnPool, sqlStatement, pgx.NamedArgs{resourceNameCol: resourceName})
}

// ReadingCountByNodeNameAndResourceName returns the count of readings associated a specific node and resource values from db
func (c *Client) ReadingCountByNodeNameAndResourceName(nodeName string, resourceName string) (int64, errors.EdgeX) {
	sqlStatement := sqlQueryCountReadingByCol(nodeNameCol, resourceNameCol)

	return getTotalRowsCount(context.Background(), c.ConnPool, sqlStatement, pgx.NamedArgs{nodeNameCol: nodeName, resourceNameCol: resourceName})
}

// ReadingCountByTimeRange returns the count of reading by origin within the time range from db
func (c *Client) ReadingCountByTimeRange(start int64, end int64) (int64, errors.EdgeX) {
	return getTotalRowsCount(context.Background(), c.ConnPool, sqlQueryCountReadingByTimeRangeCol(originCol, nil), pgx.NamedArgs{startTimeCondition: start, endTimeCondition: end})
}

// ReadingCountByNodeNameAndTimeRange returns the count of readings by origin within the time range and the specified node from db
func (c *Client) ReadingCountByNodeNameAndTimeRange(nodeName string, start int64, end int64) (int64, errors.EdgeX) {
	return getTotalRowsCount(context.Background(), c.ConnPool, sqlQueryCountReadingByTimeRangeCol(originCol, nil, nodeNameCol),
		pgx.NamedArgs{startTimeCondition: start, endTimeCondition: end, nodeNameCol: nodeName})
}

// ReadingCountByResourceNameAndTimeRange returns the count of readings by origin within the time range and the specified resource from db
func (c *Client) ReadingCountByResourceNameAndTimeRange(resourceName string, start int64, end int64) (int64, errors.EdgeX) {
	return getTotalRowsCount(context.Background(), c.ConnPool, sqlQueryCountReadingByTimeRangeCol(originCol, nil, resourceNameCol),
		pgx.NamedArgs{startTimeCondition: start, endTimeCondition: end, resourceNameCol: resourceName})
}

// ReadingCountByNodeNameAndResourceNameAndTimeRange returns the count of readings by origin within the time range
// associated with the specified node and resource from db
func (c *Client) ReadingCountByNodeNameAndResourceNameAndTimeRange(nodeName string, resourceName string, start int64, end int64) (int64, errors.EdgeX) {
	return getTotalRowsCount(context.Background(),
		c.ConnPool,
		sqlQueryCountReadingByTimeRangeCol(originCol, nil, nodeNameCol, resourceNameCol),
		pgx.NamedArgs{startTimeCondition: start, endTimeCondition: end, nodeNameCol: nodeName, resourceNameCol: resourceName})
}

// ReadingCountByNodeNameAndResourceNamesAndTimeRange returns the count of readings by origin within the time range
// associated with the specified node and resourceName slice from db
func (c *Client) ReadingCountByNodeNameAndResourceNamesAndTimeRange(nodeName string, resourceNames []string, start int64, end int64) (int64, errors.EdgeX) {
	return getTotalRowsCount(context.Background(),
		c.ConnPool,
		sqlQueryCountReadingByTimeRangeCol(originCol, []string{resourceNameCol}, nodeNameCol, resourceNameCol),
		pgx.NamedArgs{startTimeCondition: start, endTimeCondition: end, nodeNameCol: nodeName, resourceNameCol: resourceNames})
}

func (c *Client) LatestReadingByOffset(offset uint32) (model.Reading, errors.EdgeX) {
	ctx := context.Background()

	readings, err := queryReadings(ctx, c.ConnPool, sqlQueryAllReadingWithPaginationDescByCol(originCol), pgx.NamedArgs{offsetCondition: offset, limitCondition: 1})
	if err != nil {
		return nil, errors.NewCommonEdgeX(errors.KindDatabaseError, "failed to query all readings", err)
	}

	if len(readings) == 0 {
		return nil, errors.NewCommonEdgeX(errors.KindServerError, fmt.Sprintf("no reading found with offset '%d'", offset), err)
	}
	return readings[0], nil
}

// queryReadings queries the data rows with given sql statement and passed args, converts the rows to map and unmarshal the data rows to the Reading model slice
func queryReadings(ctx context.Context, connPool *pgxpool.Pool, sql string, args pgx.NamedArgs) ([]model.Reading, errors.EdgeX) {
	rows, err := connPool.Query(ctx, sql, args)
	if err != nil {
		return nil, pgClient.WrapDBError("query failed", err)
	}

	readings, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Reading, error) {
		var reading model.Reading

		readingDBModel, err := pgx.RowToStructByNameLax[dbModels.Reading](row)
		if err != nil {
			return nil, pgClient.WrapDBError("failed to convert row to map", err)
		}

		// convert the BaseReading fields to BaseReading struct defined in contract
		baseReading := readingDBModel.GetBaseReading()

		if readingDBModel.BinaryValue != nil {
			// reading type is BinaryReading
			binaryReading := model.BinaryReading{
				BaseReading: baseReading,
				MediaType:   *readingDBModel.MediaType,
				BinaryValue: readingDBModel.BinaryValue,
			}
			reading = binaryReading
		} else if readingDBModel.ObjectValue != nil {
			// reading type is ObjectReading
			objReading := model.ObjectReading{
				BaseReading: baseReading,
				ObjectValue: readingDBModel.ObjectValue,
			}
			reading = objReading
		} else if readingDBModel.Value != nil {
			// reading type is SimpleReading
			simpleReading := model.SimpleReading{
				BaseReading: baseReading,
				Value:       *readingDBModel.Value,
			}
			reading = simpleReading
		} else if readingDBModel.NumericValue != nil {
			// reading type is NumericReading
			val, err := numericReadingVal(readingDBModel.ValueType, readingDBModel.NumericValue)
			if err != nil {
				return nil, pgClient.WrapDBError("read numeric value", err)
			}
			reading = model.NumericReading{
				BaseReading:  baseReading,
				NumericValue: val,
			}
		} else {
			// reading type is NullReading
			nullReading := model.NullReading{
				BaseReading: baseReading,
				Value:       nil,
			}
			reading = nullReading
		}

		return reading, nil
	})

	if err != nil {
		return nil, errors.NewCommonEdgeXWrapper(err)
	}

	return readings, nil
}

func numericReadingVal(valueType string, numericValue *pgtype.Numeric) (any, errors.EdgeX) {
	switch valueType {
	case common.ValueTypeFloat32, common.ValueTypeFloat64:
		val, err := numericValue.Float64Value()
		if err != nil {
			return nil, pgClient.WrapDBError("read the float value", err)
		}
		return val.Float64, nil
	case common.ValueTypeUint8, common.ValueTypeUint16, common.ValueTypeUint32, common.ValueTypeUint64:
		parsedUint, err := numericToUint64(numericValue)
		if err != nil {
			return nil, pgClient.WrapDBError("failed to parse numeric reading value to uint", err)
		}
		return parsedUint, nil
	case common.ValueTypeInt8, common.ValueTypeInt16, common.ValueTypeInt32, common.ValueTypeInt64:
		pgIntNum, err := numericValue.Int64Value()
		if err != nil || !pgIntNum.Valid {
			return nil, pgClient.WrapDBError("failed to parse numeric value to int64", err)
		}
		return pgIntNum.Int64, nil
	default:
		return nil, errors.NewCommonEdgeX(errors.KindServerError, fmt.Sprintf("unexpected value type '%s", valueType), nil)
	}
}

// deleteReadingsByOriginAndEventId delete the data rows with given sql statement and passed args
func deleteReadingsByOriginAndEventId(ctx context.Context, tx pgx.Tx, origin int64, eventId string) errors.EdgeX {
	sqlStatement := sqlDeleteTimeRangeByColumn(readingTableName, originCol, eventIdFKCol)
	commandTag, err := tx.Exec(
		ctx,
		sqlStatement,
		pgx.NamedArgs{endTimeCondition: origin, eventIdFKCol: eventId},
	)
	if commandTag.RowsAffected() == 0 {
		return errors.NewCommonEdgeX(errors.KindContractInvalid, "no reading found", nil)
	}
	if err != nil {
		return pgClient.WrapDBError("reading(s) delete failed", err)
	}
	return nil
}

// deleteReadingsBySubQuery delete the readings with event_id in the range of the sub query
func deleteReadingsBySubQuery(ctx context.Context, tx pgx.Tx, subQuerySql string, args pgx.NamedArgs) errors.EdgeX {
	sqlStatement := sqlDeleteByColumns(readingTableName, eventIdFKCol)
	subQueryCond := "ANY ( " + subQuerySql + " )"
	sqlStatement = strings.Replace(sqlStatement, "$1", subQueryCond, 1)

	commandTag, err := tx.Exec(
		ctx,
		sqlStatement,
		args,
	)
	if commandTag.RowsAffected() == 0 {
		return errors.NewCommonEdgeX(errors.KindEntityDoesNotExist, fmt.Sprintf("no reading found; SQL statement: %s", sqlStatement), nil)
	}
	if err != nil {
		return pgClient.WrapDBError("reading(s) delete failed", err)
	}
	return nil
}

// addReadingsInTx converts reading interface to BinaryReading/ObjectReading/SimpleReading structs first based on the reading value type
// and then perform the CopyFromSlice transaction to insert readings in batch
func (c *Client) addReadingsInTx(tx pgx.Tx, readings []model.Reading, eventId string) error {
	var readingDBModels []dbModels.Reading

	for _, r := range readings {
		baseReading := r.GetBaseReading()
		if baseReading.Id == "" {
			baseReading.Id = uuid.New().String()
		} else {
			_, err := uuid.Parse(baseReading.Id)
			if err != nil {
				return errors.NewCommonEdgeX(errors.KindInvalidId, "uuid parsing failed", err)
			}
		}

		var readingDBModel dbModels.Reading
		switch contractReadingModel := r.(type) {
		case model.BinaryReading:
			// convert BinaryReading struct to Reading DB model
			readingDBModel = dbModels.Reading{
				BaseReading: baseReading,
				BinaryReading: dbModels.BinaryReading{
					BinaryValue: contractReadingModel.BinaryValue,
					MediaType:   &contractReadingModel.MediaType,
				},
			}
		case model.ObjectReading:
			// convert ObjectReading struct to Reading DB model
			readingDBModel = dbModels.Reading{
				BaseReading: baseReading,
				ObjectReading: dbModels.ObjectReading{
					ObjectValue: contractReadingModel.ObjectValue,
				},
			}
		case model.SimpleReading:
			// convert SimpleReading struct to Reading DB model
			readingDBModel = dbModels.Reading{
				BaseReading: baseReading,
			}
			switch contractReadingModel.ValueType {
			case common.ValueTypeFloat32:
				var numericVal pgtype.Numeric
				if err := numericVal.ScanScientific(contractReadingModel.Value); err != nil {
					return errors.NewCommonEdgeX(errors.KindContractInvalid, fmt.Sprintf("invalid numeric value '%s'", contractReadingModel.Value), err)
				}
				readingDBModel.NumericReading = dbModels.NumericReading{NumericValue: &numericVal}
			case common.ValueTypeFloat64:
				var numericVal pgtype.Numeric
				if err := numericVal.ScanScientific(contractReadingModel.Value); err != nil {
					return errors.NewCommonEdgeX(errors.KindContractInvalid, fmt.Sprintf("invalid numeric value '%s'", contractReadingModel.Value), err)
				}
				readingDBModel.NumericReading = dbModels.NumericReading{NumericValue: &numericVal}
			case common.ValueTypeUint8, common.ValueTypeUint16, common.ValueTypeUint32, common.ValueTypeUint64,
				common.ValueTypeInt8, common.ValueTypeInt16, common.ValueTypeInt32, common.ValueTypeInt64:
				var val pgtype.Numeric
				if err := val.Scan(contractReadingModel.Value); err != nil {
					return errors.NewCommonEdgeX(errors.KindContractInvalid, fmt.Sprintf("invalid numeric value '%s'", contractReadingModel.Value), err)
				}
				readingDBModel.NumericReading = dbModels.NumericReading{NumericValue: &val}
			default:
				readingDBModel.SimpleReading = dbModels.SimpleReading{Value: &contractReadingModel.Value}
			}
		case model.NullReading:
			readingDBModel = dbModels.Reading{
				BaseReading: baseReading,
			}
		default:
			return errors.NewCommonEdgeX(errors.KindContractInvalid, "failed to convert reading to none of BinaryReading/ObjectReading/SimpleReading structs", nil)
		}
		readingDBModels = append(readingDBModels, readingDBModel)
	}

	// insert readingDBModels slice in batch
	_, err := tx.CopyFrom(
		context.Background(),
		strings.Split(readingTableName, "."),
		insertReadingCols,
		pgx.CopyFromSlice(len(readingDBModels), func(i int) ([]any, error) {
			var objectValueBytes []byte
			var err error

			r := readingDBModels[i]
			if r.ObjectValue != nil {
				objectValueBytes, err = json.Marshal(r.ObjectValue)
				if err != nil {
					return nil, errors.NewCommonEdgeX(errors.KindServerError, "unable to JSON marshal reading ObjectValue", err)
				}
			}

			nodeInfoId, err := c.nodeInfoIdByReading(r)
			if err != nil {
				return nil, errors.NewCommonEdgeX(errors.KindServerError, "unable to retrieve nodeInfo", err)
			}

			return []any{
				eventId,
				nodeInfoId,
				r.Origin,
				r.Value,
				r.NumericValue,
				r.BinaryValue,
				objectValueBytes,
			}, nil
		}),
	)
	if err != nil {
		return pgClient.WrapDBError("failed to insert readings in batch", err)
	}

	return nil
}
