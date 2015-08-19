package models

import (
	"database/sql"
	"log"
	"strconv"
	"unicode"
)

type Place struct {
	PK             int
	City           string
	Arrondissement string

	DepartmentPK int
}

type ByCity []Place

func (d ByCity) Len() int           { return len(d) }
func (d ByCity) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d ByCity) Less(i, j int) bool { return d[i].City < d[j].City }

type ByArrondissement []Place

func (d ByArrondissement) Len() int      { return len(d) }
func (d ByArrondissement) Swap(i, j int) { d[i], d[j] = d[j], d[i] }
func (d ByArrondissement) Less(i, j int) bool {
	ai, err := toInt(d[i].Arrondissement)
	if err != nil {
		log.Print("ByArrondissement.Less: " + err.Error())
	}
	aj, err := toInt(d[j].Arrondissement)
	if err != nil {
		log.Print("ByArrondissement.Less: " + err.Error())
	}
	return ai < aj
}

func toInt(s string) (int, error) {
	for i, v := range s {
		if !unicode.IsDigit(v) {
			s = s[:i]
			break
		}
	}
	return strconv.Atoi(s)
}

func CreateTablePlaces() error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS pollbc_places (
		pk serial PRIMARY KEY,
		city text,
		arrondissement text,
		department_pk serial REFERENCES pollbc_departements(pk),
		UNIQUE(city, arrondissement, department_pk)
	);`)
	return err
}

func HasPlace(place Place) (bool, error) {
	var pk int
	err := db.QueryRow("SELECT pk FROM pollbc_places WHERE city=$1 AND arrondissement=$2 AND department_pk=$3",
		place.City, place.Arrondissement, place.DepartmentPK).Scan(&pk)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func InsertPlace(place Place) error {
	_, err := db.Exec("INSERT INTO pollbc_places (city, arrondissement, department_pk) VALUES ($1, $2, $3)",
		place.City, place.Arrondissement, place.DepartmentPK)
	return err
}

func SelectPlaces() ([]Place, error) {
	rows, err := db.Query("SELECT * FROM pollbc_places")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPlaces(rows)
}

func SelectPlacesWhereDepartmentPK(dptPK int) ([]Place, error) {
	rows, err := db.Query("SELECT * FROM pollbc_places WHERE department_pk=$1", dptPK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPlaces(rows)
}

func scanPlaces(rows *sql.Rows) ([]Place, error) {
	var places []Place
	for rows.Next() {
		place := Place{}
		err := rows.Scan(&place.PK, &place.City, &place.Arrondissement, &place.DepartmentPK)
		if err != nil {
			return places, err
		}

		places = append(places, place)
	}
	if err := rows.Err(); err != nil {
		return places, err
	}
	return places, nil
}

func SelectPKFromPlaces(place Place) (pk int, err error) {
	err = db.QueryRow("SELECT pk FROM pollbc_places WHERE city=$1 AND arrondissement=$2 AND department_pk=$3",
		place.City, place.Arrondissement, place.DepartmentPK).Scan(&pk)
	return pk, err
}

func SelectDepartmentPKWherePK(pk int) (dptPK int, err error) {
	err = db.QueryRow("SELECT department_pk FROM pollbc_places WHERE pk=$1", pk).Scan(&dptPK)
	return dptPK, err
}
