package scraper

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/queue"
)

const (
	maxBooks = 100
	threads  = 4
	//selects the top of the table
	tableSel  = "body > table.c > tbody"
	userAgent = "Mozilla/5.0 (Linux; Android 7.1.1; Moto Z2 Play Build/NPSS26.118-19-1-2; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/86.0.4240.198 Mobile Safari/537.36 [FB_IAB/FB4A;FBAV/296.0.0.44.119;]"
)

func downloadFile(wg *sync.WaitGroup, name string, output string, url string) {
	wg.Add(1)

	if output[len(output)-1:] != "/" {
		name = fmt.Sprintf("%v/%v", output, name)
	} else {
		name = fmt.Sprintf("%v%v", output, name)
	}
	description := fmt.Sprintf("✅)[ 📖 ] -> %v", name)

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	// Create the file
	out, err := os.Create(name)
	if err != nil {
		fmt.Println(err)
	}
	defer func() {
		resp.Body.Close()
		out.Close()
		wg.Done()
	}()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	fmt.Println(description)
}
func createTableScraperCB(urlChan chan string) func(table *colly.HTMLElement) {

	return func(table *colly.HTMLElement) {

		table.DOM.Children().Each(func(i int, column *goquery.Selection) {

			column.Children().Each(func(nthChild int, row *goquery.Selection) {

				//the 9th child contains book download link
				if nthChild == 9 {
					link := row.ChildrenFiltered("a")
					if url, ok := link.Attr("href"); ok {
						urlChan <- url
					}

				}

			})
		})
		close(urlChan)

	}
}

func fmtURL(title string) string {
	searchParam := url.QueryEscape(strings.Trim(title, " "))
	return fmt.Sprintf("https://libgen.is/search.php?&sort=year&req=%v&res=100", searchParam)
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

		clones[i].Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 2, Delay: 4 * time.Second})
		clones[i].OnError(func(r *colly.Response, err error) {
			fmt.Println(err.Error())
		})
		clones[i].UserAgent = userAgent
	}
	return clones
}
func Scrape(title string, output string, mirrors []bool) {
	var downloadWG sync.WaitGroup
	colly.Async(true)
	URL := fmtURL(title)
	urlChan := make(chan string, maxBooks)
	// first getting the table data links title etc
	collectors := createCollectors(2)

	collectors[0].OnHTML(tableSel, createTableScraperCB(urlChan))
	collectors[0].Visit(URL)

	collectors[0].Wait()
	//adding that to the queue
	queue, _ := queue.New(threads, &queue.InMemoryQueueStorage{MaxSize: 10000})
	for url := range urlChan {
		fmt.Printf("queuing %v\n", url)
		queue.AddURL(url)
	}
	for idx, mirror := range mirrors {
		if mirror {
			selector := fmt.Sprintf("#download > ul > li:nth-child(%v) > a", idx+1)
			collectors[1].OnHTML(
				selector,
				func(e *colly.HTMLElement) {

					URL := e.Request.AbsoluteURL(e.Attr("href"))
					sURL := strings.Split(URL, "?")
					sLen := len(sURL)
					//parsing and getting the query
					var query string = ""
					if sLen > 1 {
						// disregard first element
						for i := 1; i < len(sURL); i++ {
							if strings.Contains(sURL[i], "filename") {
								query = sURL[i]
								break
							}
						}
					}
					if query != "" {
						qMap, _ := url.ParseQuery(query)
						filename := qMap.Get("filename")
						go downloadFile(&downloadWG, filename, output, URL)
					}
				})
		}
	}

	queue.Run(collectors[1])
	collectors[1].Wait()
	downloadWG.Wait()
}
