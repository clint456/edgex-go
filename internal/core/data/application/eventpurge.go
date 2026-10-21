//
// Copyright (C) 2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/clint456/edgex-go/internal/core/data/container"
	dbModels "github.com/clint456/edgex-go/internal/pkg/infrastructure/models"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
)

// AsyncPurgeEvent purge events and related readings according to the retention capability.
func (a *CoreDataApp) AsyncPurgeEvent(ctx context.Context, dic *di.Container) errors.EdgeX {
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	config := container.ConfigurationFrom(dic.Get)
	interval, err := time.ParseDuration(config.Retention.Interval)
	if err != nil {
		return errors.NewCommonEdgeX(errors.KindContractInvalid, "retention interval parse failed", err)
	}
	if interval <= 0 {
		lc.Infof("Event retention is disabled because the retention interval is `%s`.", interval)
		return nil
	}

	// purge events by auto event
	a.asyncPurgeEventOnce.Do(func() {
		go func() {
			timer := time.NewTimer(interval)
			defer timer.Stop() // ensure timer is stopped to prevent resource leak
			for {
				timer.Reset(interval) // since event deletion might take lots of time, restart the timer to recount the time
				select {
				case <-ctx.Done():
					lc.Info("Exiting event retention")
					return
				case <-timer.C:
					err = purgeEvents(dic)
					if err != nil {
						lc.Errorf("Failed to purge events and readings, %v", err)
					}
				}
			}
		}()
	})

	return nil
}

func purgeEvents(dic *di.Container) errors.EdgeX {
	nodeInfoCache := container.NodeInfoCacheFrom(dic.Get)
	nodeInfoMap := nodeInfoCache.CloneNodeInfoMapWithSourceName()

	ep := newEventPurgeExecutor(nodeInfoMap)

	// Purge events matched conditions defined in the retention policies of auto events
	if err := ep.purgeEventsByAutoEvent(dic); err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	// Purge the remaining node & none-node events not covered by retention policies and auto events
	if err := ep.purgeEventsByNodeInfo(dic); err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	return nil
}

type eventPurgeExecutor struct {
	nodeInfoMap map[int]dbModels.NodeInfo
	mutex       sync.Mutex
}

func newEventPurgeExecutor(nodeInfoMap map[int]dbModels.NodeInfo) *eventPurgeExecutor {
	return &eventPurgeExecutor{nodeInfoMap: nodeInfoMap}
}

// cullMatchingNodeInfo evaluates all entries in the nodeInfoMap from cache against the provided nodeName and sourceName regular expression patterns.
// If both nodeName and sourceName patterns match, the entry is removed from nodeInfoMap.
func (ep *eventPurgeExecutor) cullMatchingNodeInfo(nodeName, sourceName string, lc logger.LoggingClient) {
	ep.mutex.Lock()
	defer ep.mutex.Unlock()

	for id, nodeInfo := range ep.nodeInfoMap {
		lc.Debugf("Checking nodeInfo id '%d' with node '%s', source '%s' from the purge event list", id, nodeName, nodeInfo.SourceName)

		nodeMatched, _ := regexp.MatchString(nodeName, nodeInfo.NodeName)
		if nodeMatched {
			sourceMatched, _ := regexp.MatchString(sourceName, nodeInfo.SourceName)
			if sourceMatched {
				lc.Debugf("Removing nodeInfo id '%d' with node '%s', source '%s' from the purge event list", id, nodeName, nodeInfo.SourceName)
				delete(ep.nodeInfoMap, id)
			}
		}
	}
}

// purgeEventsByAutoEvent iterates over the auto events defined for each node and removes any events that match the retention policy conditions specified in the auto event.
func (ep *eventPurgeExecutor) purgeEventsByAutoEvent(dic *di.Container) errors.EdgeX {
	config := container.ConfigurationFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	nodes := container.NodeStoreFrom(dic.Get).Nodes()

	lc.Debug("Starting purge events by auto events ......")

	for _, node := range nodes {
		for _, autoEvent := range node.AutoEvents {
			// apply the default retention policy, when maxCap/minCap/duration are not specified or equal to zero/empty string
			// which are mentioned in the documentation
			if autoEvent.Retention.MaxCap == 0 {
				autoEvent.Retention.MaxCap = config.Retention.DefaultMaxCap
			}
			if autoEvent.Retention.MinCap == 0 {
				autoEvent.Retention.MinCap = config.Retention.DefaultMinCap
			}
			if autoEvent.Retention.Duration == "" {
				autoEvent.Retention.Duration = config.Retention.DefaultDuration
			}

			ep.cullMatchingNodeInfo(node.Name, autoEvent.SourceName, lc)

			err := handleEventRetention(
				node.Name,
				autoEvent.SourceName,
				autoEvent.Retention.MaxCap,
				autoEvent.Retention.MinCap,
				autoEvent.Retention.Duration,
				dic,
			)
			if err != nil {
				lc.Errorf("failed to execute event retention for node '%s', source '%s', maxCap: '%d', minCap: '%d', duration '%s': %v",
					node.Name, autoEvent.SourceName, autoEvent.Retention.MaxCap, autoEvent.Retention.MinCap, autoEvent.Retention.Duration, err)
				continue
			}
		}
	}
	return nil
}

// purgeEventsByNodeInfo traverses the node info list and applies the retention settings from config
// based on the node name and source name in each node info entry.
func (ep *eventPurgeExecutor) purgeEventsByNodeInfo(dic *di.Container) errors.EdgeX {
	config := container.ConfigurationFrom(dic.Get)
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)

	lc.Debug("Starting purge events by retention policies on default config level ......")

	for _, nodeInfo := range ep.nodeInfoMap {
		err := handleEventRetention(
			nodeInfo.NodeName,
			nodeInfo.SourceName,
			config.Retention.DefaultMaxCap,
			config.Retention.DefaultMinCap,
			config.Retention.DefaultDuration,
			dic,
		)
		if err != nil {
			lc.Errorf(
				"failed to execute event retention for node info with node name: '%s', source name: '%s', error: %v",
				nodeInfo.NodeName,
				nodeInfo.SourceName,
				err,
			)
			continue
		}
	}
	return nil
}

func handleEventRetention(
	nodeName string,
	sourceName string,
	maxCap int64,
	minCap int64,
	durationStr string,
	dic *di.Container,
) errors.EdgeX {
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)

	if maxCap > 0 {
		isReach, err := isReachMaxCap(nodeName, sourceName, maxCap, dic)
		if err != nil {
			return errors.NewCommonEdgeXWrapper(err)
		}
		if !isReach {
			lc.Debugf(
				"Skip the event retention for the node with source name: '%s' because max capacity '%d' not reached",
				sourceName,
				maxCap,
			)
			return nil
		}
	}

	validDuration, duration := common.ParseDurationWithDay(durationStr)
	if !validDuration {
		return errors.NewCommonEdgeX(errors.KindServerError, fmt.Sprintf("retention duration '%s' parse failed", durationStr), nil)
	}

	lc.Debugf("Starting execute event retention for node name: '%s', source name: '%s', maxCap: %d, minCap: %d, duration: %s",
		nodeName, sourceName, maxCap, minCap, durationStr)

	if duration > 0 {
		return timeBasedEventRetention(nodeName, sourceName, minCap, duration, dic)
	}
	return countBasedEventRetention(nodeName, sourceName, minCap, dic)
}

func isReachMaxCap(nodeName, sourceName string, maxCap int64, dic *di.Container) (bool, errors.EdgeX) {
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	dbClient := container.DBClientFrom(dic.Get)
	// Check high watermark, count events by limit instead of count all to improve performance
	count, err := dbClient.EventCountByNodeNameAndSourceNameAndLimit(nodeName, sourceName, int(maxCap))
	if err != nil {
		return false, errors.NewCommonEdgeXWrapper(err)
	}
	if count < maxCap {
		lc.Debugf(
			"Skip the event retention for the source name `%s` of node `%s`, the event number `%d` is less than the max capacity `%d`.",
			sourceName, nodeName, count, maxCap,
		)
		return false, nil
	}
	return true, nil
}

func timeBasedEventRetention(nodeName, sourceName string, minCap int64, duration time.Duration, dic *di.Container) errors.EdgeX {
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	dbClient := container.DBClientFrom(dic.Get)

	if minCap <= 0 {
		lc.Debugf("MinCap is disabled, purge events by duration '%d' and nodeName '%s', and sourceName '%s'", duration, nodeName, sourceName)
		err := dbClient.DeleteEventsByAgeAndNodeNameAndSourceName(duration.Nanoseconds(), nodeName, sourceName)
		if err != nil {
			return errors.NewCommonEdgeX(errors.Kind(err),
				fmt.Sprintf("failed to delete events and readings with specific nodeName '%s', sourceName '%s', and duration '%s'",
					nodeName, sourceName, duration), err)
		}
	} else {
		lc.Debugf("MinCap is '%d', purge events by duration '%d' and nodeName '%s', and sourceName '%s' to meet the minCap",
			minCap, duration, nodeName, sourceName)
		// Find the event that the age is within the duration and use offset as minCap to keep data
		// SELECT * FROM core_data.event WHERE event.origin <= $1 and nodename=$2 and sourcename=$3 ORDER BY origin desc offset $4;
		event, err := dbClient.LatestEventByNodeNameAndSourceNameAndAgeAndOffset(nodeName, sourceName, duration.Nanoseconds(), minCap)
		if errors.Kind(err) == errors.KindEntityDoesNotExist {
			lc.Debugf("Skip the event retention for the event source '%s' of node '%s', the event number might equal or less than the minCap", sourceName, nodeName)
			return nil
		} else if err != nil {
			return errors.NewCommonEdgeX(errors.Kind(err),
				fmt.Sprintf("failed to delete events and readings with specific nodeName '%s', sourceName '%s', duration '%s', and minCap '%d'",
					nodeName, sourceName, duration, minCap), err)
		}

		age := time.Now().UnixNano() - event.Origin
		err = dbClient.DeleteEventsByAgeAndNodeNameAndSourceName(age, nodeName, sourceName)
		if err != nil {
			return errors.NewCommonEdgeX(errors.Kind(err), fmt.Sprintf("failed to delete events and readings with specific nodeName '%s', sourceName '%s', and minCap '%d'",
				nodeName, sourceName, minCap), err)
		}
	}
	return nil
}

func countBasedEventRetention(nodeName, sourceName string, minCap int64, dic *di.Container) errors.EdgeX {
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	dbClient := container.DBClientFrom(dic.Get)

	if minCap <= 0 {
		lc.Debugf("MinCap is disabled, purge events by nodeName '%s' and sourceName '%s'", nodeName, sourceName)
		err := dbClient.DeleteEventsByNodeNameAndSourceName(nodeName, sourceName)
		if err != nil {
			return errors.NewCommonEdgeX(errors.Kind(err), fmt.Sprintf("failed to delete events and readings with specific nodeName '%s', sourceName '%s', and minCap '%d'",
				nodeName, sourceName, minCap), err)
		}
	} else {
		lc.Debugf("MinCap is '%d', purge events by nodeName '%s' and sourceName '%s' to meet the minCap",
			minCap, nodeName, sourceName)
		event, err := dbClient.LatestEventByNodeNameAndSourceNameAndOffset(nodeName, sourceName, minCap)
		if errors.Kind(err) == errors.KindEntityDoesNotExist {
			lc.Debugf("Skip the event retention for the auto event source '%s' of node '%s', the event number might equal or less than the minCap", sourceName, nodeName)
			return nil
		} else if err != nil {
			return errors.NewCommonEdgeX(errors.Kind(err), fmt.Sprintf("failed to delete events and readings with specific nodeName '%s', sourceName '%s', and minCap '%d'",
				nodeName, sourceName, minCap), err)
		}
		age := time.Now().UnixNano() - event.Origin
		err = dbClient.DeleteEventsByAgeAndNodeNameAndSourceName(age, nodeName, sourceName)
		if err != nil {
			return errors.NewCommonEdgeX(errors.Kind(err), fmt.Sprintf("failed to delete events and readings with specific nodeName '%s', sourceName '%s', and minCap '%d'",
				nodeName, sourceName, minCap), err)
		}
	}
	return nil
}
