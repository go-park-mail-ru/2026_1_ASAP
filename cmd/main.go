package main

import (
	"net/http"
	"github.com/go-chi/chi"
)

func main() {
	mux := chi.NewRouter()

	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/logout", logoutHandler)
	mux.HandleFunc("/register", registerHandler)
	mux.HandleFunc("/", mainHandler)

	http.ListenAndServe(":8080", mux)
}


func loginHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from login page"))
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from register page"))
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from logout page"))
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from main page"))
}