package sync_test

import (
	"fmt"
	"testing"

	"github.com/EyobAshenaki/chroma-go/sync"
)

func TestSyncStart(t *testing.T) {
	fmt.Println("Sync test start")

	sync.Init()

	if err := sync.Start(); err != nil {
		t.Errorf("Start sync error: %s", err)
	}

	sync.Close()

	// if _, err := sync.GetAllChannels(); err != nil {
	// 	t.Errorf("Start sync error: %s", err)
	// }

	// postParams := url.Values{
	// 	"since": {""},
	// }

	// postParams.Set("since", "1717873778562")

	// fmt.Println("Param: ", postParams)
	// channelId := "znqgwqwykb887k9zpn7z1m47mc"

	// if _, err := sync.FetchPostsForPage(channelId, postParams); err != nil {
	// 	t.Errorf("Start sync error: %s", err)
	// }

	fmt.Println("Sync test end")
}
