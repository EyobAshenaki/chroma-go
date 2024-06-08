package sync_test

import (
	"fmt"
	"testing"

	sync "github.com/EyobAshenaki/chroma-go/sync"
)

func TestSyncStart(t *testing.T) {
	// var wg sync.WaitGroup
	// wg.Add(1)

	// go func() {
	// 	defer wg.Done()
	// 	fmt.Println("Hello")
	// }()

	// wg.Wait()

	fmt.Println("Sync test start")

	if err := sync.Start(); err != nil {
		t.Errorf("Start sync error: %s", err)
	}

	fmt.Println("Sync test end")
}
