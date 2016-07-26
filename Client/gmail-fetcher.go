package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/pkg/browser"
	"golang.org/x/net/context"
	"golang.org/x/net/html"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func webGmailGetClient(w http.ResponseWriter, r *http.Request, page string) *http.Client {
	ctx := context.Background()

	b, err := ioutil.ReadFile("client_id.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
		return nil
	}

	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
		return nil
	}

	cacheFile, err := tokenCacheFile()
	if err != nil {
		log.Fatalf("Unable to get path to cached credential file. %v", err)
		return nil
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		authCode := r.FormValue("token")

		if len(authCode) > 0 {
			tok, err = config.Exchange(oauth2.NoContext, authCode)
			if err != nil {
				log.Fatalf("Unable to retrieve token from web %v", err)
				return nil
			}
		} else {
			// no local token -> prompt for gmail-login and return
			htmlBody := `<h1>Login to Gmail</h1>
        <p>Please click <a href="{{index . 0}}">here</a> and enter your authorization code:
        <p><form action="/{{index . 1}}/" method="POST">
          <div><input type="text" name="token"></div>
          <div><input type="submit" value="Continue"></div>
        </form></p>`

			authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
			browser.OpenURL(authURL)

			t, _ := template.New("gmail-auth").Parse(htmlBody)
			t.Execute(w, []string{authURL, page})
			return nil
		}
	}

	saveToken(cacheFile, tok)
	client := config.Client(ctx, tok)
	return client
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)
	browser.OpenURL(authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir,
		url.QueryEscape("gmail-fetcher.json")), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.Create(file)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func webGmailFetch(w http.ResponseWriter, r *http.Request) {
	client := webGmailGetClient(w, r, "gmailFetch")

	if client == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	srv, err := gmail.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve gmail Client %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	webGmailListMails(w, r, srv)
}

func webGmailListMails(w http.ResponseWriter, req *http.Request, srv *gmail.Service) {
	user := "me"

	pageToken := ""
	if len(req.URL.Path) > len("/gmailFetch/") {
		pageToken = req.URL.Path[len("/gmailFetch/"):]
	}

	r, err := srv.Users.Threads.List(user).LabelIds("INBOX").MaxResults(30).PageToken(pageToken).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve threads. %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	htmlBody := `<h1>Gmail threads</h1>
    <ul>
      {{range .Threads}}
      <li><a href="/gmailView/{{.Id}}">{{.Id}}</a>: {{.Snippet}}</li>
      {{end}}
    </ul>
    <p><h2><a href="/gmailFetch/{{.NextPageToken}}">Next Page</a></h2></p>`

	t, _ := template.New("gmail-threads").Parse(htmlBody)
	t.Execute(w, r)
}

func webGmailView(w http.ResponseWriter, r *http.Request) {
	client := webGmailGetClient(w, r, "gmailView")

	threadID := r.URL.Path[len("/gmailView/"):]

	srv, err := gmail.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve gmail Client %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	webGmailViewThread(w, srv, threadID)
}

func stripTags(in string) string {
	out := ""

	z := html.NewTokenizer(bytes.NewBufferString(in))

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			return out
		} else if tt == html.TextToken {
			out += " "
			out += z.Token().Data
		}
	}
}

type MailMessage struct {
	Body    string
	Subject string
	ID      string
	Short   string
}

func base64dec(in string) string {
	// messages from gmail are not base64 per se, they are a set of base64 strings separated by "-"
	ret := ""
	parts := strings.Split(in, "-")
	for _, part := range parts {
		out, _ := base64.StdEncoding.DecodeString(part)
		ret += string(out)
	}
	return ret
}

type ClassificationResult struct {
	Category string
	Score    float64
}

type ByScore []interface{}

func (a ByScore) Len() int      { return len(a) }
func (a ByScore) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByScore) Less(i, j int) bool {
	return a[i].(map[string]interface{})["second"].(float64) > a[j].(map[string]interface{})["second"].(float64)
}

func getClassification(message string) []ClassificationResult {
	r, err := http.Post("http://localhost:8099/classify", "text/plain", bytes.NewBufferString(message))
	if err != nil {
		panic(err)
	}
	resp, _ := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	var result []interface{}
	json.Unmarshal(resp, &result)
	fmt.Println("Received", len(result), "potential classes")

	sort.Sort(ByScore(result))
	ret := []ClassificationResult{}
	for i := 0; i < 5 && i < len(result); i++ {
		ret = append(ret, ClassificationResult{result[i].(map[string]interface{})["first"].(string), result[i].(map[string]interface{})["second"].(float64)})
	}
	return ret
}

func webGmailViewThread(w http.ResponseWriter, srv *gmail.Service, threadID string) {
	user := "me"
	r, err := srv.Users.Threads.Get(user, threadID).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve thread. %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	funcMap := template.FuncMap{
		"base64dec": base64dec,
	}

	mails := []MailMessage{}
	for _, message := range r.Messages {
		msg := MailMessage{}

		for _, header := range message.Payload.Headers {
			if header.Name == "Subject" {
				msg.Subject = header.Value
				break
			}
		}

		msg.ID = message.Id
		msg.Short = message.Snippet
		if len(message.Payload.Parts) > 0 {
			// message is multipart
			for _, part := range message.Payload.Parts {
				if part.MimeType == "text/plain" {
					msg.Body += part.Body.Data
				}
			}
			if len(msg.Body) == 0 {
				msg.Body = message.Payload.Parts[0].Body.Data
			}
		} else {
			msg.Body = message.Payload.Body.Data
		}
		mails = append(mails, msg)
	}

	htmlBody := `<h1>Messages</h1>
      <ul>
        {{range .}}
        <li>{{.Subject}}: {{.Short}}</li>
        {{end}}
      </ul>`

	combinedMessages := ""
	for _, msg := range mails {
		if len(msg.Body) > 0 {
			combinedMessages += " " + base64dec(msg.Body)
		}
	}

	// fallback
	if len(combinedMessages) == 0 {
		combinedMessages = mails[0].Short
	}
	classifyResult := getClassification(combinedMessages)

	htmlBody += `<p><h2>Classification Scores:</h2><ul>`
	for _, c := range classifyResult {
		htmlBody += "<li>" + c.Category + ": " + strconv.FormatFloat(c.Score, 'g', -1, 64) + "</li>"
	}
	htmlBody += `</ul></p>`
	t, _ := template.New("gmail-thead").Funcs(funcMap).Parse(htmlBody)
	t.Execute(w, mails)
}
