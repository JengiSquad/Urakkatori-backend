package routes

import (
	"database/sql"
	"net/http"
)

var db *sql.DB

func Router(databaseConn *sql.DB) {
	db = databaseConn
	http.HandleFunc("/yap", yapHandler)
	http.HandleFunc("/tag/definition", tagHandler)
	http.HandleFunc("/post", PostHandler)
}

func yapHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, Yap!"))
}
