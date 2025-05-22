package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/database"
	logicfunction "gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/logicFunction"
)

type Image struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type PostRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Images      []Image  `json:"images"`
	PosterID    string   `json:"poster_id,omitempty"`
	Tags        []string `json:"tags"`
}

func PostHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getPosts(w, r)
	case http.MethodPost:
		createPost(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getPosts(w http.ResponseWriter, _ *http.Request) {
	rows, err := database.QueryDB(db, `SELECT id, "Posted", title, description, "Images", theme, poster_id, tags FROM public."Posts"`)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	posts, err := logicfunction.RowsToJSON(rows)
	if err != nil {
		http.Error(w, "Failed to convert rows to JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(posts)
}

func createPost(w http.ResponseWriter, r *http.Request) {
	// Get session token from header
	sessionToken := r.Header.Get("Session-Token")
	if sessionToken == "" {
		http.Error(w, "Missing Session-Token header", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req PostRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	imagesJSON, err := json.Marshal(req.Images)
	if err != nil {
		http.Error(w, "Invalid images", http.StatusBadRequest)
		return
	}

	// Generate all parent ltree tags for each tag
	tagSet := make(map[string]struct{})
	for _, tag := range req.Tags {
		if tag == "" {
			continue
		}
		parts := strings.Split(tag, ".")
		for i := 1; i <= len(parts); i++ {
			parent := strings.Join(parts[:i], ".")
			tagSet[parent] = struct{}{}
		}
	}
	// Convert set to slice
	var allTags []string
	for tag := range tagSet {
		allTags = append(allTags, tag)
	}
	if len(allTags) == 0 {
		http.Error(w, "At least one tag is required", http.StatusBadRequest)
		return
	}
	// Convert tags to Postgres text[] format
	tagsPG := "{" + strings.Join(allTags, ",") + "}"

	query := `INSERT INTO public."Posts" (title, description, "Images", tags) VALUES ($1, $2, $3, $4) RETURNING id`
	var id int64
	err = db.QueryRow(query, req.Title, req.Description, string(imagesJSON), tagsPG).Scan(&id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database insert error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"id": %d}`, id)
}
