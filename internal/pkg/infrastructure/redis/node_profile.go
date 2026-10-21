//
// Copyright (C) 2020-2021 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"encoding/json"
	"fmt"

	pkgCommon "github.com/clint456/edgex-go/internal/pkg/common"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"

	"github.com/gomodule/redigo/redis"
)

const (
	ProfileCollection             = "md|dp"
	ProfileCollectionName         = ProfileCollection + DBKeySeparator + common.Name
	ProfileCollectionLabel        = ProfileCollection + DBKeySeparator + common.Label
	ProfileCollectionModel        = ProfileCollection + DBKeySeparator + common.Model
	ProfileCollectionManufacturer = ProfileCollection + DBKeySeparator + common.Manufacturer
)

// profileStoredKey return the node profile's stored key which combines the collection name and object id
func profileStoredKey(id string) string {
	return CreateKey(ProfileCollection, id)
}

// profileNameExists whether the node profile exists by name
func profileNameExists(conn redis.Conn, name string) (bool, errors.EdgeX) {
	exists, err := objectNameExists(conn, ProfileCollectionName, name)
	if err != nil {
		return false, errors.NewCommonEdgeXWrapper(err)
	}
	return exists, nil
}

// profileIdExists checks whether the node profile exists by id
func profileIdExists(conn redis.Conn, id string) (bool, errors.EdgeX) {
	exists, err := objectIdExists(conn, profileStoredKey(id))
	if err != nil {
		return false, errors.NewCommonEdgeXWrapper(err)
	}
	return exists, nil
}

// sendAddProfileCmd send redis command for adding node profile
func sendAddProfileCmd(conn redis.Conn, storedKey string, dp models.Profile) errors.EdgeX {
	m, err := json.Marshal(dp)
	if err != nil {
		return errors.NewCommonEdgeX(errors.KindContractInvalid, "unable to JSON marshal node profile for Redis persistence", err)
	}
	_ = conn.Send(SET, storedKey, m)
	_ = conn.Send(ZADD, ProfileCollection, 0, storedKey)
	_ = conn.Send(HSET, ProfileCollectionName, dp.Name, storedKey)
	_ = conn.Send(ZADD, CreateKey(ProfileCollectionManufacturer, dp.Manufacturer), dp.Modified, storedKey)
	_ = conn.Send(ZADD, CreateKey(ProfileCollectionModel, dp.Model), dp.Modified, storedKey)
	for _, label := range dp.Labels {
		_ = conn.Send(ZADD, CreateKey(ProfileCollectionLabel, label), dp.Modified, storedKey)
	}
	return nil
}

// addProfile adds a node profile to DB
func addProfile(conn redis.Conn, dp models.Profile) (models.Profile, errors.EdgeX) {
	// query node profile name and id to avoid the conflict
	exists, edgeXerr := profileIdExists(conn, dp.Id)
	if edgeXerr != nil {
		return dp, errors.NewCommonEdgeXWrapper(edgeXerr)
	} else if exists {
		return dp, errors.NewCommonEdgeX(errors.KindDuplicateName, fmt.Sprintf("node profile id %s exists", dp.Id), edgeXerr)
	}

	exists, edgeXerr = profileNameExists(conn, dp.Name)
	if edgeXerr != nil {
		return dp, errors.NewCommonEdgeXWrapper(edgeXerr)
	} else if exists {
		return dp, errors.NewCommonEdgeX(errors.KindDuplicateName, fmt.Sprintf("node profile name %s exists", dp.Name), edgeXerr)
	}

	ts := pkgCommon.MakeTimestamp()
	// For Redis DB, the PUT or PATCH operation will removes the old object and add the modified one,
	// so the Created is not zero value and we shouldn't set the timestamp again.
	if dp.Created == 0 {
		dp.Created = ts
	}
	dp.Modified = ts

	storedKey := profileStoredKey(dp.Id)
	_ = conn.Send(MULTI)
	edgeXerr = sendAddProfileCmd(conn, storedKey, dp)
	if edgeXerr != nil {
		return dp, errors.NewCommonEdgeXWrapper(edgeXerr)
	}
	_, err := conn.Do(EXEC)
	if err != nil {
		edgeXerr = errors.NewCommonEdgeX(errors.KindDatabaseError, "node profile creation failed", err)
	}

	return dp, edgeXerr
}

// profileById query node profile by id from DB
func profileById(conn redis.Conn, id string) (profile models.Profile, edgeXerr errors.EdgeX) {
	edgeXerr = getObjectById(conn, profileStoredKey(id), &profile)
	if edgeXerr != nil {
		return profile, errors.NewCommonEdgeX(errors.Kind(edgeXerr), fmt.Sprintf("fail to query node profile by id %s", id), edgeXerr)
	}
	return
}

// profileByName query node profile by name from DB
func profileByName(conn redis.Conn, name string) (profile models.Profile, edgeXerr errors.EdgeX) {
	edgeXerr = getObjectByHash(conn, ProfileCollectionName, name, &profile)
	if edgeXerr != nil {
		return profile, errors.NewCommonEdgeX(errors.Kind(edgeXerr), fmt.Sprintf("fail to query node profile by name %s", name), edgeXerr)
	}
	return
}

// sendDeleteProfileCmd send redis command for deleting node profile
func sendDeleteProfileCmd(conn redis.Conn, storedKey string, dp models.Profile) {
	_ = conn.Send(DEL, storedKey)
	_ = conn.Send(ZREM, ProfileCollection, storedKey)
	_ = conn.Send(HDEL, ProfileCollectionName, dp.Name)
	_ = conn.Send(ZREM, CreateKey(ProfileCollectionManufacturer, dp.Manufacturer), storedKey)
	_ = conn.Send(ZREM, CreateKey(ProfileCollectionModel, dp.Model), storedKey)
	for _, label := range dp.Labels {
		_ = conn.Send(ZREM, CreateKey(ProfileCollectionLabel, label), storedKey)
	}
}

func deleteProfile(conn redis.Conn, dp models.Profile) errors.EdgeX {
	storedKey := profileStoredKey(dp.Id)
	_ = conn.Send(MULTI)
	sendDeleteProfileCmd(conn, storedKey, dp)
	_, err := conn.Do(EXEC)
	if err != nil {
		return errors.NewCommonEdgeX(errors.KindDatabaseError, "node profile deletion failed", err)
	}
	return nil
}

// updateProfile updates a node profile to DB
func updateProfile(conn redis.Conn, dp models.Profile) (edgeXerr errors.EdgeX) {
	var oldProfile models.Profile
	oldProfile, edgeXerr = profileById(conn, dp.Id)
	if edgeXerr == nil {
		if dp.Name != oldProfile.Name {
			return errors.NewCommonEdgeX(errors.KindContractInvalid, fmt.Sprintf("node profile name '%s' not match the exsting '%s' ", dp.Name, oldProfile.Name), nil)
		}
	} else {
		oldProfile, edgeXerr = profileByName(conn, dp.Name)
		if edgeXerr != nil {
			return errors.NewCommonEdgeXWrapper(edgeXerr)
		}
	}

	dp.Id = oldProfile.Id
	dp.Created = oldProfile.Created
	dp.Modified = pkgCommon.MakeTimestamp()

	storedKey := profileStoredKey(dp.Id)
	_ = conn.Send(MULTI)
	sendDeleteProfileCmd(conn, storedKey, oldProfile)
	edgeXerr = sendAddProfileCmd(conn, storedKey, dp)
	if edgeXerr != nil {
		return errors.NewCommonEdgeXWrapper(edgeXerr)
	}
	_, err := conn.Do(EXEC)
	if err != nil {
		return errors.NewCommonEdgeX(errors.KindDatabaseError, "node profile update failed", err)
	}

	return nil
}

// deleteProfileById deletes the node profile by id
func deleteProfileById(conn redis.Conn, id string) errors.EdgeX {
	profile, err := profileById(conn, id)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	err = deleteProfile(conn, profile)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	return nil
}

// deleteProfileByName deletes the node profile by name
func deleteProfileByName(conn redis.Conn, name string) errors.EdgeX {
	profile, err := profileByName(conn, name)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}

	// Check the associated Node and ProvisionWatcher existence
	nodes, err := nodesByProfileName(conn, 0, 1, name)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	if len(nodes) > 0 {
		return errors.NewCommonEdgeX(errors.KindStatusConflict, "fail to delete the node profile when associated node exists", nil)
	}
	provisionWatchers, err := provisionWatchersByProfileName(conn, 0, 1, name)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	if len(provisionWatchers) > 0 {
		return errors.NewCommonEdgeX(errors.KindStatusConflict, "fail to delete the node profile when associated provisionWatcher exists", nil)
	}

	err = deleteProfile(conn, profile)
	if err != nil {
		return errors.NewCommonEdgeXWrapper(err)
	}
	return nil
}

// profilesByLabels query node profile with offset and limit
func profilesByLabels(conn redis.Conn, offset int, limit int, labels []string) (profiles []models.Profile, edgeXerr errors.EdgeX) {
	objects, edgeXerr := getObjectsByLabelsAndSomeRange(conn, ZREVRANGE, ProfileCollection, labels, offset, limit)
	if edgeXerr != nil {
		return profiles, errors.NewCommonEdgeXWrapper(edgeXerr)
	}

	profiles = make([]models.Profile, len(objects))
	for i, in := range objects {
		dp := models.Profile{}
		err := json.Unmarshal(in, &dp)
		if err != nil {
			return []models.Profile{}, errors.NewCommonEdgeX(errors.KindDatabaseError, "node profile format parsing failed from the database", err)
		}
		profiles[i] = dp
	}
	return profiles, nil
}

// profilesByModel query node profiles by offset, limit and model
func profilesByModel(conn redis.Conn, offset int, limit int, model string) (profiles []models.Profile, edgeXerr errors.EdgeX) {
	objects, err := getObjectsByRevRange(conn, CreateKey(ProfileCollectionModel, model), offset, limit)
	if err != nil {
		return profiles, errors.NewCommonEdgeXWrapper(err)
	}

	profiles = make([]models.Profile, len(objects))
	for i, in := range objects {
		dp := models.Profile{}
		err := json.Unmarshal(in, &dp)
		if err != nil {
			return profiles, errors.NewCommonEdgeX(errors.KindContractInvalid, "node profile parsing failed", err)
		}
		profiles[i] = dp
	}
	return profiles, nil
}

// profilesByManufacturer query node profiles by offset, limit and manufacturer
func profilesByManufacturer(conn redis.Conn, offset int, limit int, manufacturer string) (profiles []models.Profile, edgeXerr errors.EdgeX) {
	objects, err := getObjectsByRevRange(conn, CreateKey(ProfileCollectionManufacturer, manufacturer), offset, limit)
	if err != nil {
		return profiles, errors.NewCommonEdgeXWrapper(err)
	}

	profiles = make([]models.Profile, len(objects))
	for i, in := range objects {
		dp := models.Profile{}
		err := json.Unmarshal(in, &dp)
		if err != nil {
			return profiles, errors.NewCommonEdgeX(errors.KindContractInvalid, "node profile parsing failed", err)
		}
		profiles[i] = dp
	}
	return profiles, nil
}

// profilesByManufacturerAndModel query node profiles by offset, limit, manufacturer and model
func profilesByManufacturerAndModel(conn redis.Conn, offset int, limit int, manufacturer string, model string) (profiles []models.Profile, edgeXerr errors.EdgeX) {
	var redisKeys []string
	redisKeys = append(redisKeys, CreateKey(ProfileCollectionManufacturer, manufacturer))
	redisKeys = append(redisKeys, CreateKey(ProfileCollectionModel, model))

	objects, err := intersectionObjectsByKeys(conn, offset, limit, redisKeys...)
	if err != nil {
		return profiles, errors.NewCommonEdgeXWrapper(err)
	}

	profiles = make([]models.Profile, len(objects))
	for i, in := range objects {
		dp := models.Profile{}
		err := json.Unmarshal(in, &dp)
		if err != nil {
			return profiles, errors.NewCommonEdgeX(errors.KindContractInvalid, "node profile parsing failed", err)
		}
		profiles[i] = dp
	}
	return profiles, nil
}
