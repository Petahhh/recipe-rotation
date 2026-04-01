package main

import (
	"database/sql"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"recipe-rotation-2/internal/recipes"

	_ "modernc.org/sqlite"
)

var recipeBankPageTmpl = template.Must(template.New("recipeBank").Parse(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>Recipe Bank</title></head>
<body>
<h1>Recipe Bank</h1>
<form method="post" action="/recipe-bank">
<p><label>Name <input type="text" name="name"></label></p>
<p><label>Link <input type="url" name="link"></label></p>
<p><label>Ingredients <textarea name="ingredients" rows="4" cols="40"></textarea></label></p>
<p><button type="submit">Add recipe</button></p>
</form>
{{range .}}
<article class="recipe-card" data-recipe-name="{{.Name}}">
<h2>{{.Name}}</h2>
<p><a href="{{.Link}}">{{.Link}}</a></p>
<pre>{{.Ingredients}}</pre>
<p><a href="/recipe-bank/{{.ID}}/edit">Edit</a></p>
<form method="post" action="/recipe-bank/{{.ID}}/delete">
<p><button type="submit">Delete</button></p>
</form>
</article>
{{end}}
</body>
</html>`))

var recipeEditPageTmpl = template.Must(template.New("recipeEdit").Parse(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>Edit recipe</title></head>
<body>
<h1>Edit recipe</h1>
<form method="post" action="/recipe-bank/{{.ID}}/edit">
<p><label>Name <input type="text" name="name" value="{{.Name}}"></label></p>
<p><label>Link <input type="url" name="link" value="{{.Link}}"></label></p>
<p><label>Ingredients <textarea name="ingredients" rows="4" cols="40">{{.Ingredients}}</textarea></label></p>
<p><button type="submit">Save</button></p>
</form>
<p><a href="/recipe-bank">Back to Recipe Bank</a></p>
</body>
</html>`))

type server struct {
	store    *recipes.Store
	tmpl     *template.Template
	editTmpl *template.Template
}

func newMux(store *recipes.Store) http.Handler {
	srv := &server{store: store, tmpl: recipeBankPageTmpl, editTmpl: recipeEditPageTmpl}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", srv.homeHandler)
	mux.HandleFunc("GET /recipe-bank", srv.recipeBankGetHandler)
	mux.HandleFunc("GET /recipe-bank/{id}/edit", srv.recipeBankEditGetHandler)
	mux.HandleFunc("POST /recipe-bank/{id}/edit", srv.recipeBankEditPostHandler)
	mux.HandleFunc("POST /recipe-bank", srv.recipeBankPostHandler)
	mux.HandleFunc("POST /recipe-bank/{id}/delete", srv.recipeBankDeletePostHandler)
	return mux
}

func (s *server) homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>Recipe rotation</title></head>
<body>
<h1>recipe rotation</h1>
<p><a href="/recipe-bank">Recipe Bank</a></p>
</body>
</html>`))
}

func (s *server) recipeBankGetHandler(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.List(r.Context())
	if err != nil {
		log.Printf("recipe list: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.Execute(w, list); err != nil {
		log.Printf("recipe bank template: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (s *server) recipeBankPostHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	link := r.FormValue("link")
	ingredients := r.FormValue("ingredients")

	if _, err := s.store.Create(r.Context(), name, link, ingredients); err != nil {
		log.Printf("recipe create: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", "/recipe-bank")
	w.WriteHeader(http.StatusSeeOther)
}

func (s *server) recipeBankEditGetHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}
	rec, err := s.store.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, recipes.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		log.Printf("recipe get: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.editTmpl.Execute(w, rec); err != nil {
		log.Printf("recipe edit template: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (s *server) recipeBankEditPostHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	link := r.FormValue("link")
	ingredients := r.FormValue("ingredients")
	if err := s.store.Update(r.Context(), id, name, link, ingredients); err != nil {
		if errors.Is(err, recipes.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		log.Printf("recipe update: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", "/recipe-bank")
	w.WriteHeader(http.StatusSeeOther)
}

func (s *server) recipeBankDeletePostHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}
	if err := s.store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, recipes.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		log.Printf("recipe delete: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", "/recipe-bank")
	w.WriteHeader(http.StatusSeeOther)
}

func openRecipeDB() (*sql.DB, error) {
	path := os.Getenv("RECIPE_DB_PATH")
	if path == "" {
		path = "data/recipes.db"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := recipes.Migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func main() {
	db, err := openRecipeDB()
	if err != nil {
		log.Fatalf("recipe db: %v", err)
	}
	defer func() { _ = db.Close() }()

	store := recipes.NewStore(db)

	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}

	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, newMux(store)))
}
