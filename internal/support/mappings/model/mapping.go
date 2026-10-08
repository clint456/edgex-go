package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type NorthPoint struct {
	NorthDeviceName   string `json:"northDeviceName"`
	NorthProfileName  string `json:"northProfileName"`
	NorthResourceName string `json:"northResourceName"`
}

type SouthPoint struct {
	SouthDeviceName   string `json:"southDeviceName"`
	SouthProfileName  string `json:"southProfileName"`
	SouthResourceName string `json:"southResourceName"`
}

// 南北向映射模型
type Mapping struct {
	ID         string         `json:"id"`
	NorthPoint NorthPoint     `json:"northPoint"`
	SouthPoint SouthPoint     `json:"southPoint"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Created    int64          `json:"created"`
	Modified   int64          `json:"modified"`
}

var ErrInvalidMapping = errors.New("northPoint and southPoint fields are required; metadata must be JSON-serializable")

func (m Mapping) Validate() error {
	if !m.NorthPoint.Valid() || !m.SouthPoint.Valid() {
		return ErrInvalidMapping
	}
	if _, err := json.Marshal(m.Metadata); err != nil {
		return ErrInvalidMapping
	}
	return nil
}

func (p NorthPoint) Valid() bool {
	return strings.TrimSpace(p.NorthDeviceName) != "" && strings.TrimSpace(p.NorthProfileName) != "" && strings.TrimSpace(p.NorthResourceName) != ""
}

func (p SouthPoint) Valid() bool {
	return strings.TrimSpace(p.SouthDeviceName) != "" && strings.TrimSpace(p.SouthProfileName) != "" && strings.TrimSpace(p.SouthResourceName) != ""
}

func (m Mapping) PrepareForCreate() (Mapping, error) {
	if err := m.Validate(); err != nil {
		return Mapping{}, err
	}
	if m.ID == "" {
		m.ID = uuid.NewString()
	} else if _, err := uuid.Parse(m.ID); err != nil {
		return Mapping{}, ErrInvalidMapping
	}
	now := time.Now().UTC().UnixMilli()
	m.Created, m.Modified = now, now
	if m.Metadata == nil {
		m.Metadata = map[string]any{}
	}
	return m, nil
}
