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
	err = models.CreateTableDepartements()
	if err != nil {
		log.Print(err)
	}
	err = models.CreateTablePlaces()
	if err != nil {
		log.Print(err)
	}
	err = models.CreateTableAnnounces()
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
			place, dpt, err := queryPlace(n)
			if err != nil {
				log.Print(err)
				continue
			}

			var ok bool
			ok, err = models.HasDepartment(dpt)
			if err != nil {
				log.Print(err)
			} else if !ok {
				err := models.InsertDepartment(dpt)
				if err != nil {
					log.Print(err)
				}
			}
			dptID, err := models.SelectIDFromDepartment(dpt)
			if err != nil {
				log.Print(err)
			}
			place.DepartmentID = dptID

			ok, err = models.HasPlace(place)
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

func deleteOldAnnounces() {
	for {
		deleted, err := models.DeleteAnnounces()
		if err != nil {
			log.Print(err)
		}
		if deleted != 0 {
			log.Printf("Number of old announces deleted:\t%d\n", deleted)
		}
		time.Sleep(time.Minute)
	}
}

func serveHTTP(w http.ResponseWriter, r *http.Request) {
	var ann []models.Announce
	var departments []models.Department
	var places []models.Place
	dptMap := make(map[int]models.Department)
	placesMap := make(map[int]models.Place)
	printDpts := false

	q := map[string][]string(r.URL.Query())
	placeIDsQuery := q["placeID"]
	departmentIDsQuery := q["departmentID"]

	if placeIDsQuery != nil {
		for _, placeID := range placeIDsQuery {
			placeID, err := strconv.Atoi(placeID)
			if err != nil {
				log.Print(err)
			}
			dptID, err := models.SelectDepartmentIDWhereID(placeID)
			if err != nil {
				log.Print(err)
			}
			dpt, err := models.SelectDepartmentWhereID(dptID)
			if err != nil {
				log.Print(err)
			}
			departments = append(departments, dpt)
			newAnn, err := models.SelectAnnouncesWherePlaceID(placeID)
			if err != nil {
				log.Print(err)
			}
			ann = append(ann, newAnn...)
		}
		for _, dpt := range departments {
			departmentPlaces, err := models.SelectPlacesWhereDepartmentID(dpt.ID)
			if err != nil {
				log.Print(err)
			}
			for _, place := range departmentPlaces {
				places = append(places, place)
			}
		}
	} else if departmentIDsQuery != nil {
		for _, dptID := range departmentIDsQuery {
			dptID, err := strconv.Atoi(dptID)
			if err != nil {
				log.Print(err)
			}
			dpt, err := models.SelectDepartmentWhereID(dptID)
			if err != nil {
				log.Print(err)
			}
			departments = append(departments, dpt)
			places, err = models.SelectPlacesWhereDepartmentID(dptID)
			if err != nil {
				log.Print(err)
			}
			for _, place := range places {
				newAnn, err := models.SelectAnnouncesWherePlaceID(place.ID)
				if err != nil {
					log.Print(err)
				}
				ann = append(ann, newAnn...)
			}
		}
	} else {
		printDpts = true
		var err error
		ann, err = models.SelectAnnounces()
		if err != nil {
			log.Print(err)
		}
		departments, err = models.SelectDepartments()
		if err != nil {
			log.Print(err)
		}
		sort.Sort(models.ByName(departments))
		places, err = models.SelectPlaces()
		if err != nil {
			log.Print(err)
		}
	}

	sort.Sort(models.ByDate(ann))
	if len(ann) > 35 {
		ann = ann[:35]
	}

	if places[0].City != "" {
		sort.Sort(models.ByCity(places))
	} else {
		sort.Sort(models.ByArrondissement(places))
	}
	for _, d := range departments {
		dptMap[d.ID] = d
	}
	for _, p := range places {
		placesMap[p.ID] = p
	}

	data := struct {
		Departments []models.Department
		Places      []models.Place
		Announces   []models.Announce
		DptMap      map[int]models.Department
		PlaceMap    map[int]models.Place
		Location    *time.Location
		PrintDpts   bool
	}{departments, places, ann, dptMap, placesMap, paris, printDpts}
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
	go deleteOldAnnounces()

	http.Handle("/css/", http.FileServer(http.Dir("static")))
	http.Handle("/js/", http.FileServer(http.Dir("static")))
	http.HandleFunc("/", serveHTTP)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
