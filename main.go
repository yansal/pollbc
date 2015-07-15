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

type Announce struct {
	ID    string
	Date  time.Time
	Place string
	Price string
	Title string

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
	db, err := sql.Open("postgres", "")
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("CREATE TABLE pollbc_announces (id varchar PRIMARY KEY, date timestamp with time zone, price varchar, place varchar, title varchar, fetched timestamp with time zone)")
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
			id := queryID(n)
			if id == "" {
				continue
			}
			var tmp string
			err := h.db.QueryRow("SELECT id FROM pollbc_announces WHERE id=$1", id).Scan(&tmp)
			if err != nil && err != sql.ErrNoRows {
				log.Print(err)
			}
			if err == nil {
				continue
			}
			count++

			ann := Announce{ID: id, Fetched: time.Now()}
			ann.Date = queryDate(n)
			ann.Place = queryPlace(n)
			ann.Price = queryPrice(n)
			ann.Title = queryTitle(n)
			_, err = h.db.Exec("INSERT INTO pollbc_announces (id, date, price, place, title, fetched) VALUES ($1, $2, $3, $4, $5, $6)",
				ann.ID, ann.Date, ann.Price, ann.Place, ann.Title, ann.Fetched)
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
	t := template.Must(template.ParseFiles("template.html"))
	ann := make([]Announce, 0)
	rows, err := h.db.Query("SELECT * FROM pollbc_announces")
	for rows.Next() {
		var id, price, place, title string
		var date, fetched time.Time
		err := rows.Scan(&id, &date, &price, &place, &title, &fetched)
		if err != nil {
			log.Print(err)
		}

		ann = append(ann, Announce{ID: id, Date: date, Price: price, Place: place, Title: title, Fetched: fetched})
	}

	sort.Sort(ByDate(ann))
	err = t.Execute(w, ann)
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
