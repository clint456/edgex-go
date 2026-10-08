package interfaces

import (
	"context"

	"github.com/clint456/edgex-go/internal/support/mappings/model"
)

type Repository interface {
	Add(context.Context, model.Mapping) (model.Mapping, error)
	Update(context.Context, model.Mapping) (model.Mapping, error)
	Delete(context.Context, string) error
	ByID(context.Context, string) (model.Mapping, error)
	ByNorthPoint(context.Context, model.NorthPoint) (model.Mapping, error)
	BySouthPoint(context.Context, model.SouthPoint) (model.Mapping, error)
	ByNorthPoints(context.Context, []model.NorthPoint) ([]model.Mapping, error)
	BySouthPoints(context.Context, []model.SouthPoint) ([]model.Mapping, error)
	ByNorthDeviceName(context.Context, string, int, int) ([]model.Mapping, int64, error)
	ByNorthProfileName(context.Context, string, int, int) ([]model.Mapping, int64, error)
	BySouthDeviceName(context.Context, string, int, int) ([]model.Mapping, int64, error)
	BySouthProfileName(context.Context, string, int, int) ([]model.Mapping, int64, error)
	DeleteByNorthDeviceName(context.Context, string) ([]model.Mapping, error)
	DeleteByNorthProfileName(context.Context, string) ([]model.Mapping, error)
	DeleteBySouthDeviceName(context.Context, string) ([]model.Mapping, error)
	DeleteBySouthProfileName(context.Context, string) ([]model.Mapping, error)
	All(context.Context, int, int) ([]model.Mapping, int64, error)
}
