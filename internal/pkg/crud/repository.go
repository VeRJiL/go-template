package crud

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/VeRJiL/go-template/internal/pkg/modules"
)

// GenericRepository implements the Repository interface for any entity
type GenericRepository[T modules.Entity] struct {
	db        *sql.DB
	tableName string
	entityType reflect.Type
}

// NewGenericRepository creates a new generic repository
func NewGenericRepository[T modules.Entity](db *sql.DB, entity T) *GenericRepository[T] {
	return &GenericRepository[T]{
		db:         db,
		tableName:  entity.GetTableName(),
		entityType: reflect.TypeOf(entity),
	}
}

// Create inserts a new entity into the database
func (r *GenericRepository[T]) Create(ctx context.Context, entity *T) error {
	// Set timestamps if entity supports them
	if timestampable, ok := any(entity).(modules.Timestampable); ok {
		now := time.Now().Unix()
		timestampable.SetCreatedAt(now)
		timestampable.SetUpdatedAt(now)
	}

	// Validate entity
	if err := (*entity).Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Build insert query
	columns, placeholders, values := r.buildInsertQuery(entity)
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id",
		r.tableName, columns, placeholders)

	// Execute query
	var id uint
	err := r.db.QueryRowContext(ctx, query, values...).Scan(&id)
	if err != nil {
		return fmt.Errorf("failed to create entity: %w", err)
	}

	// Set the generated ID
	(*entity).SetID(id)
	return nil
}

// GetByID retrieves an entity by its ID
func (r *GenericRepository[T]) GetByID(ctx context.Context, id uint) (*T, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE id = $1", r.tableName)

	// Add soft delete check if supported
	if r.supportsSoftDelete() {
		query += " AND deleted_at IS NULL"
	}

	row := r.db.QueryRowContext(ctx, query, id)

	entity := new(T)
	err := r.scanEntity(row, entity)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("entity with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	return entity, nil
}

// Update updates an existing entity
func (r *GenericRepository[T]) Update(ctx context.Context, entity *T) error {
	// Set updated timestamp if entity supports it
	if timestampable, ok := any(entity).(modules.Timestampable); ok {
		timestampable.SetUpdatedAt(time.Now().Unix())
	}

	// Validate entity
	if err := (*entity).Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Build update query
	setClauses, values := r.buildUpdateQuery(entity)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d",
		r.tableName, setClauses, len(values)+1)

	values = append(values, (*entity).GetID())

	// Execute query
	result, err := r.db.ExecContext(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("failed to update entity: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("entity with ID %d not found", (*entity).GetID())
	}

	return nil
}

// Delete removes an entity (soft delete if supported, hard delete otherwise)
func (r *GenericRepository[T]) Delete(ctx context.Context, id uint) error {
	if r.supportsSoftDelete() {
		return r.softDelete(ctx, id)
	}
	return r.hardDelete(ctx, id)
}

// List retrieves entities with filtering and pagination
func (r *GenericRepository[T]) List(ctx context.Context, filters modules.ListFilters) ([]*T, int64, error) {
	// Build query with filters
	baseQuery := fmt.Sprintf("FROM %s", r.tableName)
	whereClause, args := r.buildWhereClause(filters)
	if whereClause != "" {
		baseQuery += " WHERE " + whereClause
	}

	// Add soft delete check if supported
	if r.supportsSoftDelete() {
		if whereClause != "" {
			baseQuery += " AND deleted_at IS NULL"
		} else {
			baseQuery += " WHERE deleted_at IS NULL"
		}
	}

	// Get total count
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count entities: %w", err)
	}

	// Build select query with pagination and sorting
	selectQuery := "SELECT * " + baseQuery

	// Add sorting
	if filters.SortBy != "" {
		direction := "ASC"
		if strings.ToUpper(filters.SortOrder) == "DESC" {
			direction = "DESC"
		}
		selectQuery += fmt.Sprintf(" ORDER BY %s %s", filters.SortBy, direction)
	}

	// Add pagination
	if filters.Limit > 0 {
		selectQuery += fmt.Sprintf(" LIMIT %d", filters.Limit)
	}
	if filters.Offset > 0 {
		selectQuery += fmt.Sprintf(" OFFSET %d", filters.Offset)
	}

	// Execute query
	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list entities: %w", err)
	}
	defer rows.Close()

	// Scan results
	var entities []*T
	for rows.Next() {
		entity := new(T)
		if err := r.scanEntity(rows, entity); err != nil {
			return nil, 0, fmt.Errorf("failed to scan entity: %w", err)
		}
		entities = append(entities, entity)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("row iteration error: %w", err)
	}

	return entities, total, nil
}

// Exists checks if an entity exists by ID
func (r *GenericRepository[T]) Exists(ctx context.Context, id uint) (bool, error) {
	query := fmt.Sprintf("SELECT 1 FROM %s WHERE id = $1", r.tableName)

	if r.supportsSoftDelete() {
		query += " AND deleted_at IS NULL"
	}

	var exists int
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return true, nil
}

// Helper methods

func (r *GenericRepository[T]) buildInsertQuery(entity *T) (string, string, []interface{}) {
	entityValue := reflect.ValueOf(*entity)
	entityType := reflect.TypeOf(*entity)

	var columns []string
	var placeholders []string
	var values []interface{}

	placeholder := 1
	for i := 0; i < entityValue.NumField(); i++ {
		field := entityType.Field(i)
		value := entityValue.Field(i)

		// Skip ID field (auto-generated)
		if strings.ToLower(field.Name) == "id" {
			continue
		}

		// Get database column name from tag or use field name
		dbTag := field.Tag.Get("db")
		if dbTag == "" || dbTag == "-" {
			dbTag = strings.ToLower(field.Name)
		}

		columns = append(columns, dbTag)
		placeholders = append(placeholders, fmt.Sprintf("$%d", placeholder))
		values = append(values, value.Interface())
		placeholder++
	}

	return strings.Join(columns, ", "), strings.Join(placeholders, ", "), values
}

func (r *GenericRepository[T]) buildUpdateQuery(entity *T) (string, []interface{}) {
	entityValue := reflect.ValueOf(*entity)
	entityType := reflect.TypeOf(*entity)

	var setClauses []string
	var values []interface{}

	placeholder := 1
	for i := 0; i < entityValue.NumField(); i++ {
		field := entityType.Field(i)
		value := entityValue.Field(i)

		// Skip ID field
		if strings.ToLower(field.Name) == "id" {
			continue
		}

		// Get database column name from tag or use field name
		dbTag := field.Tag.Get("db")
		if dbTag == "" || dbTag == "-" {
			dbTag = strings.ToLower(field.Name)
		}

		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", dbTag, placeholder))
		values = append(values, value.Interface())
		placeholder++
	}

	return strings.Join(setClauses, ", "), values
}

func (r *GenericRepository[T]) buildWhereClause(filters modules.ListFilters) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Add search condition if provided
	if filters.Search != "" {
		// This is a simple implementation - in production, you'd want to specify searchable fields
		conditions = append(conditions, fmt.Sprintf("LOWER(CAST(id AS TEXT)) LIKE LOWER($%d)", argIndex))
		args = append(args, "%"+filters.Search+"%")
		argIndex++
	}

	// Add custom filters
	for key, value := range filters.Filters {
		if value != "" {
			conditions = append(conditions, fmt.Sprintf("%s = $%d", key, argIndex))
			args = append(args, value)
			argIndex++
		}
	}

	return strings.Join(conditions, " AND "), args
}

func (r *GenericRepository[T]) scanEntity(scanner interface{ Scan(...interface{}) error }, entity *T) error {
	entityValue := reflect.ValueOf(entity).Elem()

	// Prepare scan destinations
	var scanDests []interface{}
	for i := 0; i < entityValue.NumField(); i++ {
		field := entityValue.Field(i)
		scanDests = append(scanDests, field.Addr().Interface())
	}

	return scanner.Scan(scanDests...)
}

func (r *GenericRepository[T]) supportsSoftDelete() bool {
	var entity T
	_, ok := any(entity).(modules.SoftDeletable)
	return ok
}

func (r *GenericRepository[T]) softDelete(ctx context.Context, id uint) error {
	now := time.Now().Unix()
	query := fmt.Sprintf("UPDATE %s SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL", r.tableName)

	result, err := r.db.ExecContext(ctx, query, now, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete entity: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("entity with ID %d not found", id)
	}

	return nil
}

func (r *GenericRepository[T]) hardDelete(ctx context.Context, id uint) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", r.tableName)

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("entity with ID %d not found", id)
	}

	return nil
}