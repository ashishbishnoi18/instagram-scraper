package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/embedtools/instagram-scraper/scraper"
	"github.com/embedtools/instagram-scraper/types"
)

func main() {
	if os.Getenv("MODULE_CANARY_NETWORK") == "" {
		fmt.Println("SKIP: set MODULE_CANARY_NETWORK=1 to run canary tests")
		os.Exit(0)
	}

	client, err := scraper.New(
		scraper.WithProxyURL(os.Getenv("PROXY_URL")),
		scraper.WithCurlBinPath(os.Getenv("CURL_BIN_PATH")),
	)
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	passed, failed := 0, 0

	fmt.Print("Testing ListTopics... ")
	topicsResult, err := client.ListTopics(ctx, &types.ListTopicsInput{})
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
		failed++
	} else {
		fmt.Printf("OK (%d topics)\n", topicsResult.Count)
		passed++
	}

	fmt.Printf("\n--- Results: %d passed, %d failed ---\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}
