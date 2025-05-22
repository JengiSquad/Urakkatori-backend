package routes

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/database"
)

func UserHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		getDisplayname(w, r)
	//case http.MethodPost:
	//	createChat(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getDisplayname(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		UserID string `json:"userid"`
	}
	var req requestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		http.Error(w, "userid is required", http.StatusBadRequest)
		return
	}

	query := fmt.Sprintf(`SELECT raw_user_meta_data FROM auth.users WHERE id = '%s'`, req.UserID)
	rows, err := database.QueryDB(db, query)
	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	if !rows.Next() {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var rawUserMetaData sql.NullString
	if err := rows.Scan(&rawUserMetaData); err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var metaData map[string]interface{}
	if err := json.Unmarshal([]byte(rawUserMetaData.String), &metaData); err != nil {
		http.Error(w, "Failed to parse user meta data", http.StatusInternalServerError)
		return
	}

	displayName, ok := metaData["display_name"].(string)
	if !ok {
		http.Error(w, "display_name not found", http.StatusNotFound)
		return
	}

	resp := map[string]string{"display_name": displayName}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
