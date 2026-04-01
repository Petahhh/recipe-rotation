package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"recipe-rotation-2/internal/recipes"

	_ "modernc.org/sqlite"
)

func newTestMux(t *testing.T) http.Handler {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := recipes.Migrate(db); err != nil {
		_ = db.Close()
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return newMux(recipes.NewStore(db))
}

// muxRoundTripper runs requests through the given handler (no real network).
type muxRoundTripper struct {
	mux http.Handler
}

func (rt *muxRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	rt.mux.ServeHTTP(rec, req)
	return rec.Result(), nil
}

// TestRecipeBankPOSTPersistsRecipeVisibleOnGET documents task 6: POST must save a recipe
// so a subsequent GET /recipe-bank response body includes name, link, and ingredients.
func TestRecipeBankPOSTPersistsRecipeVisibleOnGET(t *testing.T) {
	t.Parallel()

	const (
		wantName        = "Test Pasta Primavera"
		wantLink        = "https://example.com/recipes/primavera"
		wantIngredients = "pasta\nolive oil\nvegetables"
	)

	mux := newTestMux(t)
	client := &http.Client{
		Transport:     &muxRoundTripper{mux: mux},
		CheckRedirect: func(*http.Request, []*http.Request) error { return nil },
	}

	form := url.Values{}
	form.Set("name", wantName)
	form.Set("link", wantLink)
	form.Set("ingredients", wantIngredients)

	postReq, err := http.NewRequest(http.MethodPost, "http://example.com/recipe-bank", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	postResp, err := client.Do(postReq)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, postResp.Body)
	postResp.Body.Close()

	// Follow redirect chain from POST, then always re-fetch GET /recipe-bank to assert persistence.
	for postResp.StatusCode >= 300 && postResp.StatusCode < 400 {
		loc := postResp.Header.Get("Location")
		if loc == "" {
			t.Fatalf("POST /recipe-bank: redirect %d but no Location", postResp.StatusCode)
		}
		locURL, err := postResp.Request.URL.Parse(loc)
		if err != nil {
			t.Fatalf("POST /recipe-bank: bad Location %q: %v", loc, err)
		}
		next, err := http.NewRequest(http.MethodGet, locURL.String(), nil)
		if err != nil {
			t.Fatal(err)
		}
		postResp, err = client.Do(next)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, postResp.Body)
		postResp.Body.Close()
	}

	getReq, err := http.NewRequest(http.MethodGet, "http://example.com/recipe-bank", nil)
	if err != nil {
		t.Fatal(err)
	}
	getResp, err := client.Do(getReq)
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	getBody, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(getBody)

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /recipe-bank after POST: want status %d, got %d; body=%q", http.StatusOK, getResp.StatusCode, body)
	}
	if !strings.Contains(body, wantName) {
		t.Fatalf("GET /recipe-bank: response body must include submitted name %q; got %q", wantName, body)
	}
	if !strings.Contains(body, wantLink) {
		t.Fatalf("GET /recipe-bank: response body must include submitted link %q; got %q", wantLink, body)
	}
	if !strings.Contains(body, wantIngredients) {
		t.Fatalf("GET /recipe-bank: response body must include submitted ingredients %q; got %q", wantIngredients, body)
	}
}

func TestHomeReturnsHTMLWithRecipeBankLink(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	newTestMux(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: want status %d, got %d", http.StatusOK, rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(strings.ToLower(ct), "text/html") {
		t.Fatalf("GET /: want HTML content type, got Content-Type %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `href="/recipe-bank"`) && !strings.Contains(body, `href='/recipe-bank'`) {
		t.Fatalf("GET /: body must link to /recipe-bank; got %q", body)
	}
}

func TestRecipeBankGETReturns200AndHTML(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/recipe-bank", nil)
	rec := httptest.NewRecorder()
	newTestMux(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /recipe-bank: want status %d, got %d", http.StatusOK, rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(strings.ToLower(ct), "text/html") {
		t.Fatalf("GET /recipe-bank: want HTML content type, got Content-Type %q", ct)
	}
}

// TestRecipeBankGETIncludesCreateRecipeForm documents the HTML contract for task 4:
//   - POST form with action /recipe-bank
//   - recipe name:   <input ... name="name" ...>   (id="recipe-name" optional)
//   - recipe link:   <input ... name="link" ...>   (id="recipe-link" optional)
//   - ingredients:   <textarea ... name="ingredients">...</textarea>
func TestRecipeBankGETIncludesCreateRecipeForm(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/recipe-bank", nil)
	rec := httptest.NewRecorder()
	newTestMux(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /recipe-bank: want status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	lower := strings.ToLower(body)

	if !strings.Contains(lower, "<form") {
		t.Fatal(`GET /recipe-bank: want a <form> for create-recipe`)
	}
	if !strings.Contains(lower, `method="post"`) && !strings.Contains(lower, `method='post'`) {
		t.Fatal(`GET /recipe-bank: want form method="post"`)
	}
	if !strings.Contains(body, `action="/recipe-bank"`) && !strings.Contains(body, `action='/recipe-bank'`) {
		t.Fatal(`GET /recipe-bank: want form action="/recipe-bank"`)
	}
	if !strings.Contains(body, `name="name"`) {
		t.Fatal(`GET /recipe-bank: want recipe name field name="name"`)
	}
	if !strings.Contains(body, `name="link"`) {
		t.Fatal(`GET /recipe-bank: want recipe link field name="link"`)
	}
	if !strings.Contains(lower, "<textarea") || !strings.Contains(body, `name="ingredients"`) {
		t.Fatal(`GET /recipe-bank: want ingredients as <textarea name="ingredients">`)
	}
}

// TestRecipeBankGETIncludesEditAndDeletePerCard (v2 id 7) requires each recipe card to expose
// an edit link and a POST delete form for that row's id.
func TestRecipeBankGETIncludesEditAndDeletePerCard(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := recipes.Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	store := recipes.NewStore(db)
	id, err := store.Create(ctx, "Solo Pie", "https://example.com/pie", "crust\nfilling")
	if err != nil {
		t.Fatal(err)
	}

	mux := newMux(store)
	req := httptest.NewRequest(http.MethodGet, "/recipe-bank", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /recipe-bank: want status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	wantEdit := fmt.Sprintf(`/recipe-bank/%d/edit`, id)
	if !strings.Contains(body, `href="`+wantEdit+`"`) && !strings.Contains(body, `href='`+wantEdit+`'`) {
		t.Fatalf("GET /recipe-bank: want edit link href %q in body; got %q", wantEdit, body)
	}

	wantDeleteAction := fmt.Sprintf(`/recipe-bank/%d/delete`, id)
	delAttr := `action="` + wantDeleteAction + `"`
	idx := strings.Index(body, delAttr)
	if idx < 0 {
		delAttr = `action='` + wantDeleteAction + `'`
		idx = strings.Index(body, delAttr)
	}
	if idx < 0 {
		t.Fatalf("GET /recipe-bank: want delete form action %q in body; got %q", wantDeleteAction, body)
	}
	formStart := strings.LastIndex(body[:idx], "<form")
	if formStart < 0 {
		t.Fatal("GET /recipe-bank: delete action must appear inside a <form>")
	}
	formEndRel := strings.Index(body[idx:], "</form>")
	if formEndRel < 0 {
		t.Fatal("GET /recipe-bank: unclosed delete form")
	}
	formEnd := idx + formEndRel + len("</form>")
	seg := strings.ToLower(body[formStart:formEnd])
	if !strings.Contains(seg, `method="post"`) && !strings.Contains(seg, `method='post'`) {
		t.Fatalf("GET /recipe-bank: delete form must use POST; segment=%q", seg)
	}
	if !strings.Contains(seg, `type="submit"`) && !strings.Contains(seg, `type='submit'`) {
		t.Fatalf("GET /recipe-bank: delete form must include submit; segment=%q", seg)
	}
}

// TestRecipeBankGETListsMultipleSeededRecipesAsCards documents task 8 HTML contract:
//   - Every stored recipe is one <article class="recipe-card"> (not merely a section).
//   - Each card has data-recipe-name set to that recipe’s display name (exact match).
//   - Response must include both names when two recipes exist in the store.
func TestRecipeBankGETListsMultipleSeededRecipesAsCards(t *testing.T) {
	const (
		nameA = "Alpha Soup"
		nameB = "Beta Stew"
	)

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := recipes.Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	store := recipes.NewStore(db)
	ctx := context.Background()
	if _, err := store.Create(ctx, nameA, "https://alpha.example/r", "broth\nnoodles"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(ctx, nameB, "https://beta.example/r", "beans\nspices"); err != nil {
		t.Fatal(err)
	}

	mux := newMux(store)
	req := httptest.NewRequest(http.MethodGet, "/recipe-bank", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /recipe-bank: want status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	lower := strings.ToLower(body)

	if c := strings.Count(lower, `<article class="recipe-card"`); c < 2 {
		t.Fatalf("GET /recipe-bank: want at least two <article class=\"recipe-card\"> (one per recipe); got %d; body=%q", c, body)
	}
	if !strings.Contains(body, `data-recipe-name="`+nameA+`"`) {
		t.Fatalf("GET /recipe-bank: want data-recipe-name=%q on a card; body=%q", nameA, body)
	}
	if !strings.Contains(body, `data-recipe-name="`+nameB+`"`) {
		t.Fatalf("GET /recipe-bank: want data-recipe-name=%q on a card; body=%q", nameB, body)
	}
}

// TestRecipeBankPOSTDeleteRemovesRecipe (v2 id 9): after delete, the recipe must not appear on GET /recipe-bank.
func TestRecipeBankPOSTDeleteRemovesRecipe(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := recipes.Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	store := recipes.NewStore(db)

	const wantName = "To Be Deleted Soup"
	id, err := store.Create(ctx, wantName, "https://example.com/del", "water\nsalt")
	if err != nil {
		t.Fatal(err)
	}

	mux := newMux(store)
	// Do not follow 303 from POST delete so we assert redirect headers, then GET explicitly.
	clientNoFollow := &http.Client{
		Transport: &muxRoundTripper{mux: mux},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	client := &http.Client{
		Transport:     &muxRoundTripper{mux: mux},
		CheckRedirect: func(*http.Request, []*http.Request) error { return nil },
	}

	delURL := fmt.Sprintf("http://example.com/recipe-bank/%d/delete", id)
	postReq, err := http.NewRequest(http.MethodPost, delURL, strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	postResp, err := clientNoFollow.Do(postReq)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, postResp.Body)
	postResp.Body.Close()

	if postResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST %s: want status %d, got %d", delURL, http.StatusSeeOther, postResp.StatusCode)
	}
	loc := postResp.Header.Get("Location")
	if loc == "" {
		t.Fatalf("POST %s: want Location header", delURL)
	}
	locURL, err := postReq.URL.Parse(loc)
	if err != nil {
		t.Fatalf("POST %s: bad Location %q: %v", delURL, loc, err)
	}
	if locURL.Path != "/recipe-bank" {
		t.Fatalf("POST %s: want redirect to /recipe-bank, got %q", delURL, locURL.Path)
	}

	getReq, err := http.NewRequest(http.MethodGet, "http://example.com/recipe-bank", nil)
	if err != nil {
		t.Fatal(err)
	}
	getResp, err := client.Do(getReq)
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	getBody, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(getBody)

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /recipe-bank after delete: want status %d, got %d; body=%q", http.StatusOK, getResp.StatusCode, body)
	}
	if strings.Contains(body, wantName) {
		t.Fatalf("GET /recipe-bank: body must not include deleted recipe name %q; got %q", wantName, body)
	}
}

func TestRecipeBankPOSTDeleteUnknownIDReturns404(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/recipe-bank/99999/delete", nil)
	rec := httptest.NewRecorder()
	newTestMux(t).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("POST delete unknown id: want status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

// TestRecipeBankGETEditPrefillsForm (v2 id 11): edit page must expose name/link/ingredients for the seeded row.
func TestRecipeBankGETEditPrefillsForm(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := recipes.Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	store := recipes.NewStore(db)

	const (
		wantName        = "Edit Test Curry"
		wantLink        = "https://example.com/curry"
		wantIngredients = "spice\nrice"
	)
	id, err := store.Create(ctx, wantName, wantLink, wantIngredients)
	if err != nil {
		t.Fatal(err)
	}

	mux := newMux(store)
	editPath := fmt.Sprintf("/recipe-bank/%d/edit", id)
	req := httptest.NewRequest(http.MethodGet, editPath, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: want status %d, got %d; body=%q", editPath, http.StatusOK, rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(strings.ToLower(ct), "text/html") {
		t.Fatalf("GET %s: want HTML content type, got Content-Type %q", editPath, ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `name="name"`) {
		t.Fatalf("GET %s: want name field name=\"name\"; body=%q", editPath, body)
	}
	if !strings.Contains(body, `name="link"`) {
		t.Fatalf("GET %s: want link field name=\"link\"; body=%q", editPath, body)
	}
	if !strings.Contains(body, `name="ingredients"`) {
		t.Fatalf("GET %s: want ingredients field name=\"ingredients\"; body=%q", editPath, body)
	}
	if !strings.Contains(body, `value="`+wantName+`"`) && !strings.Contains(body, `value='`+wantName+`'`) {
		t.Fatalf("GET %s: want name input value %q; body=%q", editPath, wantName, body)
	}
	if !strings.Contains(body, `value="`+wantLink+`"`) && !strings.Contains(body, `value='`+wantLink+`'`) {
		t.Fatalf("GET %s: want link input value %q; body=%q", editPath, wantLink, body)
	}
	if !strings.Contains(body, wantIngredients) {
		t.Fatalf("GET %s: want ingredients text %q in body; body=%q", editPath, wantIngredients, body)
	}
}

// TestRecipeBankPOSTEditUpdatesRecipeVisibleOnGET (v2 id 13): POST edit form persists; list shows new values, not originals.
func TestRecipeBankPOSTEditUpdatesRecipeVisibleOnGET(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := recipes.Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	store := recipes.NewStore(db)

	const (
		oldName        = "Original Chowder"
		oldLink        = "https://example.com/old-chowder"
		oldIngredients = "potato\ncream"
		newName        = "Updated Bisque"
		newLink        = "https://example.com/new-bisque"
		newIngredients = "lobster\nsherry"
	)
	id, err := store.Create(ctx, oldName, oldLink, oldIngredients)
	if err != nil {
		t.Fatal(err)
	}

	mux := newMux(store)
	clientNoFollow := &http.Client{
		Transport: &muxRoundTripper{mux: mux},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	client := &http.Client{
		Transport:     &muxRoundTripper{mux: mux},
		CheckRedirect: func(*http.Request, []*http.Request) error { return nil },
	}

	editURL := fmt.Sprintf("http://example.com/recipe-bank/%d/edit", id)
	form := url.Values{}
	form.Set("name", newName)
	form.Set("link", newLink)
	form.Set("ingredients", newIngredients)
	postReq, err := http.NewRequest(http.MethodPost, editURL, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	postResp, err := clientNoFollow.Do(postReq)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, postResp.Body)
	postResp.Body.Close()

	if postResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST %s: want status %d, got %d", editURL, http.StatusSeeOther, postResp.StatusCode)
	}
	loc := postResp.Header.Get("Location")
	if loc == "" {
		t.Fatalf("POST %s: want Location header", editURL)
	}
	locURL, err := postReq.URL.Parse(loc)
	if err != nil {
		t.Fatalf("POST %s: bad Location %q: %v", editURL, loc, err)
	}
	if locURL.Path != "/recipe-bank" {
		t.Fatalf("POST %s: want redirect to /recipe-bank, got %q", editURL, locURL.Path)
	}

	getReq, err := http.NewRequest(http.MethodGet, "http://example.com/recipe-bank", nil)
	if err != nil {
		t.Fatal(err)
	}
	getResp, err := client.Do(getReq)
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	getBody, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(getBody)

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /recipe-bank after edit: want status %d, got %d; body=%q", http.StatusOK, getResp.StatusCode, body)
	}
	if !strings.Contains(body, newName) {
		t.Fatalf("GET /recipe-bank: want updated name %q in body; got %q", newName, body)
	}
	if !strings.Contains(body, newLink) {
		t.Fatalf("GET /recipe-bank: want updated link %q in body; got %q", newLink, body)
	}
	if !strings.Contains(body, newIngredients) {
		t.Fatalf("GET /recipe-bank: want updated ingredients %q in body; got %q", newIngredients, body)
	}
	if strings.Contains(body, oldName) {
		t.Fatalf("GET /recipe-bank: body must not include original name %q; got %q", oldName, body)
	}
}
