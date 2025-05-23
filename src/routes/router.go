package routes

import (
	"database/sql"
	"net/http"
)

var db *sql.DB

func Router(databaseConn *sql.DB) {
	db = databaseConn
	http.HandleFunc("/yap", yapHandler)
	http.HandleFunc("/tag/definition", tagHandler)
	http.HandleFunc("/post", PostHandler)
	http.HandleFunc("/match", MatchHandler)
	http.HandleFunc("/chats", ChatHandler)
	http.HandleFunc("/chats/sendmessage", ChatMessageHandler)
	http.HandleFunc("/chats/getchat", ChatIdHandler)
	http.HandleFunc("/user/uuid", UUIDHandler)
	http.HandleFunc("/user/displayname", UserHandler)
	http.HandleFunc("/user/posts", PostsByUUIDHandler)
	http.HandleFunc("/feed", FeedHandler)
}

func yapHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, Yap!"))
}
