package sync

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/EyobAshenaki/chroma-go/sync"
)

type MessageChan chan [][]byte

type Broker struct {
	clients        map[MessageChan]bool
	newClients     chan MessageChan
	closingClients chan MessageChan
	message        MessageChan
	sync           *sync.Sync
	ctx            context.Context
	ctxCancel      context.CancelFunc
}

// Spawn a go routine handles the addition & removal of
// clients, as well as the broadcasting of messages out
// to clients that are currently attached.
func (b *Broker) listen() {
	go func() {
		for {
			select {
			case s := <-b.newClients:
				b.clients[s] = true

				fmt.Println("***")
				log.Println("Added new client")
				fmt.Println("***")
			case s := <-b.closingClients:
				delete(b.clients, s)
				close(s)

				fmt.Println("***")
				log.Println("Removed client")
				fmt.Println("***")
			case msg := <-b.message:
				for client := range b.clients {
					client <- msg
				}

				fmt.Println("***")
				log.Printf("Broadcast message to %d clients", len(b.clients))
				fmt.Println("***")
			}
		}
	}()
}

// Implement the http.Handler interface.
// This allows us to wrap HTTP handlers
// http://golang.org/pkg/net/http/#Handler
func (b *Broker) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Make sure that the writer supports flushing.
	flusher, ok := rw.(http.Flusher)
	if !ok {
		http.Error(rw, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// Set the headers related to event streaming.
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel in which the current client receives
	// messages when they occur.
	messageChan := make(chan [][]byte)

	// Add this client to the map of those that should
	// receive updates
	b.newClients <- messageChan

	// // Remove this client when this handler exits
	// defer func() {
	// 	log.Println("Removing all connections")
	// 	b.closingClients <- messageChan
	// }()

	if req.URL.Path == "/sync/stop" {
		log.Printf("stop sync: %v\n", req.URL.Path)
		b.ctxCancel()
		b.sync.StopTicker()
	}

	// Listen to connection close
	notify := req.Context().Done()

	// Remove from active clients when the connection closes
	go func() {
		<-notify
		b.closingClients <- messageChan
	}()

	var percentageChan chan [][]byte

	if b.sync == nil {
		log.Println("Initializing sync...")

		// Initialize sync
		b.sync = sync.GetSyncInstance()
		b.sync.InitializeStore()

		log.Println(b.sync.IsTickerNil())
	}

	if b.sync.IsTickerNil() {
		log.Println("Sync is not running...")

		percentageChan = make(chan [][]byte)

		ctxWithCancel, cancelCtx := context.WithCancel(context.Background())

		b.ctx = ctxWithCancel
		b.ctxCancel = cancelCtx

		go func() {
			go func() {
				defer b.sync.StopSync()
				err := b.sync.StartSync(ctxWithCancel, percentageChan)
				if err != nil {
					log.Println(fmt.Errorf("error while syncing: %s", err))
					http.Error(
						rw,
						fmt.Sprintf("error while syncing: %s", err),
						http.StatusInternalServerError,
					)
					return
				}
			}()

			for {
				select {
				case <-ctxWithCancel.Done():
					log.Println("Stop sync!")

					var response [][]byte

					response = append(response, []byte("event: stop\n"))
					response = append(response, []byte("data: {}\n"))
					response = append(response, []byte("\n"))

					b.message <- response

					b.sync.SetTickerNil()
					return
				case percent, ok := <-percentageChan:
					if !ok {
						log.Printf("Percentage channel closed!")
						return
					}

					fmt.Printf("Syncing posts... %v%% complete\n", string(percent[1]))
					fmt.Println()
					fmt.Println("-------------------------------------")
					fmt.Println()

					b.message <- percent
				}
			}
		}()
	}

	// Block and wait for messages to be broadcasted
	for {
		// Get the message when broadcasted
		msgs, open := <-messageChan

		// If our channel is closed, the client must have disconnected so break
		if !open {
			break
		}

		for _, msg := range msgs {
			// Write data to the ResponseWriter
			io.Writer.Write(rw, msg)
		}

		// Flush/send the data immediately instead of buffering it for later
		flusher.Flush()
	}

	// Done.
	log.Println("Finished HTTP request at ", req.URL.Path)
}

// Broker factory
func NewSSEServer() (broker *Broker) {
	// Instantiate a broker
	broker = &Broker{
		clients:        make(map[MessageChan]bool),
		newClients:     make(chan MessageChan),
		closingClients: make(chan MessageChan),
		message:        make(MessageChan),
	}

	// Set it running - listening and broadcasting events
	go broker.listen()

	return broker
}
