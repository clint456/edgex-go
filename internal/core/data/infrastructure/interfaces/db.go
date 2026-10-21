//
// Copyright (C) 2020-2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package interfaces

import (
	"github.com/clint456/edgex-go/internal/pkg/infrastructure/models"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	model "github.com/edgexfoundry/go-mod-core-contracts/v4/models"
)

type DBClient interface {
	CloseSession()

	AddEvent(e model.Event) (model.Event, errors.EdgeX)
	EventById(id string) (model.Event, errors.EdgeX)
	DeleteEventById(id string) errors.EdgeX
	EventTotalCount() (int64, errors.EdgeX)
	EventCountByNodeName(nodeName string) (int64, errors.EdgeX)
	EventCountByTimeRange(start int64, end int64) (int64, errors.EdgeX)
	EventCountByNodeNameAndSourceNameAndLimit(nodeName, sourceName string, limit int) (int64, errors.EdgeX)
	AllEvents(offset int, limit int) ([]model.Event, errors.EdgeX)
	EventsByNodeName(offset int, limit int, name string) ([]model.Event, errors.EdgeX)
	DeleteEventsByNodeName(nodeName string) errors.EdgeX
	DeleteEventsByNodeNameAndSourceName(nodeName, sourceName string) errors.EdgeX
	EventsByTimeRange(start int64, end int64, offset int, limit int) ([]model.Event, errors.EdgeX)
	DeleteEventsByAge(age int64) errors.EdgeX
	DeleteEventsByAgeAndNodeNameAndSourceName(age int64, nodeName, sourceName string) errors.EdgeX
	ReadingTotalCount() (int64, errors.EdgeX)
	AllReadings(offset int, limit int) ([]model.Reading, errors.EdgeX)
	ReadingsByTimeRange(start int64, end int64, offset int, limit int) ([]model.Reading, errors.EdgeX)
	ReadingsByResourceName(offset int, limit int, resourceName string) ([]model.Reading, errors.EdgeX)
	ReadingsByNodeName(offset int, limit int, name string) ([]model.Reading, errors.EdgeX)
	ReadingsByNodeNameAndResourceName(nodeName string, resourceName string, offset int, limit int) ([]model.Reading, errors.EdgeX)
	ReadingsByNodeNameAndResourceNameAndTimeRange(nodeName string, resourceName string, start int64, end int64, offset int, limit int) ([]model.Reading, errors.EdgeX)
	ReadingCountByNodeName(nodeName string) (int64, errors.EdgeX)
	ReadingCountByResourceName(resourceName string) (int64, errors.EdgeX)
	ReadingCountByResourceNameAndTimeRange(resourceName string, start int64, end int64) (int64, errors.EdgeX)
	ReadingCountByNodeNameAndResourceName(nodeName string, resourceName string) (int64, errors.EdgeX)
	ReadingCountByNodeNameAndResourceNameAndTimeRange(nodeName string, resourceName string, start int64, end int64) (int64, errors.EdgeX)
	ReadingCountByTimeRange(start int64, end int64) (int64, errors.EdgeX)
	ReadingsByResourceNameAndTimeRange(resourceName string, start int64, end int64, offset int, limit int) ([]model.Reading, errors.EdgeX)
	ReadingsByNodeNameAndResourceNamesAndTimeRange(nodeName string, resourceNames []string, start int64, end int64, offset, limit int) ([]model.Reading, errors.EdgeX)
	ReadingCountByNodeNameAndResourceNamesAndTimeRange(nodeName string, resourceName []string, start int64, end int64) (int64, errors.EdgeX)
	ReadingsByNodeNameAndTimeRange(nodeName string, start int64, end int64, offset int, limit int) ([]model.Reading, errors.EdgeX)
	ReadingCountByNodeNameAndTimeRange(nodeName string, start int64, end int64) (int64, errors.EdgeX)
	LatestReadingByOffset(offset uint32) (model.Reading, errors.EdgeX)
	LatestEventByNodeNameAndSourceNameAndOffset(nodeName string, sourceName string, offset int64) (model.Event, errors.EdgeX)
	LatestEventByNodeNameAndSourceNameAndAgeAndOffset(nodeName string, sourceName string, age, offset int64) (model.Event, errors.EdgeX)

	AllNodeInfos(offset int, limit int) ([]models.NodeInfo, errors.EdgeX)

	AllReadingsAggregation(aggregateFunc string, offset int, limit int) ([]model.Reading, errors.EdgeX)
	AllReadingsAggregationByTimeRange(aggregateFun string, start, end int64, offset int, limit int) ([]model.Reading, errors.EdgeX)
	ReadingsAggregationByResourceName(resourceName string, aggregateFunc string, offset int, limit int) ([]model.Reading, errors.EdgeX)
	ReadingsAggregationByResourceNameAndTimeRange(resourceName string, aggregateFun string, start int64, end int64, offset int, limit int) ([]model.Reading, errors.EdgeX)
	ReadingsAggregationByNodeName(nodeName string, aggregateFunc string, offset int, limit int) ([]model.Reading, errors.EdgeX)
	ReadingsAggregationByNodeNameAndTimeRange(nodeName string, aggregateFun string, start int64, end int64, offset int, limit int) ([]model.Reading, errors.EdgeX)
	ReadingsAggregationByNodeNameAndResourceName(nodeName string, resourceName string, aggregateFunc string, offset int, limit int) ([]model.Reading, errors.EdgeX)
	ReadingsAggregationByNodeNameAndResourceNameAndTimeRange(nodeName string, resourceName string, aggregateFunc string, start int64, end int64, offset int, limit int) ([]model.Reading, errors.EdgeX)
}
