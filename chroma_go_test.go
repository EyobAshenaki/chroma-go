package chroma_go_test

import (
	"context"
	"log"
	"testing"

	chroma_go "github.com/EyobAshenaki/chroma-go"
)

func TestChromaGo(t *testing.T) {
	t.Log("Test start")
	// godotenv.Load()

	// log.Println(os.LookupEnv("OPENAI_API_KEY"))

	client, err := chroma_go.GetChromaInstance()
	if err != nil {
		log.Println(err)
		t.FailNow()
	}

	collection, colError := client.GetOrCreateCollection("mattermost")
	if colError != nil {
		log.Println(colError)
		t.FailNow()
	}

	log.Println(collection)

	documentCount, countError := collection.Count(context.Background())
	if countError != nil {
		log.Printf("error counting document in collection: %v", countError)
	}

	log.Printf("Documents count: %v \n", documentCount)

	t.Log("Test end")
}
