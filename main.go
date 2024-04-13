package main

import "github.com/mfojtik/go-sofar-prometheus/pkg/scraper"

func main() {
	s := scraper.New("", 1234)
	_, err := s.Scrape()
	if err != nil {
		panic(err)
	}
}
