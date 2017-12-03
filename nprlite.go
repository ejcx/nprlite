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
	titleTemplate    = `<title>%s Lite</title>`
	seperator        = "|"
)

var (
	content         string
	currentPolitics string
	categories      string
	fp              = gofeed.NewParser()

	categoryUrls = map[string]Page{
		"1014": Page{Num: "1014", Category: "Politics", Url: "/politics"},
		"1003": Page{Num: "1003", Category: "National", Url: "/national"},
		"1013": Page{Num: "1013", Category: "Education", Url: "/education"},
		"1006": Page{Num: "1006", Category: "Business", Url: "/business"},
		"1019": Page{Num: "1019", Category: "Technology", Url: "/technology"},
		"1007": Page{Num: "1007", Category: "Science", Url: "/science"},
		"1128": Page{Num: "1128", Category: "Health", Url: "/health"},
		"1001": Page{Num: "1001", Category: "Top Stories", Url: "/"},
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
	for num, page := range categoryUrls {
		if page.Url != r.URL.Path {
			continue
		}
		fetcher(w, r, num, page.Category)
	}
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
	// Manually add the top stories to the front.
	categories += fmt.Sprintf(categoryTemplate, "", "/", "Top Stories")
	for _, page := range categoryUrls {
		// Set up an http handler for this.
		http.HandleFunc(page.Url, index)

		// Skip adding top stories to the header.
		if page.Url == "/" {
			continue
		}
		// Set up the category list for the header.
		categories += fmt.Sprintf(categoryTemplate, seperator, page.Url, page.Category)
	}
}
