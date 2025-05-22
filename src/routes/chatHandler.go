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

func ChatHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getChats(w, r)
	case http.MethodPost:
		createChat(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func ChatMessageHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		chatMessage(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func ChatIdHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		getChatById(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func chatMessage(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req struct {
		ChatID  string `json:"chatid"`
		Message string `json:"message"`
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
	var chatsRaw json.RawMessage
	err = db.QueryRow(`
		SELECT user_id_a, user_id_b, chats
		FROM chat
		WHERE id = $1
	`, chatID).Scan(&userA, &userB, &chatsRaw)
	if err == sql.ErrNoRows {
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	} else if err != nil {
		fmt.Printf("Database error: %v\n", err)
		http.Error(w, "Failed to fetch chat", http.StatusInternalServerError)
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

	// Parse chats field (should be an array)
	var chats []map[string]interface{}
	if len(chatsRaw) == 0 {
		chats = []map[string]interface{}{}
	} else {
		if err := json.Unmarshal(chatsRaw, &chats); err != nil {
			// Try to fix broken format (object with one key)
			var obj map[string]json.RawMessage
			if err2 := json.Unmarshal(chatsRaw, &obj); err2 == nil && len(obj) == 1 {
				for k := range obj {
					if err3 := json.Unmarshal([]byte(k), &chats); err3 != nil {
						fmt.Printf("Invalid chats JSON: %v\nRaw: %s\n", err3, k)
						http.Error(w, "Invalid chat format", http.StatusInternalServerError)
						return
					}
					break
				}
			} else {
				fmt.Printf("Invalid chats JSON: %v\nRaw: %s\n", err, string(chatsRaw))
				http.Error(w, "Invalid chat format", http.StatusInternalServerError)
				return
			}
		}
	}

	// Get current timestamp (milliseconds)
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)

	// Append new message
	newMsg := map[string]interface{}{
		"sender":    sender,
		"message":   req.Message,
		"timestamp": fmt.Sprintf("%d", timestamp),
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
	_, err = db.Exec(`UPDATE chat SET chats = $1 WHERE id = $2`, chatsBytes, chatID)
	if err != nil {
		fmt.Printf("Error updating chat: %v\n", err)
		http.Error(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func getChatById(w http.ResponseWriter, r *http.Request) {
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
		SELECT id, chats, user_id_a, user_id_b
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

func getChats(w http.ResponseWriter, r *http.Request) {
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
        SELECT id, chats, user_id_a, user_id_b 
        FROM chat 
        WHERE user_id_a = $1 OR user_id_b = $1
    `, userUUID)
	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		http.Error(w, "Failed to retrieve chats", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Structure to hold each chat
	type Chat struct {
		ID      int16           `json:"id"`
		Chats   json.RawMessage `json:"chats"`
		UserIDA uuid.UUID       `json:"user_id_a"`
		UserIDB uuid.UUID       `json:"user_id_b"`
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

func createChat(w http.ResponseWriter, r *http.Request) {
	// Implement the logic to create a chat
	w.Write([]byte("Create Chat"))
}

// Replace this with your actual database connection function
func getDBConnection() *sql.DB {
	// This is a placeholder - implement based on your project's database setup
	// For example: return database.GetConnection()
	return nil
}
