package chroma_go

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"

	chroma "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/collection"
	"github.com/amikos-tech/chroma-go/types"
	"github.com/amikos-tech/chroma-go/where"
	"github.com/joho/godotenv"
)

var instance ChromaClient

type ChromaClient struct {
	client *chroma.Client
}

var once sync.Once

func GetChromaInstance() *ChromaClient {
	fmt.Println("Connecting to Chroma ...")

	once.Do(func() {
		instance.client, _ = chroma.NewClient("http://localhost:8000")
	})

	ctx := context.Background()

	_, err := instance.client.Heartbeat(ctx)

	if err != nil {
		log.Println("... Failed to connect")
		log.Fatalf("Error while  trying to connect to chroma %v \n", err)
	}

	fmt.Println("... Connected to Chroma")

	return &instance
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
		context.Background(),
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

func (chromaClient *ChromaClient) Query(query string, mmChannelIds []interface{}) chroma.QueryResults {
	mattermostCollectionType := "mattermost"
	slackCollectionType := "slack"

	// get the mattermost collections
	mattermostCollection, mmError := chromaClient.GetOrCreateCollection(mattermostCollectionType)
	if mmError != nil {
		log.Fatalf("error getting mattermost collection: %v", mmError)
	}

	// get the slack collections
	slackCollection, slkError := chromaClient.GetOrCreateCollection(slackCollectionType)
	if slkError != nil {
		log.Fatalf("error getting slack collection: %v", slkError)
	}

	// list of channel ids user belongs to
	channelIds := mmChannelIds

	queryTexts := []string{query}
	n_results := int32(5)
	inExpression := map[string]interface{}{}

	if len(channelIds) > 0 {
		inExpr, whrError := where.Where(where.In("channel_id", channelIds))
		if whrError != nil {
			log.Fatalf("error while building where clause: %v \n", whrError)
		}
		inExpression = inExpr
	}

	// query the mattermost collection
	mmResponse, mmResError := mattermostCollection.Query(
		context.Background(),
		queryTexts,
		n_results,
		inExpression,
		nil,
		nil,
	)
	if mmResError != nil {
		log.Fatalf("error while querying mattermost collection: %v \n", mmResError)
	}

	// query the slack collection
	slkResponse, slkResError := slackCollection.Query(
		context.Background(),
		queryTexts,
		n_results,
		nil,
		nil,
		nil,
	)
	if slkResError != nil {
		log.Fatalf("error while querying slack collection: %v \n", slkResError)
	}

	// log the response
	responseJSON, _ := json.MarshalIndent(slkResponse, "->", "  ")
	fmt.Println(string(responseJSON))

	// combine the slack and mattermost responses
	response := chroma.QueryResults{
		Documents: append(mmResponse.Documents, slkResponse.Documents...),
		Ids:       append(mmResponse.Ids, slkResponse.Ids...),
		Metadatas: append(mmResponse.Metadatas, slkResponse.Metadatas...),
		Distances: append(mmResponse.Distances, slkResponse.Distances...),
	}

	// filter out the combined results with distances below a certain threshold defined in "max_chroma_distance"

	// TODO: replace this variable with the user defined one
	max_chroma_distance := float32(0.19)

	filteredResponse := chroma.QueryResults{
		Documents: make([][]string, len(response.Documents)),
		Ids:       make([][]string, len(response.Ids)),
		Metadatas: make([][]map[string]interface{}, len(response.Metadatas)),
		Distances: make([][]float32, len(response.Distances)),
	}

	for i, distances := range response.Distances {
		for j, distance := range distances {
			if distance < max_chroma_distance {
				// Ensure that the sub-slices are initialized
				if filteredResponse.Documents[i] == nil {
					filteredResponse.Documents[i] = []string{}
				}
				if filteredResponse.Ids[i] == nil {
					filteredResponse.Ids[i] = []string{}
				}
				if filteredResponse.Metadatas[i] == nil {
					filteredResponse.Metadatas[i] = []map[string]interface{}{}
				}
				if filteredResponse.Distances[i] == nil {
					filteredResponse.Distances[i] = []float32{}
				}

				filteredResponse.Documents[i] = append(filteredResponse.Documents[i], response.Documents[i][j])
				filteredResponse.Ids[i] = append(filteredResponse.Ids[i], response.Ids[i][j])
				filteredResponse.Metadatas[i] = append(filteredResponse.Metadatas[i], response.Metadatas[i][j])
				filteredResponse.Distances[i] = append(filteredResponse.Distances[i], response.Distances[i][j])
			}
		}
	}

	// return if no relevant data is found
	if len(filteredResponse.Documents) == 0 {
		fmt.Println("No relevant data found")
		return filteredResponse
	}

	// // print the response
	// responseJSON, _ := json.MarshalIndent(filteredResponse, "->", "  ")
	// fmt.Println(string(responseJSON))

	return filteredResponse
}
