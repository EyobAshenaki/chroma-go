package chroma_go

import (
	"fmt"

	chroma "github.com/amikos-tech/chroma-go"
)

func Connect() string {
	fmt.Println("Connecting to Chroma...")

	client, connectionErr := chroma.NewClient("http://localhost:8001")

	if connectionErr != nil {
		fmt.Println("Error connecting to Chroma:", connectionErr)
		return "Error connecting to Chroma"
	}

	fmt.Println("Connected to Chroma:", client)

	// // ctx := context.TODO()
	// ctx := context.Background()
	// // ctx = context.WithValue(ctx, "name", "eyob")
	// // dosth(ctx)

	// count, err := client.CountCollections(ctx)

	// if err != nil {
	// 	fmt.Println("Error Connection Count:", err)
	// 	return "Error Connection Count"
	// }

	// fmt.Println("Connected to Chroma:", count)

	// res, err := client.Heartbeat(ctx)

	// if err != nil {
	// 	fmt.Println("Error Heartbeat:", err)
	// 	return "Error Heartbeat"
	// }

	// fmt.Println("Connected to Chroma:", res["nanosecond heartbeat"])

	return "Connected"
}

// func dosth(ctx context.Context) {
// 	fmt.Printf("Doing something! My name is %s\n", ctx.Value("name"))
// }
