package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
