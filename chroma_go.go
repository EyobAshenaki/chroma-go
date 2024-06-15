package chroma_go

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"

	chroma "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/collection"
	openai "github.com/amikos-tech/chroma-go/openai"
	"github.com/amikos-tech/chroma-go/types"
	"github.com/joho/godotenv"
)

var instance ChromaClient

type ChromaClient struct {
	client *chroma.Client
}

var once sync.Once

func Connect() (*ChromaClient, error) {
	fmt.Println("Connecting to Chroma ...")

	once.Do(func() {
		instance.client, _ = chroma.NewClient("http://localhost:8000")
	})

	ctx := context.Background()

	_, err := instance.client.Heartbeat(ctx)

	if err != nil {
		fmt.Println("... Failed to connect")
		return nil, errors.New("connection refused")
	}

	fmt.Println("... Connected to Chroma")

	return &instance, nil
}

func (chromaClient *ChromaClient) GetOrCreateCollection(collectionType string) error {
	godotenv.Load()

	if collectionType == "" {
		collectionType = "mattermost"
	}

	if chromaClient == nil {
		return errors.New("chroma db is not connected")
	}

	openaiEvalFunc, err := openai.NewOpenAIEmbeddingFunction(os.Getenv("OPENAI_API_KEY"))
	if err != nil {
		log.Fatalf("Error creating OpenAI embedding function: %s \n", err)
	}

	collectionName := collectionType + "messages"

	newCollection, err := chromaClient.client.NewCollection(
		context.TODO(),
		collection.WithName(collectionName),
		collection.WithMetadata("key1", "value1"),
		collection.WithEmbeddingFunction(openaiEvalFunc),
		collection.WithHNSWDistanceFunction(types.COSINE),
		collection.WithCreateIfNotExist(true),
	)
	if err != nil {
		log.Fatalf("Error creating collection: %s \n", err)
	}

	// TODO: Create a record set and add the data that I want to embed

	// Create a new record set with to hold the records to insert
	recordSet, err := types.NewRecordSet(
		types.WithEmbeddingFunction(openaiEvalFunc),
		types.WithIDGenerator(types.NewULIDGenerator()),
	)
	if err != nil {
		log.Fatalf("Error creating record set: %s \n", err)
	}

	// Add a few records to the record set
	recordSet.WithRecord(types.WithDocument("My name is John. And I have two dogs."), types.WithMetadata("key1", "value1"))
	recordSet.WithRecord(types.WithDocument("My name is Jane. I am a data scientist."), types.WithMetadata("key2", "value2"))

	// log.Println("Record set: ", recordSet)
	// log.Println("Collection: ", newCollection)

	// Build and validate the record set (this will create embeddings if not already present)
	_, err = recordSet.BuildAndValidate(context.TODO())
	if err != nil {
		log.Fatalf("Error validating record set: %s \n", err)
	}

	// // Add the records to the collection
	// _, err = newCollection.AddRecords(context.Background(), recordSet)
	// if err != nil {
	// 	log.Fatalf("Error adding documents: %s \n", err)
	// }

	// TODO: Next get the embeddings and other necessary data from the record set and upsert then to chroma

	// Count the number of documents in the collection
	countDocs, qrerr := newCollection.Count(context.TODO())
	if qrerr != nil {
		log.Fatalf("Error counting documents: %s \n", qrerr)
	}

	log.Println("Number of documents is: ", countDocs)

	return nil
}
