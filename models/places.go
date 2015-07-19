package models

import "database/sql"

type Place struct {
	ID             int
	City           string
	Department     string
	Arrondissement string
}

func CreateTablePlaces() error {
	_, err := db.Exec("CREATE TABLE pollbc_places (id serial PRIMARY KEY, city varchar, department varchar, arrondissement varchar)")
	return err
}

func HasPlace(place Place) (bool, error) {
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

func InsertPlace(place Place) error {
	_, err := db.Exec("INSERT INTO pollbc_places (city, department, arrondissement) VALUES ($1, $2, $3)",
		place.City, place.Department, place.Arrondissement)
	return err
}

func SelectIDFromPlaces(place Place) (id int, err error) {
	err = db.QueryRow("SELECT id FROM pollbc_places WHERE city=$1 AND department=$2 AND arrondissement=$3",
		place.City, place.Department, place.Arrondissement).Scan(&id)
	return id, err
}

func SelectDistinctDepartmentFromPlaces() ([]string, error) {
	rows, err := db.Query("SELECT DISTINCT department FROM pollbc_places")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	departments := make([]string, 0)
	for rows.Next() {
		var department string
		err := rows.Scan(&department)
		if err != nil {
			return departments, err
		}
		departments = append(departments, department)
	}
	if err := rows.Err(); err != nil {
		return departments, err
	}
	return departments, nil
}

func SelectDepartmentWhereID(id int) (dpt string, err error) {
	err = db.QueryRow("SELECT department FROM pollbc_places WHERE id=$1", id).Scan(&dpt)
	return dpt, err
}

func SelectPlaces() (map[int]Place, error) {
	rows, err := db.Query("SELECT * FROM pollbc_places")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPlaces(rows)
}

func SelectPlacesWhereDepartment(department string) (map[int]Place, error) {
	rows, err := db.Query("SELECT * FROM pollbc_places WHERE department=$1", department)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPlaces(rows)
}

func scanPlaces(rows *sql.Rows) (map[int]Place, error) {
	places := make(map[int]Place)
	for rows.Next() {
		place := Place{}
		err := rows.Scan(&place.ID, &place.City, &place.Department, &place.Arrondissement)
		if err != nil {
			return places, err
		}

		places[place.ID] = place
	}
	if err := rows.Err(); err != nil {
		return places, err
	}
	return places, nil
}
