package main

import (
	"bget/internal/scraper"
	"flag"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func main() {

	var (
		output     string
		title      string
		cloudflare bool
		ipfs       bool
		infura     bool
		piniata    bool
	)
	flag.StringVar(&title, "t", "", "-t \"Some Book Title\"")
	flag.StringVar(&output, "o", "", "-o \"Some Output Directory\"")
	flag.BoolVar(&cloudflare, "cloudflare", false, "cloudflare mirror")
	flag.BoolVar(&ipfs, "ipfs", false, "ipfs mirrror")
	flag.BoolVar(&infura, "infura", false, "infura mirrror")
	flag.BoolVar(&piniata, "pinata", false, "pinata mirrror")
	flag.Parse()

	var withMirrors = []bool{
		//important that they are in the order of nth(idx) css child selectors
		cloudflare,
		ipfs,
		infura,
		piniata,
	}
	count := 0

	for i := 0; i < len(withMirrors); i++ {
		if withMirrors[i] {
			count++
		}
	}
	if title == "" {
		log.Fatal("No title inputed")

	}
	if count == 0 {
		log.Fatal("Select a mirror")
	}
	if strings.Contains(output, "~") {
		if homeDir, err := os.UserHomeDir(); err != nil {
			log.Fatal(err)
		} else {
			output = strings.Replace(output, "~", homeDir, 1)

		}
	}
	if !path.IsAbs(output) {
		if absp, err := filepath.Abs(output); err != nil {
			log.Fatal(err)
		} else {
			output = absp
		}
	}
	scraper.Scrape(title, output, withMirrors)
}
