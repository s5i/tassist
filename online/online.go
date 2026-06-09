package online

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/s5i/tassist/settings"
	"golang.org/x/net/html"
)

func New(stStorage *settings.Storage) (*Online, error) {
	st := stStorage.Preset()

	return &Online{
		domain:           st.OnlineSource.Domain,
		serverCookie:     st.OnlineSource.ServerCookie,
		serverTypeCookie: st.OnlineSource.ServerTypeCookie,
		headers:          http.Header{"User-Agent": {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36"}},
	}, nil
}

type Online struct {
	domain           string
	serverCookie     string
	serverTypeCookie string
	headers          http.Header

	onlineMu  sync.Mutex
	online    int
	hasOnline bool
}

func (o *Online) Run(ctx context.Context) error {
	for {
		online, err := o.get(ctx)
		if err != nil {
			log.Printf("Online count fetch failed: %v", err)
		}

		o.onlineMu.Lock()
		o.online = online
		o.hasOnline = err == nil
		o.onlineMu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Minute):
		}
	}
}

func (o *Online) Get() (int, bool) {
	o.onlineMu.Lock()
	defer o.onlineMu.Unlock()

	return o.online, o.hasOnline
}

func (o *Online) get(ctx context.Context) (int, error) {
	addr := fmt.Sprintf("https://%s", o.domain)
	req, err := http.NewRequestWithContext(ctx, "GET", addr, nil)
	if err != nil {
		return 0, fmt.Errorf("http.NewRequestWithContext(%q) failed: %v", addr, err)
	}

	for k, v := range o.headers {
		for _, v := range v {
			req.Header.Add(k, v)
		}
	}
	if o.serverCookie != "" {
		req.AddCookie(cookie("server", o.serverCookie))
	}
	if o.serverTypeCookie != "" {
		req.AddCookie(cookie("serverType", o.serverTypeCookie))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("http.DefaultClient.Do(%q) failed: %v", addr, err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("html.Parse failed: %v", err)
	}

	for n := range doc.Descendants() {
		if n.Type != html.ElementNode || n.Data != "a" {
			continue
		}
		for _, a := range n.Attr {
			if a.Key != "href" || a.Val != "/stats/online" {
				continue
			}

			for t := range n.Descendants() {
				if t.Type != html.TextNode || !strings.HasPrefix(t.Data, "Currently:") {
					continue
				}

				onlineStr := strings.TrimSpace(strings.TrimPrefix(t.Data, "Currently:"))
				if online, err := strconv.Atoi(onlineStr); err == nil {
					return online, nil
				}
			}

		}
	}

	return 0, fmt.Errorf("online count not found")
}

func cookie(name, value string) *http.Cookie {
	return &http.Cookie{
		Name:   name,
		Value:  value,
		Path:   "/",
		MaxAge: int(31 * 24 * time.Hour / time.Second),
	}
}
