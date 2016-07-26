package main

import (
	"bufio"
	"bytes"
	"container/heap"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	"github.com/jteeuwen/go-pkg-xmlx"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// QuoraAnswer represents a Quora answer and question
type QuoraAnswer struct {
	Question   string
	Answer     string
	Categories []string
	ID         string
	URL        string
}

func populateHeader(req *http.Request) {
	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("accept-language", "de-DE,de;q=0.8,en-US;q=0.6,en;q=0.4")
	req.Header.Set("cache-control", "max-age=0")
	req.Header.Set("cookie", `m-b="GbiM-63PLV_A1gIoGVuewA\075\075"; m-css_v=08e0dd11ea429f2f18; m-f=kisNMBYEAkki-x6lpNjEbzKh5-f_EeF_Exup; _ga=GA1.2.737719950.1440961339; m-depd=02051fe361f19eeb; m-s="nTCGT4wFo4cKoh30_7bP7Q\075\075"; m-t="B9-aQDNMFLosSl4b1oLr8w\075\075"; m-tz=-480`)
	req.Header.Set("dnt", "1")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 6.3; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/48.0.2564.97 Safari/537.36")
	req.Header.Set("upgrade-insecure-requests", "1")
}

// fetches additional categories for an article from its website, which involves a bit of HTML parsing
func fetchQuoraCategories(httpClient *http.Client, answer *QuoraAnswer) {
	if len(answer.Categories) > 1 {
		// answer already has more than one category, skip this one
		return
	}

	req, err := http.NewRequest("GET", answer.URL, nil)
	check(err)

	populateHeader(req)

	resp, err := httpClient.Do(req)
	check(err)
	defer resp.Body.Close()

	document := html.NewTokenizer(resp.Body)
	found := false
	for {
		tt := document.Next()
		switch tt {
		case html.ErrorToken:
			return
		case html.TextToken:
			if found {
				// replace whitespace with "-"" to simplify later usage
				category := strings.Replace(string(document.Text()), " ", "-", -1)
				duplicate := false
				for ix := range answer.Categories {
					if category == answer.Categories[ix] {
						// category already in list, skip this one
						duplicate = true
						break
					}
				}

				if !duplicate {
					answer.Categories = append(answer.Categories, category)
				}
			}
			break
		case html.StartTagToken:
			tn, hasAttributes := document.TagName()
			_, attrVal, _ := document.TagAttr()
			if hasAttributes && len(tn) == 4 && bytes.Equal(tn, []byte("span")) && len(attrVal) == 23 && bytes.Equal(attrVal, []byte("TopicNameSpan TopicName")) {
				found = true
			} else {
				found = false
			}
			break
		case html.EndTagToken:
			found = false
			break
		}
	}
}

// fetches the top 50 articles from a quora category (or whatever comes into the RSS feed)
func parseQuoraCategory(name string) []QuoraAnswer {
	// just parse the rss feed since quora's website is just a mess...
	// unfortunately that allows us only to get 50 articles per category
	resp, _ := http.Get("https://www.quora.com/topic/" + url.QueryEscape(name) + "/rss")

	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	feed := xmlx.New()
	err := feed.LoadBytes(body, nil)

	check(err)

	// load collected answers from the file first
	collectedAnswers := loadCategoryFromFile("trainingData/" + name + ".json")

	// remember articles we loaded already
	seenAnswers := map[string]bool{}
	for ix := range collectedAnswers {
		// index articles via URL, since GUID isn't permanent
		seenAnswers[collectedAnswers[ix].URL] = true
	}

	nodes := feed.SelectNodes("", "item")

	newCount := 0
	for node := range nodes {
		// only load new articles
		if !seenAnswers[nodes[node].S("", "link")] {
			newCount++
			collectedAnswers = append(collectedAnswers, QuoraAnswer{
				Question:   nodes[node].S("", "title"),
				Answer:     nodes[node].S("", "description"),
				Categories: []string{name},
				ID:         nodes[node].S("", "guid"),
				URL:        nodes[node].S("", "link")})
		}
	}

	fmt.Println("found", newCount, "new articles!")
	return collectedAnswers
}

func exportToFile(posts *[]QuoraAnswer, filename string) {
	f, err := os.Create(filename)
	check(err)

	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	err = enc.Encode(posts)
	check(err)

	w.Flush()
}

func convertShortNumber(number string) int {

	multiplier := 1.0
	if strings.HasSuffix(number, "m") {
		number = number[:len(number)-1]
		multiplier = 1000000.0
	} else if strings.HasSuffix(number, "k") {
		number = number[:len(number)-1]
		multiplier = 1000.0
	}

	parsed, _ := strconv.ParseFloat(number, 64)
	return int(multiplier * parsed)
}

func getQuoraTopicFollowerCount(httpClient *http.Client, topicName string) int {
	req, err := http.NewRequest("GET", "https://www.quora.com/topic/"+url.QueryEscape(topicName), nil)
	check(err)

	populateHeader(req)

	resp, err := httpClient.Do(req)
	check(err)
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	bodyStr := string(body)

	countString := "0"
	index := strings.Index(bodyStr, `"count">`)

	if index != -1 {
		indexEnd := strings.Index(bodyStr[index:], "<")

		if indexEnd != -1 {
			countString = bodyStr[index+8 : index+indexEnd]
		}
	}

	return convertShortNumber(countString)
}

func loadCategoryFromFile(filename string) []QuoraAnswer {
	var collectedAnswers []QuoraAnswer

	// only load if the file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return collectedAnswers
	}

	f, err := os.Open(filename)
	check(err)

	r := bufio.NewReader(f)
	dec := json.NewDecoder(r)
	err = dec.Decode(&collectedAnswers)
	check(err)

	return collectedAnswers
}

// QuoraCategory holds information about a Quora category
type QuoraCategory struct {
	Name          string
	FollowerCount int
	Index         int
}

type QCQueue []*QuoraCategory

func (pq QCQueue) Len() int { return len(pq) }

func (pq QCQueue) Less(i, j int) bool {
	// categories with a bigger follower count have a higher priority
	return pq[i].FollowerCount > pq[j].FollowerCount
}

func (pq QCQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *QCQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*QuoraCategory)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *QCQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

func fetchAndExportQuoraCategory(categoryName string, parsedCategories *map[string]int) {
	fmt.Println("fetching articles...")
	answers := parseQuoraCategory(categoryName)
	fmt.Print("fetching article categories:  0%")

	client := &http.Client{}

	for ix := range answers {
		fetchQuoraCategories(client, &answers[ix])

		for j := range answers[ix].Categories {
			(*parsedCategories)[answers[ix].Categories[j]] += 1
		}
		fmt.Printf("\b\b\b\b%3d%%", int(ix*100/len(answers)))
	}

	exportToFile(&answers, "trainingData/"+categoryName+".json")
	fmt.Print("\b\b\b\bDone!\n")
}

func crawlAndExportQuoraCategories(maxCount int) {

	client := &http.Client{}
	// a few starting categories, for each category we also count the number of articles associated with it
	parsedCategories := map[string]int{
		"Physics":                      5,
		"Politics":                     5,
		"United-Kingdom":               5,
		"Education":                    5,
		"The-United-States-of-America": 5,
		"Luck":    5,
		"Science": 5,
		"Germany": 5,
		"History": 5,
	}

	openQueue := QCQueue{}
	visited := map[string]bool{}
	seen := map[string]bool{}
	count := 0

	heap.Init(&openQueue)
	for {
		fmt.Print("adding newly found categories:  0%")
		c := 0
		for category := range parsedCategories {
			if !visited[category] && !seen[category] && parsedCategories[category] > 1 {
				heap.Push(&openQueue, &QuoraCategory{Name: category, FollowerCount: getQuoraTopicFollowerCount(client, category)})
				seen[category] = true
			}
			c++
			fmt.Printf("\b\b\b\b%3d%%", int(c*100/len(parsedCategories)))
		}
		fmt.Println("\b\b\b\bDone!")

		// we don't want to crawl more than maxCount categories, or maybe we do?
		if count >= maxCount || len(openQueue) == 0 {
			break
		}

		// reset parsed categories
		parsedCategories = map[string]int{}
		topCategory := heap.Pop(&openQueue).(*QuoraCategory)
		visited[topCategory.Name] = true

		fmt.Println("visiting", topCategory.Name, "with", topCategory.FollowerCount, "followers")
		fetchAndExportQuoraCategory(topCategory.Name, &parsedCategories)
		count++
	}

}
