package recipes

import (
	"context"
	"database/sql"
)

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
