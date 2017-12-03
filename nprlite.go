package nprlite

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mmcdole/gofeed"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
)

const (
	styleTemplate = `
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta http-equiv="X-UA-Compatible" content="ie=edge">
<style>
  body {
	max-width: 650px;
	margin: 2em auto 4em;
	padding: 0 1rem;
	line-height: 1.5;
	font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol";
	-webkit-font-smoothing: antialiased;
}

img {
	max-width: 100%;
	height: auto;
}

.categories {
	word-break: break-word;
}

.category {
  padding: 0 5px 0 5px;
}
</style>`

	pageTemplate = `<!DOCTYPE html>
<html>
<head>
%s
%s
</head>
<body>
<div class="categories">%s</div>
<h3>%s</h3>
<ul>
%s
</ul>
</body>
</html>`

	articleTemplate = `<!DOCTYPE html>
<html>
<head>
NPR News
%s
</head>
<body>
<div>%s</div>
</body>
</html>`

	contentTemplate      = `<li><a href="%s">%s</a></li>`
	articleEntryTemplate = `<p>%s</p>`
	categoryTemplate     = `%s<a class="category" href="%s">%s</a>`
	titleTemplate        = `<title>%s Lite</title>`
	seperator            = "|"
)

var (
	NOPAGE          = errors.New("NOPAGE")
	content         string
	currentPolitics string
	categories      string
	fp              = gofeed.NewParser()

	categoryUrls = []Page{
		Page{Num: "1014", Category: "Politics", Url: "/politics"},
		Page{Num: "1003", Category: "National", Url: "/national"},
		Page{Num: "1013", Category: "Education", Url: "/education"},
		Page{Num: "1006", Category: "Business", Url: "/business"},
		Page{Num: "1019", Category: "Technology", Url: "/technology"},
		Page{Num: "1007", Category: "Science", Url: "/science"},
		Page{Num: "1128", Category: "Health", Url: "/health"},
		Page{Num: "1001", Category: "Headlines", Url: "/"},
	}
)

type Page struct {
	Num      string
	Category string
	Url      string
}

type Error string

func (e Error) Error() string { return string(e) }

func fetcher(w http.ResponseWriter, r *http.Request, num, category string) {
	currentPage, err := getnews(w, r, num, category)
	if err != nil {
		log.Printf("Could not fetch %s: %s", category, err)
		return
	}
	fmt.Fprint(w, currentPage)
}

func story(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		err := errors.New("Invalid story id")
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	article, err := getarticle(w, r, id)
	if err != nil {
		err := errors.New("Couldn't fetch story")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if article == "" {
		return
	}
	fmt.Fprintf(w, articleTemplate, styleTemplate, article)
}

func index(w http.ResponseWriter, r *http.Request) {
	for _, page := range categoryUrls {
		if page.Url != r.URL.Path {
			continue
		}
		fetcher(w, r, page.Num, page.Category)
	}
}

func parsearticle(body io.Reader) (string, error) {
	buf, err := ioutil.ReadAll(body)
	if err != nil {
		return "", err
	}
	pieces := strings.Split(string(buf), "<p><a href=\"/\">Home</a></p>")
	if len(pieces) != 2 {
		return "", nil
	}
	articleFooter := pieces[1]
	articlePieces := strings.Split(articleFooter, "<ul>")
	if len(pieces) != 2 {
		return "", nil
	}
	article := articlePieces[0]

	return article, nil
}

func gotoReferrerOrHome(w http.ResponseWriter, r *http.Request) {
	if r.Referer() != "" {
		http.Redirect(w, r, r.Referer(), 301)
	} else {
		http.Redirect(w, r, "/", 301)
	}
}

func getarticle(w http.ResponseWriter, r *http.Request, id string) (string, error) {
	client := urlfetch.Client(appengine.NewContext(r))
	resp, err := client.Get("http://text.npr.org/s.php?sId=" + id)
	if err != nil {
		return "", err
	}
	if resp.StatusCode == 404 {
		gotoReferrerOrHome(w, r)
		return "", nil
	}
	body, err := parsearticle(resp.Body)
	// Some of NPR's pages just don't exist with this api. How
	// about we just provide an error message and blame npr.
	if body == "" {
		gotoReferrerOrHome(w, r)
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return body, nil
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
		// Strip out the story id from the story url.
		re := regexp.MustCompile("https://www.npr.org.*/[0-9]{4}/[0-9]{1,2}/[0-9]{1,2}/([0-9]+).*")
		submatches := re.FindAllStringSubmatch(item.Link, -1)
		url := item.Link
		if len(submatches) != 0 {
			url = "/story/" + submatches[0][1]
		}

		// add this story to the story list.
		contentItems += fmt.Sprintf(contentTemplate, url, item.Title)
	}
	title := fmt.Sprintf(titleTemplate, feed.Title)
	currentPage := fmt.Sprintf(pageTemplate, title, styleTemplate, categories, category, contentItems)
	return currentPage, nil
}

func init() {
	// Manually add the top stories to the front.
	r := mux.NewRouter()
	categories += fmt.Sprintf(categoryTemplate, "", "/", "Headlines")
	for _, page := range categoryUrls {
		// Set up an http handler for this.
		r.HandleFunc(page.Url, index)

		// Skip adding top stories to the header.
		if page.Url == "/" {
			continue
		}
		// Set up the category list for the header.
		categories += fmt.Sprintf(categoryTemplate, seperator, page.Url, page.Category)
	}
	r.HandleFunc("/story/{id:[0-9]+}", story)
	http.Handle("/", r)
}
