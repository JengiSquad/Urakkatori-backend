package routes

import (
	"fmt"
	"net/http"

	"gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/database"
	logicfunction "gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/logicFunction"
)

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
