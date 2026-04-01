package recipes

import "database/sql"

const createRecipesTable = `
CREATE TABLE IF NOT EXISTS recipes (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	link TEXT NOT NULL,
	ingredients TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

// Migrate applies schema migrations for the recipe store.
func Migrate(db *sql.DB) error {
	_, err := db.Exec(createRecipesTable)
	return err
}
