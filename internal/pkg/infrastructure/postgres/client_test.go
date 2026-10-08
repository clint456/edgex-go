//
// Copyright (C) 2024-2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	dataInterfaces "github.com/clint456/edgex-go/internal/core/data/infrastructure/interfaces"
	keeperInterfaces "github.com/clint456/edgex-go/internal/core/keeper/infrastructure/interfaces"
	metadataInterfaces "github.com/clint456/edgex-go/internal/core/metadata/infrastructure/interfaces"
)

// Check the implementation of Postgres satisfies the DB client
var _ dataInterfaces.DBClient = &Client{}
var _ metadataInterfaces.DBClient = &Client{}
var _ keeperInterfaces.DBClient = &Client{}
