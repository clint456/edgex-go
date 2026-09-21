//
// Copyright (C) 2021 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package redis

import dataInterfaces "github.com/edgexfoundry/edgex-go/internal/core/data/infrastructure/interfaces"
import keeperInterfaces "github.com/edgexfoundry/edgex-go/internal/core/keeper/infrastructure/interfaces"
import metadataInterfaces "github.com/edgexfoundry/edgex-go/internal/core/metadata/infrastructure/interfaces"

// Check the implementation of Redis satisfies the DB client
var _ dataInterfaces.DBClient = &Client{}
var _ metadataInterfaces.DBClient = &Client{}
var _ keeperInterfaces.DBClient = &Client{}
