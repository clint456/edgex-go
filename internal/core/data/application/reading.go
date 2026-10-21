//
// Copyright (C) 2021-2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cast"

	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"

	"github.com/clint456/edgex-go/internal/core/data/container"
	"github.com/clint456/edgex-go/internal/core/data/query"
	"github.com/clint456/edgex-go/internal/pkg/utils"
)

// ReadingTotalCount return the count of all of readings currently stored in the database and error if any
func (a *CoreDataApp) ReadingTotalCount(dic *di.Container) (int64, errors.EdgeX) {
	dbClient := container.DBClientFrom(dic.Get)

	count, err := dbClient.ReadingTotalCount()
	if err != nil {
		return 0, errors.NewCommonEdgeXWrapper(err)
	}

	return count, nil
}

// AllReadings query events by offset, and limit
func (a *CoreDataApp) AllReadings(parms query.Parameters, dic *di.Container) (readings []dtos.BaseReading, totalCount int64, err errors.EdgeX) {
	dbClient := container.DBClientFrom(dic.Get)

	readingModels, err := dbClient.AllReadings(parms.Offset, parms.Limit)
	if err != nil {
		return readings, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	readings, err = convertReadingModelsToDTOs(readingModels)
	processNumericReadings(parms.Numeric, readings)

	if parms.Offset < 0 {
		return readings, 0, err // skip total count
	}
	totalCount, err = dbClient.ReadingTotalCount()
	if err != nil {
		return readings, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	cont, err := utils.CheckCountRange(totalCount, parms.Offset, parms.Limit)
	if !cont {
		return []dtos.BaseReading{}, totalCount, err
	}
	return readings, totalCount, err
}

// ReadingsByResourceName query readings with offset, limit, and resource name
func (a *CoreDataApp) ReadingsByResourceName(parms query.Parameters, resourceName string, dic *di.Container) (readings []dtos.BaseReading, totalCount int64, err errors.EdgeX) {
	if resourceName == "" {
		return readings, totalCount, errors.NewCommonEdgeX(errors.KindContractInvalid, "resourceName is empty", nil)
	}
	dbClient := container.DBClientFrom(dic.Get)

	readingModels, err := dbClient.ReadingsByResourceName(parms.Offset, parms.Limit, resourceName)
	if err != nil {
		return readings, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	readings, err = convertReadingModelsToDTOs(readingModels)
	processNumericReadings(parms.Numeric, readings)

	if parms.Offset < 0 {
		return readings, 0, err // skip total count
	}
	totalCount, err = dbClient.ReadingCountByResourceName(resourceName)
	if err != nil {
		return readings, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	cont, err := utils.CheckCountRange(totalCount, parms.Offset, parms.Limit)
	if !cont {
		return []dtos.BaseReading{}, totalCount, err
	}
	return readings, totalCount, err
}

// ReadingsByNodeName query readings with offset, limit, and node name
func (a *CoreDataApp) ReadingsByNodeName(parms query.Parameters, name string, dic *di.Container) (readings []dtos.BaseReading, totalCount int64, err errors.EdgeX) {
	if name == "" {
		return readings, totalCount, errors.NewCommonEdgeX(errors.KindContractInvalid, "name is empty", nil)
	}
	dbClient := container.DBClientFrom(dic.Get)

	readingModels, err := dbClient.ReadingsByNodeName(parms.Offset, parms.Limit, name)
	if err != nil {
		return readings, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	readings, err = convertReadingModelsToDTOs(readingModels)
	processNumericReadings(parms.Numeric, readings)

	if parms.Offset < 0 {
		return readings, 0, err // skip total count
	}
	totalCount, err = dbClient.ReadingCountByNodeName(name)
	if err != nil {
		return readings, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	cont, err := utils.CheckCountRange(totalCount, parms.Offset, parms.Limit)
	if !cont {
		return []dtos.BaseReading{}, totalCount, err
	}
	return readings, totalCount, err
}

// ReadingsByTimeRange query readings with offset, limit and time range
func (a *CoreDataApp) ReadingsByTimeRange(parms query.Parameters, dic *di.Container) (readings []dtos.BaseReading, totalCount int64, err errors.EdgeX) {
	dbClient := container.DBClientFrom(dic.Get)

	readingModels, err := dbClient.ReadingsByTimeRange(parms.Start, parms.End, parms.Offset, parms.Limit)
	if err != nil {
		return readings, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	readings, err = convertReadingModelsToDTOs(readingModels)
	processNumericReadings(parms.Numeric, readings)

	if parms.Offset < 0 {
		return readings, 0, err // skip total count
	}
	totalCount, err = dbClient.ReadingCountByTimeRange(parms.Start, parms.End)
	if err != nil {
		return readings, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	cont, err := utils.CheckCountRange(totalCount, parms.Offset, parms.Limit)
	if !cont {
		return []dtos.BaseReading{}, totalCount, err
	}
	return readings, totalCount, err
}

func convertReadingModelsToDTOs(readingModels []models.Reading) (readings []dtos.BaseReading, err errors.EdgeX) {
	readings = make([]dtos.BaseReading, len(readingModels))
	for i, r := range readingModels {
		readings[i] = dtos.FromReadingModelToDTO(r)
	}
	return readings, nil
}

// ReadingCountByNodeName return the count of all of readings associated with given node and error if any
func (a *CoreDataApp) ReadingCountByNodeName(nodeName string, dic *di.Container) (int64, errors.EdgeX) {
	if nodeName == "" {
		return 0, errors.NewCommonEdgeX(errors.KindContractInvalid, "name is empty", nil)
	}
	dbClient := container.DBClientFrom(dic.Get)
	count, err := dbClient.ReadingCountByNodeName(nodeName)
	if err != nil {
		return 0, errors.NewCommonEdgeXWrapper(err)
	}

	return count, nil
}

// ReadingsByResourceNameAndTimeRange returns readings by resource name and specified time range. Readings are sorted in descending order of origin time.
func (a *CoreDataApp) ReadingsByResourceNameAndTimeRange(resourceName string, parms query.Parameters, dic *di.Container) (readings []dtos.BaseReading, totalCount int64, err errors.EdgeX) {
	if resourceName == "" {
		return readings, totalCount, errors.NewCommonEdgeX(errors.KindContractInvalid, "resourceName is empty", nil)
	}
	dbClient := container.DBClientFrom(dic.Get)

	readingModels, err := dbClient.ReadingsByResourceNameAndTimeRange(resourceName, parms.Start, parms.End, parms.Offset, parms.Limit)
	if err != nil {
		return readings, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	readings, err = convertReadingModelsToDTOs(readingModels)
	processNumericReadings(parms.Numeric, readings)

	if parms.Offset < 0 {
		return readings, 0, err // skip total count
	}
	totalCount, err = dbClient.ReadingCountByResourceNameAndTimeRange(resourceName, parms.Start, parms.End)
	if err != nil {
		return readings, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	cont, err := utils.CheckCountRange(totalCount, parms.Offset, parms.Limit)
	if !cont {
		return []dtos.BaseReading{}, totalCount, err
	}
	return readings, totalCount, err
}

// ReadingsByNodeNameAndResourceName query readings with offset, limit, node name and its associated resource name
func (a *CoreDataApp) ReadingsByNodeNameAndResourceName(nodeName string, resourceName string, parms query.Parameters, dic *di.Container) (readings []dtos.BaseReading, totalCount int64, err errors.EdgeX) {
	if nodeName == "" {
		return readings, totalCount, errors.NewCommonEdgeX(errors.KindContractInvalid, "node name is empty", nil)
	}
	if resourceName == "" {
		return readings, totalCount, errors.NewCommonEdgeX(errors.KindContractInvalid, "resource name is empty", nil)
	}

	dbClient := container.DBClientFrom(dic.Get)

	readingModels, err := dbClient.ReadingsByNodeNameAndResourceName(nodeName, resourceName, parms.Offset, parms.Limit)
	if err != nil {
		return readings, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	readings, err = convertReadingModelsToDTOs(readingModels)
	processNumericReadings(parms.Numeric, readings)

	if parms.Offset < 0 {
		return readings, 0, err // skip total count
	}
	totalCount, err = dbClient.ReadingCountByNodeNameAndResourceName(nodeName, resourceName)
	if err != nil {
		return readings, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	cont, err := utils.CheckCountRange(totalCount, parms.Offset, parms.Limit)
	if !cont {
		return []dtos.BaseReading{}, totalCount, err
	}
	return readings, totalCount, err
}

// ReadingsByNodeNameAndResourceNameAndTimeRange query readings with offset, limit, node name, its associated resource name and specified time range
func (a *CoreDataApp) ReadingsByNodeNameAndResourceNameAndTimeRange(nodeName string, resourceName string, parms query.Parameters, dic *di.Container) (readings []dtos.BaseReading, totalCount int64, err errors.EdgeX) {
	readingModels, err := getFilteredReadingModelsByResource(nodeName, resourceName, parms, dic)
	if err != nil {
		return readings, totalCount, err
	}
	readings, err = convertReadingModelsToDTOs(readingModels)
	processNumericReadings(parms.Numeric, readings)

	if parms.Offset < 0 {
		return readings, 0, err // skip total count
	}
	dbClient := container.DBClientFrom(dic.Get)
	totalCount, err = dbClient.ReadingCountByNodeNameAndResourceNameAndTimeRange(nodeName, resourceName, parms.Start, parms.End)
	if err != nil {
		return readings, totalCount, errors.NewCommonEdgeXWrapper(err)
	}
	cont, err := utils.CheckCountRange(totalCount, parms.Offset, parms.Limit)
	if !cont {
		return []dtos.BaseReading{}, totalCount, err
	}
	return readings, totalCount, err
}

// ReadingsByNodeNameAndResourceNamesAndTimeRange query readings with offset, limit, node name, its associated resource name and specified time range
func (a *CoreDataApp) ReadingsByNodeNameAndResourceNamesAndTimeRange(nodeName string, resourceNames []string, parms query.Parameters, dic *di.Container) (readings []dtos.BaseReading, totalCount int64, err errors.EdgeX) {
	if nodeName == "" {
		return readings, totalCount, errors.NewCommonEdgeX(errors.KindContractInvalid, "node name is empty", nil)
	}

	if parms.Offset >= 0 {
		totalCount, err = getFilteredReadingCount(nodeName, resourceNames, parms.Start, parms.End, dic)
		if err != nil {
			return readings, totalCount, errors.NewCommonEdgeXWrapper(err)
		}
		if cont, err := utils.CheckCountRange(totalCount, parms.Offset, parms.Limit); !cont {
			return []dtos.BaseReading{}, totalCount, err
		}
	}

	readingModels, err := getFilteredReadingModelsByMultipleResources(nodeName, resourceNames, parms, dic)
	if err != nil {
		return readings, totalCount, errors.NewCommonEdgeXWrapper(err)
	}

	readings, err = convertReadingModelsToDTOs(readingModels)
	processNumericReadings(parms.Numeric, readings)
	return readings, totalCount, err
}

func getFilteredReadingModelsByResource(nodeName, resourceName string, parms query.Parameters, dic *di.Container) (readingModels []models.Reading, err errors.EdgeX) {
	if nodeName == "" {
		return readingModels, errors.NewCommonEdgeX(errors.KindContractInvalid, "node name is empty", nil)
	}
	if resourceName == "" {
		return readingModels, errors.NewCommonEdgeX(errors.KindContractInvalid, "resource name is empty", nil)
	}

	dbClient := container.DBClientFrom(dic.Get)

	readingModels, err = dbClient.ReadingsByNodeNameAndResourceNameAndTimeRange(nodeName, resourceName, parms.Start, parms.End, parms.Offset, parms.Limit)
	if err != nil {
		return readingModels, errors.NewCommonEdgeXWrapper(err)
	}

	return readingModels, nil
}

func getFilteredReadingCount(nodeName string, resourceNames []string, start, end int64, dic *di.Container) (int64, errors.EdgeX) {
	dbClient := container.DBClientFrom(dic.Get)
	if len(resourceNames) > 0 {
		return dbClient.ReadingCountByNodeNameAndResourceNamesAndTimeRange(nodeName, resourceNames, start, end)
	}
	return dbClient.ReadingCountByNodeNameAndTimeRange(nodeName, start, end)
}

func getFilteredReadingModelsByMultipleResources(nodeName string, resourceNames []string, parms query.Parameters, dic *di.Container) (readingModels []models.Reading, err errors.EdgeX) {
	dbClient := container.DBClientFrom(dic.Get)

	if len(resourceNames) > 0 {
		return dbClient.ReadingsByNodeNameAndResourceNamesAndTimeRange(nodeName, resourceNames, parms.Start, parms.End, parms.Offset, parms.Limit)
	}
	return dbClient.ReadingsByNodeNameAndTimeRange(nodeName, parms.Start, parms.End, parms.Offset, parms.Limit)
}

func processNumericReadings(isNumeric bool, readings []dtos.BaseReading) {
	for i := range readings {
		if isNumeric {
			convertToNumeric(&readings[i])
		} else {
			convertToString(&readings[i])
		}
	}
}

// convertToNumeric converts SimpleReadding value to NumericReading value
func convertToNumeric(reading *dtos.BaseReading) {
	if len(reading.Value) == 0 {
		return
	}
	switch reading.ValueType {
	case common.ValueTypeInt8, common.ValueTypeInt16, common.ValueTypeInt32, common.ValueTypeInt64:
		reading.NumericValue = cast.ToInt64(reading.Value)
	case common.ValueTypeUint8, common.ValueTypeUint16, common.ValueTypeUint32, common.ValueTypeUint64:
		reading.NumericValue = cast.ToUint64(reading.Value)
	case common.ValueTypeFloat32, common.ValueTypeFloat64:
		reading.NumericValue = cast.ToFloat64(reading.Value)
	case common.ValueTypeInt8Array, common.ValueTypeInt16Array, common.ValueTypeInt32Array, common.ValueTypeInt64Array:
		arrayValue, err := parseArray(reading.Value, cast.ToInt64)
		if err != nil {
			return
		}
		reading.NumericValue = arrayValue
	case common.ValueTypeUint8Array, common.ValueTypeUint16Array, common.ValueTypeUint32Array, common.ValueTypeUint64Array:
		arrayValue, err := parseArray(reading.Value, cast.ToUint64)
		if err != nil {
			return
		}
		reading.NumericValue = arrayValue
	case common.ValueTypeFloat32Array, common.ValueTypeFloat64Array:
		arrayValue, err := parseArray(reading.Value, cast.ToFloat64)
		if err != nil {
			return
		}
		reading.NumericValue = arrayValue
	default:
		// skip not matched data type like bool, string
		return
	}
	reading.Value = ""
}

// convertToNumeric converts NumericReading value to SimpleReadding value
func convertToString(reading *dtos.BaseReading) {
	if reading.NumericValue == nil {
		return
	}
	switch reading.ValueType {
	case common.ValueTypeFloat32:
		reading.Value = strconv.FormatFloat(cast.ToFloat64(reading.NumericValue), 'e', -1, 32)
	case common.ValueTypeFloat64:
		reading.Value = strconv.FormatFloat(cast.ToFloat64(reading.NumericValue), 'e', -1, 64)
	default:
		reading.Value = fmt.Sprintf("%v", reading.NumericValue)
	}
	reading.NumericValue = nil
}

func parseArray[T any](val string, conv func(any) T) ([]T, error) {
	if len(val) <= 2 {
		return nil, fmt.Errorf("invalid array string: length must be greater than 2, got %d", len(val))
	}

	parts := strings.Split(val[1:len(val)-1], ",") // trim "[" and "]"
	result := make([]T, len(parts))
	for i, p := range parts {
		result[i] = conv(strings.TrimSpace(p))
	}

	return result, nil
}
