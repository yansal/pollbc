package models

import (
	"database/sql"
	"time"
)

type Announce struct {
	PK    int
	URL   string
	Date  time.Time
	Price string
	Title string

	Fetched time.Time

	PlacePK int
}

func CreateTableAnnounces() error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS pollbc_announces (
		pk serial PRIMARY KEY,
		url text UNIQUE NOT NULL,
		date timestamp with time zone NOT NULL,
		price text,
		title text NOT NULL,
		fetched timestamp with time zone NOT NULL,
		place_pk serial REFERENCES pollbc_places(pk)
	);`)
	return err
}

func HasAnnounce(url string) (bool, error) {
	var pk int
	err := db.QueryRow("SELECT pk FROM pollbc_announces WHERE url=$1", url).Scan(&pk)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func InsertAnnounce(ann Announce) error {
	_, err := db.Exec("INSERT INTO pollbc_announces (url, date, price, title, fetched, place_pk) VALUES ($1, $2, $3, $4, $5, $6)",
		ann.URL, ann.Date, ann.Price, ann.Title, ann.Fetched, ann.PlacePK)
	return err
}

func SelectAnnounces() ([]Announce, error) {
	rows, err := db.Query("SELECT * FROM pollbc_announces ORDER BY date DESC LIMIT 35")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAnnounces(rows)
}

func SelectAnnouncesWherePlacePK(placePK int) ([]Announce, error) {
	rows, err := db.Query("SELECT * FROM pollbc_announces WHERE place_pk=$1 ORDER BY date DESC LIMIT 35", placePK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAnnounces(rows)
}

func scanAnnounces(rows *sql.Rows) ([]Announce, error) {
	ann := make([]Announce, 0)
	for rows.Next() {
		a := Announce{}
		err := rows.Scan(&a.PK, &a.URL, &a.Date, &a.Price, &a.Title, &a.Fetched, &a.PlacePK)
		if err != nil {
			return ann, err
		}

		ann = append(ann, a)
	}
	if err := rows.Err(); err != nil {
		return ann, err
	}
	return ann, nil
}

func DeleteAnnounces() (int64, error) {
	res, err := db.Exec("DELETE FROM pollbc_announces WHERE date < NOW() - interval '1 week'")
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
