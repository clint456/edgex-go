//
// Copyright (C) 2020-2023 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"fmt"
	"github.com/gomodule/redigo/redis"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	model "github.com/edgexfoundry/go-mod-core-contracts/v4/models"

	"github.com/clint456/edgex-go/internal/pkg/db"
	redisClient "github.com/clint456/edgex-go/internal/pkg/db/redis"

	"github.com/google/uuid"
)

type Client struct {
	*redisClient.Client
	loggingClient logger.LoggingClient
}

func NewClient(config db.Configuration, logger logger.LoggingClient) (*Client, errors.EdgeX) {
	var err error
	dc := &Client{}
	dc.Client, err = redisClient.NewClient(config, logger)
	dc.loggingClient = logger
	if err != nil {
		return nil, errors.NewCommonEdgeX(errors.KindDatabaseError, "redis client creation failed", err)
	}

	return dc, nil
}

func closeRedisConnection(conn redis.Conn, logger logger.LoggingClient) {
	err := conn.Close()
	if err != nil {
		logger.Errorf("error occured while closing the redis connection: %s", err.Error())
	}
}

// AddEvent adds a new event
func (c *Client) AddEvent(e model.Event) (model.Event, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)
	if e.Id != "" {
		_, err := uuid.Parse(e.Id)
		if err != nil {
			return model.Event{}, errors.NewCommonEdgeX(errors.KindInvalidId, "uuid parsing failed", err)
		}
	}

	return addEvent(conn, e)
}

// EventById gets an event by id
func (c *Client) EventById(id string) (event model.Event, edgeXerr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	event, edgeXerr = eventById(conn, id)
	if edgeXerr != nil {
		return event, errors.NewCommonEdgeX(errors.Kind(edgeXerr), fmt.Sprintf("fail to query event by id %s", id), edgeXerr)
	}

	return
}

// DeleteEventById removes an event by id
func (c *Client) DeleteEventById(id string) (edgeXerr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	edgeXerr = deleteEventById(conn, id)
	if edgeXerr != nil {
		return errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return
}

// Add a new node profle
func (c *Client) AddProfile(dp model.Profile) (model.Profile, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	if dp.Id != "" {
		_, err := uuid.Parse(dp.Id)
		if err != nil {
			return model.Profile{}, errors.NewCommonEdgeX(errors.KindInvalidId, "ID failed UUID parsing", err)
		}
	} else {
		dp.Id = uuid.New().String()
	}

	return addProfile(conn, dp)
}

// UpdateProfile updates a new node profile
func (c *Client) UpdateProfile(dp model.Profile) errors.EdgeX {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)
	return updateProfile(conn, dp)
}

// ProfileNameExists checks the node profile exists by name
func (c *Client) ProfileNameExists(name string) (bool, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)
	return profileNameExists(conn, name)
}

// AddNodeService adds a new node service
func (c *Client) AddNodeService(ds model.NodeService) (model.NodeService, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	if len(ds.Id) == 0 {
		ds.Id = uuid.New().String()
	}

	return addNodeService(conn, ds)
}

// NodeServiceByName gets a node service by name
func (c *Client) NodeServiceByName(name string) (nodeService model.NodeService, edgeXerr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	nodeService, edgeXerr = nodeServiceByName(conn, name)
	if edgeXerr != nil {
		return nodeService, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return
}

// NodeServiceById gets a node service by id
func (c *Client) NodeServiceById(id string) (nodeService model.NodeService, edgeXerr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	nodeService, edgeXerr = nodeServiceById(conn, id)
	if edgeXerr != nil {
		return nodeService, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return
}

// DeleteNodeServiceById deletes a node service by id
func (c *Client) DeleteNodeServiceById(id string) errors.EdgeX {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	edgeXerr := deleteNodeServiceById(conn, id)
	if edgeXerr != nil {
		return errors.NewCommonEdgeX(errors.Kind(edgeXerr), fmt.Sprintf("fail to delete the node service with id %s", id), edgeXerr)
	}

	return nil
}

// DeleteNodeServiceByName deletes a node service by name
func (c *Client) DeleteNodeServiceByName(name string) errors.EdgeX {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	edgeXerr := deleteNodeServiceByName(conn, name)
	if edgeXerr != nil {
		return errors.NewCommonEdgeX(errors.Kind(edgeXerr), fmt.Sprintf("fail to delete the node service with name %s", name), edgeXerr)
	}

	return nil
}

// NodeServiceNameExists checks the node service exists by name
func (c *Client) NodeServiceNameExists(name string) (bool, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)
	return nodeServiceNameExist(conn, name)
}

// UpdateNodeService updates a node service
func (c *Client) UpdateNodeService(ds model.NodeService) errors.EdgeX {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)
	return updateNodeService(conn, ds)
}

// ProfileById gets a node profile by id
func (c *Client) ProfileById(id string) (profile model.Profile, err errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	profile, err = profileById(conn, id)
	if err != nil {
		return profile, errors.NewCommonEdgeXWrapper(err)
	}

	return
}

// ProfileByName gets a node profile by name
func (c *Client) ProfileByName(name string) (profile model.Profile, edgeXerr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	profile, edgeXerr = profileByName(conn, name)
	if edgeXerr != nil {
		return profile, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return
}

// DeleteProfileById deletes a node profile by id
func (c *Client) DeleteProfileById(id string) errors.EdgeX {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	edgeXerr := deleteProfileById(conn, id)
	if edgeXerr != nil {
		return errors.NewCommonEdgeX(errors.Kind(edgeXerr), fmt.Sprintf("fail to delete the node profile with id %s", id), edgeXerr)
	}

	return nil
}

// DeleteProfileByName deletes a node profile by name
func (c *Client) DeleteProfileByName(name string) errors.EdgeX {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	edgeXerr := deleteProfileByName(conn, name)
	if edgeXerr != nil {
		return errors.NewCommonEdgeX(errors.Kind(edgeXerr), fmt.Sprintf("fail to delete the node profile with name %s", name), edgeXerr)
	}

	return nil
}

// AllProfiles query node profiles with offset and limit
func (c *Client) AllProfiles(offset int, limit int, labels []string) ([]model.Profile, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	profiles, edgeXerr := profilesByLabels(conn, offset, limit, labels)
	if edgeXerr != nil {
		return profiles, errors.NewCommonEdgeXWrapper(edgeXerr)
	}
	return profiles, nil
}

// ProfilesByModel query node profiles with offset, limit and model
func (c *Client) ProfilesByModel(offset int, limit int, model string) ([]model.Profile, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	profiles, edgeXerr := profilesByModel(conn, offset, limit, model)
	if edgeXerr != nil {
		return profiles, errors.NewCommonEdgeXWrapper(edgeXerr)
	}
	return profiles, nil
}

// ProfilesByManufacturer query node profiles with offset, limit and manufacturer
func (c *Client) ProfilesByManufacturer(offset int, limit int, manufacturer string) ([]model.Profile, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	profiles, edgeXerr := profilesByManufacturer(conn, offset, limit, manufacturer)
	if edgeXerr != nil {
		return profiles, errors.NewCommonEdgeXWrapper(edgeXerr)
	}
	return profiles, nil
}

// ProfilesByManufacturerAndModel query node profiles with offset, limit, manufacturer and model
func (c *Client) ProfilesByManufacturerAndModel(offset int, limit int, manufacturer string, model string) ([]model.Profile, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	profiles, edgeXerr := profilesByManufacturerAndModel(conn, offset, limit, manufacturer, model)
	if edgeXerr != nil {
		return profiles, errors.NewCommonEdgeXWrapper(edgeXerr)
	}
	return profiles, nil
}

// EventTotalCount returns the total count of Event from the database
func (c *Client) EventTotalCount() (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberNumber(conn, ZCARD, EventsCollection)
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// EventCountByNodeName returns the count of Event associated a specific Node from the database
func (c *Client) EventCountByNodeName(nodeName string) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberNumber(conn, ZCARD, CreateKey(EventsCollectionNodeName, nodeName))
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// EventCountByTimeRange returns the count of Event by time range
func (c *Client) EventCountByTimeRange(startTime int64, endTime int64) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberCountByScoreRange(conn, EventsCollectionOrigin, startTime, endTime)
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// AllNodeServices returns multiple node services per query criteria, including
// offset: the number of items to skip before starting to collect the result set
// limit: The numbers of items to return
// labels: allows for querying a given object by associated user-defined labels
func (c *Client) AllNodeServices(offset int, limit int, labels []string) (nodeServices []model.NodeService, edgeXerr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	nodeServices, edgeXerr = nodeServicesByLabels(conn, offset, limit, labels)
	if edgeXerr != nil {
		return nodeServices, errors.NewCommonEdgeXWrapper(edgeXerr)
	}
	return nodeServices, nil
}

// Add a new node
func (c *Client) AddNode(d model.Node) (model.Node, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	if len(d.Id) == 0 {
		d.Id = uuid.New().String()
	}

	return addNode(conn, d)
}

// DeleteNodeById deletes a node by id
func (c *Client) DeleteNodeById(id string) errors.EdgeX {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	edgeXerr := deleteNodeById(conn, id)
	if edgeXerr != nil {
		return errors.NewCommonEdgeX(errors.Kind(edgeXerr), fmt.Sprintf("fail to delete the node with id %s", id), edgeXerr)
	}

	return nil
}

// DeleteNodeByName deletes a node by name
func (c *Client) DeleteNodeByName(name string) errors.EdgeX {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	edgeXerr := deleteNodeByName(conn, name)
	if edgeXerr != nil {
		return errors.NewCommonEdgeX(errors.Kind(edgeXerr), fmt.Sprintf("fail to delete the node with name %s", name), edgeXerr)
	}

	return nil
}

// NodesByServiceName query nodes by offset, limit and name
func (c *Client) NodesByServiceName(offset int, limit int, name string) (nodes []model.Node, edgeXerr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	nodes, edgeXerr = nodesByServiceName(conn, offset, limit, name)
	if edgeXerr != nil {
		return nodes, errors.NewCommonEdgeX(errors.Kind(edgeXerr),
			fmt.Sprintf("fail to query nodes by offset %d, limit %d and name %s", offset, limit, name), edgeXerr)
	}
	return nodes, nil
}

// NodeIdExists checks the node existence by id
func (c *Client) NodeIdExists(id string) (bool, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)
	exists, err := nodeIdExists(conn, id)
	if err != nil {
		return exists, errors.NewCommonEdgeX(errors.Kind(err), fmt.Sprintf("fail to check the node existence by id %s", id), err)
	}
	return exists, nil
}

// NodeNameExists checks the node existence by name
func (c *Client) NodeNameExists(name string) (bool, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)
	exists, err := nodeNameExists(conn, name)
	if err != nil {
		return exists, errors.NewCommonEdgeX(errors.Kind(err), fmt.Sprintf("fail to check the node existence by name %s", name), err)
	}
	return exists, nil
}

// NodeById gets a node by id
func (c *Client) NodeById(id string) (node model.Node, edgeXerr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	node, edgeXerr = nodeById(conn, id)
	if edgeXerr != nil {
		return node, errors.NewCommonEdgeX(errors.Kind(edgeXerr), fmt.Sprintf("fail to query node by id %s", id), edgeXerr)
	}

	return
}

// NodeByName gets a node by name
func (c *Client) NodeByName(name string) (node model.Node, edgeXerr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	node, edgeXerr = nodeByName(conn, name)
	if edgeXerr != nil {
		return node, errors.NewCommonEdgeX(errors.Kind(edgeXerr), fmt.Sprintf("fail to query node by name %s", name), edgeXerr)
	}

	return
}

// NodesByProfileName query nodes by offset, limit and profile name
func (c *Client) NodesByProfileName(offset int, limit int, profileName string) (nodes []model.Node, edgeXerr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	nodes, edgeXerr = nodesByProfileName(conn, offset, limit, profileName)
	if edgeXerr != nil {
		return nodes, errors.NewCommonEdgeX(errors.Kind(edgeXerr),
			fmt.Sprintf("fail to query nodes by offset %d, limit %d and name %s", offset, limit, profileName), edgeXerr)
	}
	return nodes, nil
}

// Update a node
func (c *Client) UpdateNode(d model.Node) errors.EdgeX {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	return updateNode(conn, d)
}

// AllEvents query events by offset and limit
func (c *Client) AllEvents(offset int, limit int) ([]model.Event, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	events, edgeXerr := c.allEvents(conn, offset, limit)
	if edgeXerr != nil {
		return events, errors.NewCommonEdgeX(errors.Kind(edgeXerr),
			fmt.Sprintf("fail to query events by offset %d and limit %d", offset, limit), edgeXerr)
	}
	return events, nil
}

// AllNodes query the nodes with offset, limit, and labels
func (c *Client) AllNodes(offset int, limit int, labels []string) ([]model.Node, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	nodes, edgeXerr := nodesByLabels(conn, offset, limit, labels)
	if edgeXerr != nil {
		return nodes, errors.NewCommonEdgeXWrapper(edgeXerr)
	}
	return nodes, nil
}

func (c *Client) NodeTree(parent string, levels int, offset int, limit int, labels []string) (int64, []model.Node, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	totalCount, nodes, edgeXerr := nodeTree(conn, parent, levels, offset, limit, labels)
	if edgeXerr != nil {
		return totalCount, nodes, errors.NewCommonEdgeXWrapper(edgeXerr)
	}
	return totalCount, nodes, nil
}

// EventsByNodeName query events by offset, limit and node name
func (c *Client) EventsByNodeName(offset int, limit int, name string) (events []model.Event, edgeXerr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	events, edgeXerr = eventsByNodeName(conn, offset, limit, name)
	if edgeXerr != nil {
		return events, errors.NewCommonEdgeX(errors.Kind(edgeXerr),
			fmt.Sprintf("fail to query events by offset %d, limit %d and name %s", offset, limit, name), edgeXerr)
	}
	return events, nil
}

// EventsByTimeRange query events by time range, offset, and limit
func (c *Client) EventsByTimeRange(startTime int64, endTime int64, offset int, limit int) (events []model.Event, edgeXerr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	events, edgeXerr = eventsByTimeRange(conn, startTime, endTime, offset, limit)
	if edgeXerr != nil {
		return events, errors.NewCommonEdgeX(errors.Kind(edgeXerr),
			fmt.Sprintf("fail to query events by time range %v ~ %v, offset %d, and limit %d", startTime, endTime, offset, limit), edgeXerr)
	}
	return events, nil
}

// ReadingTotalCount returns the total count of Event from the database
func (c *Client) ReadingTotalCount() (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberNumber(conn, ZCARD, ReadingsCollection)
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// AllReadings query events by offset, limit, and labels
func (c *Client) AllReadings(offset int, limit int) ([]model.Reading, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	readings, edgeXerr := allReadings(conn, offset, limit)
	if edgeXerr != nil {
		return readings, errors.NewCommonEdgeX(errors.Kind(edgeXerr),
			fmt.Sprintf("fail to query readings by offset %d, and limit %d", offset, limit), edgeXerr)
	}
	return readings, nil
}

// ReadingsByTimeRange query readings by time range, offset, and limit
func (c *Client) ReadingsByTimeRange(start int64, end int64, offset int, limit int) (readings []model.Reading, edgeXerr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	readings, edgeXerr = readingsByTimeRange(conn, start, end, offset, limit)
	if edgeXerr != nil {
		return readings, errors.NewCommonEdgeX(errors.Kind(edgeXerr),
			fmt.Sprintf("fail to query readings by time range %v ~ %v, offset %d, and limit %d", start, end, offset, limit), edgeXerr)
	}
	return readings, nil
}

// ReadingsByResourceName query readings by offset, limit and resource name
func (c *Client) ReadingsByResourceName(offset int, limit int, resourceName string) (readings []model.Reading, edgeXerr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	readings, edgeXerr = readingsByResourceName(conn, offset, limit, resourceName)
	if edgeXerr != nil {
		return readings, errors.NewCommonEdgeX(errors.Kind(edgeXerr),
			fmt.Sprintf("fail to query readings by offset %d, limit %d and resourceName %s", offset, limit, resourceName), edgeXerr)
	}
	return readings, nil
}

// ReadingsByNodeName query readings by offset, limit and node name
func (c *Client) ReadingsByNodeName(offset int, limit int, name string) (readings []model.Reading, edgeXerr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	readings, edgeXerr = readingsByNodeName(conn, offset, limit, name)
	if edgeXerr != nil {
		return readings, errors.NewCommonEdgeX(errors.Kind(edgeXerr),
			fmt.Sprintf("fail to query readings by offset %d, limit %d and name %s", offset, limit, name), edgeXerr)
	}
	return readings, nil
}

// ReadingCountByNodeName returns the count of Readings associated a specific Node from the database
func (c *Client) ReadingCountByNodeName(nodeName string) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberNumber(conn, ZCARD, CreateKey(ReadingsCollectionNodeName, nodeName))
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// ReadingCountByResourceName returns the count of Readings associated a specific Resource from the database
func (c *Client) ReadingCountByResourceName(resourceName string) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberNumber(conn, ZCARD, CreateKey(ReadingsCollectionResourceName, resourceName))
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// ReadingCountByResourceNameAndTimeRange returns the count of Readings associated a specific Resource from the database within specified time range
func (c *Client) ReadingCountByResourceNameAndTimeRange(resourceName string, startTime int64, endTime int64) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberCountByScoreRange(conn, CreateKey(ReadingsCollectionResourceName, resourceName), startTime, endTime)
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// ReadingCountByNodeNameAndResourceName returns the count of Readings associated with specified Resource and Node from the database
func (c *Client) ReadingCountByNodeNameAndResourceName(nodeName string, resourceName string) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberNumber(conn, ZCARD, CreateKey(ReadingsCollectionNodeNameResourceName, nodeName, resourceName))
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// ReadingCountByNodeNameAndResourceNameAndTimeRange returns the count of Readings associated with specified Resource and Node from the database within specified time range
func (c *Client) ReadingCountByNodeNameAndResourceNameAndTimeRange(nodeName string, resourceName string, start int64, end int64) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberCountByScoreRange(conn, CreateKey(ReadingsCollectionNodeNameResourceName, nodeName, resourceName), start, end)
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// ReadingCountByNodeNameAndResourceNamesAndTimeRange returns the count of readings by origin within the time range
// associated with the specified node and resourceName slice from db
func (c *Client) ReadingCountByNodeNameAndResourceNamesAndTimeRange(nodeName string, resourceNames []string, start int64, end int64) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	readings, err := readingsByNodeNameAndResourceNamesAndTimeRange(conn, nodeName, resourceNames, start, end, 0, -1)
	if err != nil {
		return 0, errors.NewCommonEdgeX(errors.Kind(err),
			fmt.Sprintf("fail to query readings by nodeName %s, resourceNames %v and time range %v ~ %v", nodeName, resourceNames, start, end), err)
	}

	return int64(len(readings)), nil
}

// ReadingCountByTimeRange returns the count of Readings from the database within specified time range
func (c *Client) ReadingCountByTimeRange(start int64, end int64) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberCountByScoreRange(conn, ReadingsCollectionOrigin, start, end)
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// ReadingsByResourceNameAndTimeRange query readings by resourceName and specified time range. Readings are sorted in descending order of origin time.
func (c *Client) ReadingsByResourceNameAndTimeRange(resourceName string, start int64, end int64, offset int, limit int) (readings []model.Reading, err errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	readings, err = readingsByResourceNameAndTimeRange(conn, resourceName, start, end, offset, limit)
	if err != nil {
		return readings, errors.NewCommonEdgeX(errors.Kind(err),
			fmt.Sprintf("fail to query readings by resourceName %s and time range %v ~ %v, offset %d, and limit %d", resourceName, start, end, offset, limit), err)
	}
	return readings, nil
}

func (c *Client) ReadingsByNodeNameAndResourceName(nodeName string, resourceName string, offset int, limit int) (readings []model.Reading, err errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	readings, err = readingsByNodeNameAndResourceName(conn, nodeName, resourceName, offset, limit)
	if err != nil {
		return readings, errors.NewCommonEdgeX(errors.Kind(err),
			fmt.Sprintf("fail to query readings by nodeName %s and resourceName %s", nodeName, resourceName), err)
	}

	return readings, nil
}

func (c *Client) ReadingsByNodeNameAndResourceNameAndTimeRange(nodeName string, resourceName string, start int64, end int64, offset int, limit int) (readings []model.Reading, err errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	readings, err = readingsByNodeNameAndResourceNameAndTimeRange(conn, nodeName, resourceName, start, end, offset, limit)
	if err != nil {
		return readings, errors.NewCommonEdgeX(errors.Kind(err),
			fmt.Sprintf("fail to query readings by nodeName %s, resourceName %s and time range %v ~ %v", nodeName, resourceName, start, end), err)
	}

	return readings, nil
}

func (c *Client) ReadingsByNodeNameAndResourceNamesAndTimeRange(nodeName string, resourceNames []string, start, end int64, offset, limit int) (readings []model.Reading, err errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	readings, err = readingsByNodeNameAndResourceNamesAndTimeRange(conn, nodeName, resourceNames, start, end, offset, limit)
	if err != nil {
		return readings, errors.NewCommonEdgeX(errors.Kind(err),
			fmt.Sprintf("fail to query readings by nodeName %s, resourceNames %v and time range %v ~ %v", nodeName, resourceNames, start, end), err)
	}

	return readings, nil
}

func (c *Client) ReadingsByNodeNameAndTimeRange(nodeName string, start int64, end int64, offset int, limit int) (readings []model.Reading, err errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	readings, err = readingsByNodeNameAndTimeRange(conn, nodeName, start, end, offset, limit)
	if err != nil {
		return readings, errors.NewCommonEdgeX(errors.Kind(err),
			fmt.Sprintf("fail to query readings by nodeName %s, and time range %v ~ %v", nodeName, start, end), err)
	}

	return readings, nil
}

func (c *Client) ReadingCountByNodeNameAndTimeRange(nodeName string, start int64, end int64) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberCountByScoreRange(conn, CreateKey(ReadingsCollectionNodeName, nodeName), start, end)
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// AddProvisionWatcher adds a new provision watcher
func (c *Client) AddProvisionWatcher(pw model.ProvisionWatcher) (model.ProvisionWatcher, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	if len(pw.Id) == 0 {
		pw.Id = uuid.New().String()
	}

	return addProvisionWatcher(conn, pw)
}

// ProvisionWatcherById gets a provision watcher by id
func (c *Client) ProvisionWatcherById(id string) (provisionWatcher model.ProvisionWatcher, edgexErr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	provisionWatcher, edgexErr = provisionWatcherById(conn, id)
	if edgexErr != nil {
		return provisionWatcher, errors.NewCommonEdgeX(errors.Kind(edgexErr), fmt.Sprintf("failed to query provision watcher by id %s", id), edgexErr)
	}

	return
}

// ProvisionWatcherByName gets a provision watcher by name
func (c *Client) ProvisionWatcherByName(name string) (provisionWatcher model.ProvisionWatcher, edgexErr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	provisionWatcher, edgexErr = provisionWatcherByName(conn, name)
	if edgexErr != nil {
		return provisionWatcher, errors.NewCommonEdgeXWrapper(edgexErr)
	}

	return
}

// ProvisionWatchersByServiceName query provision watchers by offset, limit and service name
func (c *Client) ProvisionWatchersByServiceName(offset int, limit int, name string) (provisionWatchers []model.ProvisionWatcher, edgexErr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	provisionWatchers, edgexErr = provisionWatchersByServiceName(conn, offset, limit, name)
	if edgexErr != nil {
		return provisionWatchers, errors.NewCommonEdgeX(errors.Kind(edgexErr),
			fmt.Sprintf("failed to query provision watcher by offset %d, limit %d and service name %s", offset, limit, name), edgexErr)
	}

	return
}

// ProvisionWatchersByProfileName query provision watchers by offset, limit and profile name
func (c *Client) ProvisionWatchersByProfileName(offset int, limit int, name string) (provisionWatchers []model.ProvisionWatcher, edgexErr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	provisionWatchers, edgexErr = provisionWatchersByProfileName(conn, offset, limit, name)
	if edgexErr != nil {
		return provisionWatchers, errors.NewCommonEdgeX(errors.Kind(edgexErr),
			fmt.Sprintf("failed to query provision watcher by offset %d, limit %d and profile name %s", offset, limit, name), edgexErr)
	}

	return
}

// AllProvisionWatchers query provision watchers with offset, limit and labels
func (c *Client) AllProvisionWatchers(offset int, limit int, labels []string) (provisionWatchers []model.ProvisionWatcher, edgexErr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	provisionWatchers, edgexErr = provisionWatchersByLabels(conn, offset, limit, labels)
	if edgexErr != nil {
		return provisionWatchers, errors.NewCommonEdgeXWrapper(edgexErr)
	}

	return
}

// DeleteProvisionWatcherByName deletes a provision watcher by name
func (c *Client) DeleteProvisionWatcherByName(name string) errors.EdgeX {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	edgeXerr := deleteProvisionWatcherByName(conn, name)
	if edgeXerr != nil {
		return errors.NewCommonEdgeX(errors.Kind(edgeXerr), fmt.Sprintf("failed to delete the provision watcher with name %s", name), edgeXerr)
	}

	return nil
}

// Update a provision watcher
func (c *Client) UpdateProvisionWatcher(pw model.ProvisionWatcher) errors.EdgeX {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	return updateProvisionWatcher(conn, pw)
}

// ProfileCountByLabels returns the total count of Node Profiles with labels specified.  If no label is specified, the total count of all node profiles will be returned.
func (c *Client) ProfileCountByLabels(labels []string) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberCountByLabels(conn, ZREVRANGE, ProfileCollection, labels)
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// ProfileCountByManufacturer returns the count of Node Profiles associated with specified manufacturer
func (c *Client) ProfileCountByManufacturer(manufacturer string) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberNumber(conn, ZCARD, CreateKey(ProfileCollectionManufacturer, manufacturer))
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// ProfileCountByModel returns the count of Node Profiles associated with specified model
func (c *Client) ProfileCountByModel(model string) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberNumber(conn, ZCARD, CreateKey(ProfileCollectionModel, model))
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// ProfileCountByManufacturerAndModel returns the count of Node Profiles associated with specified manufacturer and model
func (c *Client) ProfileCountByManufacturerAndModel(manufacturer, model string) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	profiles, edgeXerr := profilesByManufacturerAndModel(conn, 0, -1, manufacturer, model)
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return int64(len(profiles)), nil
}

func (c *Client) InUseResourceCount() (int64, errors.EdgeX) {
	c.loggingClient.Warn("InUseResourceCount function didn't implement")
	return 0, nil
}

// NodeServiceCountByLabels returns the total count of Node Services with labels specified.  If no label is specified, the total count of all node services will be returned.
func (c *Client) NodeServiceCountByLabels(labels []string) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberCountByLabels(conn, ZREVRANGE, NodeServiceCollection, labels)
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// NodeCountByLabels returns the total count of Nodes with labels specified.  If no label is specified, the total count of all nodes will be returned.
func (c *Client) NodeCountByLabels(labels []string) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberCountByLabels(conn, ZREVRANGE, NodeCollection, labels)
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// NodeCountByProfileName returns the count of Nodes associated with specified profile
func (c *Client) NodeCountByProfileName(profileName string) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberNumber(conn, ZCARD, CreateKey(NodeCollectionProfileName, profileName))
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// NodeCountByServiceName returns the count of Nodes associated with specified service
func (c *Client) NodeCountByServiceName(serviceName string) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberNumber(conn, ZCARD, CreateKey(NodeCollectionServiceName, serviceName))
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// ProvisionWatcherCountByLabels returns the total count of Provision Watchers with labels specified.  If no label is specified, the total count of all provision watchers will be returned.
func (c *Client) ProvisionWatcherCountByLabels(labels []string) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberCountByLabels(conn, ZREVRANGE, ProvisionWatcherCollection, labels)
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// ProvisionWatcherCountByServiceName returns the count of Provision Watcher associated with specified service
func (c *Client) ProvisionWatcherCountByServiceName(name string) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberNumber(conn, ZCARD, CreateKey(ProvisionWatcherCollectionServiceName, name))
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// ProvisionWatcherCountByProfileName returns the count of Provision Watcher associated with specified profile
func (c *Client) ProvisionWatcherCountByProfileName(name string) (int64, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	count, edgeXerr := getMemberNumber(conn, ZCARD, CreateKey(ProvisionWatcherCollectionProfileName, name))
	if edgeXerr != nil {
		return 0, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return count, nil
}

// LatestReadingByOffset returns a latest reading by offset
func (c *Client) LatestReadingByOffset(offset uint32) (model.Reading, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	reading, edgeXerr := latestReadingByOffset(conn, int(offset))
	if edgeXerr != nil {
		return nil, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	return reading, nil
}

// KeeperKeys returns the values stored for the specified key or with the same key prefix
func (c *Client) KeeperKeys(key string, keyOnly bool, isRaw bool) (kvs []model.KVResponse, edgeXerr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	key = replaceKeyDelimiterForDB(key)
	kvs, edgeXerr = keeperKeys(conn, key, keyOnly, isRaw)
	if edgeXerr != nil {
		// replace the key delimiter from colon(:) to slash(/)
		errRespKey := replaceKeyDelimiterForKeeper(key)
		return kvs, errors.NewCommonEdgeX(errors.Kind(edgeXerr), fmt.Sprintf("fail to get key %s", errRespKey), edgeXerr)
	}

	for _, kv := range kvs {
		// check if the KVResponse interface is KV struct
		if convertedKV, ok := kv.(*model.KVS); ok {
			// convert KVResponse interface to KV struct and replace the key delimiter
			convertedKV.SetKey(replaceKeyDelimiterForKeeper(convertedKV.Key))
		} else {
			// dereference the KeyOnly pointer and convert to string
			originKey := string(*kv.(*model.KeyOnly))
			// convert KVResponse interface to KeyOnly struct and replace the key delimiter
			kv.(*model.KeyOnly).SetKey(replaceKeyDelimiterForKeeper(originKey))
		}
	}
	return kvs, nil
}

// AddKeeperKeys returns the values stored for the specified key or with the same key prefix
func (c *Client) AddKeeperKeys(kv model.KVS, isFlatten bool) (keys []model.KeyOnly, edgeXerr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	kv.Key = replaceKeyDelimiterForDB(kv.Key)
	keys, edgeXerr = addKeeperKeys(conn, kv, isFlatten)
	if edgeXerr != nil {
		return nil, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	// replace the key delimiter in the response for Keeper
	for i, key := range keys {
		// convert the KeyOnly type to string
		originKey := string(key)

		// replace the key delimiter from colon(:) to slash(/)
		keys[i].SetKey(replaceKeyDelimiterForKeeper(originKey))
	}
	return keys, nil
}

// DeleteKeeperKeys delete the specified key or keys with the same prefix
func (c *Client) DeleteKeeperKeys(key string, prefixMatch bool) (kvs []model.KeyOnly, edgeXerr errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	key = replaceKeyDelimiterForDB(key)
	kvs, edgeXerr = deleteKeeperKeys(conn, key, prefixMatch)
	if edgeXerr != nil {
		return kvs, errors.NewCommonEdgeX(errors.Kind(edgeXerr), fmt.Sprintf("fail to delete key %s", replaceKeyDelimiterForKeeper(key)), edgeXerr)
	}

	for i, kv := range kvs {
		// convert the KeyOnly type to string
		originKey := string(kv)

		// replace the key delimiter from colon(:) to slash(/)
		kvs[i].SetKey(replaceKeyDelimiterForKeeper(originKey))
	}
	return kvs, nil
}

func (c *Client) AddRegistration(r model.Registration) (model.Registration, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	r, err := addRegistration(conn, r)
	if err != nil {
		return model.Registration{}, errors.NewCommonEdgeXWrapper(err)
	}

	return r, nil
}

func (c *Client) DeleteRegistrationByServiceId(id string) errors.EdgeX {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	err := deleteRegistrationByServiceId(conn, id)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	return nil
}

func (c *Client) Registrations() ([]model.Registration, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	registries, err := registrations(conn, 0, -1)
	if err != nil {
		return nil, errors.NewCommonEdgeXWrapper(err)
	}

	return registries, nil
}

func (c *Client) RegistrationByServiceId(id string) (model.Registration, errors.EdgeX) {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	r, err := registrationById(conn, id)
	if err != nil {
		return model.Registration{}, errors.NewCommonEdgeXWrapper(err)
	}

	return r, nil
}

func (c *Client) UpdateRegistration(r model.Registration) errors.EdgeX {
	conn := c.Pool.Get()
	defer closeRedisConnection(conn, c.loggingClient)

	err := updateRegistration(conn, r)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	return nil
}
