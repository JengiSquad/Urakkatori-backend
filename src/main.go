package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/auth"
	"gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/database"
	"gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/routes"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file:", err)
	}
	auth.InitializeAuth()

	db, err := database.OpenDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	router := routes.Router(db)

	log.Fatal(http.ListenAndServe(":8080", router))
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, World!")
}
