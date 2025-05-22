package routes

import (
	"encoding/json"
	"net/http"
	"sort"
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

func SendMatches(postID int, w http.ResponseWriter, r *http.Request) {
	rows, err := database.QueryDB(db, `SELECT poster_id, tags FROM public."Posts" WHERE id = $1`, postID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	originalPost, err := logicfunction.RowsToJSONObject(rows)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Fix: extract first result from "results" array and parse tags/poster_id
	results, ok := originalPost["results"].([]map[string]interface{})
	if !ok || len(results) == 0 {
		http.Error(w, "No results found for original post", http.StatusInternalServerError)
		return
	}
	first := results[0]

	// Parse poster_id
	originalPostPosterID, ok := first["poster_id"].(string)
	if !ok {
		http.Error(w, "Failed to parse poster_id from original post", http.StatusInternalServerError)
		return
	}

	// Parse tags (should be a string like "{01,01.02,01.02.01,01.02.01.02}")
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var kaikkiIhmisarvot []Ihmisarvot

	userData, err := logicfunction.RowsToUserTagLevelList(rows)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, user := range userData {
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

	// Sort kaikkiIhmisarvot by TotalLevel descending and assign ranks
	sort.Slice(kaikkiIhmisarvot, func(i, j int) bool {
		return kaikkiIhmisarvot[i].TotalLevel > kaikkiIhmisarvot[j].TotalLevel
	})

	var kaikkiIhmisarvotRanked []IhmisarvotRanked
	for i, v := range kaikkiIhmisarvot {
		kaikkiIhmisarvotRanked = append(kaikkiIhmisarvotRanked, IhmisarvotRanked{
			UserID:     v.UserID,
			TotalLevel: v.TotalLevel,
			Rank:       i + 1,
		})
	}
	topKymppi := kaikkiIhmisarvotRanked
	if len(topKymppi) > 10 {
		topKymppi = topKymppi[:10]
	}
	var userIDs []string
	for _, v := range topKymppi {
		userIDs = append(userIDs, v.UserID)
	}

	// Return early if no userIDs
	if len(userIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	inClause := ""
	args := make([]interface{}, len(userIDs))
	for i, id := range userIDs {
		inClause += "$" + strconv.Itoa(i+1)
		if i < len(userIDs)-1 {
			inClause += ","
		}
		args[i] = id
	}

	query := `SELECT id, poster_id, tags FROM public."Posts" WHERE poster_id IN (` + inClause + `)`
	rows, err = database.QueryDB(db, query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	topKymppiPosts, err := logicfunction.RowsToPostRowList(rows)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rows, err = database.QueryDB(db, `SELECT id, skill FROM public.user_skill WHERE id = $1`, originalPostPosterID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var ogJaba []Ihmisarvot

	userData, err = logicfunction.RowsToUserTagLevelList(rows)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(userData) == 0 {
		http.Error(w, "No user data found for original poster", http.StatusInternalServerError)
		return
	}
	ogData := userData[0].TagLevels
	for _, rawPorn := range topKymppiPosts {
		for _, tag := range rawPorn.Tags {
			for _, og := range ogData {
				if og.Tag == tag && og.Level != -1 {
					found := false
					for i := range ogJaba {
						if ogJaba[i].UserID == rawPorn.PosterID {
							ogJaba[i].TotalLevel += og.Level
							found = true
							break
						}
					}
					if !found {
						ogJaba = append(ogJaba, Ihmisarvot{
							UserID:     rawPorn.PosterID,
							TotalLevel: og.Level,
						})
					}
				}
			}
		}
	}

	// Sort ogJaba by TotalLevel descending
	sort.Slice(ogJaba, func(i, j int) bool {
		return ogJaba[i].TotalLevel > ogJaba[j].TotalLevel
	})
	/*
		// Assume kaikkiIhmisarvotRanked and ogJaba are already sorted by rank/level descending
		topOgJaba := ogJaba
		if len(ogJaba) > 10 {
			topOgJaba = ogJaba[:10]
		}

		// Build sets for fast lookup
		topKymppiSet := make(map[string]struct{})
		for _, v := range topKymppi {
			topKymppiSet[v.UserID] = struct{}{}
		}
		topOgJabaSet := make(map[string]struct{})
		for _, v := range topOgJaba {
			topOgJabaSet[v.UserID] = struct{}{}
		}

		// Find mutual matches (users in both topKymppi and topOgJaba)
		var mutualMatches []string
		for userID := range topKymppiSet {
			if _, ok := topOgJabaSet[userID]; ok {
				mutualMatches = append(mutualMatches, userID)
			}
		}

		// Define the struct for mutual match info
		type MutualMatch struct {
			OgPosterID  string `json:"ogposterId"`
			OgPostID    int    `json:"ogpostId"`
			OthersID    string `json:"othersId"`
			OtherPostID int    `json:"otherpostId"`
		}

		var mutualMatchList []MutualMatch

		// You need to know the original post's poster and postID
		ogPosterID := originalPostPosterID
		ogPostID := postID

		// Build a map from userID to their postID for topKymppiPosts
		userToPostID := make(map[string]string)
		for _, post := range topKymppiPosts {
			userToPostID[post.PosterID] = post.ID
		}

		// For each mutual match, collect the info
		for _, userID := range mutualMatches {
			otherPostIDStr, ok := userToPostID[userID]
			if !ok {
				continue
			}
			// Try to convert otherPostIDStr to int, fallback to 0 if not possible
			otherPostID := 0
			if idInt, err := strconv.Atoi(otherPostIDStr); err == nil {
				otherPostID = idInt
			}
			mutualMatchList = append(mutualMatchList, MutualMatch{
				OgPosterID:  ogPosterID,
				OgPostID:    ogPostID,
				OthersID:    userID,
				OtherPostID: otherPostID,
			})
		}*/

	// Marshal mutualMatchList to JSON and write to response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(topKymppi); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
