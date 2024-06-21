package sync_test

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/EyobAshenaki/chroma-go/sync"
)

func TestSyncStart(t *testing.T) {
	fmt.Println("Sync test start")

	// mmSync := sync.GetSyncInstance()
	// mmSync.InitializeStore()

	// c, err := mmSync.StartSync()

	// if err != nil {
	// 	t.Errorf("Start sync error: %s", err)
	// }

	// for result := range c {
	// 	fmt.Printf("Syncing posts... %.2f%% complete\n", result)
	// }

	// mmSync.CloseStore()

	// ----------------------------------------------
	// ticker := time.NewTicker(2 * time.Second)
	// fmt.Println("Ticker set and ready! ", time.Now())

	// ctx, cancelCtx := context.WithCancel(context.Background())

	// go func() {
	// 	for {
	// 		select {
	// 		case t := <-ticker.C:
	// 			func() {
	// 				fmt.Println("Current time: ", t)
	// 				time.Sleep(3 * time.Second)
	// 			}()

	// 		case <-ctx.Done():
	// 			if err := ctx.Err(); err != nil {
	// 				fmt.Printf("doAnother err: %s\n", err)
	// 			}
	// 			fmt.Println("Done")
	// 			ticker.Stop()
	// 			return
	// 		}
	// 	}
	// }()

	// // give the ticker some time to run before being canceled
	// time.Sleep(60 * time.Second)

	// cancelCtx()

	// ----------------------------------------------

	// * idea *
	// the program should return a tuple of a bool indicating the
	// the fetching is complete and an int indicating the fetched
	// posts completion percentage

	// ----------------------------------------------

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

	broker := sync.NewSSEServer()

	// Make b the HTTP handler for "/sync/start". This is
	// possible since it has a ServeHTTP method. That method
	// is called in a separate goroutine for each
	// request to "/sync/start".
	http.Handle("/sync/start", broker)

	go func() {
		mmSync := sync.GetSyncInstance()

		mmSync.InitializeStore()
		defer mmSync.CloseStore()

		percentageChan, err := mmSync.StartSync()
		if err != nil {
			fmt.Println(fmt.Errorf("Start sync error: %s", err))
		}

		defer mmSync.StopSync()

		for percent := range percentageChan {
			fmt.Printf("Syncing posts... %.2f%% complete\n", percent)
			fmt.Println()
			fmt.Println("-------------------------------------")
			fmt.Println()

			msgInByte := []byte(strconv.FormatFloat(percent, 'f', -1, 64))

			broker.SendMessage(msgInByte)
		}
	}()

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
