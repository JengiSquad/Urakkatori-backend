package routes

import (
	"database/sql"
	"net/http"

	"github.com/gorilla/mux"
)

var db *sql.DB

func Router(databaseConn *sql.DB) http.Handler {
	r := mux.NewRouter()
	db = databaseConn

	//r.HandleFunc("/yap", yapHandler).Methods("GET")
	r.HandleFunc("/tag/definition", TagHandler).Methods("GET")

	r.HandleFunc("/post", GetPostsHandler).Methods("GET")
	r.HandleFunc("/post", CreatePostHandler).Methods("POST")
	r.HandleFunc("/post", DeletePostHandler).Methods("DELETE")

	//r.HandleFunc("/match", MatchHandler).Methods("GET")

	r.HandleFunc("/chats", GetChatsHandler).Methods("GET")
	r.HandleFunc("/chats", CreateChatHandler).Methods("POST")
	r.HandleFunc("/chats/sendmessage", ChatMessageHandler).Methods("POST")
	r.HandleFunc("/chats/getchat", GetChatByIdHandler).Methods("POST")

	r.HandleFunc("/user/uuid", GetUUIDHandler).Methods("GET")
	r.HandleFunc("/user/displayname", getDisplaynameHandler).Methods("POST")
	r.HandleFunc("/user/posts", GetPostsByUUIDHandler).Methods("POST")

	r.HandleFunc("/feed", FeedHandler).Methods("GET")

	r.HandleFunc("/user/skills", SkillsHandler).Methods("POST")
	return r
}

func yapHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, Yap!"))
}
