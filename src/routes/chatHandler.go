package routes

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid" // Assuming you use this for UUID handling
	"gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/auth"
	logicfunction "gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/logicFunction"
	// Import your database connection package
	// For example: "yourproject/database"
)

func ChatMessageHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req struct {
		ChatID  string `json:"chatid"`
		Message string `json:"message"`
	}

	type ChatMessage struct {
		Sender    string `json:"sender"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert ChatID from string to int16
	var chatID int16
	if _, err := fmt.Sscanf(req.ChatID, "%d", &chatID); err != nil {
		http.Error(w, "Invalid chatid format", http.StatusBadRequest)
		return
	}

	// Get user UUID from token
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
	userUUIDParsed, err := uuid.Parse(userUUID)
	if err != nil {
		http.Error(w, "Invalid user UUID", http.StatusUnauthorized)
		return
	}

	// Fetch chat participants and chats field
	var userA, userB uuid.UUID
	var last_updated int64
	var chatsRaw json.RawMessage
	err = db.QueryRow(`
		SELECT user_id_a, user_id_b, messages, last_updated
		FROM chat
		WHERE id = $1
	`, chatID).Scan(&userA, &userB, &chatsRaw, &last_updated)
	fmt.Printf("Fetching chatID: %d\n", chatID)
	fmt.Println("Raw chats from DB:", string(chatsRaw))
	if err == sql.ErrNoRows {
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	} else if err != nil {
		fmt.Printf("Database error: %v\n", err)
		http.Error(w, "Failed to fetch chat", http.StatusInternalServerError)
		return
	}

	if err != nil {
		fmt.Printf("Error marshalling chats: %v\n", err)
		http.Error(w, "Failed to process message", http.StatusInternalServerError)
		return
	}

	// Check if user is a participant
	var sender string
	if userUUIDParsed == userA {
		sender = "user_id_a"
	} else if userUUIDParsed == userB {
		sender = "user_id_b"
	} else {
		http.Error(w, "Forbidden: not a participant", http.StatusForbidden)
		return
	}

	var chats []ChatMessage

	if len(chatsRaw) == 0 {
		chats = []ChatMessage{}
	} else {

		err := json.Unmarshal(chatsRaw, &chats)
		if err != nil {
			fmt.Printf("Error unmarshalling chats: %v\n", err)
			http.Error(w, "Failed to process message", http.StatusInternalServerError)
			return
		}

	}

	// Get current timestamp (milliseconds)
	timestamp := int(time.Now().UnixNano() / int64(time.Millisecond))

	// Append new message
	newMsg := ChatMessage{
		Sender:    sender,
		Message:   req.Message,
		Timestamp: fmt.Sprintf("%d", timestamp),
	}
	chats = append(chats, newMsg)

	// Marshal back to JSON
	chatsBytes, err := json.Marshal(chats)
	if err != nil {
		fmt.Printf("Error marshalling chats: %v\n", err)
		http.Error(w, "Failed to process message", http.StatusInternalServerError)
		return
	}

	// Update chats in DB
	_, err = db.Exec(`UPDATE chat SET messages = $1 WHERE id = $2`, chatsBytes, chatID)
	if err != nil {
		fmt.Printf("Error updating chat: %v\n", err)
		http.Error(w, "Failed to save message", http.StatusInternalServerError)
		return
	}
	_, err = db.Exec(`UPDATE chat SET last_updated = $1 WHERE id = $2`, int(timestamp), chatID)
	if err != nil {
		fmt.Printf("Error updating chat: %v\n", err)
		http.Error(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func GetChatByIdHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChatID string `json:"chatid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert ChatID from string to int16
	var chatID int16
	if _, err := fmt.Sscanf(req.ChatID, "%d", &chatID); err != nil {
		http.Error(w, "Invalid chatid format", http.StatusBadRequest)
		return
	}

	// Use parameterized query to fetch the chat by id
	rows, err := db.Query(`
		SELECT id, messages, user_id_a, user_id_b
		FROM chat
		WHERE id = $1
	`, chatID)
	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		http.Error(w, "Failed to retrieve chat", http.StatusInternalServerError)
		return
	}

	chat_json, err := logicfunction.RowsToJSONObject(rows)
	if err != nil {
		fmt.Printf("Error parsing rows to database object: %v\n", err)
		http.Error(w, "Failed parsing chat", http.StatusInternalServerError)
		return
	}

	chatJSONBytes, err := json.Marshal(chat_json)
	if err != nil {
		fmt.Printf("Error marshalling chat JSON: %v\n", err)
		http.Error(w, "Failed to process chat", http.StatusInternalServerError)
		return
	}
	w.Write(chatJSONBytes)
}

func GetChatsHandler(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetToken(r)
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	userUUID, err := auth.ExtractUserUUID(token)
	if err != nil {
		http.Error(w, "Invalid user UUID", http.StatusBadRequest)
		return
	}

	// Use parameterized query to avoid SQL injection and type errors
	rows, err := db.Query(`
        SELECT id, messages, user_id_a, user_id_b, last_updated 
        FROM chat 
        WHERE user_id_a = $1 OR user_id_b = $1
    `, userUUID)
	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		http.Error(w, "Failed to retrieve messages", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Structure to hold each chat
	type Chat struct {
		ID          int16           `json:"id"`
		Chats       json.RawMessage `json:"chats"`
		UserIDA     uuid.UUID       `json:"user_id_a"`
		UserIDB     uuid.UUID       `json:"user_id_b"`
		LastUpdated int64           `json:"last_updated"`
	}

	// Collect all chats as JSON, then decode to []Chat
	chatsBytes, err := logicfunction.RowsToJSON(rows)
	if err != nil {
		fmt.Printf("Error converting rows to JSON: %v\n", err)
		http.Error(w, "Failed to process chats", http.StatusInternalServerError)
		return
	}

	// If chatsBytes is a base64-encoded string, decode it first
	var chatsArr []Chat
	// Try direct unmarshal first
	err = json.Unmarshal(chatsBytes, &chatsArr)

	if err != nil {
		fmt.Printf("Error unmarshalling chats JSON: %v\n", err)
		http.Error(w, "Failed to process chats", http.StatusInternalServerError)
		return
	}

	userUUIDParsed, err := uuid.Parse(userUUID)
	if err != nil {
		http.Error(w, "Invalid user UUID", http.StatusBadRequest)
		return
	}
	for i, chat := range chatsArr {
		if chat.UserIDB == userUUIDParsed {
			chatsArr[i].UserIDA, chatsArr[i].UserIDB = chatsArr[i].UserIDB, chatsArr[i].UserIDA
		}
	}

	chatsJSON, err := json.Marshal(chatsArr)
	if err != nil {
		fmt.Printf("Error marshalling chats JSON: %v\n", err)
		http.Error(w, "Failed to process chats", http.StatusInternalServerError)
		return
	}

	w.Write(chatsJSON)
}

func CreateChatHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserAID string `json:"user_a_id"`
		UserBID string `json:"user_b_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserAID == "" || req.UserBID == "" {
		http.Error(w, "user_a_id and user_b_id are required", http.StatusBadRequest)
		return
	}

	// Get the largest existing chat ID
	var maxID sql.NullInt16
	err := db.QueryRow("SELECT MAX(id) FROM chat").Scan(&maxID)
	if err != nil {
		fmt.Printf("Error getting max chat ID: %v\n", err)
		http.Error(w, "Failed to create chat", http.StatusInternalServerError)
		return
	}

	// Calculate next ID (start from 1 if no chats exist)
	var nextID int16 = 1
	if maxID.Valid {
		nextID = maxID.Int16 + 1
	}

	// Create new chat with specific ID
	timestamp := int(time.Now().UnixNano() / int64(time.Millisecond))
	_, err = db.Exec(`
		INSERT INTO chat (id, user_id_a, user_id_b, messages, last_updated) 
		VALUES ($1, $2, $3, '[]', $4)
	`, nextID, req.UserAID, req.UserBID, timestamp)

	if err != nil {
		fmt.Printf("Error creating chat: %v\n", err)
		http.Error(w, "Failed to create chat", http.StatusInternalServerError)
		return
	}

	resp := map[string]int16{"chat_id": nextID}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Replace this with your actual database connection function
func getDBConnection() *sql.DB {
	// This is a placeholder - implement based on your project's database setup
	// For example: return database.GetConnection()
	return nil
}
