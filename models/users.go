package models

type User struct {
	PK    int
	Email string
}

func CreateTableUsers() error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS pollbc_users (
		pk serial PRIMARY KEY,
		email text UNIQUE NOT NULL
	);`)
	return err
}
