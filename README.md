# bget
library genesis web scraper first 100 results

## Flags
  * **Science (optional)**
    
     * ```-science``` - scrapes scientific articles
  
  * **Mirrors (required at least one)**
    
    * ```-cloudflare``` - uses cloudflare download link
    * ```-ipfs``` - uses ipfs download link
    * ```-infura``` - uses infura download link
    * ```-pinata``` - uses pinata download link
  
  * **Output (optional)**
    
     * ```-o /your/output/dir``` - where all downloaded files will be saved or in working directory if not specified
  
  * **Title**

    * ```-t "Any book title"``` - searches and scrapes the first 100 results of book title

### Example
```bash
go run main.go -t "some book title" -cloudflare -o /output/directory/
```
