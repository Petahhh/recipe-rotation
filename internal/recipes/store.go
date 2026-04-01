package recipes

import (
	"context"
	"database/sql"
	"errors"
)

// ErrNotFound is returned when no recipe matches the requested id.
var ErrNotFound = errors.New("recipe not found")

// Store reads and writes recipes in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore returns a recipe store backed by db. Caller must run Migrate on db.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Create inserts a recipe and returns its id.
func (s *Store) Create(ctx context.Context, name, link, ingredients string) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO recipes (name, link, ingredients) VALUES (?, ?, ?)`,
		name, link, ingredients,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// List returns all recipes.
func (s *Store) List(ctx context.Context) ([]Recipe, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, link, ingredients FROM recipes ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Recipe
	for rows.Next() {
		var r Recipe
		if err := rows.Scan(&r.ID, &r.Name, &r.Link, &r.Ingredients); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Delete removes the recipe with the given id. ErrNotFound if no row was deleted.
func (s *Store) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM recipes WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
