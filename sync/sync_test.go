package sync_test

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	sse "github.com/EyobAshenaki/chroma-go/server_side_event"
)

func TestSyncStart(t *testing.T) {
	fmt.Println("Sync test start")

	// if _, err := sync.GetAllChannels(); err != nil {
	// 	t.Errorf("Start sync error: %s", err)
	// }

	// ----------------------------------------------

	// postParams := url.Values{
	// 	"since": {""},
	// }

	// postParams.Set("since", "1717873778562")

	// fmt.Println("Param: ", postParams)
	// channelId := "znqgwqwykb887k9zpn7z1m47mc"

	// if _, err := sync.FetchPostsForPage(channelId, postParams); err != nil {
	// 	t.Errorf("Start sync error: %s", err)
	// }

	// ----------------------------------------------

	broker := sse.NewSSEServer()

	http.Handle("/sync/start", broker)

	http.Handle("/sync/stop", broker)

	err := http.ListenAndServe(":3333", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}

	// ----------------------------------------------

	fmt.Println("Sync test end")
}
