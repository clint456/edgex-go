// Copyright (C) 2025 IOTech Ltd

package cache

import (
	"sync"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
)

// ActiveNodeStore used to store active nodes
type ActiveNodeStore interface {
	Contains(nodeName string) bool
	Add(node models.Node)
	Remove(nodeName string)
	RemoveAll()
	Nodes() map[string]models.Node
}

type activeNodeStore struct {
	lc            logger.LoggingClient
	activeNodeMap map[string]models.Node
	mutex         sync.RWMutex
}

// Contains check if node is in the active node list
func (s *activeNodeStore) Contains(nodeName string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if _, ok := s.activeNodeMap[nodeName]; ok {
		return true
	}
	return false
}

// Add adds node into the active node list
func (s *activeNodeStore) Add(node models.Node) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, ok := s.activeNodeMap[node.Name]; !ok {
		s.lc.Debugf("adding %s into activeNodeStore...", node.Name)
		s.activeNodeMap[node.Name] = node
		return
	}
	s.lc.Infof("activeNodeStore already contains %s. skip adding...", node.Name)
}

// Remove removes node out of the active node list.
func (s *activeNodeStore) Remove(nodeName string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, ok := s.activeNodeMap[nodeName]; !ok {
		s.lc.Infof("activeNodeStore does not contain %s. skip removing...", nodeName)
	} else {
		s.lc.Debugf("removing %s out of activeNodeStore...", nodeName)
		delete(s.activeNodeMap, nodeName)
	}
}

// RemoveAll removes all node out of the active node list.
func (s *activeNodeStore) RemoveAll() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if len(s.activeNodeMap) == 0 {
		return
	}
	clear(s.activeNodeMap)
}

func (s *activeNodeStore) Nodes() map[string]models.Node {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	// since the node map might change, return the copy of nodes to prevent unexpected result
	clonedNodes := make(map[string]models.Node, len(s.activeNodeMap))
	for nodeName, node := range s.activeNodeMap {
		clonedNodes[nodeName] = node.Clone()
	}
	return clonedNodes
}

func NodeStore(dic *di.Container) ActiveNodeStore {
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	return &activeNodeStore{activeNodeMap: make(map[string]models.Node), lc: lc}
}
