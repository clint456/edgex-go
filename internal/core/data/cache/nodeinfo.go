//
// Copyright (C) 2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"crypto/md5" // #nosec
	"fmt"
	"sync"

	"github.com/clint456/edgex-go/internal/pkg/infrastructure/models"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
)

// NodeInfoCache used to store node_info id
type NodeInfoCache interface {
	GetNodeInfoId(nodeInfo models.NodeInfo) (int, bool)
	CloneNodeInfoMapWithSourceName() map[int]models.NodeInfo
	Add(nodeInfo models.NodeInfo)
	Remove(nodeInfo models.NodeInfo)
}

type nodeInfoCache struct {
	lc               logger.LoggingClient
	nodeInfoKeyIdMap map[string]int
	nodeInfoMap      map[int]models.NodeInfo
	mutex            sync.RWMutex
}

func NewNodeInfoCache(dic *di.Container, nodeInfos []models.NodeInfo) NodeInfoCache {
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	cache := &nodeInfoCache{
		lc:               lc,
		nodeInfoMap:      make(map[int]models.NodeInfo, len(nodeInfos)),
		nodeInfoKeyIdMap: make(map[string]int, len(nodeInfos)),
	}

	for _, nodeInfo := range nodeInfos {
		// Add id/nodeInfo to nodeInfoMap
		if _, exists := cache.nodeInfoMap[nodeInfo.Id]; exists {
			lc.Warnf("duplicate NodeInfo ID %d exists in nodeInfoMap, skip adding the nodeInfo id to cache", nodeInfo.Id)
		} else {
			cache.nodeInfoMap[nodeInfo.Id] = nodeInfo
		}

		// Add key/nodeInfoId to nodeInfoKeyIdMap
		key := cache.cacheKey(nodeInfo)
		if cachedId, exists := cache.nodeInfoKeyIdMap[key]; exists {
			lc.Warnf("duplicate cache key %q detected for both NodeInfo ID %d (exists) and %d (new), skip adding the new nodeInfo id to cache", key, cachedId, nodeInfo.Id)
		} else {
			cache.nodeInfoKeyIdMap[key] = nodeInfo.Id
		}
	}

	return cache
}

// GetNodeInfoId returns nodeInfoId from the cache
func (s *nodeInfoCache) GetNodeInfoId(nodeInfo models.NodeInfo) (int, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	key := s.cacheKey(nodeInfo)
	if id, ok := s.nodeInfoKeyIdMap[key]; ok {
		return id, true
	}
	return 0, false
}

// CloneNodeInfoMapWithSourceName returns node info map with source name from the cache
func (s *nodeInfoCache) CloneNodeInfoMapWithSourceName() map[int]models.NodeInfo {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	newNodeInfoMap := make(map[int]models.NodeInfo)
	for id, nodeInfo := range s.nodeInfoMap {
		// Only copy the nodeInfo with SourceName into the new map
		if nodeInfo.SourceName != "" {
			newNodeInfoMap[id] = nodeInfo
		}
	}
	return newNodeInfoMap
}

func (s *nodeInfoCache) cacheKey(nodeInfo models.NodeInfo) string {
	data := [][]byte{
		[]byte(nodeInfo.NodeName), []byte(nodeInfo.ProfileName), []byte(nodeInfo.SourceName),
		[]byte(nodeInfo.ResourceName), []byte(nodeInfo.ValueType), []byte(nodeInfo.Units),
		[]byte(nodeInfo.MediaType), []byte(fmt.Sprintf("%v", nodeInfo.Tags)),
	}
	var byteData []byte
	for _, b := range data {
		byteData = append(byteData, b...)
	}
	return fmt.Sprintf("%x", md5.Sum(byteData)) // #nosec
}

// Add adds nodeInfoId and NodeInfo data into the cache
func (s *nodeInfoCache) Add(nodeInfo models.NodeInfo) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	key := s.cacheKey(nodeInfo)
	if _, ok := s.nodeInfoKeyIdMap[key]; !ok {
		s.lc.Debugf("adding nodeInfoId with id '%d' into cache...", nodeInfo.Id)
		s.nodeInfoKeyIdMap[key] = nodeInfo.Id
	} else {
		s.lc.Infof("nodeInfoIdCache already contains %d. skip adding...", nodeInfo.Id)
	}

	if _, ok := s.nodeInfoMap[nodeInfo.Id]; !ok {
		s.lc.Debugf("adding nodeInfo with id '%d' into nodeInfoMap cache...", nodeInfo.Id)
		s.nodeInfoMap[nodeInfo.Id] = nodeInfo
	} else {
		s.lc.Infof("nodeInfoMap cache already contains nodeInfo id '%d'. skip adding...", nodeInfo.Id)
	}
}

// Remove removes nodeInfoId and NodeInfo data out of the cache.
func (s *nodeInfoCache) Remove(nodeInfo models.NodeInfo) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	key := s.cacheKey(nodeInfo)
	if _, ok := s.nodeInfoKeyIdMap[key]; !ok {
		s.lc.Infof("nodeInfoIdCache does not contain %d. skip removing...", nodeInfo.Id)
	} else {
		s.lc.Debugf("removing nodeInfo id '%d' out of nodeInfo cache...", nodeInfo.Id)
		delete(s.nodeInfoKeyIdMap, key)
	}

	if _, ok := s.nodeInfoMap[nodeInfo.Id]; !ok {
		s.lc.Infof("nodeInfo with id '%d' not exists in nodeInfoMap cache, skip removing.", nodeInfo.Id)
		s.nodeInfoMap[nodeInfo.Id] = nodeInfo
	} else {
		s.lc.Debugf("Removing nodeInfo with id '%d' from nodeInfoMap cache ...", nodeInfo.Id)
		delete(s.nodeInfoMap, nodeInfo.Id)
	}
}
