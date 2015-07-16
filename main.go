package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	_ "github.com/yansal/pollbc/Godeps/_workspace/src/github.com/lib/pq"
)

type Place struct {
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

type Handler struct {
	db *sql.DB
}

func NewHandler() *Handler {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("CREATE TABLE pollbc_announces (id varchar PRIMARY KEY, date timestamp with time zone, price varchar, placeID serial, title varchar, fetched timestamp with time zone)")
	if err != nil {
		log.Print(err)
	}

	_, err = db.Exec("CREATE TABLE pollbc_place (id serial PRIMARY KEY, city varchar, department varchar, arrondissement varchar)")
	if err != nil {
		log.Print(err)
	}

	h := Handler{db: db}
	go h.poll()
	return &h
}

func (h *Handler) poll() {
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
			department, city, arrondissement := queryPlace(n)
			if department == "" && arrondissement == "" {
				log.Print("error: department and arrondissement are both null string")
				continue
			}

			var placeID int
			var err error
			err = h.db.QueryRow("SELECT id FROM pollbc_place WHERE city=$1 AND department=$2 AND arrondissement=$3",
				city, department, arrondissement).Scan(&placeID)
			if err == sql.ErrNoRows {
				_, err = h.db.Exec("INSERT INTO pollbc_place (city, department, arrondissement) VALUES ($1, $2, $3)", city, department, arrondissement)
				if err != nil {
					log.Print(err)
				}
				err = h.db.QueryRow("SELECT id FROM pollbc_place WHERE city=$1 AND department=$2", city, department).Scan(&placeID)
				if err != nil {
					log.Print(err)
				}
			} else if err != nil {
				log.Print(err)
			}

			id := queryID(n)
			if id == "" {
				continue
			}
			var tmp string
			err = h.db.QueryRow("SELECT id FROM pollbc_announces WHERE id=$1", id).Scan(&tmp)
			if err != nil && err != sql.ErrNoRows {
				log.Print(err)
			}
			if err == nil {
				continue
			}
			count++

			ann := Announce{ID: id, Fetched: time.Now()}
			ann.Date = queryDate(n)
			ann.PlaceID = placeID
			ann.Price = queryPrice(n)
			ann.Title = queryTitle(n)
			_, err = h.db.Exec("INSERT INTO pollbc_announces (id, date, price, placeID, title, fetched) VALUES ($1, $2, $3, $4, $5, $6)",
				ann.ID, ann.Date, ann.Price, ann.PlaceID, ann.Title, ann.Fetched)
			if err != nil {
				log.Print(err)
			}
		}

		if count > 0 {
			// TODO: Notify by email?
			log.Printf("Number of new announces fetched:\t%d\n", count)
		}
		time.Sleep(5 * time.Second)
	}
}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ann := make([]Announce, 0)
	places := make(map[int]Place)
	rows, err := h.db.Query("SELECT * FROM pollbc_announces")
	for rows.Next() {
		var id, price, title string
		var placeID int
		var date, fetched time.Time
		err := rows.Scan(&id, &date, &price, &placeID, &title, &fetched)
		if err != nil {
			log.Print(err)
		}
		if _, ok := places[placeID]; !ok {
			var id int
			var city, department, arrondissement string
			err = h.db.QueryRow("SELECT * FROM pollbc_place WHERE id=$1", placeID).Scan(&id, &city, &department, &arrondissement)
			if err != nil {
				log.Print(err)
			}
			places[id] = Place{City: city, Department: department, Arrondissement: arrondissement}
		}

		ann = append(ann, Announce{ID: id, Date: date, Price: price, PlaceID: placeID, Title: title, Fetched: fetched})
	}
	sort.Sort(ByDate(ann))

	data := struct {
		Announces []Announce
		Places    map[int]Place
	}{ann, places}
	t := template.Must(template.ParseFiles("template.html"))
	err = t.Execute(w, data)
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

	h := NewHandler()
	log.Fatal(http.ListenAndServe(":"+port, h))
}
