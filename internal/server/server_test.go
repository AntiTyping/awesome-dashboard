package server

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AntiTyping/awesome-dashboard/internal/store"
)

func newTestServer(t *testing.T) (*store.Store, *httptest.Server) {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	mux := http.NewServeMux()
	New(st).RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return st, ts
}

func getItems(t *testing.T, ts *httptest.Server) (*http.Response, string) {
	t.Helper()
	resp, err := http.Get(ts.URL + "/api/items")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, strings.TrimSpace(string(b))
}

func TestGetItems_Empty(t *testing.T) {
	_, ts := newTestServer(t)
	resp, body := getItems(t, ts)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type %q", ct)
	}
	if body != "[]" {
		t.Fatalf("body %q", body)
	}
}

func TestGetItems_WithData(t *testing.T) {
	st, ts := newTestServer(t)
	st.Add("A", "sub a", "red")
	st.Add("B", "sub b", "blue")
	resp, body := getItems(t, ts)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var items []store.Item
	if err := json.Unmarshal([]byte(body), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items", len(items))
	}
	if items[0].Title != "A" || items[0].Subtitle != "sub a" || items[0].Color != "red" || items[0].ID == "" {
		t.Fatalf("bad item %+v", items[0])
	}
	if items[1].Title != "B" || items[1].Subtitle != "sub b" || items[1].Color != "blue" {
		t.Fatalf("bad item %+v", items[1])
	}
}

func TestGetItems_OrderedByCreatedAt(t *testing.T) {
	st, ts := newTestServer(t)
	st.Add("A", "", "")
	st.Add("B", "", "")
	_, body := getItems(t, ts)
	var items []store.Item
	if err := json.Unmarshal([]byte(body), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Title != "A" {
		t.Fatalf("unexpected order %+v", items)
	}
}

type sseClient struct {
	resp  *http.Response
	lines chan string
}

func connectSSE(t *testing.T, ts *httptest.Server) *sseClient {
	t.Helper()
	resp, err := http.Get(ts.URL + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type %q", ct)
	}
	c := &sseClient{resp: resp, lines: make(chan string, 16)}
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			c.lines <- sc.Text()
		}
		close(c.lines)
	}()
	t.Cleanup(func() { resp.Body.Close() })
	return c
}

func (c *sseClient) expectRefresh(t *testing.T) {
	t.Helper()
	for _, want := range []string{"data: refresh", ""} {
		select {
		case l, ok := <-c.lines:
			if !ok || l != want {
				t.Fatalf("got %q ok=%v, want %q", l, ok, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for refresh")
		}
	}
}

func notify(t *testing.T, ts *httptest.Server) {
	t.Helper()
	resp, err := http.Post(ts.URL+"/api/notify", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("notify status %d", resp.StatusCode)
	}
}

func TestNotifyTriggersSSE(t *testing.T) {
	_, ts := newTestServer(t)
	c := connectSSE(t, ts)
	notify(t, ts)
	c.expectRefresh(t)
}

func TestSSEMultipleClients(t *testing.T) {
	_, ts := newTestServer(t)
	var cs []*sseClient
	for i := 0; i < 3; i++ {
		cs = append(cs, connectSSE(t, ts))
	}
	notify(t, ts)
	for _, c := range cs {
		c.expectRefresh(t)
	}
}

func TestSSEClientDisconnect(t *testing.T) {
	_, ts := newTestServer(t)
	gone := connectSSE(t, ts)
	stay := connectSSE(t, ts)
	gone.resp.Body.Close()
	time.Sleep(100 * time.Millisecond)
	notify(t, ts)
	stay.expectRefresh(t)
	notify(t, ts)
	stay.expectRefresh(t)
}
