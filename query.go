package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/yansal/pollbc/Godeps/_workspace/src/golang.org/x/net/html"
	"github.com/yansal/pollbc/Godeps/_workspace/src/golang.org/x/net/html/charset"
	"github.com/yansal/pollbc/models"
)

func fetch() (*html.Node, error) {
	r, err := http.Get("http://www.leboncoin.fr/colocations/offres/ile_de_france")
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	contentType := r.Header.Get("Content-Type")
	reader, err := charset.NewReader(r.Body, contentType)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(reader)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func queryAnnounces(doc *html.Node) []*html.Node {
	var nodes []*html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "class" && a.Val == "list_item clearfix trackable" {
					nodes = append(nodes, n.Parent)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	if len(nodes) == 0 {
		log.Print("queryAnnounces: len(nodes) == 0")
	}
	return nodes
}

func queryURL(n *html.Node) (string, error) {
	for _, a := range n.FirstChild.NextSibling.Attr {
		if a.Key == "href" {
			return "http:" + a.Val, nil
		}
	}
	return "", errors.New("Can't find href in html node")
}

func queryDate(n *html.Node) (time.Time, error) {
	var dateNode *html.Node
	var f func(*html.Node)
	count := 0
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "p" {
			for _, a := range n.Attr {
				if a.Key == "class" && a.Val == "item_supp" {
					count++
					if count == 3 {
						dateNode = n
						return
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)

	split := strings.Split(strings.TrimSpace(dateNode.FirstChild.Data), ", ")
	if len(split) != 2 {
		return time.Time{}, fmt.Errorf("date is %+v", split)
	}
	date := split[0]
	clock := split[1]

	now := time.Now().In(paris)
	var y, d int
	var mon time.Month
	switch date {
	case "Aujourd'hui":
		y, mon, d = now.Date()
	case "Hier":
		y, mon, d = now.AddDate(0, 0, -1).Date()
	default:
		split := strings.Split(date, " ")
		var err error
		d, err = strconv.Atoi(split[0])
		if err != nil {
			return time.Time{}, err
		}

		switch split[1] {
		case "janvier":
			mon = time.January
		case "fevrier":
			mon = time.February
		case "mars":
			mon = time.March
		case "avril":
			mon = time.April
		case "mai":
			mon = time.May
		case "juin":
			mon = time.June
		case "juillet":
			mon = time.July
		case "aout":
			mon = time.August
		case "septembre":
			mon = time.September
		case "octobre":
			mon = time.October
		case "novembre":
			mon = time.November
		case "decembre":
			mon = time.December
		default:
			return time.Time{}, errors.New("Problem parsing the month")
		}

		thisYear, _, _ := now.Date()
		if time.Date(thisYear, mon, d, 0, 0, 0, 0, time.Local).Before(now) {
			y = thisYear
		} else {
			y = thisYear - 1
		}
	}

	split = strings.Split(clock, ":")
	h, err := strconv.Atoi(split[0])
	if err != nil {
		return time.Time{}, err
	}
	min, err := strconv.Atoi(split[1])
	if err != nil {
		return time.Time{}, err
	}

	return time.Date(y, mon, d, h, min, 0, 0, time.Local), nil
}

func queryPlace(n *html.Node) (models.Place, models.Department, error) {
	var placeNode *html.Node
	var f func(*html.Node)
	count := 0
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "p" {
			for _, a := range n.Attr {
				if a.Key == "class" && a.Val == "item_supp" {
					count++
					if count == 2 {
						placeNode = n
						return
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	place := models.Place{}
	dpt := models.Department{}
	if placeNode == nil {
		// TODO render node
		return place, dpt, errors.New("Can't find place in html node")
	}

	placeString := strings.Join(strings.Fields(strings.TrimSpace(placeNode.FirstChild.Data)), " ")
	split := strings.Split(placeString, "/")
	switch len(split) {
	case 1:
		fields := strings.Fields(split[0])
		switch len(fields) {
		case 1:
			dpt.Name = fields[0]
		case 2:
			dpt.Name = fields[0]
			place.Arrondissement = fields[1]
		default:
			return place, dpt, fmt.Errorf("queryPlace: can't parse %v", fields)
		}
	case 2:
		place.City = strings.TrimSpace(split[0])
		dpt.Name = strings.TrimSpace(split[1])
		if place.City == "" {
			return place, dpt, fmt.Errorf("queryPlace: city is null string in %v", split)
		}
		if dpt.Name == "" {
			return place, dpt, fmt.Errorf("queryPlace: department is null string in %v", split)
		}
	default:
		return place, dpt, fmt.Errorf("queryPlace: can't parse %v", split)
	}
	return place, dpt, nil
}

func queryPrice(n *html.Node) string {
	var priceNode *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "h3" {
			for _, a := range n.Attr {
				if a.Key == "class" && a.Val == "item_price" {
					priceNode = n
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	if priceNode == nil {
		return ""
	}
	return strings.TrimSpace(priceNode.FirstChild.Data)
}

func queryTitle(n *html.Node) string {
	var titleNode *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "h2" {
			for _, a := range n.Attr {
				if a.Key == "class" && a.Val == "item_title" {
					titleNode = n
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return strings.TrimSpace(titleNode.FirstChild.Data)
}
