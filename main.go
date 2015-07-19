package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	_ "github.com/yansal/pollbc/Godeps/_workspace/src/github.com/lib/pq"
)

var paris *time.Location

func init() {
	var err error
	paris, err = time.LoadLocation("Europe/Paris")
	if err != nil {
		log.Fatal(err)
	}
}

type Place struct {
	ID             int
	City           string
	Department     string
	Arrondissement string
}

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

var db *sql.DB

func init() {
	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	err = createTableAnnounces(db)
	if err != nil {
		log.Print(err)
	}

	err = createTablePlaces(db)
	if err != nil {
		log.Print(err)
	}

}

func poll() {
	for {
		doc, err := fetch()
		if err != nil {
			log.Print(err)
			time.Sleep(time.Minute)
			continue
		}
		nodes := queryAnnounces(doc)

		count := 0
		for _, n := range nodes {
			place, err := queryPlace(n)
			if err != nil {
				log.Print(err)
				continue
			}

			ok, err := hasPlace(db, place)
			if err != nil {
				log.Print(err)
			} else if !ok {
				err := insertPlace(db, place)
				if err != nil {
					log.Print(err)
				}
			}
			placeID, err := selectIDFromPlaces(db, place)
			if err != nil {
				log.Print(err)
			}

			id := queryID(n)
			if id == "" {
				continue
			}
			ok, err = hasAnnounce(db, id)
			if err != nil {
				log.Print(err)
			} else if !ok {
				count++
				ann := Announce{ID: id, Fetched: time.Now()}
				ann.Date = queryDate(n)
				ann.PlaceID = placeID
				ann.Price = queryPrice(n)
				ann.Title = queryTitle(n)
				err := insertAnnounce(db, ann)
				if err != nil {
					log.Print(err)
				}
			}
		}

		if count > 0 {
			// TODO: Notify by email?
			log.Printf("Number of new announces fetched:\t%d\n", count)
		}
		time.Sleep(5 * time.Second)
	}
}

func serveHTTP(w http.ResponseWriter, r *http.Request) {
	var ann []Announce
	var departments []string
	places := make(map[int]Place)

	q := map[string][]string(r.URL.Query())
	placeIDsQuery := q["placeID"]
	departmentsQuery := q["department"]

	if placeIDsQuery != nil {
		departments := make(map[string]struct{})
		for _, placeID := range placeIDsQuery {
			placeID, err := strconv.Atoi(placeID)
			if err != nil {
				log.Print(err)
			}
			dpt, err := selectDepartmentWhereID(db, placeID)
			if err != nil {
				log.Print(err)
			}
			departments[dpt] = struct{}{}
			newAnn, err := selectAnnouncesWherePlaceID(db, placeID)
			if err != nil {
				log.Print(err)
			}
			ann = append(ann, newAnn...)
		}
		for dpt := range departments {
			departmentPlaces, err := selectPlacesWhereDepartment(db, dpt)
			if err != nil {
				log.Print(err)
			}
			for id, place := range departmentPlaces {
				places[id] = place
			}
		}
	} else if departmentsQuery != nil {
		for _, department := range departmentsQuery {
			departmentPlaces, err := selectPlacesWhereDepartment(db, department)
			if err != nil {
				log.Print(err)
			}
			for id, place := range departmentPlaces {
				places[id] = place
				newAnn, err := selectAnnouncesWherePlaceID(db, id)
				if err != nil {
					log.Print(err)
				}
				ann = append(ann, newAnn...)
			}
		}
	} else {
		var err error
		ann, err = selectAnnounces(db)
		if err != nil {
			log.Print(err)
		}
		departments, err = selectDistinctDepartmentFromPlaces(db)
		if err != nil {
			log.Print(err)
		}
		sort.Strings(departments)
		places, err = selectPlaces(db)
		if err != nil {
			log.Print(err)
		}
	}

	sort.Sort(ByDate(ann))
	if len(ann) > 35 {
		ann = ann[:35]
	}

	data := struct {
		Announces   []Announce
		Places      map[int]Place
		Departments []string
	}{ann, places, departments}
	t := template.Must(template.ParseFiles("template.html"))
	err := t.Execute(w, data)
	if err != nil {
		log.Print(err)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}
	log.Printf("Listening on port %v", port)

	go poll()
	log.Fatal(http.ListenAndServe(":"+port, http.HandlerFunc(serveHTTP)))
}
