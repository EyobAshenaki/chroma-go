package search_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/EyobAshenaki/chroma-go/search"
	"github.com/rs/cors"
)

func TestSearch(t *testing.T) {
	t.Log("Test start")

	mux := http.NewServeMux()
	mux.HandleFunc("/search", handleSearch)

	handler := cors.Default().Handler(mux)

	err := http.ListenAndServe(":4501", handler)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}

	t.Log("Test end")
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	hasQuery := r.URL.Query().Has("query")
	if !hasQuery {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	query := r.URL.Query().Get("query")

	searchResponse := search.Search(query)

	searchResponseJSON, err := json.Marshal(searchResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	io.Writer.Write(w, searchResponseJSON)

	responseJSON, _ := json.MarshalIndent(searchResponse, "->", "  ")
	fmt.Println(string(responseJSON))
}
