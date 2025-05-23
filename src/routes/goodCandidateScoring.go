package routes

import (
	"fmt"
	"strconv"
	"strings"

	"gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/database"
	logicfunction "gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/logicFunction"
)

type Ihmisarvot struct {
	UserID     string `json:"userid"`
	TotalLevel int    `json:"level"`
}
type IhmisarvotRanked struct {
	UserID     string `json:"userid"`
	TotalLevel int    `json:"level"`
	Rank       int    `json:"rank"`
}

type FeedRow struct {
	ID    string  `json:"id"`
	Posts []int64 `json:"posts"`
}

func AddConnections(postID int) {
	rows, err := database.QueryDB(db, `SELECT poster_id, tags FROM public."Posts" WHERE id = $1`, postID)
	if err != nil {
		fmt.Println("Error querying database:", err)
		return
	}
	originalPost, err := logicfunction.RowsToJSONObject(rows)
	if err != nil {
		fmt.Println("Error converting rows to JSON:", err)
		return
	}
	results, ok := originalPost["results"].([]map[string]interface{})
	if !ok || len(results) == 0 {
		fmt.Println("No results found for original post")
		return
	}
	first := results[0]

	originalPostPosterID, ok := first["poster_id"].(string)
	if !ok {
		fmt.Println("Failed to parse poster_id from original post")
		return
	}

	var originalPostTags []string
	switch v := first["tags"].(type) {
	case string:
		trimmed := strings.Trim(v, "{}")
		if trimmed != "" {
			originalPostTags = strings.Split(trimmed, ",")
		}
	case []interface{}:
		for _, t := range v {
			if s, ok := t.(string); ok {
				originalPostTags = append(originalPostTags, s)
			}
		}
	case []string:
		originalPostTags = v
	}

	rows, err = database.QueryDB(db, `SELECT id, skill FROM user_skill WHERE id != $1`, originalPostPosterID)
	if err != nil {
		fmt.Println("Error querying user_skill:", err)
		return
	}

	var kaikkiIhmisarvot []Ihmisarvot

	userData, err := logicfunction.RowsToUserTagLevelList(rows)
	if err != nil {
		fmt.Println("Error converting user data:", err)
		return
	}

	for _, user := range userData {
		found := false
		for _, v := range kaikkiIhmisarvot {
			if v.UserID == user.ID {
				found = true
				break
			}
		}
		if !found {
			kaikkiIhmisarvot = append(kaikkiIhmisarvot, Ihmisarvot{
				UserID:     user.ID,
				TotalLevel: 0,
			})
		}
		for _, tag := range user.TagLevels {
			for _, originalTag := range originalPostTags {
				if tag.Tag == originalTag && tag.Level != -1 {
					found := false
					for i := range kaikkiIhmisarvot {
						if kaikkiIhmisarvot[i].UserID == user.ID {
							kaikkiIhmisarvot[i].TotalLevel += tag.Level
							found = true
							break
						}
					}
					if !found {
						kaikkiIhmisarvot = append(kaikkiIhmisarvot, Ihmisarvot{
							UserID:     user.ID,
							TotalLevel: tag.Level,
						})
					}
				}
			}
		}
	}

	var scoredUserIDs []string
	for _, v := range kaikkiIhmisarvot {
		if v.TotalLevel > 0 {
			scoredUserIDs = append(scoredUserIDs, v.UserID)
		}
	}

	if len(scoredUserIDs) == 0 {
		return
	}

	// Ensure each user has a feed row; if not, insert with empty posts array
	for _, id := range scoredUserIDs {
		var exists bool
		row := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM feed WHERE id = $1)`, id)
		if err := row.Scan(&exists); err != nil {
			fmt.Println("Failed to check feed existence for user:", id, "error:", err)
			return
		}
		if !exists {
			_, err := database.QueryDB(db, `INSERT INTO feed (id, posts) VALUES ($1, '{}')`, id)
			if err != nil {
				fmt.Println("Failed to insert feed for user:", id)
				fmt.Println("Error:", err)
				return
			}
		}
	}

	// Add the new postID to each user's feed in the database (always append, even if already present)
	for _, id := range scoredUserIDs {
		_, err := database.QueryDB(db, `UPDATE feed SET posts = array_append(posts, $1) WHERE id = $2`, postID, id)
		if err != nil {
			fmt.Println("Failed to update feed for user:", id, "error:", err)
			return
		}
	}

	// Build the IN clause and args
	inClause := ""
	args := make([]interface{}, len(scoredUserIDs))
	for i, id := range scoredUserIDs {
		inClause += "$" + strconv.Itoa(i+1)
		if i < len(scoredUserIDs)-1 {
			inClause += ","
		}
		args[i] = id
	}

	// Query the feed table
	query := `SELECT id, posts FROM feed WHERE id IN (` + inClause + `)`
	rows, err = database.QueryDB(db, query, args...)
	if err != nil {
		fmt.Println("Failed to query feeds:", err)
		return
	}
	defer rows.Close()

	var feeds []FeedRow

	for rows.Next() {
		var id string
		var postsRaw []byte
		if err := rows.Scan(&id, &postsRaw); err != nil {
			fmt.Println("Failed to scan feed row:", err)
			return
		}
		var posts []int64
		postsStr := strings.Trim(string(postsRaw), "{}")
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
		feeds = append(feeds, FeedRow{ID: id, Posts: posts})
	}

	feedMap := make(map[string][]int64)
	for _, feed := range feeds {
		feedMap[feed.ID] = feed.Posts
	}
	return
}

/*
func SendMatches(postID int, w http.ResponseWriter, r *http.Request) {
	connections := addConnection(postID)
	if connections == nil {
		http.Error(w, "Failed to get connections", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(connections); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}*/
