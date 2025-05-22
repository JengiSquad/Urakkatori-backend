package routes

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/auth"
	"gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/database"
	logicfunction "gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/logicFunction"
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

func UUIDHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getUUID(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func PostsByUUIDHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		getPostsByUUID(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getPostsByUUID(w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		UUID string `json:"uuid"`
	}
	var req requestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.UUID == "" {
		http.Error(w, "uuid is required", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(`SELECT * FROM public."Posts" WHERE poster_id = $1`, req.UUID)
	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		http.Error(w, "Failed to retrieve posts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Check if there are any rows
	hasRows := rows.Next()
	if !hasRows {
		http.Error(w, fmt.Sprintf("No posts found for user with UUID %s", req.UUID), http.StatusNotFound)
		return
	}

	// Reset the rows cursor since we moved it forward with rows.Next()
	rows, err = db.Query(`SELECT * FROM public."Posts" WHERE poster_id = $1`, req.UUID)
	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		http.Error(w, "Failed to retrieve posts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	postsJSON, err := logicfunction.RowsToJSON(rows)
	if err != nil {
		fmt.Printf("Error converting rows to JSON: %v\n", err)
		http.Error(w, "Failed to process posts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(postsJSON)
}

func getUUID(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetToken(r)
	if err != nil {
		http.Error(w, "Token not found", http.StatusUnauthorized)
		return
	}
	// If you use Bearer token in Authorization header, use:
	// tokenStr := r.Header.Get("Authorization")

	// Use your auth package to extract UUID
	userUUID, err := auth.ExtractUserUUID(token)
	if err != nil {
		http.Error(w, "Invalid user token", http.StatusUnauthorized)
		return
	}

	resp := map[string]string{"uuid": userUUID}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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
