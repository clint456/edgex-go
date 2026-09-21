//
// Copyright (C) 2024-2025 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	data "github.com/edgexfoundry/edgex-go/internal/core/data/embed"
	keeper "github.com/edgexfoundry/edgex-go/internal/core/keeper/embed"
	metadata "github.com/edgexfoundry/edgex-go/internal/core/metadata/embed"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
)

// constants relate to the postgres db table names
const (
	configTableName           = keeper.SchemaName + ".config"
	eventTableName            = data.SchemaName + ".event"
	deviceInfoTableName       = data.SchemaName + ".device_info"
	deviceServiceTableName    = metadata.SchemaName + ".device_service"
	deviceProfileTableName    = metadata.SchemaName + ".device_profile"
	deviceTableName           = metadata.SchemaName + ".device"
	provisionWatcherTableName = metadata.SchemaName + ".provision_watcher"
	readingTableName          = data.SchemaName + ".reading"
	registryTableName         = keeper.SchemaName + ".registry"
)

// constants relate to the common db table column names
const (
	contentCol  = "content"
	createdCol  = "created"
	idCol       = "id"
	modifiedCol = "modified"
	statusCol   = "status"
	nameCol     = "name"
)

// constants relate to the named arguments as specified for SQL conditions
const (
	offsetCondition      = common.Offset
	limitCondition       = common.Limit
	startTimeCondition   = common.Start
	endTimeCondition     = common.End
	jsonContentCondition = "jsonContent"
	categoryCondition    = "category"
	labelsCondition      = "labels"
)

// constants relate to the event/reading postgres db table column names
const (
	deviceNameCol     = "devicename"
	resourceNameCol   = "resourcename"
	profileNameCol    = "profilename"
	sourceNameCol     = "sourcename"
	originCol         = "origin"
	valueTypeCol      = "valuetype"
	unitsCol          = "units"
	tagsCol           = "tags"
	eventIdFKCol      = "event_id"
	deviceInfoIdFKCol = "device_info_id"
	valueCol          = "value"
	numericValueCol   = "numeric_value"
	binaryValueCol    = "binaryvalue"
	mediaTypeCol      = "mediatype"
	objectValueCol    = "objectvalue"
	markDeletedCol    = "mark_deleted"
)

// constants relate to the keeper postgres db table column names
const (
	keyCol = "key"
)

// constants relate to the field names in the content column
const (
	createdField      = "Created"
	labelsField       = "Labels"
	parentField       = "Parent"
	manufacturerField = "Manufacturer"
	modelField        = "Model"
	nameField         = "Name"
	profileNameField  = "ProfileName"
	serviceIdField    = "ServiceId"
	serviceNameField  = "ServiceName"
)
