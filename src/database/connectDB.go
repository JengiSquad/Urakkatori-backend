package database

import (
	"database/sql"
	"fmt"
	"os"
)

func OpenDB() (*sql.DB, error) {
	host := os.Getenv("SUPABASE_HOST")
	port := os.Getenv("SUPABASE_PORT")
	user := os.Getenv("SUPABASE_USER")
	password := os.Getenv("SUPABASE_PASSWORD")
	dbname := os.Getenv("SUPABASE_DATABASE")
	pool_mode := os.Getenv("SUPABASE_POOL_MODE")
	if host == "" || port == "" || user == "" || password == "" || dbname == "" || pool_mode == "" {
		return nil, fmt.Errorf("database environment variables are not set")
	}
	psqlPort := 5432
	fmt.Sscanf(port, "%d", &psqlPort)
	psqlConString := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s pool_mode=%s sslmode=disable",
		host, psqlPort, user, password, dbname, pool_mode)
	return sql.Open("postgres", psqlConString)
}

func QueryDB(db *sql.DB, queryString string, args ...interface{}) (*sql.Rows, error) {
	return db.Query(queryString, args...)
}
