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

func CreateTableUsersPlaces() error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS pollbc_users_places (
		user_pk serial REFERENCES pollbc_users(pk),
		place_pk serial REFERENCES pollbc_places(pk),
		PRIMARY KEY (user_pk, place_pk)
	);`)
	return err
}
