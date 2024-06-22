package search_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/EyobAshenaki/chroma-go/search"
)

func TestSearch(t *testing.T) {
	t.Log("Test start")

	searchResponse := search.Search("testing")

	responseJSON, _ := json.MarshalIndent(searchResponse, "->", "  ")
	fmt.Println(string(responseJSON))

	t.Log("Test end")
}
