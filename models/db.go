package models

import (
	"database/sql"

	_ "github.com/yansal/pollbc/Godeps/_workspace/src/github.com/lib/pq"
)

var db *sql.DB

func InitDB(datasourceName string) {
	var err error
	db, err = sql.Open("postgres", datasourceName)
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}
}
