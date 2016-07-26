package main

import (
	"net/http"
	"strconv"
	"text/template"

	"github.com/pkg/browser"
)

func webCrawlerQuora(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) > len("/crawlerQuora/") {
		subPage := r.URL.Path[len("/crawlerQuora/"):]

		if subPage == "crawlCategories" {
			amount, _ := strconv.ParseInt(r.FormValue("amount"), 10, 64)

			htmlBody := `<h1>Quora Crawler</h1>
        <p><h2>Crawling {{.}} different categories from Quora... This can take some time, you can follow the progress in the command window!</h2></p>
          `

			t, _ := template.New("quora-crawling").Parse(htmlBody)
			t.Execute(w, amount)

			crawlAndExportQuoraCategories(int(amount))
		} else if subPage == "fetchCategory" {
			category := r.FormValue("category")

			htmlBody := `<h1>Quora Crawler</h1>
        <p><h2>Fetching articles for Quora category {{.}}... This can take some time, you can follow the progress in the command window!</h2></p>
          `

			t, _ := template.New("quora-crawling").Parse(htmlBody)
			t.Execute(w, category)

			empty := map[string]int{}
			fetchAndExportQuoraCategory(category, &empty)
		}

	} else {
		htmlBody := `<h1>Quora Crawler</h1>
    <p><form action="/crawlerQuora/crawlCategories" method="POST">
      <div>Enter the amount of categories you want to crawl: <input type="text" name="amount"></div>
      <div><input type="submit" value="Go"></div>
    </form></p>
    <p><form action="/crawlerQuora/fetchCategory" method="POST">
      <div>Fetch articles from a specific category: <input type="text" name="category"></div>
      <div><input type="submit" value="Go"></div>
    </form></p>
      `

		w.Write([]byte(htmlBody))
	}
}

func webCrawlerMain(w http.ResponseWriter, r *http.Request) {
	htmlBody := `<h1>Crawler</h1>
    <p><h2><a href="/crawlerQuora">Quora</a></p>
    <p><h2><a href="/crawlerMedium">Medium</a></p>
    `

	w.Write([]byte(htmlBody))
}

func webMain(w http.ResponseWriter, r *http.Request) {
	htmlBody := `<h1>Mail Classifier</h1>
    <p><h2><a href="/gmailFetch">E-Mails from Gmail</a></p>
    <p><h2><a href="/crawlerMain">Crawler</a></p>
    `

	w.Write([]byte(htmlBody))
}

func main() {
	http.HandleFunc("/", webMain)
	http.HandleFunc("/gmailFetch/", webGmailFetch)
	http.HandleFunc("/gmailView/", webGmailView)
	http.HandleFunc("/crawlerMain", webCrawlerMain)
	http.HandleFunc("/crawlerQuora/", webCrawlerQuora)
	http.ListenAndServe(":8080", nil)
	browser.OpenURL("http://localhost:8080")
}
