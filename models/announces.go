package models

import (
	"database/sql"
	"time"
)

type Announce struct {
	ID    string
	Date  time.Time
	Price string
	Title string

	PlaceID int

	Fetched time.Time
}

type ByDate []Announce

func (d ByDate) Len() int           { return len(d) }
func (d ByDate) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d ByDate) Less(i, j int) bool { return d[i].Date.After(d[j].Date) }

func CreateTableAnnounces() error {
	_, err := db.Exec("CREATE TABLE pollbc_announces (id varchar PRIMARY KEY, date timestamp with time zone, price varchar, placeID serial, title varchar, fetched timestamp with time zone)")
	return err
}

func HasAnnounce(id string) (bool, error) {
	err := db.QueryRow("SELECT id FROM pollbc_announces WHERE id=$1", id).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func InsertAnnounce(ann Announce) error {
	_, err := db.Exec("INSERT INTO pollbc_announces (id, date, price, placeID, title, fetched) VALUES ($1, $2, $3, $4, $5, $6)",
		ann.ID, ann.Date, ann.Price, ann.PlaceID, ann.Title, ann.Fetched)
	return err
}

func SelectAnnounces() ([]Announce, error) {
	rows, err := db.Query("SELECT * FROM pollbc_announces")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAnnounces(rows)
}

func SelectAnnouncesWherePlaceID(placeID int) ([]Announce, error) {
	rows, err := db.Query("SELECT * FROM pollbc_announces WHERE placeID=$1", placeID)
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
		err := rows.Scan(&a.ID, &a.Date, &a.Price, &a.PlaceID, &a.Title, &a.Fetched)
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
