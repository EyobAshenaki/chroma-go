package chroma_go_test

import (
	"log"
	"testing"
)

func TestChromaGo(t *testing.T) {
	t.Log("Test start")

	// client := chroma_go.GetChromaInstance()

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

	// ----------------------------------------------------

	// mmChannelIds := []interface{}{
	// 	// "9unddga5zin75goadjbwipz9tr",
	// 	// "jdwhwtegcpb3tei871ohw9sz9y",
	// }

	// client.Query("testing", mmChannelIds)

	// ------------------------------------------

	var Reset = "\033[0m"
	var Red = "\033[31m"
	// var Green = "\033[32m"
	// var Yellow = "\033[33m"
	var Blue = "\033[34m"
	// var Purple = "\033[35m"
	// var Cyan = "\033[36m"
	// var Gray = "\033[37m"
	// var White = "\033[97m"

	// Reset helps so that the color doesn't bleed in to the next print statement
	println(Red + "This is Blue" + Reset)
	log.Println(Blue + "This is Blue" + Reset)

	t.Log("Test end")
}
