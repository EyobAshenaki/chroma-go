package sync_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	sse "github.com/EyobAshenaki/chroma-go/server_side_event"
	"github.com/EyobAshenaki/chroma-go/sync"
	"github.com/rs/cors"
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
	mux := http.NewServeMux()

	broker := sse.NewSSEServer()

	mux.Handle("/sync/start", broker)

	mux.Handle("/sync/stop", broker)

	mux.HandleFunc("/sync/is_fetch_in_progress", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		mmSync := sync.GetSyncInstance()

		isFetchInProgress, err := mmSync.GetIsFetchInProgress()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		io.Writer.Write(w, []byte(strconv.FormatBool(isFetchInProgress)))
	})

	mux.HandleFunc("/sync/is_sync_in_progress", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		mmSync := sync.GetSyncInstance()

		isSyncInProgress, err := mmSync.GetIsSyncInProgress()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		io.Writer.Write(w, []byte(strconv.FormatBool(isSyncInProgress)))
	})

	mux.HandleFunc("/sync/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Make sure that the writer supports flushing.
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		// Set the headers related to event streaming.
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		flusher.Flush()

		previousIsSyncInProgress := true
		previousIsFetchInProgress := false

		mmSync := sync.GetSyncInstance()

		for idx := 0; ; idx++ {
			isSyncInProgress, err := mmSync.GetIsSyncInProgress()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			isFetchInProgress, err := mmSync.GetIsFetchInProgress()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if isSyncInProgress != previousIsSyncInProgress || isFetchInProgress != previousIsFetchInProgress {
				previousIsSyncInProgress = isSyncInProgress
				previousIsFetchInProgress = isFetchInProgress

				io.Writer.Write(w, []byte(fmt.Sprintf("id: %v\n", idx)))
				io.Writer.Write(w, []byte("event: onStatusChange\n"))
				io.Writer.Write(w, []byte("data: {\"is_sync_in_progress\": "+strconv.FormatBool(isSyncInProgress)+", \"is_fetch_in_progress\": "+strconv.FormatBool(isFetchInProgress)+"}\n"))
				io.Writer.Write(w, []byte("\n"))
				flusher.Flush()
			}
		}
	})

	mux.HandleFunc("/sync/fetch_interval", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" && r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		mmSync := sync.GetSyncInstance()

		if r.Method == "GET" {
			fetchInterval, err := mmSync.GetFetchInterval()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			io.Writer.Write(w, []byte(strconv.Itoa(fetchInterval)))
		}

		if r.Method == "POST" {
			hasFetchInterval := r.URL.Query().Has("fetch_interval")
			if !hasFetchInterval {
				http.Error(w, "fetch_interval query field not found", http.StatusBadRequest)
				return
			}

			fetchInterval := r.URL.Query().Get("fetch_interval")
			if fetchInterval == "" {
				http.Error(w, "fetch_interval query field is empty", http.StatusBadRequest)
				return
			}

			fetchIntervalInt, parseError := strconv.Atoi(fetchInterval)
			if parseError != nil {
				http.Error(w, parseError.Error(), http.StatusBadRequest)
				return
			}

			fetchDuration := time.Duration(fetchIntervalInt) * time.Hour

			err := mmSync.UpdateFetchInterval(fetchDuration)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			io.Writer.Write(w, []byte("Fetch interval updated successfully"))
		}
	})

	mux.HandleFunc("/sync/last_fetched_at", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		mmSync := sync.GetSyncInstance()

		if r.Method == "GET" {
			lastFetchedAt, err := mmSync.GetLastFetchedAt()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			io.Writer.Write(w, []byte(strconv.Itoa(int(lastFetchedAt.UnixMilli()))))
		}
	})

	mux.HandleFunc("/sync/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		isReset, err := sync.ResetVectorStore()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		io.Writer.Write(w, []byte(fmt.Sprintf("Vector store reset successfully: %v", isReset)))
	})

	handler := cors.Default().Handler(mux)

	err := http.ListenAndServe(":3333", handler)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}

	// ----------------------------------------------

	fmt.Println("Sync test end")
}
