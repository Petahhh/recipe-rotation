package e2e

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestEndpointServesTraffic(t *testing.T) {
	t.Parallel()

	const ip = "34.60.141.247"
	url := fmt.Sprintf("http://%s:80/", ip)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("expected endpoint %s to serve traffic, but request failed: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		t.Fatalf("expected 2xx response from %s, got status %d", url, resp.StatusCode)
	}
}
