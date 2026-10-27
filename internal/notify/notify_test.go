package notify

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func TestSend_PostsToNotifyEndpoint(t *testing.T) {
	called := make(chan bool, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called <- r.Method == http.MethodPost && r.URL.Path == "/api/notify"
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	Send(port)

	select {
	case ok := <-called:
		if !ok {
			t.Fatal("expected POST /api/notify")
		}
	default:
		t.Fatal("notify endpoint was not called")
	}
}

func TestSend_IgnoresErrors(t *testing.T) {
	Send(99999)
}
