//
// Copyright (C) 2021 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package redis

import dataInterfaces "github.com/clint456/edgex-go/internal/core/data/infrastructure/interfaces"
import keeperInterfaces "github.com/clint456/edgex-go/internal/core/keeper/infrastructure/interfaces"
import metadataInterfaces "github.com/clint456/edgex-go/internal/core/metadata/infrastructure/interfaces"

// Check the implementation of Redis satisfies the DB client
var _ dataInterfaces.DBClient = &Client{}
var _ metadataInterfaces.DBClient = &Client{}
var _ keeperInterfaces.DBClient = &Client{}
