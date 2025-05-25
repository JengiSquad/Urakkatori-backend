package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/auth"
)

func FeedHandler(w http.ResponseWriter, r *http.Request) {
	userUUID, err := auth.GetUUID(r)
	if err != nil {
		http.Error(w, "Getting UUID went wrong", http.StatusUnauthorized)
		return
	}

	// Query the feed table for this user
	fmt.Println(userUUID)
	row := db.QueryRow(`SELECT posts FROM feed WHERE id = $1`, userUUID)
	var postsRaw []byte
	if err := row.Scan(&postsRaw); err != nil {
		http.Error(w, "Feed not found", http.StatusNotFound)
		return
	}

	// Parse Postgres array format like {1,2,3}
	var posts []int64
	postsStr := string(postsRaw)
	postsStr = strings.Trim(postsStr, "{}")
	if postsStr == "" {
		posts = []int64{}
	} else {
		postParts := strings.Split(postsStr, ",")
		for _, p := range postParts {
			val, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64)
			if err == nil {
				posts = append(posts, val)
			}
		}
	}

	fmt.Println(string(postsRaw))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(posts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
