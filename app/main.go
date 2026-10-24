// TILFacts: condenses TIL into soylent TIL.
//
// Double-click the app and TILFacts opens in your web browser. It reads
// Reddit's r/todayilearned RSS feed on this computer, serves the page at
// http://127.0.0.1:8015, and quits on its own about 10 seconds after the last
// TILFacts tab closes, or right away when you answer "Yes" to
// "Would you like to close?".
//
// Why a program and not just a page: Reddit closed its anonymous .json feeds
// on May 28, 2026, and browsers don't let a page opened from a file read the
// RSS feed. See ../README.md for building.
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed TILFacts.html
var page []byte

const (
	host        = "127.0.0.1"
	defaultPort = 8015 // a nod to 2015
	userAgent   = "TILFacts/2.0 (personal reader; +https://github.com/scriptkittie/TILFacts)"

	// Being polite to Reddit.
	cacheFor   = 90 * time.Second // reuse an answer instead of asking again
	minGap     = 2 * time.Second  // never ask more often than this
	coolDown   = time.Minute      // after Reddit says "slow down", don't ask at all for this long
	maxRetryIn = 10 * time.Second // longest we'll wait for the one retry after a "slow down"

	// Closing by itself.
	closeGrace   = 10 * time.Second // after the last tab closes (long enough to survive a reload)
	tabTimeout   = 4 * time.Minute  // a tab that stops checking in (sleep, crash) counts as closed
	firstVisitBy = 3 * time.Minute  // quit if the browser never shows up
)

// TILFACTS_FEED_URL points the app at a test feed instead of Reddit.
var feedURL = envOr("TILFACTS_FEED_URL", "https://www.reddit.com/r/todayilearned/%s/.rss")

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

var (
	sorts   = map[string]bool{"new": true, "hot": true, "rising": true, "top": true}
	periods = map[string]bool{"day": true, "week": true, "month": true, "year": true, "all": true}
	linkRe  = regexp.MustCompile(`<a href="([^"]+)">\s*\[link\]\s*</a>`)
	afterRe = regexp.MustCompile(`^t3_[a-z0-9]+$`)
	client  = &http.Client{Timeout: 15 * time.Second}
)

// ---------- Reddit ----------

type feedErr struct {
	status int
	msg    string
}

func (e *feedErr) Error() string { return e.msg }

var errSlowDown = &feedErr{http.StatusTooManyRequests, "Reddit asked us to slow down."}

var (
	feedMu    sync.Mutex // one request to Reddit at a time
	cache     = map[string]cachedFeed{}
	lastReq   time.Time
	coolUntil time.Time
)

type cachedFeed struct {
	at   time.Time
	body []byte
}

func fetchFeed(u string) ([]byte, error) {
	feedMu.Lock()
	defer feedMu.Unlock()
	if c, ok := cache[u]; ok && time.Since(c.at) < cacheFor {
		return c.body, nil
	}
	if time.Now().Before(coolUntil) {
		return nil, errSlowDown
	}
	time.Sleep(time.Until(lastReq.Add(minGap)))

	res, err := get(u)
	if err == nil && res.StatusCode == http.StatusTooManyRequests {
		// Wait as long as Reddit asks (within reason), then try exactly once more.
		wait := 5 * time.Second
		if s, e := strconv.Atoi(res.Header.Get("Retry-After")); e == nil {
			wait = time.Duration(s) * time.Second
		}
		res.Body.Close()
		time.Sleep(min(max(wait, minGap), maxRetryIn))
		res, err = get(u)
	}
	if err != nil {
		return nil, &feedErr{http.StatusGatewayTimeout, "Couldn't reach Reddit."}
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == http.StatusTooManyRequests:
		coolUntil = time.Now().Add(coolDown)
		return nil, errSlowDown
	case res.StatusCode != http.StatusOK:
		return nil, &feedErr{http.StatusBadGateway, fmt.Sprintf("Reddit answered %d.", res.StatusCode)}
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, &feedErr{http.StatusGatewayTimeout, "Reddit's answer was cut off."}
	}
	cache[u] = cachedFeed{time.Now(), body}
	return body, nil
}

func get(u string) (*http.Response, error) {
	lastReq = time.Now()
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/atom+xml")
	return client.Do(req)
}

type atomFeed struct {
	Entries []struct {
		ID    string `xml:"id"`
		Title string `xml:"title"`
		Link  struct {
			Href string `xml:"href,attr"`
		} `xml:"link"`
		Content   string `xml:"content"`
		Published string `xml:"published"`
		Updated   string `xml:"updated"`
		Author    struct {
			Name string `xml:"name"`
		} `xml:"author"`
	} `xml:"entry"`
}

// post is one TIL, in the shape the page reads.
type post struct {
	ID      string `json:"id"`      // e.g. t3_1abcde
	Title   string `json:"title"`   // the original Reddit title
	URL     string `json:"url"`     // the source the post links to
	Domain  string `json:"domain"`  // its site, e.g. en.wikipedia.org
	Thread  string `json:"thread"`  // the Reddit comments page
	Created int64  `json:"created"` // Unix seconds
	Author  string `json:"author"`
}

func parseFeed(body []byte) ([]post, error) {
	var f atomFeed
	if err := xml.Unmarshal(body, &f); err != nil {
		return nil, &feedErr{http.StatusBadGateway, "Reddit sent something that isn't a feed."}
	}
	posts := make([]post, 0, len(f.Entries))
	for _, e := range f.Entries {
		id, title, thread := strings.TrimSpace(e.ID), strings.TrimSpace(e.Title), e.Link.Href
		if !afterRe.MatchString(id) || title == "" || !strings.HasPrefix(thread, "https://") {
			continue
		}
		// The post's own link is the "[link]" anchor in the HTML content;
		// a text post links to itself.
		link, domain := thread, "reddit.com"
		if m := linkRe.FindStringSubmatch(e.Content); m != nil {
			if u, err := url.Parse(html.UnescapeString(m[1])); err == nil && (u.Scheme == "https" || u.Scheme == "http") {
				link = u.String()
				if strings.TrimRight(link, "/") != strings.TrimRight(thread, "/") {
					domain = strings.TrimPrefix(u.Hostname(), "www.")
				}
			}
		}
		stamp := e.Published
		if stamp == "" {
			stamp = e.Updated
		}
		var created int64
		if t, err := time.Parse(time.RFC3339, stamp); err == nil {
			created = t.Unix()
		}
		posts = append(posts, post{
			ID: id, Title: title, URL: link, Domain: domain, Thread: thread, Created: created,
			Author: strings.TrimSpace(strings.TrimPrefix(e.Author.Name, "/u/")),
		})
	}
	return posts, nil
}

// listing answers /api/listing?sort=new|hot|rising|top&t=day..all&limit=1..100&after=t3_x
func listing(q url.Values) (map[string]any, error) {
	sort := q.Get("sort")
	if !sorts[sort] {
		sort = "new"
	}
	limit, err := strconv.Atoi(q.Get("limit"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 100
	}
	params := url.Values{"limit": {strconv.Itoa(limit)}}
	if t := q.Get("t"); sort == "top" && periods[t] {
		params.Set("t", t)
	}
	if a := q.Get("after"); afterRe.MatchString(a) {
		params.Set("after", a)
	}
	body, err := fetchFeed(fmt.Sprintf(feedURL, sort) + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	posts, err := parseFeed(body)
	if err != nil {
		return nil, err
	}
	var after any // null when there's nothing more to ask for
	if len(posts) >= limit {
		after = posts[len(posts)-1].ID
	}
	return map[string]any{"posts": posts, "after": after}, nil
}

// ---------- open tabs ----------

// Each TILFacts tab checks in with an id every 30 seconds and says goodbye
// when it closes. The app quits once no tabs are left.
var (
	tabsMu     sync.Mutex
	tabs       = map[string]time.Time{}
	gone       = map[string]bool{} // tabs that said goodbye; ignore their late check-ins
	everOpened bool
	emptySince time.Time // when the last tab left
	quit       = make(chan struct{})
	quitOnce   sync.Once
)

var tabRe = regexp.MustCompile(`^[a-z0-9]{6,32}$`)

func tabHello(id string) {
	if !tabRe.MatchString(id) {
		return
	}
	tabsMu.Lock()
	if !gone[id] {
		tabs[id] = time.Now()
		everOpened = true
	}
	tabsMu.Unlock()
}

func tabBye(id string) {
	if !tabRe.MatchString(id) {
		return
	}
	tabsMu.Lock()
	delete(tabs, id)
	gone[id] = true
	if len(tabs) == 0 {
		emptySince = time.Now()
	}
	tabsMu.Unlock()
}

func shouldQuit(started time.Time) bool {
	tabsMu.Lock()
	defer tabsMu.Unlock()
	for id, seen := range tabs {
		if time.Since(seen) > tabTimeout {
			delete(tabs, id)
			if len(tabs) == 0 {
				emptySince = time.Now()
			}
		}
	}
	if !everOpened {
		return time.Since(started) > firstVisitBy
	}
	return len(tabs) == 0 && time.Since(emptySince) > closeGrace
}

func stop() { quitOnce.Do(func() { close(quit) }) }

// ---------- web server ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func handler(port int) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	})
	mux.HandleFunc("GET /api/ping", func(w http.ResponseWriter, r *http.Request) {
		tabHello(r.URL.Query().Get("tab"))
		writeJSON(w, http.StatusOK, map[string]string{"app": "TILFacts"})
	})
	mux.HandleFunc("POST /api/bye", func(w http.ResponseWriter, r *http.Request) {
		tabBye(r.URL.Query().Get("tab"))
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /api/quit", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"closed": "TILFacts"})
		go func() { time.Sleep(300 * time.Millisecond); stop() }()
	})
	mux.HandleFunc("GET /api/listing", func(w http.ResponseWriter, r *http.Request) {
		data, err := listing(r.URL.Query())
		if err != nil {
			var fe *feedErr
			if !errors.As(err, &fe) {
				fe = &feedErr{http.StatusInternalServerError, "Something went wrong."}
			}
			writeJSON(w, fe.status, map[string]string{"error": fe.msg})
			return
		}
		writeJSON(w, http.StatusOK, data)
	})

	// Answer only this computer's own pages: the address must be ours, and any
	// page calling in must come from it (stops other websites poking the app).
	self := fmt.Sprintf("%s:%d", host, port)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != self && r.Host != fmt.Sprintf("localhost:%d", port) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if o := r.Header.Get("Origin"); o != "" && o != "http://"+r.Host {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		mux.ServeHTTP(w, r)
	})
}

func openBrowser(u string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	case "darwin":
		cmd = exec.Command("open", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	_ = cmd.Start()
}

// alreadyRunning reports whether TILFacts is already serving on this port.
func alreadyRunning(port int) bool {
	c := &http.Client{Timeout: time.Second}
	res, err := c.Get(fmt.Sprintf("http://%s:%d/api/ping", host, port))
	if err != nil {
		return false
	}
	defer res.Body.Close()
	var v struct{ App string }
	return json.NewDecoder(res.Body).Decode(&v) == nil && v.App == "TILFacts"
}

func main() {
	port := defaultPort
	if len(os.Args) > 1 {
		if p, err := strconv.Atoi(os.Args[1]); err == nil {
			port = p
		}
	}
	// Double-clicked again while it's open: just show it.
	if alreadyRunning(port) {
		openBrowser(fmt.Sprintf("http://%s:%d/", host, port))
		return
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		ln, err = net.Listen("tcp", host+":0") // something else has our port
		if err != nil {
			os.Exit(1)
		}
		port = ln.Addr().(*net.TCPAddr).Port
	}
	srv := &http.Server{Handler: handler(port), ReadHeaderTimeout: 10 * time.Second}
	go srv.Serve(ln)

	started := time.Now()
	openBrowser(fmt.Sprintf("http://%s:%d/", host, port))

	go func() {
		for range time.Tick(2 * time.Second) {
			if shouldQuit(started) {
				stop()
				return
			}
		}
	}()

	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
