//
// Copyright (C) 2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
)

func (c *Client) AllReadingsAggregation(aggregateFunc string, offset, limit int) ([]models.Reading, errors.EdgeX) {
	c.loggingClient.Warn("AllReadingsAggregation function didn't implement")
	return nil, nil
}

func (c *Client) AllReadingsAggregationByTimeRange(aggregateFun string, start, end int64, offset, limit int) ([]models.Reading, errors.EdgeX) {
	c.loggingClient.Warn("AllReadingsAggregationByTimeRange function didn't implement")
	return nil, nil
}

func (c *Client) ReadingsAggregationByResourceName(resourceName string, aggregateFunc string, offset int, limit int) ([]models.Reading, errors.EdgeX) {
	c.loggingClient.Warn("ReadingsAggregationByResourceName function didn't implement")
	return nil, nil
}

func (c *Client) ReadingsAggregationByResourceNameAndTimeRange(resourceName string, aggregateFun string, start int64, end int64, offset int, limit int) ([]models.Reading, errors.EdgeX) {
	c.loggingClient.Warn("ReadingsAggregationByResourceNameAndTimeRange function didn't implement")
	return nil, nil
}

func (c *Client) ReadingsAggregationByNodeName(nodeName string, aggregateFunc string, offset int, limit int) ([]models.Reading, errors.EdgeX) {
	c.loggingClient.Warn("ReadingsAggregationByNodeName function didn't implement")
	return nil, nil
}

func (c *Client) ReadingsAggregationByNodeNameAndTimeRange(nodeName string, aggregateFun string, start int64, end int64, offset int, limit int) ([]models.Reading, errors.EdgeX) {
	c.loggingClient.Warn("ReadingsAggregationByNodeNameAndTimeRange function didn't implement")
	return nil, nil
}

func (c *Client) ReadingsAggregationByNodeNameAndResourceName(nodeName string, resourceName string, aggregateFunc string, offset int, limit int) ([]models.Reading, errors.EdgeX) {
	c.loggingClient.Warn("AllReadingsAggregation function didn't implement")
	return nil, nil
}

func (c *Client) ReadingsAggregationByNodeNameAndResourceNameAndTimeRange(nodeName string, resourceName string, aggregateFunc string, start int64, end int64, offset int, limit int) ([]models.Reading, errors.EdgeX) {
	c.loggingClient.Warn("ReadingsAggregationByNodeNameAndResourceNameAndTimeRange function didn't implement")
	return nil, nil
}
