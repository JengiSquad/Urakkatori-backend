package routes

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
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
	defer rows.Close()

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

	// --- Add postID to globalfeed first, regardless of matched users ---
	var existsGlobal bool
	rowGlobal := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM globalfeed WHERE id = 0)`)
	if err := rowGlobal.Scan(&existsGlobal); err != nil {
		fmt.Println("Failed to check globalfeed existence:", err)
		return
	}

	if !existsGlobal {
		// Create globalfeed with the current post
		_, err := db.Exec(`INSERT INTO globalfeed (id, feed) VALUES (0, ARRAY[$1]::bigint[])`, postID)
		if err != nil {
			fmt.Println("Failed to insert globalfeed row with initial post:", err)
			return
		}
		fmt.Println("Created new globalfeed with post ID:", postID)
	} else {
		// Add to existing globalfeed if not already present
		_, err = db.Exec(`
			UPDATE globalfeed 
			SET feed = array_append(feed, $1) 
			WHERE id = 0 
			AND NOT feed @> ARRAY[$1]::bigint[]`, postID)
		if err != nil {
			fmt.Println("Failed to update globalfeed with post ID:", postID, "Error:", err)
			return
		}
		fmt.Println("Added post ID to globalfeed:", postID)
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
			_, err := db.Exec(`INSERT INTO feed (id, posts) VALUES ($1, '{}')`, id)
			if err != nil {
				fmt.Println("Failed to insert feed for user:", id)
				fmt.Println("Error:", err)
				return
			}
		}
	}

	// Add the new postID to each user's feed in the database (always append, even if already present)
	for _, id := range scoredUserIDs {
		_, err := db.Exec(`UPDATE feed SET posts = array_append(posts, $1) WHERE id = $2`, postID, id)
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
}

func GetConnectionsFromGlobal(userUUID uuid.UUID) ([]int64, error) {
	// Get all global post IDs
	rows, err := database.QueryDB(db, `SELECT feed FROM globalfeed WHERE id = 0`)
	if err != nil {
		return nil, fmt.Errorf("error querying globalfeed: %w", err)
	}
	defer rows.Close()

	var postsRaw []byte
	if rows.Next() {
		if err := rows.Scan(&postsRaw); err != nil {
			return nil, fmt.Errorf("error scanning globalfeed row: %w", err)
		}
	}

	var globalPosts []int64
	postsStr := strings.Trim(string(postsRaw), "{}")
	if postsStr == "" {
		return []int64{}, nil
	} else {
		postParts := strings.Split(postsStr, ",")
		for _, p := range postParts {
			val, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64)
			if err == nil {
				globalPosts = append(globalPosts, val)
			}
		}
	}

	// Get user skills as map[tag]level
	userSkillRows, err := database.QueryDB(db, `SELECT skill FROM user_skill WHERE id = $1`, userUUID.String())
	if err != nil {
		return nil, fmt.Errorf("error querying user_skill: %w", err)
	}
	defer userSkillRows.Close()

	userSkills := make(map[string]int)
	for userSkillRows.Next() {
		var skillRaw string
		if err := userSkillRows.Scan(&skillRaw); err != nil {
			continue
		}

		// Parse JSON array format
		type SkillItem struct {
			Tag   string `json:"tag"`
			Level int    `json:"level"`
		}
		var skills []SkillItem
		if err := json.Unmarshal([]byte(skillRaw), &skills); err != nil {
			continue
		}

		for _, skill := range skills {
			userSkills[skill.Tag] = skill.Level
		}
	}

	var userFeedPosts []int64

	for _, postID := range globalPosts {
		// Get tags for this post
		postRows, err := database.QueryDB(db, `SELECT tags FROM public."Posts" WHERE id = $1`, postID)
		if err != nil {
			continue
		}
		var tagsRaw string
		if postRows.Next() {
			if err := postRows.Scan(&tagsRaw); err != nil {
				postRows.Close()
				continue
			}
		}
		postRows.Close()
		tags := []string{}
		tagsStr := strings.Trim(tagsRaw, "{}")
		if tagsStr != "" {
			tags = strings.Split(tagsStr, ",")
		}
		// Check if user has skill > 1 for any tag
		hasSkill := false
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if lvl, ok := userSkills[tag]; ok && lvl > 1 {
				hasSkill = true
				break
			}
		}
		if hasSkill {
			userFeedPosts = append(userFeedPosts, postID)
		}
	}

	// Update user's feed in the feed table (replace posts)
	var exists bool
	row := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM feed WHERE id = $1)`, userUUID.String())
	if err := row.Scan(&exists); err != nil {
		return userFeedPosts, nil // skip update on error
	}
	if !exists {
		_, _ = db.Exec(`INSERT INTO feed (id, posts) VALUES ($1, '{}')`, userUUID.String())
	}
	postsArray := "{" + strings.Trim(strings.Replace(fmt.Sprint(userFeedPosts), " ", ",", -1), "[]") + "}"
	_, _ = db.Exec(`UPDATE feed SET posts = $1 WHERE id = $2`, postsArray, userUUID.String())

	return userFeedPosts, nil
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
