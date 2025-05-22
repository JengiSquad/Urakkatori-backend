package database

import (
	"database/sql"
	"fmt"
)

func OpenDB() (*sql.DB, error) {
	const (
		host      = "aws-0-eu-north-1.pooler.supabase.com"
		port      = 5432
		user      = "postgres.uphrachnmeiwrgbzmdat"
		password  = "urakkatori1"
		dbname    = "postgres"
		pool_mode = "session"
	)
	psqlConString := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s pool_mode=%s sslmode=disable",
		host, port, user, password, dbname, pool_mode)
	return sql.Open("postgres", psqlConString)
}

func QueryDB(db *sql.DB, queryString string, args ...interface{}) (*sql.Rows, error) {
	return db.Query(queryString, args...)
}
