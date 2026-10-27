package notify

import (
	"fmt"
	"net/http"
	"time"
)

func Send(port int) {
	client := http.Client{Timeout: time.Second}
	resp, err := client.Post(fmt.Sprintf("http://localhost:%d/api/notify", port), "", nil)
	if err != nil {
		return
	}
	resp.Body.Close()
}
