package scraper

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/queue"
)

const (
	threads          = 4
	tableFictionSel  = "body > table.c > tbody"
	tableSciSel      = "body > table > tbody"
	tableNextPageSel = "body > div:nth-child(6) > div:nth-child(2) > a"
	userAgent        = "Mozilla/5.0 (Linux; Android 7.1.1; Moto Z2 Play Build/NPSS26.118-19-1-2; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/86.0.4240.198 Mobile Safari/537.36 [FB_IAB/FB4A;FBAV/296.0.0.44.119;]"
)

//Scrapes nonfiction table
func ScrapeNfTable(urlChan chan string) func(table *colly.HTMLElement) {

	return func(table *colly.HTMLElement) {

		table.DOM.Children().Each(func(_ int, row *goquery.Selection) {

			row.Children().Each(func(nthChild int, column *goquery.Selection) {

				//the 9th child contains book download link
				if nthChild == 9 {
					link := column.ChildrenFiltered("a")
					if url, ok := link.Attr("href"); ok {
						fmt.Println(url)
						urlChan <- url
					}

				}

			})
		})
		close(urlChan)

	}
}
func ScrapeSciTable(urlChan chan string, c *colly.Collector) func(table *colly.HTMLElement) {

	return func(table *colly.HTMLElement) {
		table.DOM.Children().Each(func(_ int, row *goquery.Selection) {
			row.Children().Each(func(nthChild int, row *goquery.Selection) {
				if nthChild == 0 {
					td := row.Siblings()
					td.Each(func(nthChild int, column *goquery.Selection) {
						if nthChild == 3 {
							if url, ok := column.Children().Children().Next().Children().Attr("href"); ok {

								urlChan <- url

							}
						}

					})
				}

			})
		})

	}
}

func fmtURLNf(title string) string {
	searchParam := url.QueryEscape(strings.Trim(title, " "))
	return fmt.Sprintf("https://libgen.is/search.php?&sort=year&req=%v&res=100", searchParam)
}
func fmtURLSci(title string) string {
	searchParam := url.QueryEscape(strings.Trim(title, " "))
	return fmt.Sprintf("https://libgen.is/scimag/?q=%v", searchParam)
}
func createCollectors(size int) []*colly.Collector {
	var clones = make([]*colly.Collector, size)
	clones[0] = colly.NewCollector()

	for i := 1; i < size; i++ {
		clones[i] = clones[0].Clone()
	}

	for i := 0; i < size; i++ {
		//sometimes site can have slow downloads
		clones[i].SetRequestTimeout(time.Second * 60)

		clones[i].Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 5, Delay: 6 * time.Second})
		clones[i].OnError(func(r *colly.Response, err error) {
			fmt.Println(err.Error(), r.StatusCode)
		})
		clones[i].UserAgent = userAgent
	}
	return clones
}
func Scrape(title string, output string, science bool, mirrors []bool) {
	colly.Async(true)

	if !science {
		// putting 100 here because the maximum amount of books on the first page is 100
		urlChan := make(chan string, 100)
		// first getting the table data links title etc
		collectors := createCollectors(3)
		URL := fmtURLNf(title)
		fmt.Println("searching for nonfiction books")

		collectors[0].OnHTML(tableFictionSel, ScrapeNfTable(urlChan))
		collectors[0].Visit(URL)

		collectors[0].Wait()
		//adding that to the queue
		queue, _ := queue.New(threads, &queue.InMemoryQueueStorage{MaxSize: 10000})
		for url := range urlChan {
			fmt.Printf("queuing %v\n", url)
			queue.AddURL(url)
		}
		for idx, mirror := range mirrors {
			//Iterates through mirrors and selects and downloads depending on index
			if mirror {
				selector := fmt.Sprintf("#download > ul > li:nth-child(%v) > a", idx+1)
				collectors[1].OnHTML(
					selector,
					func(e *colly.HTMLElement) {

						URL := e.Request.AbsoluteURL(e.Attr("href"))
						collectors[2].Visit(URL)

					})
			}
		}
		collectors[2].OnResponse(func(r *colly.Response) {
			file := fmt.Sprintf("%v/%v", output, r.FileName())
			r.Save(file)
			fmt.Printf("[✅] 📖 nonfiction book saved -> %v\n", file)

		})
		queue.Run(collectors[1])
		collectors[1].Wait()
		collectors[2].Wait()
	} else {
		urlChan := make(chan string, 100)
		// first getting the table data links title etc
		collectors := createCollectors(3)
		fmt.Println("searching for scientific articles")
		URL := fmtURLSci(title)
		collectors[0].OnHTML(tableSciSel, ScrapeSciTable(urlChan, collectors[0]))
		collectors[0].OnHTML(tableNextPageSel, func(h *colly.HTMLElement) {
			if h.Text == "▶" {
				slug := h.Request.AbsoluteURL(h.Attr("href"))
				URL = fmt.Sprintf("https://libgen.is%v", slug)
				fmt.Printf("collecting links on next page %v\n", URL)
				collectors[0].Visit(URL)
			}

		})
		collectors[1].OnHTML("#main", func(h *colly.HTMLElement) {
			if slugs, ok := h.DOM.Find("tbody > tr:nth-child(1) > td:nth-child(2) > a").Attr("href"); ok {
				URL := fmt.Sprintf("https://libgen.rocks/%v", slugs)
				collectors[2].Visit(URL)
			}
		})
		collectors[2].OnResponse(func(r *colly.Response) {
			file := fmt.Sprintf("%v/%v", output, r.FileName())
			r.Save(file)
			fmt.Printf("[✅] 🧬 📖 Scientific article saved -> %v\n", file)
		})
		collectors[0].Visit(URL)
		collectors[0].Wait()

		close(urlChan)

		queue, _ := queue.New(threads, &queue.InMemoryQueueStorage{MaxSize: 10000})
		count := 0
		for url := range urlChan {
			count++
			queue.AddURL(url)
		}
		fmt.Printf("Downloading %v article(s)", count)
		queue.Run(collectors[1])
		collectors[1].Wait()
		collectors[2].Wait()

	}

}
