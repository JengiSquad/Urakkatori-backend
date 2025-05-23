package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/auth"
	"gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/database"
)

func SkillsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		skillsInit(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func skillsInit(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetToken(r)
	if err != nil {
		http.Error(w, "Invalid user token", http.StatusUnauthorized)
		return
	}
	userUUID, err := auth.ExtractUserUUID(token)
	if err != nil {
		http.Error(w, "Invalid user UUID", http.StatusUnauthorized)
		return
	}

	uuidValue, err := uuid.Parse(userUUID)
	if err != nil {
		http.Error(w, "Failed to parse user UUID", http.StatusBadRequest)
		return
	}

	type SkillInput struct {
		Path  string `json:"path"`
		Level int    `json:"level"`
	}
	type RequestBody struct {
		Skills []SkillInput `json:"skills"`
	}
	type SkillOutput struct {
		Tag   string `json:"tag"`
		Level int    `json:"level"`
	}

	var req RequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert skills to required format
	var skills []SkillOutput
	for _, skill := range req.Skills {
		skills = append(skills, SkillOutput{
			Tag:   skill.Path,
			Level: skill.Level,
		})
	}

	// Convert to JSON string
	skillsJSON, err := json.Marshal(skills)
	if err != nil {
		http.Error(w, "Failed to marshal skills", http.StatusInternalServerError)
		return
	}

	// Update or insert user skills
	_, err = database.QueryDB(db, `
		INSERT INTO user_skill (id, skill) 
		VALUES ($1, $2) 
		ON CONFLICT (id) 
		DO UPDATE SET skill = $2`, userUUID, string(skillsJSON))
	if err != nil {
		http.Error(w, "Failed to update user skills", http.StatusInternalServerError)
		return
	}

	_, err = GetConnectionsFromGlobal(uuidValue)

	if err != nil {
		fmt.Println("Error getting connections:", err)
		http.Error(w, "Failed to get connections", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
