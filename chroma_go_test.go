package chroma_go_test

import (
	"testing"

	chroma_go "github.com/EyobAshenaki/chroma-go"
)

func TestChromaGo(t *testing.T) {
	t.Log("Test start")
	res := chroma_go.Connect()

	if res != "Connected" {
		t.Errorf("Expected 'Connected', got '%s'", res)
		// t.Fail()
	}
	// t.Log(res)
	t.Log("Test end")
}
