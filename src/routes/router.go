package routes

import (
	"database/sql"
	"fmt"
	"net/http"

	"gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/database"
	logicfunction "gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/logicFunction"
)

var db *sql.DB

func Router(databaseConn *sql.DB) {
	db = databaseConn
	http.HandleFunc("/Yap", yapHandler)
	http.HandleFunc("/tag/definition", tagHandler)
}

func yapHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, Yap!"))
}

func tagHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := database.QueryDB(db, `SELECT * FROM tag_definitions`)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	jsonData, err := logicfunction.RowsToJSON(rows)
	if err != nil {
		http.Error(w, "Failed to convert rows to JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}
