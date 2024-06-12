package sync_test

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	// "github.com/EyobAshenaki/chroma-go/sync"
)

func TestSyncStart(t *testing.T) {
	fmt.Println("Sync test start")

	// mmSync := sync.New()
	// mmSync.InitializeStore()

	// if err := mmSync.StartFetch(); err != nil {
	// 	t.Errorf("Start sync error: %s", err)
	// }

	// mmSync.CloseStore()

	// ----------------------------------------------
	// ticker := time.NewTicker(2 * time.Second)

	// // done := make(chan bool)
	// ctx, cancelCtx := context.WithCancel(context.Background())

	// go func() {
	// 	for {
	// 		select {
	// 		case t := <-ticker.C:
	// 			func() {
	// 				time.Sleep(3 * time.Second)
	// 				fmt.Println("Current time: ", t)
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
	// time.Sleep(6 * time.Second)

	// cancelCtx()

	// // give the ticker some time to run
	// time.Sleep(10 * time.Second)

	// // stops the ticker execution
	// done <- true
	// cancelCtx()

	// ----------------------------------------------

	http.HandleFunc("/", getRoot)
	http.HandleFunc("/hello", getHello)

	err := http.ListenAndServe(":3333", nil)
	if err != nil {
		fmt.Printf("ListenAndServe error: %s\n", err)
	}

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

	fmt.Println("Sync test end")
}

func getRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("got / request\n")
	io.WriteString(w, "This is my website!\n")
}
func getHello(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("got /hello request\n")
	io.WriteString(w, "Hello, HTTP!\n")
}
