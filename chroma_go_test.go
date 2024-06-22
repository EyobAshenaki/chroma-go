package chroma_go_test

import (
	"testing"

	chroma_go "github.com/EyobAshenaki/chroma-go"
)

func TestChromaGo(t *testing.T) {
	t.Log("Test start")

	client := chroma_go.GetChromaInstance()

	// collection, colError := client.GetOrCreateCollection("mattermost")
	// if colError != nil {
	// 	log.Println(colError)
	// 	t.FailNow()
	// }

	// log.Println(collection)

	// documentCount, countError := collection.Count(context.Background())
	// if countError != nil {
	// 	log.Printf("error counting document in collection: %v", countError)
	// }

	// log.Printf("Documents count: %v \n", documentCount)

	mmChannelIds := []interface{}{
		// "9unddga5zin75goadjbwipz9tr",
		// "jdwhwtegcpb3tei871ohw9sz9y",
	}

	client.Query("testing", mmChannelIds)

	t.Log("Test end")
}
