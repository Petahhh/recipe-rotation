package recipes

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMigrationsCreateRecipesTable(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var n int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'recipes'`,
	).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected recipes table after migrations, sqlite_master count=%d", n)
	}

	rows, err := db.Query(`PRAGMA table_info(recipes)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var colCount int
	for rows.Next() {
		colCount++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if colCount == 0 {
		t.Fatal("PRAGMA table_info(recipes) returned no rows")
	}
}
