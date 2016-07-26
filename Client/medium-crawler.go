package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

// MediumPost encapsulates a post on the platform Medium
type MediumPost struct {
	ID     string   `json:"id"`
	Author string   `json:"creatorId"`
	Lang   string   `json:"detectedLanguage"`
	Slug   string   `json:"uniqueSlug"`
	Tags   []string `json:"tags"`
	Text   string   `json:"text"`
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func processPost(post map[string]interface{}) MediumPost {
	creator := post["creator"].(map[string]interface{})

	uniqueSlug := post["uniqueSlug"].(string)
	if uniqueSlug == "" {
		uniqueSlug = post["slug"].(string) + "-" + post["id"].(string)
		fmt.Println("NOTE: no unique slug field, automatically created " + uniqueSlug)
	}

	encodedPost := MediumPost{
		ID:     post["id"].(string),
		Author: creator["username"].(string),
		Lang:   post["detectedLanguage"].(string),
		Slug:   uniqueSlug}

	return encodedPost
}

func populateHeader(req *http.Request) {
	req.Header.Set("accept", "application/json")
	req.Header.Set("accept-language", "de-DE,de;q=0.8,en-US;q=0.6,en;q=0.4")
	req.Header.Set("dnt", "1")
	req.Header.Set("origin", "https://medium.com")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 6.3; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/48.0.2564.97 Safari/537.36")
	req.Header.Set("x-obvious-cid", "web")
	req.Header.Set("x-xsrf-token", "1")
}

func fetchPostContent(post *MediumPost) {
	resp, err := http.Get("https://medium.com/@" + post.Author + "/" + post.Slug + "?format=json")
	check(err)

	defer resp.Body.Close()
	fmt.Println("response:", resp.Status)

	body, _ := ioutil.ReadAll(resp.Body)
	var encodedContent map[string]interface{}
	json.Unmarshal(body[16:], &encodedContent)

	strs := encodedContent["payload"].(map[string]interface{})
	postBody := strs["value"].(map[string]interface{})

	if postBody["content"] == nil {
		fmt.Println("ERROR: no content for post https://medium.com/@" + post.Author + "/" + post.Slug + "?format=json")
		return
	}

	postContent := postBody["content"].(map[string]interface{})
	bodyModel := postContent["bodyModel"].(map[string]interface{})

	//fmt.Print(bodyModel)
	paragraphs := bodyModel["paragraphs"].([]interface{})

	var buffer bytes.Buffer

	for _, paragraph := range paragraphs {
		// type 1 are text paragraphs
		encodedParagraph := paragraph.(map[string]interface{})
		if encodedParagraph["type"].(float64) == 1 {
			buffer.WriteString(encodedParagraph["text"].(string)) // just append the text for now
		}
	}

	post.Text = buffer.String()

	tags := postBody["virtuals"].(map[string]interface{})["tags"].([]interface{})

	for _, tag := range tags {
		post.Tags = append(post.Tags, tag.(map[string]interface{})["slug"].(string))
	}
}

func fetchTop100Content(httpClient *http.Client, dateString string, ignoredPosts []string) []byte {
	reqURL := "https://medium.com/top-100/" + dateString + "/load-more"
	jsonIgnoredPosts, _ := json.Marshal(ignoredPosts)

	var reqStr = []byte(`{"count":10,"ignore":` + string(jsonIgnoredPosts) + `}`)

	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(reqStr))

	check(err)

	populateHeader(req)
	req.Header.Set("content-length", fmt.Sprint(len(reqStr)))
	req.Header.Set("content-type", "application/json")
	req.Header.Set("referer", "https://medium.com/top-100/"+dateString)
	resp, err := httpClient.Do(req)
	check(err)

	defer resp.Body.Close()

	fmt.Println("response:", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)

	return body[16:]
}

func fetchTop100Posts(dateString string) []MediumPost {
	fmt.Println("fetching top 100 for " + dateString)
	fetchedPostIds := []string{}
	fetchedPosts := []MediumPost{}
	client := &http.Client{}

	for len(fetchedPostIds) < 100 {
		content := fetchTop100Content(client, dateString, fetchedPostIds)
		var encodedContent map[string]interface{}
		if err := json.Unmarshal(content, &encodedContent); err != nil {
			panic(err)
		}
		strs := encodedContent["payload"].(map[string]interface{})
		posts := strs["value"].([]interface{})

		if len(posts) == 0 {
			break
		}

		fmt.Printf("got %d posts\n", len(posts))
		for _, post := range posts {
			encodedPost := processPost(post.(map[string]interface{}))
			fetchedPostIds = append(fetchedPostIds, encodedPost.ID)
			fmt.Println("fetched post id " + encodedPost.ID)

			if encodedPost.Lang == "en" {
				fetchedPosts = append(fetchedPosts, encodedPost)
			}
		}
	}

	return fetchedPosts
}

func exportToFile(posts *[]MediumPost, filename string) {
	f, err := os.Create(filename)
	check(err)

	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	err = enc.Encode(posts)
	check(err)

	w.Flush()
}

func fetchAndExportTop100(monthString string) {
	fetchedPosts := fetchTop100Posts(monthString)

	for ix := range fetchedPosts {
		fetchPostContent(&fetchedPosts[ix])
	}

	exportToFile(&fetchedPosts, monthString+".json")
}

func main() {
	t := time.Date(2014, 2, 1, 1, 1, 1, 1, time.UTC)
	for t.Before(time.Now()) {
		t = t.AddDate(0, 1, 0)
		fetchAndExportTop100(strings.ToLower(t.Format("January-2006")))
	}
}
