package main

import (
	"fmt"
	"github.com/gocolly/colly"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
)

const (
	host = "https://www.silver.ru"
	lastPageId = 19
	paralLmt = 20
)

type TrackInfo struct {
	title string
	dlink string
}

var dchan = make(chan TrackInfo)

func main()  {
	fList, err := os.Create("dizgo_list.txt")
	if err != nil {
		log.Fatalf("create dizgo list: %v", err)
	}
	defer fList.Close()

	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		log.Printf("visiting %q...", r.URL)
	})

	c.OnError(func(_ *colly.Response, err error) {
		log.Fatalf("something went wrong: %v", err)
	})

	c.OnHTML(".blog td>a[href]", func(e *colly.HTMLElement) {
		if err := c.Visit(e.Request.AbsoluteURL(e.Attr("href"))); err != nil {
			log.Fatalf("visit %q: %v", e.Request.AbsoluteURL(e.Attr("href")), err)
		}
	})

	c.OnHTML(".blog-detail", func(e *colly.HTMLElement) {
		title := e.ChildText(".title>h2")
		titleDec, err := url.QueryUnescape(title)
		if err != nil {
			log.Fatalf("decode title %q: %v", title, err)
		}
		titleDec = strings.ReplaceAll(titleDec, "\u00a0", " ")
		dlink := e.Request.AbsoluteURL(e.ChildAttr("audio", "src"))
		fmt.Fprintf(fList, "%s\t%s\n", titleDec, dlink)

		dchan <- TrackInfo{titleDec, dlink}
	})

	startDload()
	for suffix := 1; suffix <= lastPageId; suffix++ {
		strconv.Itoa(suffix)
		url := fmt.Sprintf("%s/programms/mozcow_dizcow_hi_fi_edition/vipyski-prigrammi/?PAGEN_1=6&PAGEN_1=%d",
			host, suffix)
		if err := c.Visit(url); err != nil {
			log.Fatalf("visit %q: %v", err)
		}
	}

	close(dchan)
}

func startDload() {
	for i := 0; i < paralLmt; i++ {
		go func() {
			for {
				track, ok := <-dchan
				if !ok {
					break
				}
				log.Printf("downloading %q", track.title)
				fTrack, err := os.Create(path.Join("tracks", track.title+".mp3"))
				if err != nil {
					log.Fatalf("create track file: %v", err)
				}
				resp, err := http.Get(track.dlink)
				if err != nil {
					log.Fatalf("download track: %v", err)
				}
				if _, err := io.Copy(fTrack, resp.Body); err != nil {
					log.Fatalf("downloading track: %v", err)
				}
				resp.Body.Close()
			}
		}()
	}
}