package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func main() {
	connStr := "host=localhost port=5432 user=postgres password=postgres dbname=subscription_db sslmode=disable"

	fmt.Println("Connecting with:", connStr)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println("Open error:", err)
		return
	}

	err = db.Ping()
	if err != nil {
		fmt.Println("Ping error:", err)
		return
	}

	fmt.Println("Connected successfully!")
}
