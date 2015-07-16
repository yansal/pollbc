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

	err = createTableAnnounces(db)
	if err != nil {
		log.Print(err)
	}

	err = createTablePlaces(db)
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

			place := Place{Department: department, City: city, Arrondissement: arrondissement}
			ok, err := hasPlace(h.db, place)
			if err != nil {
				log.Print(err)
			} else if !ok {
				err := insertPlace(h.db, place)
				if err != nil {
					log.Print(err)
				}
			}
			placeID, err := selectIDFromPlaces(h.db, place)
			if err != nil {
				log.Print(err)
			}

			id := queryID(n)
			if id == "" {
				continue
			}
			ok, err = hasAnnounce(h.db, id)
			if err != nil {
				log.Print(err)
			} else if !ok {
				count++
				ann := Announce{ID: id, Fetched: time.Now()}
				ann.Date = queryDate(n)
				ann.PlaceID = placeID
				ann.Price = queryPrice(n)
				ann.Title = queryTitle(n)
				err := insertAnnounce(h.db, ann)
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
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var ann []Announce
	var err error
	placeID := r.URL.Query().Get("placeID")
	if placeID == "" {
		ann, err = selectAnnounces(h.db)
		if err != nil {
			log.Print(err)
		}
	} else {
		placeID, err := strconv.Atoi(placeID)
		if err != nil {
			log.Print(err)
		}
		ann, err = selectAnnouncesWherePlaceID(h.db, placeID)
		if err != nil {
			log.Print(err)
		}
	}

	places := make(map[int]Place)
	for _, a := range ann {
		_, ok := places[a.PlaceID]
		if ok {
			continue
		}
		place, err := selectPlace(h.db, a.PlaceID)
		if err != nil {
			log.Print(err)
			continue
		}
		places[a.PlaceID] = place
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
