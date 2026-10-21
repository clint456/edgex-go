//
// Copyright (C) 2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"github.com/clint456/edgex-go/internal/pkg/cache"

	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
)

// NodeStoreInterfaceName contains the name of the cache.ActiveNodeStore implementation in the DIC.
var NodeStoreInterfaceName = di.TypeInstanceToName((*cache.ActiveNodeStore)(nil))

// NodeStoreFrom helper function queries the DIC and returns the cache.ActiveNodeStore implementation.
func NodeStoreFrom(get di.Get) cache.ActiveNodeStore {
	return get(NodeStoreInterfaceName).(cache.ActiveNodeStore)
}
