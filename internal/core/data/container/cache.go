//
// Copyright (C) 2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"github.com/clint456/edgex-go/internal/core/data/cache"

	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
)

var NodeInfoCacheInterfaceName = di.TypeInstanceToName((*cache.NodeInfoCache)(nil))

// NodeInfoCacheFrom helper function queries the DIC and returns the cache.NodeInfoCache implementation.
func NodeInfoCacheFrom(get di.Get) cache.NodeInfoCache {
	return get(NodeInfoCacheInterfaceName).(cache.NodeInfoCache)
}
