package main

import (
	"database/sql"
	"time"
)

func createTableAnnounces(db *sql.DB) error {
	_, err := db.Exec("CREATE TABLE pollbc_announces (id varchar PRIMARY KEY, date timestamp with time zone, price varchar, placeID serial, title varchar, fetched timestamp with time zone)")
	return err
}

func createTablePlaces(db *sql.DB) error {
	_, err := db.Exec("CREATE TABLE pollbc_places (id serial PRIMARY KEY, city varchar, department varchar, arrondissement varchar)")
	return err
}

func hasPlace(db *sql.DB, place Place) (bool, error) {
	var id int
	err := db.QueryRow("SELECT id FROM pollbc_places WHERE city=$1 AND department=$2 AND arrondissement=$3",
		place.City, place.Department, place.Arrondissement).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func insertPlace(db *sql.DB, place Place) error {
	_, err := db.Exec("INSERT INTO pollbc_places (city, department, arrondissement) VALUES ($1, $2, $3)",
		place.City, place.Department, place.Arrondissement)
	return err
}

func selectIDFromPlaces(db *sql.DB, place Place) (id int, err error) {
	err = db.QueryRow("SELECT id FROM pollbc_places WHERE city=$1 AND department=$2 AND arrondissement=$3",
		place.City, place.Department, place.Arrondissement).Scan(&id)
	return id, err
}

func hasAnnounce(db *sql.DB, id string) (bool, error) {
	err := db.QueryRow("SELECT id FROM pollbc_announces WHERE id=$1", id).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func insertAnnounce(db *sql.DB, ann Announce) error {
	_, err := db.Exec("INSERT INTO pollbc_announces (id, date, price, placeID, title, fetched) VALUES ($1, $2, $3, $4, $5, $6)",
		ann.ID, ann.Date, ann.Price, ann.PlaceID, ann.Title, ann.Fetched)
	return err
}

func selectAnnounces(db *sql.DB) ([]Announce, error) {
	rows, err := db.Query("SELECT * FROM pollbc_announces")
	if err != nil {
		return nil, err
	}
	return scanAnnounces(rows)
}

func selectAnnouncesWherePlaceID(db *sql.DB, placeID int) ([]Announce, error) {
	rows, err := db.Query("SELECT * FROM pollbc_announces WHERE placeID=$1", placeID)
	if err != nil {
		return nil, err
	}
	return scanAnnounces(rows)
}

func scanAnnounces(rows *sql.Rows) ([]Announce, error) {
	ann := make([]Announce, 0)
	for rows.Next() {
		var id, price, title string
		var placeID int
		var date, fetched time.Time
		err := rows.Scan(&id, &date, &price, &placeID, &title, &fetched)
		if err != nil {
			return ann, err
		}

		paris, err := time.LoadLocation("Europe/Paris")
		if err != nil {
			return ann, err
		}
		fetched = fetched.In(paris)

		ann = append(ann, Announce{ID: id, Date: date, Price: price, PlaceID: placeID, Title: title, Fetched: fetched})
	}
	return ann, nil
}

func selectPlace(db *sql.DB, id int) (Place, error) {
	var city, department, arrondissement string
	err := db.QueryRow("SELECT * FROM pollbc_places WHERE id=$1", id).Scan(&id, &city, &department, &arrondissement)
	if err != nil {
		return Place{}, err
	}
	return Place{City: city, Department: department, Arrondissement: arrondissement}, nil
}
