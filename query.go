package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
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
	nodes := make([]*html.Node, 0)
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, a := range n.Attr {
				if a.Key == "class" && a.Val == "lbc" {
					nodes = append(nodes, n.Parent)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return nodes
}

func queryID(n *html.Node) string {
	for _, a := range n.Attr {
		if a.Key == "href" {
			return a.Val
		}
	}
	return ""
}

func queryDate(n *html.Node) time.Time {
	var dateNode *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, a := range n.Attr {
				if a.Key == "class" && a.Val == "date" {
					dateNode = n
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)

	date := dateNode.FirstChild.NextSibling.FirstChild.Data
	clock := dateNode.FirstChild.NextSibling.NextSibling.NextSibling.FirstChild.Data

	now := time.Now()
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
			log.Fatal(err)
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
			log.Fatal("Problem parsing the month")
		}

		thisYear, _, _ := now.Date()
		if time.Date(thisYear, mon, d, 0, 0, 0, 0, time.Local).Before(now) {
			y = thisYear
		} else {
			y = thisYear - 1
		}
	}

	split := strings.Split(clock, ":")
	h, err := strconv.Atoi(split[0])
	if err != nil {
		log.Fatal(err)
	}
	min, err := strconv.Atoi(split[1])
	if err != nil {
		log.Fatal(err)
	}

	return time.Date(y, mon, d, h, min, 0, 0, time.Local)
}

func queryPlace(n *html.Node) string {
	var placeNode *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, a := range n.Attr {
				if a.Key == "class" && a.Val == "placement" {
					placeNode = n
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	if placeNode == nil {
		return ""
	}
	return strings.Join(strings.Fields(strings.TrimSpace(placeNode.FirstChild.Data)), " ")
}

func queryPrice(n *html.Node) string {
	var priceNode *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, a := range n.Attr {
				if a.Key == "class" && a.Val == "price" {
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
				if a.Key == "class" && a.Val == "title" {
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
