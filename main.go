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
	var err error
	q := map[string][]string(r.URL.Query())
	placeIDs, ok := q["placeID"]
	if !ok {
		ann, err = selectAnnounces(db)
		if err != nil {
			log.Print(err)
		}
	} else {
		for _, placeID := range placeIDs {
			placeID, err := strconv.Atoi(placeID)
			if err != nil {
				log.Print(err)
			}
			newAnn, err := selectAnnouncesWherePlaceID(db, placeID)
			if err != nil {
				log.Print(err)
			}
			ann = append(ann, newAnn...)
		}
	}

	places := make(map[int]Place)
	for _, a := range ann {
		_, ok := places[a.PlaceID]
		if ok {
			continue
		}
		place, err := selectPlace(db, a.PlaceID)
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

	go poll()
	log.Fatal(http.ListenAndServe(":"+port, http.HandlerFunc(serveHTTP)))
}
