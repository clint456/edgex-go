package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrepareForCreate(t *testing.T) {
	mapping, err := (Mapping{
		NorthPoint: NorthPoint{NorthNodeName: "north-node", NorthProfileName: "north-profile", NorthResourceName: "north-resource"},
		SouthPoint: SouthPoint{SouthNodeName: "south-node", SouthProfileName: "south-profile", SouthResourceName: "south-resource"},
	}).PrepareForCreate()
	require.NoError(t, err)
	require.NotEmpty(t, mapping.ID)
	require.NotZero(t, mapping.Created)
	require.Equal(t, mapping.Created, mapping.Modified)
	require.NotNil(t, mapping.Metadata)
}

func TestMappingUsesNorthAndSouthJSONFields(t *testing.T) {
	mapping := Mapping{
		NorthPoint: NorthPoint{NorthNodeName: "north-node", NorthProfileName: "north-profile", NorthResourceName: "north-resource"},
		SouthPoint: SouthPoint{SouthNodeName: "south-node", SouthProfileName: "south-profile", SouthResourceName: "south-resource"},
	}
	payload, err := json.Marshal(mapping)
	require.NoError(t, err)
	require.JSONEq(t, `{"id":"","northPoint":{"northNodeName":"north-node","northProfileName":"north-profile","northResourceName":"north-resource"},"southPoint":{"southNodeName":"south-node","southProfileName":"south-profile","southResourceName":"south-resource"},"created":0,"modified":0}`, string(payload))
}

func TestValidateRejectsIncompleteMapping(t *testing.T) {
	err := (Mapping{NorthPoint: NorthPoint{NorthNodeName: "north-node"}}).Validate()
	require.ErrorIs(t, err, ErrInvalidMapping)
}
