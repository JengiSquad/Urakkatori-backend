package routes

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lib/pq"
	"gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/database"
)

type PostRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Images      []string `json:"images"`
	Tags        []string `json:"tags"`
}

type PostResponse struct {
	ID          int16    `json:"id"`
	Posted      string   `json:"posted"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	PosterID    string   `json:"poster_id"`
	Tags        []string `json:"tags"`
	Images      []string `json:"images"`
}

func PostHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getPosts(w, r)
	case http.MethodPost:
		createPost(w, r)
	case http.MethodDelete:
		deletePost(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
func getPosts(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	var rows *sql.Rows
	var err error

	query := `SELECT id, "Posted", title, description, poster_id, tags, "Images" FROM public."Posts"`
	if id != "" {
		query += " WHERE id = $1"
		rows, err = database.QueryDB(db, query, id)
	} else {
		rows, err = database.QueryDB(db, query)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []PostResponse
	for rows.Next() {
		var post PostResponse
		var tagsArr, imagesArr []string
		var posterID sql.NullString
		err := rows.Scan(&post.ID, &post.Posted, &post.Title, &post.Description, &posterID, pq.Array(&tagsArr), pq.Array(&imagesArr))
		if err != nil {
			http.Error(w, "Failed to scan row", http.StatusInternalServerError)
			return
		}
		post.Tags = tagsArr
		post.Images = imagesArr
		if posterID.Valid {
			post.PosterID = posterID.String
		}
		posts = append(posts, post)
	}

	w.Header().Set("Content-Type", "application/json")
	if id != "" && len(posts) == 1 {
		json.NewEncoder(w).Encode(posts[0])
	} else {
		json.NewEncoder(w).Encode(posts)
	}
}

func createPost(w http.ResponseWriter, r *http.Request) {
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
	var allTags []string
	for tag := range tagSet {
		allTags = append(allTags, tag)
	}
	if len(allTags) == 0 {
		http.Error(w, "At least one tag is required", http.StatusBadRequest)
		return
	}

	query := `INSERT INTO public."Posts" (title, description, "Images", tags) VALUES ($1, $2, $3, $4) RETURNING id`
	var id int16
	err = db.QueryRow(query, req.Title, req.Description, pq.Array(req.Images), pq.Array(allTags)).Scan(&id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database insert error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"id": %d}`, id)
}

func deletePost(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	query := `DELETE FROM public."Posts" WHERE id = $1`
	res, err := database.QueryDB(db, query, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database delete error: %v", err), http.StatusInternalServerError)
		return
	}
	defer res.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message": "Post with id %s deleted"}`, id)
}
