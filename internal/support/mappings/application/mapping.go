package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/clint456/edgex-go/internal/support/mappings/container"
	"github.com/clint456/edgex-go/internal/support/mappings/model"
	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v4/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
	"github.com/edgexfoundry/go-mod-messaging/v4/pkg/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrNotFound = errors.New("mapping not found")
var ErrDuplicate = errors.New("mapping already exists")

const defaultChangeTopic = "edgex/support-mappings/configuration"

func Add(ctx context.Context, mapping model.Mapping, dic *di.Container) (model.Mapping, error) {
	mapping, err := mapping.PrepareForCreate()
	if err != nil {
		return model.Mapping{}, err
	}
	repo, err := container.RepositoryFrom(dic.Get)
	if err != nil {
		return model.Mapping{}, err
	}
	mapping, err = repo.Add(ctx, mapping)
	if err != nil {
		return model.Mapping{}, mapConflict(err)
	}
	return mapping, publishChange(ctx, "created", mapping, dic)
}

func Update(ctx context.Context, mapping model.Mapping, dic *di.Container) (model.Mapping, error) {
	if err := mapping.Validate(); err != nil || !validID(mapping.ID) {
		return model.Mapping{}, model.ErrInvalidMapping
	}
	repo, err := container.RepositoryFrom(dic.Get)
	if err != nil {
		return model.Mapping{}, err
	}
	previous, err := repo.ByID(ctx, mapping.ID)
	if err != nil {
		return model.Mapping{}, mapNotFound(err)
	}
	mapping.Created = previous.Created
	mapping.Modified = time.Now().UTC().UnixMilli()
	mapping, err = repo.Update(ctx, mapping)
	if err != nil {
		return model.Mapping{}, mapConflict(err)
	}
	return mapping, publishChange(ctx, "updated", mapping, dic)
}

func Delete(ctx context.Context, id string, dic *di.Container) error {
	if !validID(id) {
		return model.ErrInvalidMapping
	}
	repo, err := container.RepositoryFrom(dic.Get)
	if err != nil {
		return err
	}
	previous, err := repo.ByID(ctx, id)
	if err != nil {
		return mapNotFound(err)
	}
	if err = repo.Delete(ctx, id); err != nil {
		return mapNotFound(err)
	}
	return publishChange(ctx, "deleted", previous, dic)
}

func validID(id string) bool {
	if strings.TrimSpace(id) == "" {
		return false
	}
	_, err := uuid.Parse(id)
	return err == nil
}

func Forward(ctx context.Context, point model.NorthPoint, dic *di.Container) (model.Mapping, error) {
	if !point.Valid() {
		return model.Mapping{}, model.ErrInvalidMapping
	}
	repo, err := container.RepositoryFrom(dic.Get)
	if err != nil {
		return model.Mapping{}, err
	}
	m, err := repo.ByNorthPoint(ctx, point)
	return m, mapNotFound(err)
}

func Reverse(ctx context.Context, point model.SouthPoint, dic *di.Container) (model.Mapping, error) {
	if !point.Valid() {
		return model.Mapping{}, model.ErrInvalidMapping
	}
	repo, err := container.RepositoryFrom(dic.Get)
	if err != nil {
		return model.Mapping{}, err
	}
	m, err := repo.BySouthPoint(ctx, point)
	return m, mapNotFound(err)
}

func Batch(ctx context.Context, direction string, northPoints []model.NorthPoint, southPoints []model.SouthPoint, dic *di.Container) ([]model.Mapping, error) {
	if direction == "forward" {
		if len(northPoints) == 0 || len(northPoints) > 1000 || len(southPoints) != 0 {
			return nil, model.ErrInvalidMapping
		}
		for _, point := range northPoints {
			if !point.Valid() {
				return nil, model.ErrInvalidMapping
			}
		}
	} else if direction == "reverse" {
		if len(southPoints) == 0 || len(southPoints) > 1000 || len(northPoints) != 0 {
			return nil, model.ErrInvalidMapping
		}
		for _, point := range southPoints {
			if !point.Valid() {
				return nil, model.ErrInvalidMapping
			}
		}
	} else {
		return nil, model.ErrInvalidMapping
	}
	repo, err := container.RepositoryFrom(dic.Get)
	if err != nil {
		return nil, err
	}
	if direction == "forward" {
		return repo.ByNorthPoints(ctx, northPoints)
	}
	return repo.BySouthPoints(ctx, southPoints)
}

func ByNorthDeviceName(ctx context.Context, name string, offset, limit int, dic *di.Container) ([]model.Mapping, int64, error) {
	if strings.TrimSpace(name) == "" || offset < 0 || limit < 1 || limit > 1000 {
		return nil, 0, model.ErrInvalidMapping
	}
	repo, err := container.RepositoryFrom(dic.Get)
	if err != nil {
		return nil, 0, err
	}
	return repo.ByNorthDeviceName(ctx, name, offset, limit)
}

func ByNorthProfileName(ctx context.Context, name string, offset, limit int, dic *di.Container) ([]model.Mapping, int64, error) {
	if strings.TrimSpace(name) == "" || offset < 0 || limit < 1 || limit > 1000 {
		return nil, 0, model.ErrInvalidMapping
	}
	repo, err := container.RepositoryFrom(dic.Get)
	if err != nil {
		return nil, 0, err
	}
	return repo.ByNorthProfileName(ctx, name, offset, limit)
}

func BySouthDeviceName(ctx context.Context, name string, offset, limit int, dic *di.Container) ([]model.Mapping, int64, error) {
	if strings.TrimSpace(name) == "" || offset < 0 || limit < 1 || limit > 1000 {
		return nil, 0, model.ErrInvalidMapping
	}
	repo, err := container.RepositoryFrom(dic.Get)
	if err != nil {
		return nil, 0, err
	}
	return repo.BySouthDeviceName(ctx, name, offset, limit)
}

func BySouthProfileName(ctx context.Context, name string, offset, limit int, dic *di.Container) ([]model.Mapping, int64, error) {
	if strings.TrimSpace(name) == "" || offset < 0 || limit < 1 || limit > 1000 {
		return nil, 0, model.ErrInvalidMapping
	}
	repo, err := container.RepositoryFrom(dic.Get)
	if err != nil {
		return nil, 0, err
	}
	return repo.BySouthProfileName(ctx, name, offset, limit)
}

func DeleteByNorthDeviceName(ctx context.Context, name string, dic *di.Container) error {
	if strings.TrimSpace(name) == "" {
		return model.ErrInvalidMapping
	}
	repo, err := container.RepositoryFrom(dic.Get)
	if err != nil {
		return err
	}
	mappings, err := repo.DeleteByNorthDeviceName(ctx, name)
	if err != nil {
		return err
	}
	return publishDeleted(ctx, mappings, dic)
}

func DeleteByNorthProfileName(ctx context.Context, name string, dic *di.Container) error {
	if strings.TrimSpace(name) == "" {
		return model.ErrInvalidMapping
	}
	repo, err := container.RepositoryFrom(dic.Get)
	if err != nil {
		return err
	}
	mappings, err := repo.DeleteByNorthProfileName(ctx, name)
	if err != nil {
		return err
	}
	return publishDeleted(ctx, mappings, dic)
}

func DeleteBySouthDeviceName(ctx context.Context, name string, dic *di.Container) error {
	if strings.TrimSpace(name) == "" {
		return model.ErrInvalidMapping
	}
	repo, err := container.RepositoryFrom(dic.Get)
	if err != nil {
		return err
	}
	mappings, err := repo.DeleteBySouthDeviceName(ctx, name)
	if err != nil {
		return err
	}
	return publishDeleted(ctx, mappings, dic)
}

func DeleteBySouthProfileName(ctx context.Context, name string, dic *di.Container) error {
	if strings.TrimSpace(name) == "" {
		return model.ErrInvalidMapping
	}
	repo, err := container.RepositoryFrom(dic.Get)
	if err != nil {
		return err
	}
	mappings, err := repo.DeleteBySouthProfileName(ctx, name)
	if err != nil {
		return err
	}
	return publishDeleted(ctx, mappings, dic)
}

func publishDeleted(ctx context.Context, mappings []model.Mapping, dic *di.Container) error {
	for _, mapping := range mappings {
		if err := publishChange(ctx, "deleted", mapping, dic); err != nil {
			return err
		}
	}
	return nil
}

func All(ctx context.Context, offset, limit int, dic *di.Container) ([]model.Mapping, int64, error) {
	if offset < 0 || limit < 1 || limit > 1000 {
		return nil, 0, model.ErrInvalidMapping
	}
	repo, err := container.RepositoryFrom(dic.Get)
	if err != nil {
		return nil, 0, err
	}
	return repo.All(ctx, offset, limit)
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func mapConflict(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicate
	}
	return err
}

func publishChange(ctx context.Context, action string, mapping model.Mapping, dic *di.Container) error {
	config := container.ConfigurationFrom(dic.Get)
	topic := strings.TrimSpace(config.Writable.ChangeTopic)
	if topic == "" {
		topic = defaultChangeTopic
	}
	messageClient := bootstrapContainer.MessagingClientFrom(dic.Get)
	envelope := types.NewMessageEnvelope(map[string]any{
		"action": action, "mapping": mapping, "changedAt": time.Now().UTC().UnixMilli(),
	}, ctx)
	return messageClient.Publish(envelope, topic)
}
