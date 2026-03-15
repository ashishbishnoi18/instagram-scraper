package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/embedtools/instagram-scraper/scraper"
	"github.com/embedtools/instagram-scraper/types"
)

func main() {
	username := "natthakarnbookkeeping"
	if len(os.Args) > 1 {
		username = os.Args[1]
	}

	client, err := scraper.New(
		scraper.WithProxyURL(os.Getenv("PROXY_URL")),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "client error: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	out, err := client.GetProfile(ctx, &types.GetProfileInput{URL: username})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}
