package chroma_go

import (
	"context"
	"errors"
	"fmt"
	"sync"

	chroma "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/collection"
	"github.com/amikos-tech/chroma-go/types"
	"github.com/joho/godotenv"
)

var instance ChromaClient

type ChromaClient struct {
	client *chroma.Client
}

var once sync.Once

func GetChromaInstance() (*ChromaClient, error) {
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

func (chromaClient *ChromaClient) GetOrCreateCollection(collectionType string) (*chroma.Collection, error) {
	godotenv.Load()

	if collectionType == "" {
		collectionType = "mattermost"
	}

	if chromaClient == nil {
		return nil, errors.New("chroma db is not connected")
	}

	collectionName := collectionType + "_messages"
	metadatas := map[string]interface{}{}
	embeddingFunction := types.NewConsistentHashEmbeddingFunction()

	// Creates new collection, if the collection doesn't exist
	// Returns a collection, if the collection exists
	newCollection, err := chromaClient.client.NewCollection(
		context.TODO(),
		collection.WithName(collectionName),
		collection.WithMetadatas(metadatas),
		collection.WithEmbeddingFunction(embeddingFunction),
		collection.WithHNSWDistanceFunction(types.COSINE),
		collection.WithCreateIfNotExist(true),
	)
	if err != nil {
		return nil, err
	}

	return newCollection, nil
}
