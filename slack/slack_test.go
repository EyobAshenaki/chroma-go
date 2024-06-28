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

	fmt.Println("Slack test end")
}
