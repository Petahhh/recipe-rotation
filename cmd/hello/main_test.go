package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

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

	mux := newMux()
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
	newMux().ServeHTTP(rec, req)

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
	newMux().ServeHTTP(rec, req)

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
	newMux().ServeHTTP(rec, req)

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
