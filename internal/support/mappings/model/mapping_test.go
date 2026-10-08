package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrepareForCreate(t *testing.T) {
	mapping, err := (Mapping{
		NorthPoint: NorthPoint{NorthDeviceName: "north-device", NorthProfileName: "north-profile", NorthResourceName: "north-resource"},
		SouthPoint: SouthPoint{SouthDeviceName: "south-device", SouthProfileName: "south-profile", SouthResourceName: "south-resource"},
	}).PrepareForCreate()
	require.NoError(t, err)
	require.NotEmpty(t, mapping.ID)
	require.NotZero(t, mapping.Created)
	require.Equal(t, mapping.Created, mapping.Modified)
	require.NotNil(t, mapping.Metadata)
}

func TestMappingUsesNorthAndSouthJSONFields(t *testing.T) {
	mapping := Mapping{
		NorthPoint: NorthPoint{NorthDeviceName: "north-device", NorthProfileName: "north-profile", NorthResourceName: "north-resource"},
		SouthPoint: SouthPoint{SouthDeviceName: "south-device", SouthProfileName: "south-profile", SouthResourceName: "south-resource"},
	}
	payload, err := json.Marshal(mapping)
	require.NoError(t, err)
	require.JSONEq(t, `{"id":"","northPoint":{"northDeviceName":"north-device","northProfileName":"north-profile","northResourceName":"north-resource"},"southPoint":{"southDeviceName":"south-device","southProfileName":"south-profile","southResourceName":"south-resource"},"created":0,"modified":0}`, string(payload))
}

func TestValidateRejectsIncompleteMapping(t *testing.T) {
	err := (Mapping{NorthPoint: NorthPoint{NorthDeviceName: "north-device"}}).Validate()
	require.ErrorIs(t, err, ErrInvalidMapping)
}
