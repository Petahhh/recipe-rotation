package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func newMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", homeHandler)
	mux.HandleFunc("GET /recipe-bank", recipeBankHandler)
	return mux
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>Recipe rotation</title></head>
<body>
<h1>recipe rotation</h1>
<p><a href="/recipe-bank">Recipe Bank</a></p>
</body>
</html>`)
}

func recipeBankHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
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
</body>
</html>`)
}

func main() {
	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}

	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, newMux()))
}
