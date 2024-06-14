package sync_test

import (
	"errors"
	"fmt"
	"io"
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
	// 	fmt.Println("Received result: ", result)
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

	// mux := http.NewServeMux()
	// mux.HandleFunc("/", getRoot)
	// mux.HandleFunc("/hello", getHello)

	// ctx, cancelCtx := context.WithCancel(context.Background())
	// serverOne := &http.Server{
	// 	Addr:    ":3333",
	// 	Handler: mux,
	// 	BaseContext: func(l net.Listener) context.Context {
	// 		ctx = context.WithValue(ctx, keyServerAddr, l.Addr().String())
	// 		return ctx
	// 	},
	// }

	// // serverTwo := &http.Server{
	// // 	Addr:    ":4444",
	// // 	Handler: mux,
	// // 	BaseContext: func(l net.Listener) context.Context {
	// // 		ctx = context.WithValue(ctx, keyServerAddr, l.Addr().String())
	// // 		return ctx
	// // 	},
	// // }

	// go func() {
	// 	err := serverOne.ListenAndServe()
	// 	if errors.Is(err, http.ErrServerClosed) {
	// 		fmt.Printf("server one closed\n")
	// 	} else if err != nil {
	// 		fmt.Printf("error listening for server one: %s\n", err)
	// 	}
	// 	cancelCtx()
	// }()

	// // go func() {
	// // 	err := serverTwo.ListenAndServe()
	// // 	if errors.Is(err, http.ErrServerClosed) {
	// // 		fmt.Printf("server two closed\n")
	// // 	} else if err != nil {
	// // 		fmt.Printf("error listening for server two: %s\n", err)
	// // 	}
	// // 	cancelCtx()
	// // }()

	// <-ctx.Done()

	// ----------------------------------------------

	// http.HandleFunc("/", getRoot)
	// http.HandleFunc("/hello", getHello)

	// err := http.ListenAndServe(":3333", nil)
	// if errors.Is(err, http.ErrServerClosed) {
	// 	fmt.Printf("server closed\n")
	// } else if err != nil {
	// 	fmt.Printf("error starting server: %s\n", err)
	// 	os.Exit(1)
	// }

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

	fmt.Println("Sync test end")
}

const keyServerAddr = "serverAddr"

func getRoot(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	fmt.Printf("%s: got / request\n", ctx.Value(keyServerAddr))

	mmSync := sync.GetSyncInstance()
	mmSync.InitializeStore()

	c, err := mmSync.StartSync()

	if err != nil {
		fmt.Println(fmt.Errorf("Start sync error: %s", err))
	}

	for result := range c {
		fmt.Println("Received result: ", result)
	}

	mmSync.CloseStore()

	io.WriteString(w, "This is my website!\n")
}

func getHello(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	fmt.Printf("%s: got / request\n", ctx.Value(keyServerAddr))
	io.WriteString(w, "Hello, HTTP!\n")
}
