package main

import (
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
	"gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/database"
	"gitlab.paivola.fi/jhautalu/Urakka-Urakasta-Backend/src/routes"
)

func main() {
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
