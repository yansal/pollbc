package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/yansal/pollbc/models"
)

func init() {
	models.InitDB(os.Getenv("DATABASE_URL"))
	var err error
	err = models.CreateTableAnnounces()
	if err != nil {
		log.Print(err)
	}

	err = models.CreateTablePlaces()
	if err != nil {
		log.Print(err)
	}
}

var paris *time.Location

func init() {
	var err error
	paris, err = time.LoadLocation("Europe/Paris")
	if err != nil {
		log.Fatal(err)
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

			ok, err := models.HasPlace(place)
			if err != nil {
				log.Print(err)
			} else if !ok {
				err := models.InsertPlace(place)
				if err != nil {
					log.Print(err)
				}
			}
			placeID, err := models.SelectIDFromPlaces(place)
			if err != nil {
				log.Print(err)
			}

			id := queryID(n)
			if id == "" {
				continue
			}
			ok, err = models.HasAnnounce(id)
			if err != nil {
				log.Print(err)
			} else if !ok {
				count++
				ann := models.Announce{ID: id, Fetched: time.Now().In(paris)}
				ann.Date = queryDate(n)
				ann.PlaceID = placeID
				ann.Price = queryPrice(n)
				ann.Title = queryTitle(n)
				err := models.InsertAnnounce(ann)
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
	var ann []models.Announce
	var departments []string
	places := make(map[int]models.Place)

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
			dpt, err := models.SelectDepartmentWhereID(placeID)
			if err != nil {
				log.Print(err)
			}
			departments[dpt] = struct{}{}
			newAnn, err := models.SelectAnnouncesWherePlaceID(placeID)
			if err != nil {
				log.Print(err)
			}
			ann = append(ann, newAnn...)
		}
		for dpt := range departments {
			departmentPlaces, err := models.SelectPlacesWhereDepartment(dpt)
			if err != nil {
				log.Print(err)
			}
			for id, place := range departmentPlaces {
				places[id] = place
			}
		}
	} else if departmentsQuery != nil {
		for _, department := range departmentsQuery {
			departmentPlaces, err := models.SelectPlacesWhereDepartment(department)
			if err != nil {
				log.Print(err)
			}
			for id, place := range departmentPlaces {
				places[id] = place
				newAnn, err := models.SelectAnnouncesWherePlaceID(id)
				if err != nil {
					log.Print(err)
				}
				ann = append(ann, newAnn...)
			}
		}
	} else {
		var err error
		ann, err = models.SelectAnnounces()
		if err != nil {
			log.Print(err)
		}
		departments, err = models.SelectDistinctDepartmentFromPlaces()
		if err != nil {
			log.Print(err)
		}
		sort.Strings(departments)
		places, err = models.SelectPlaces()
		if err != nil {
			log.Print(err)
		}
	}

	sort.Sort(models.ByDate(ann))
	if len(ann) > 35 {
		ann = ann[:35]
	}

	data := struct {
		Announces   []models.Announce
		Places      map[int]models.Place
		Departments []string
		Location    *time.Location
	}{ann, places, departments, paris}
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
