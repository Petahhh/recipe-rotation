package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
)

type recipe struct {
	Name        string
	Link        string
	Ingredients string
}

var (
	recipeMu   sync.Mutex
	recipeBank []recipe
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
<section class="recipe">
<h2>{{.Name}}</h2>
<p><a href="{{.Link}}">{{.Link}}</a></p>
<pre>{{.Ingredients}}</pre>
</section>
{{end}}
</body>
</html>`))

func newMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", homeHandler)
	mux.HandleFunc("GET /recipe-bank", recipeBankGetHandler)
	mux.HandleFunc("POST /recipe-bank", recipeBankPostHandler)
	return mux
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
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

func recipeBankGetHandler(w http.ResponseWriter, r *http.Request) {
	recipeMu.Lock()
	list := append([]recipe(nil), recipeBank...)
	recipeMu.Unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := recipeBankPageTmpl.Execute(w, list); err != nil {
		log.Printf("recipe bank template: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func recipeBankPostHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	link := r.FormValue("link")
	ingredients := r.FormValue("ingredients")

	recipeMu.Lock()
	recipeBank = append(recipeBank, recipe{
		Name:        name,
		Link:        link,
		Ingredients: ingredients,
	})
	recipeMu.Unlock()

	w.Header().Set("Location", "/recipe-bank")
	w.WriteHeader(http.StatusSeeOther)
}

func main() {
	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}

	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, newMux()))
}
