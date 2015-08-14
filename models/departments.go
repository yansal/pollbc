package models

import "database/sql"

type Department struct {
	ID   int
	Name string
}

type ByName []Department

func (d ByName) Len() int           { return len(d) }
func (d ByName) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d ByName) Less(i, j int) bool { return d[i].Name < d[j].Name }

func CreateTableDepartements() error {
	_, err := db.Exec(`CREATE TABLE pollbc_departements (
		id serial PRIMARY KEY,
		name text
	);`)
	return err
}

func HasDepartment(dpt Department) (bool, error) {
	var id int
	err := db.QueryRow("SELECT id FROM pollbc_departements WHERE name=$1",
		dpt.Name).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func InsertDepartment(dpt Department) error {
	_, err := db.Exec("INSERT INTO pollbc_departements (name) VALUES ($1)",
		dpt.Name)
	return err
}

func SelectDepartments() ([]Department, error) {
	rows, err := db.Query("SELECT * FROM pollbc_departements")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var dpts []Department
	for rows.Next() {
		dpt := Department{}
		err := rows.Scan(&dpt.ID, &dpt.Name)
		if err != nil {
			return dpts, err
		}

		dpts = append(dpts, dpt)
	}
	if err := rows.Err(); err != nil {
		return dpts, err
	}
	return dpts, nil
}

func SelectDepartmentWhereID(id int) (dpt Department, err error) {
	err = db.QueryRow("SELECT * FROM pollbc_departements WHERE id=$1",
		id).Scan(&dpt.ID, &dpt.Name)
	return
}

func SelectIDFromDepartment(dpt Department) (id int, err error) {
	err = db.QueryRow("SELECT id FROM pollbc_departements WHERE name=$1",
		dpt.Name).Scan(&id)
	return id, err
}
