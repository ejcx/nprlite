package nprlite

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/mmcdole/gofeed"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
)

const (
	pageTemplate = `<!DOCTYPE html>
<html>
<head>
%s
<style>
.category {
  padding: 0 5px 0 5px;
}
</style>
</head>
<body>
<div>%s</div>
<h3>%s</h3>
<ul>
%s
</ul>
</body>
</html>`

	contentTemplate  = `<li><a target="_blank" href="%s">%s</a></li>`
	categoryTemplate = `%s<a class="category" href="%s">%s</a>`
	titleTemplate    = `<title>%s</title>`
)

var (
	content         string
	currentPolitics string
	categories      string
	fp              = gofeed.NewParser()

	categoryUrls = map[string]Page{
		"/":           Page{Num: "1001", Category: "News", Url: "/"},
		"/politics":   Page{Num: "1014", Category: "Politics", Url: "/politics"},
		"/national":   Page{Num: "1003", Category: "National", Url: "/national"},
		"/education":  Page{Num: "1013", Category: "Education", Url: "/education"},
		"/business":   Page{Num: "1006", Category: "Business", Url: "/business"},
		"/technology": Page{Num: "1019", Category: "Technology", Url: "/technology"},
		"/science":    Page{Num: "1007", Category: "Science", Url: "/science"},
		"/health":     Page{Num: "1128", Category: "Health", Url: "/health"},
	}
)

type Page struct {
	Num      string
	Category string
	Url      string
}

func fetcher(w http.ResponseWriter, r *http.Request, num, category string) {
	currentPage, err := getnews(w, r, num, category)
	if err != nil {
		log.Printf("Could not fetch %s: %s", category, err)
		return
	}
	fmt.Fprint(w, currentPage)
}

func index(w http.ResponseWriter, r *http.Request) {
	page := categoryUrls[r.URL.Path]
	fetcher(w, r, page.Num, page.Category)
}

func getnews(w http.ResponseWriter, r *http.Request, id, category string) (string, error) {
	client := urlfetch.Client(appengine.NewContext(r))
	resp, err := client.Get("https://www.npr.org/rss/rss.php?id=" + id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return "", err
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return "", err
	}
	feed, err := fp.ParseString(string(buf))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return "", err
	}

	contentItems := ""
	for _, item := range feed.Items {
		contentItems += fmt.Sprintf(contentTemplate, item.Link, item.Title)
	}
	title := fmt.Sprintf(titleTemplate, feed.Title)
	currentPage := fmt.Sprintf(pageTemplate, title, categories, category, contentItems)
	return currentPage, nil
}

func init() {
	categoryCount := 0
	for url, page := range categoryUrls {
		seperator := "|"
		if categoryCount == 0 {
			seperator = ""
		}
		// Set up an http handler for this.
		http.HandleFunc(url, index)

		// Set up the category list for the header.
		categories += fmt.Sprintf(categoryTemplate, seperator, url, page.Category)
		categoryCount++
	}
}
