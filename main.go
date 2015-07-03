package main

import (
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
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

type Data struct {
	Ann    []Announce
	Places []string
}
type Handler struct {
	Data Data
	sync.Mutex
}

func NewHandler() *Handler {
	h := Handler{}
	if _, err := os.Stat("db.json"); err == nil {
		// Load db
		b, err := ioutil.ReadFile("db.json")
		if err != nil {
			log.Fatal(err)
		}
		d := Data{}
		a := make([]Announce, 0)
		p := make([]string, 0)
		d.Ann = a
		d.Places = p
		err = json.Unmarshal(b, &d)
		if err != nil {
			log.Fatal(err)
		}
		h.Data = d
	}

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

		newAnn := make([]Announce, 0)
		for _, n := range nodes {
			id := queryID(n)
			if id == "" || h.hasID(id) {
				continue
			}

			ann := Announce{ID: id, Fetched: time.Now()}
			ann.Date = queryDate(n)
			ann.Place = queryPlace(n)
			ann.Price = queryPrice(n)
			ann.Title = queryTitle(n)
			newAnn = append(newAnn, ann)

			if !h.hasPlace(ann.Place) {
				h.Data.Places = append(h.Data.Places, ann.Place)
			}
		}

		if len(newAnn) > 0 {
			// TODO: Notify by email?
			log.Printf("Number of new announces fetched:\t%d\n", len(newAnn))

			// Save db
			h.Lock()
			h.Data.Ann = append(h.Data.Ann, newAnn...)
			j, err := json.MarshalIndent(h.Data, "", "\t")
			if err != nil {
				log.Print(err)
			}
			err = ioutil.WriteFile("db.json", j, 0600)
			if err != nil {
				log.Print(err)
			}
			h.Unlock()
		}
		time.Sleep(5 * time.Second)
	}
}

func (h *Handler) hasID(id string) bool {
	for _, a := range h.Data.Ann {
		if a.ID == id {
			return true
		}
	}
	return false
}

func (h *Handler) hasPlace(place string) bool {
	for _, p := range h.Data.Places {
		if p == place {
			return true
		}
	}
	return false
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.ParseFiles("template.html"))
	sort.Sort(ByDate(h.Data.Ann))
	sort.Strings(h.Data.Places)
	err := t.Execute(w, h.Data)
	if err != nil {
		log.Print(err)
	}
}

func main() {
	h := NewHandler()
	panic(http.ListenAndServe(":8080", h))
}
