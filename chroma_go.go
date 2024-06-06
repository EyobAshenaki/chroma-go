package chroma_go

import (
	"context"
	"fmt"

	chroma "github.com/amikos-tech/chroma-go"
)

func Connect() string {
	fmt.Println("Connecting to Chroma...")

	client, err := chroma.NewClient("localhost:8000")

	if err != nil {
		fmt.Println("Error connecting to Chroma:", err)
		return "Error connecting to Chroma"
	}

	fmt.Println("Connected to Chroma:", client)

	// ctx := context.TODO()
	ctx := context.Background()
	ctx = context.WithValue(ctx, "name", "eyob")
	dosth(ctx)

	return "Connected"
}

func dosth(ctx context.Context) {
	fmt.Printf("Doing something! My name is %s\n", ctx.Value("name"))
}
