package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/clint456/edgex-go/internal/support/mappings/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	table   = "support_mappings.mapping"
	columns = `id, north_device_name, north_profile_name, north_resource_name,
		south_device_name, south_profile_name, south_resource_name, metadata, created, modified`
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) Add(ctx context.Context, m model.Mapping) (model.Mapping, error) {
	metadata, err := json.Marshal(m.Metadata)
	if err != nil {
		return model.Mapping{}, err
	}
	query := `INSERT INTO ` + table + ` ( ` + columns + ` )
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING ` + columns
	return scan(r.pool.QueryRow(ctx, query, values(m, metadata)...))
}

func (r *Repository) Update(ctx context.Context, m model.Mapping) (model.Mapping, error) {
	metadata, err := json.Marshal(m.Metadata)
	if err != nil {
		return model.Mapping{}, err
	}
	query := `UPDATE ` + table + ` SET north_device_name=$2, north_profile_name=$3, north_resource_name=$4,
	south_device_name=$5, south_profile_name=$6, south_resource_name=$7, metadata=$8, modified=$9
WHERE id=$1 RETURNING ` + columns
	args := []any{m.ID, m.NorthPoint.NorthDeviceName, m.NorthPoint.NorthProfileName, m.NorthPoint.NorthResourceName,
		m.SouthPoint.SouthDeviceName, m.SouthPoint.SouthProfileName, m.SouthPoint.SouthResourceName, metadata, m.Modified}
	return scan(r.pool.QueryRow(ctx, query, args...))
}

func values(m model.Mapping, metadata []byte) []any {
	return []any{m.ID, m.NorthPoint.NorthDeviceName, m.NorthPoint.NorthProfileName, m.NorthPoint.NorthResourceName,
		m.SouthPoint.SouthDeviceName, m.SouthPoint.SouthProfileName, m.SouthPoint.SouthResourceName, metadata, m.Created, m.Modified}
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM `+table+` WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ByID(ctx context.Context, id string) (model.Mapping, error) {
	return scan(r.pool.QueryRow(ctx, `SELECT `+columns+` FROM `+table+` WHERE id=$1`, id))
}

func (r *Repository) ByNorthPoint(ctx context.Context, point model.NorthPoint) (model.Mapping, error) {
	query := `SELECT ` + columns + ` FROM ` + table + ` WHERE north_device_name=$1 AND north_profile_name=$2 AND north_resource_name=$3`
	return scan(r.pool.QueryRow(ctx, query, point.NorthDeviceName, point.NorthProfileName, point.NorthResourceName))
}

func (r *Repository) BySouthPoint(ctx context.Context, point model.SouthPoint) (model.Mapping, error) {
	query := `SELECT ` + columns + ` FROM ` + table + ` WHERE south_device_name=$1 AND south_profile_name=$2 AND south_resource_name=$3`
	return scan(r.pool.QueryRow(ctx, query, point.SouthDeviceName, point.SouthProfileName, point.SouthResourceName))
}

func (r *Repository) ByNorthPoints(ctx context.Context, points []model.NorthPoint) ([]model.Mapping, error) {
	values := make([][3]string, 0, len(points))
	for _, point := range points {
		values = append(values, [3]string{point.NorthDeviceName, point.NorthProfileName, point.NorthResourceName})
	}
	return r.byPoints(ctx, "north_device_name", "north_profile_name", "north_resource_name", values)
}

func (r *Repository) BySouthPoints(ctx context.Context, points []model.SouthPoint) ([]model.Mapping, error) {
	values := make([][3]string, 0, len(points))
	for _, point := range points {
		values = append(values, [3]string{point.SouthDeviceName, point.SouthProfileName, point.SouthResourceName})
	}
	return r.byPoints(ctx, "south_device_name", "south_profile_name", "south_resource_name", values)
}

func (r *Repository) byPoints(ctx context.Context, deviceColumn, profileColumn, resourceColumn string, points [][3]string) ([]model.Mapping, error) {
	if len(points) == 0 {
		return []model.Mapping{}, nil
	}
	args := make([]any, 0, len(points)*3)
	predicates := make([]string, 0, len(points))
	for i, point := range points {
		base := i*3 + 1
		predicates = append(predicates, fmt.Sprintf("(%s=$%d AND %s=$%d AND %s=$%d)", deviceColumn, base, profileColumn, base+1, resourceColumn, base+2))
		args = append(args, point[0], point[1], point[2])
	}
	query := `SELECT ` + columns + ` FROM ` + table + ` WHERE ` + strings.Join(predicates, " OR ") + ` ORDER BY north_device_name, north_profile_name, north_resource_name`
	return queryMappings(ctx, r.pool, query, args...)
}

func (r *Repository) ByNorthDeviceName(ctx context.Context, name string, offset, limit int) ([]model.Mapping, int64, error) {
	return r.byName(ctx, "north_device_name", name, offset, limit)
}

func (r *Repository) ByNorthProfileName(ctx context.Context, name string, offset, limit int) ([]model.Mapping, int64, error) {
	return r.byName(ctx, "north_profile_name", name, offset, limit)
}

func (r *Repository) BySouthDeviceName(ctx context.Context, name string, offset, limit int) ([]model.Mapping, int64, error) {
	return r.byName(ctx, "south_device_name", name, offset, limit)
}

func (r *Repository) BySouthProfileName(ctx context.Context, name string, offset, limit int) ([]model.Mapping, int64, error) {
	return r.byName(ctx, "south_profile_name", name, offset, limit)
}

func (r *Repository) byName(ctx context.Context, column, name string, offset, limit int) ([]model.Mapping, int64, error) {
	var count int64
	countQuery := fmt.Sprintf(`SELECT count(*) FROM %s WHERE %s=$1`, table, column)
	if err := r.pool.QueryRow(ctx, countQuery, name).Scan(&count); err != nil {
		return nil, 0, err
	}
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s=$1 ORDER BY north_device_name, north_profile_name, north_resource_name OFFSET $2 LIMIT $3`, columns, table, column)
	result, err := queryMappings(ctx, r.pool, query, name, offset, limit)
	return result, count, err
}

func (r *Repository) DeleteByNorthDeviceName(ctx context.Context, name string) ([]model.Mapping, error) {
	return r.deleteByName(ctx, "north_device_name", name)
}

func (r *Repository) DeleteByNorthProfileName(ctx context.Context, name string) ([]model.Mapping, error) {
	return r.deleteByName(ctx, "north_profile_name", name)
}

func (r *Repository) DeleteBySouthDeviceName(ctx context.Context, name string) ([]model.Mapping, error) {
	return r.deleteByName(ctx, "south_device_name", name)
}

func (r *Repository) DeleteBySouthProfileName(ctx context.Context, name string) ([]model.Mapping, error) {
	return r.deleteByName(ctx, "south_profile_name", name)
}

func (r *Repository) deleteByName(ctx context.Context, column, name string) ([]model.Mapping, error) {
	query := fmt.Sprintf(`DELETE FROM %s WHERE %s=$1 RETURNING %s`, table, column, columns)
	return queryMappings(ctx, r.pool, query, name)
}

func (r *Repository) All(ctx context.Context, offset, limit int) ([]model.Mapping, int64, error) {
	var count int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM `+table).Scan(&count); err != nil {
		return nil, 0, err
	}
	query := `SELECT ` + columns + ` FROM ` + table + ` ORDER BY north_device_name, north_profile_name, north_resource_name OFFSET $1 LIMIT $2`
	result, err := queryMappings(ctx, r.pool, query, offset, limit)
	return result, count, err
}

type rowScanner interface{ Scan(...any) error }

func scan(row rowScanner) (model.Mapping, error) {
	var m model.Mapping
	var metadata []byte
	err := row.Scan(&m.ID, &m.NorthPoint.NorthDeviceName, &m.NorthPoint.NorthProfileName, &m.NorthPoint.NorthResourceName,
		&m.SouthPoint.SouthDeviceName, &m.SouthPoint.SouthProfileName, &m.SouthPoint.SouthResourceName,
		&metadata, &m.Created, &m.Modified)
	if err != nil {
		return m, err
	}
	if err = json.Unmarshal(metadata, &m.Metadata); err != nil {
		return m, errors.New("invalid mapping metadata stored in database")
	}
	return m, nil
}

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func queryMappings(ctx context.Context, q queryer, query string, args ...any) ([]model.Mapping, error) {
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]model.Mapping, 0)
	for rows.Next() {
		mapping, scanErr := scan(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, mapping)
	}
	return result, rows.Err()
}
