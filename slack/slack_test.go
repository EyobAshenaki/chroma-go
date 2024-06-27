package slack_test

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/EyobAshenaki/chroma-go/slack"
)

func TestSlack(t *testing.T) {
	fmt.Println("Slack test start")

	slackClient := slack.GetSlackInstance()

	mux := http.NewServeMux()
	mux.HandleFunc("/slack/upload_zip", slackClient.HandleZipUpload)
	mux.HandleFunc("/slack/store_data", slackClient.HandleFilteredChannelData)

	err := http.ListenAndServe(":4500", mux)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}

	// ------------------------------------------------------------

	// slackClient.ReadExtractedData()

	// ------------------------------------------------------------

	// msgDateStr := "2023-06-19" + "T00:00:00Z"
	// msgDate, dateParseError := time.Parse(time.RFC3339, msgDateStr)
	// if dateParseError != nil {
	// 	log.Fatalf("error while trying to parse date: %v \n", dateParseError)
	// }

	// log.Println(msgDate)

	// ------------------------------------------------------------

	// ctx, cancelCtx := context.WithCancel(context.Background())
	// serverOne := &http.Server{
	// 	Addr:    ":3333",
	// 	Handler: mux,
	// }

	// serverTwo := &http.Server{
	// 	Addr:    ":4444",
	// 	Handler: mux,
	// }

	// go func() {
	// 	err := serverOne.ListenAndServe()
	// 	if errors.Is(err, http.ErrServerClosed) {
	// 		fmt.Printf("server one closed\n")
	// 	} else if err != nil {
	// 		fmt.Printf("error listening for server one: %s\n", err)
	// 	}
	// 	cancelCtx()
	// }()

	// go func() {
	// 	err := serverTwo.ListenAndServe()
	// 	if errors.Is(err, http.ErrServerClosed) {
	// 		fmt.Printf("server two closed\n")
	// 	} else if err != nil {
	// 		fmt.Printf("error listening for server two: %s\n", err)
	// 	}
	// 	cancelCtx()
	// }()

	// <-ctx.Done()

	fmt.Println("Slack test end")
}
