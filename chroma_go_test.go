package chroma_go_test

import (
	"log"
	"testing"

	chroma_go "github.com/EyobAshenaki/chroma-go"
)

func TestChromaGo(t *testing.T) {
	t.Log("Test start")
	// godotenv.Load()

	// log.Println(os.LookupEnv("OPENAI_API_KEY"))

	client, err := chroma_go.Connect()
	if err != nil {
		log.Println(err)
		t.FailNow()
	}

	colnErr := client.GetOrCreateCollection("mattermost")
	if colnErr != nil {
		log.Println(colnErr)
		t.FailNow()
	}

	// ctx := context.Background()

	// count, err := client.CountCollections(ctx)
	// if err != nil {
	// 	log.Println(err)
	// }

	// log.Printf("Collections found: %v\n", count)

	t.Log("Test end")
}
