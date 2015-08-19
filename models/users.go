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

func SelectUsers() ([]User, error) {
	rows, err := db.Query("SELECT * FROM pollbc_users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.PK, &user.Email)
		if err != nil {
			return users, err
		}

		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return users, err
	}
	return users, nil
}

func SelectPlacesPKWhereUserPK(pk int) ([]int, error) {
	rows, err := db.Query("SELECT place_pk FROM pollbc_users_places WHERE user_pk = $1", pk)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var placesPKs []int
	for rows.Next() {
		var placePK int
		err := rows.Scan(&placePK)
		if err != nil {
			return placesPKs, err
		}

		placesPKs = append(placesPKs, placePK)
	}
	if err := rows.Err(); err != nil {
		return placesPKs, err
	}
	return placesPKs, nil
}
