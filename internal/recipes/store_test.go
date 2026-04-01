package recipes

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestStoreCreateThenListReturnsOneRecipe(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	const (
		wantName         = "Chili"
		wantLink         = "https://example.com/chili"
		wantIngredients = "beans, tomato"
	)

	store := NewStore(db)
	id, err := store.Create(ctx, wantName, wantLink, wantIngredients)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero recipe id after Create")
	}

	list, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List: want exactly 1 recipe, got %d", len(list))
	}
	got := list[0]
	if got.ID != id {
		t.Fatalf("List recipe ID: got %d, want %d", got.ID, id)
	}
	if got.Name != wantName {
		t.Fatalf("Name: got %q, want %q", got.Name, wantName)
	}
	if got.Link != wantLink {
		t.Fatalf("Link: got %q, want %q", got.Link, wantLink)
	}
	if got.Ingredients != wantIngredients {
		t.Fatalf("Ingredients: got %q, want %q", got.Ingredients, wantIngredients)
	}
}
