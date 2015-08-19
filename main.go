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
			dptPK, err := models.SelectPKFromDepartment(dpt)
			if err != nil {
				log.Print(err)
			}
			place.DepartmentPK = dptPK

			ok, err = models.HasPlace(place)
			if err != nil {
				log.Print(err)
			} else if !ok {
				err := models.InsertPlace(place)
				if err != nil {
					log.Print(err)
				}
			}
			placePK, err := models.SelectPKFromPlaces(place)
			if err != nil {
				log.Print(err)
			}

			url, err := queryURL(n)
			if err != nil {
				log.Print(err)
				continue
			}
			ok, err = models.HasAnnounce(url)
			if err != nil {
				log.Print(err)
			} else if !ok {
				count++
				ann := models.Announce{URL: url, Fetched: time.Now().In(paris)}
				ann.Date, err = queryDate(n)
				if err != nil {
					log.Print(err)
					continue
				}
				ann.PlacePK = placePK
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
	placePKsQuery := q["placePK"]
	departmentPKsQuery := q["departmentPK"]

	if placePKsQuery != nil {
		for _, placePK := range placePKsQuery {
			placePK, err := strconv.Atoi(placePK)
			if err != nil {
				log.Print(err)
			}
			dptPK, err := models.SelectDepartmentPKWherePK(placePK)
			if err != nil {
				log.Print(err)
			}
			dpt, err := models.SelectDepartmentWherePK(dptPK)
			if err != nil {
				log.Print(err)
			}
			departments = append(departments, dpt)
			newAnn, err := models.SelectAnnouncesWherePlacePK(placePK)
			if err != nil {
				log.Print(err)
			}
			ann = append(ann, newAnn...)
		}
		for _, dpt := range departments {
			departmentPlaces, err := models.SelectPlacesWhereDepartmentPK(dpt.PK)
			if err != nil {
				log.Print(err)
			}
			for _, place := range departmentPlaces {
				places = append(places, place)
			}
		}
		if places[0].City != "" {
			sort.Sort(models.ByCity(places))
		} else {
			sort.Sort(models.ByArrondissement(places))
		}
	} else if departmentPKsQuery != nil {
		for _, dptPK := range departmentPKsQuery {
			dptPK, err := strconv.Atoi(dptPK)
			if err != nil {
				log.Print(err)
			}
			dpt, err := models.SelectDepartmentWherePK(dptPK)
			if err != nil {
				log.Print(err)
			}
			departments = append(departments, dpt)
			places, err = models.SelectPlacesWhereDepartmentPK(dptPK)
			if err != nil {
				log.Print(err)
			}
			newAnn, err := models.SelectAnnouncesWhereDepartmentPK(dptPK)
			if err != nil {
				log.Print(err)
			}
			ann = append(ann, newAnn...)
		}
		if places[0].City != "" {
			sort.Sort(models.ByCity(places))
		} else {
			sort.Sort(models.ByArrondissement(places))
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

	for _, d := range departments {
		dptMap[d.PK] = d
	}
	for _, p := range places {
		placesMap[p.PK] = p
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
