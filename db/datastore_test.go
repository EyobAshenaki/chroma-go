package db_test

import (
	"fmt"
	"testing"

	"github.com/EyobAshenaki/chroma-go/db"
)

func TestDB(t *testing.T) {
	store := db.GetDataStore("mattermost")

	putErr := store.Put("plugin", "sync_interval", []byte("60"))
	if putErr != nil {
		fmt.Println("Error: put failed with - ", putErr)
	}

	val, getErr := store.Get("plugin", "sync_interval")
	if getErr != nil {
		fmt.Println("Error: get failed with - ", getErr)
	}

	fmt.Println("Value: ", string(val))

}
