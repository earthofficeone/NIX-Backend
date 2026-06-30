package keepalive

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	interval    = 5 * time.Minute
	healthPath  = "/api/health"
	pingTimeout = 30 * time.Second
)

func Start(ctx context.Context, publicURL string) {
	publicURL = strings.TrimRight(strings.TrimSpace(publicURL), "/")
	if publicURL == "" {
		return
	}

	url := publicURL + healthPath
	client := &http.Client{Timeout: pingTimeout}

	ping := func() {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			log.Printf("keep-alive: request: %v", err)
			return
		}

		res, err := client.Do(req)
		if err != nil {
			log.Printf("keep-alive: ping failed: %v", err)
			return
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			log.Printf("keep-alive: ping status %d", res.StatusCode)
		}
	}

	log.Printf("keep-alive: started (url=%s, interval=5m)", url)
	ping()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Print("keep-alive: stopped")
				return
			case <-ticker.C:
				ping()
			}
		}
	}()
}
