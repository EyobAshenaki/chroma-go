package slack

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	chroma_go "github.com/EyobAshenaki/chroma-go"
	chroma "github.com/amikos-tech/chroma-go"
)

type PurposeDetail struct {
	Value string `json:"value"`
}

type Channel struct {
	Id          string        `json:"id"`
	Name        string        `json:"name"`
	Purpose     PurposeDetail `json:"purpose"`
	DateCreated int           `json:"created"`
}

type ChannelSpec struct {
	StoreAll  bool          `json:"store_all"`
	StoreNone bool          `json:"store_none"`
	StartDate time.Duration `json:"start_date"`
	EndDate   time.Duration `json:"end_date"`
}

type UserProfile struct {
	RealName    string `json:"real_name"`
	Name        string `json:"name"`
	AvatarImage string `json:"image_72"`
}

type Message struct {
	Id      string      `json:"client_msg_id"`
	Type    string      `json:"type"`
	Subtype string      `json:"subtype"`
	Text    string      `json:"text"`
	Time    string      `json:"ts"`
	User    UserProfile `json:"user_profile"`
}

type Slack struct {
	slackCollection  *chroma.Collection
	Channels         []Channel
	FilteredChannels map[string]ChannelSpec
}

var instance *Slack
var once sync.Once

func GetSlackInstance() *Slack {
	once.Do(func() {
		chromaClient := chroma_go.GetChromaInstance()

		newCollection, colError := chromaClient.GetOrCreateCollection("slack")
		if colError != nil {
			log.Fatalf("Error while creating / getting collection: %v \n", colError)
		}

		instance = &Slack{
			slackCollection: newCollection,
		}
	})

	return instance
}

func (slack *Slack) HandleZipUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("Upload started ...")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// The argument to FormFile must match the name attribute of the file input on the frontend
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create a new file in the uploads directory
	dst, err := os.Create(fmt.Sprintf("./%d%s", time.Now().Unix(), filepath.Ext(fileHeader.Filename)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy the uploaded file to the filesystem at the specified destination
	_, copyError := io.Copy(dst, file)
	if copyError != nil {
		http.Error(w, copyError.Error(), http.StatusInternalServerError)
		return
	}

	extractError := slack.extractDetailsFromZip(dst.Name())
	if extractError != nil {
		http.Error(w, extractError.Error(), http.StatusInternalServerError)
		return
	}

	// read extracted data from zip and store it slack.Channels
	readError := slack.readExtractedData()
	if readError != nil {
		http.Error(w, readError.Error(), http.StatusInternalServerError)
		return
	}

	// delete the zip file after extraction
	deleteError := os.Remove(dst.Name())
	if deleteError != nil {
		http.Error(w, deleteError.Error(), http.StatusInternalServerError)
		return
	}

	channelJson, jsonError := json.Marshal(slack.Channels)
	if jsonError != nil {
		http.Error(w, jsonError.Error(), http.StatusInternalServerError)
		return
	}

	io.Writer.Write(w, channelJson)
}

func (slack *Slack) HandleFilteredChannelData(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Make sure that the writer supports flushing.
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("error could not read request body: %v \n", err)
		http.Error(w, "Internal Server Error: could not read request body", http.StatusInternalServerError)
		return
	}

	unmarshalError := json.Unmarshal(requestBody, &slack.FilteredChannels)
	if unmarshalError != nil {
		log.Printf("error while trying to decode JSON: %v \n", unmarshalError)
		http.Error(w, "Internal Server Error: could not decode JSON", http.StatusInternalServerError)
		return
	}

	formattedJSON, _ := json.MarshalIndent(slack.FilteredChannels, "->", "  ")
	fmt.Println(string(formattedJSON))

	if len(slack.Channels) <= 0 {
		// read extracted data from zip and store it slack.Channels
		readError := slack.readExtractedData()
		if readError != nil {
			http.Error(
				w,
				fmt.Sprintf("slack zip file may not be uploaded. try uploading slack zip file: %v", readError.Error()),
				http.StatusInternalServerError,
			)
			return
		}
	}

	for channelId, channelSpec := range slack.FilteredChannels {
		channelName := ""
		msgStartDate, msgEndDate := time.Time{}, time.Time{}

		// get channel name for current channel
		for _, channel := range slack.Channels {
			if channel.Id == channelId {
				channelName = channel.Name
			}
		}

		log.Println("***********************************")
		log.Printf("Channel: %v \n", channelName)
		log.Println("***********************************")

		// set start and end time to get messages in between them
		if channelSpec.StoreNone {
			continue
		} else if channelSpec.StoreAll {
			msgStartDate = time.Unix(0, 0).UTC()
			msgEndDate = time.Now()
		} else {
			msgStartDate = time.Unix(int64(channelSpec.StartDate), 0).UTC()
			msgEndDate = time.Unix(int64(channelSpec.EndDate), 0).UTC()

			if msgStartDate.After(msgEndDate) {
				log.Printf("error - start date cannot be greater than end date: %v \n", err)

				http.Error(w, "error - start date cannot be greater than end date", http.StatusBadRequest)
				return
			}
		}

		file, err := os.Open(filepath.Join("extracted_slack_data", channelName))
		if err != nil {
			log.Printf("error while trying to open file: %v \n", err)
			http.Error(w, "error while trying to open file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		messageFiles, err := file.Readdirnames(0) // read all file names
		if err != nil {
			log.Printf("error while trying to read file names: %v \n", err)
			http.Error(w, "error while trying to read file names", http.StatusInternalServerError)
			return
		}
		fmt.Printf("List of files in %v: %v \n", channelName, messageFiles)

		if len(messageFiles) == 0 {
			continue
		}

		// Set the headers related to event streaming.
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		flusher.Flush()

		// get all files in the channel's folder (each file correspond to daily messages)
		for idx, messageFile := range messageFiles {
			// get the date from the file name
			msgDateStr := filepath.Base(messageFile)
			msgDateStr = msgDateStr[:len(msgDateStr)-5] + "T00:00:00Z"

			msgDate, dateParseError := time.Parse(time.RFC3339, msgDateStr)
			if dateParseError != nil {
				log.Fatalf("error while trying to parse date: %v \n", dateParseError)
				http.Error(w, "error while trying to parse date", http.StatusInternalServerError)
				return
			}

			log.Println("***********************************")
			log.Printf("Messages Date: %v \n", msgDate)
			log.Println("***********************************")

			// don't save the files that are out of the specified date range
			if msgDate.After(msgEndDate) || msgDate.Before(msgStartDate) {
				continue
			}

			// read the contents of the file (all messages sent in that channel in one day)
			messagesInFile := []Message{}
			extractJsonContentFromFile(filepath.Join(channelName, messageFile), &messagesInFile)

			// formattedMsgJSON, _ := json.MarshalIndent(messagesInFile, "->", "  ")
			// fmt.Println(string(formattedMsgJSON))

			// continue to the next file if not message are found in the current one
			if len(messagesInFile) <= 0 {
				continue
			}

			metadatas := []map[string]interface{}{}
			documents := []string{}
			ids := []string{}

			for _, message := range messagesInFile {
				// filter message based on type and subtype
				if message.Type != "message" || message.Id == "" || message.Text == "" {
					continue
				}

				// TODO: replace slack handles like user mentions with user's name
				// message.Text = replaceSlackHandles(message.Text)

				ids = append(ids, message.Id)
				documents = append(documents, message.Text)
				metadatas = append(metadatas, map[string]interface{}{
					"source":       "sl",
					"access":       "pub",
					"user_name":    message.User.RealName,
					"channel_name": channelName,
					"msg_date":     msgDate.Unix(),
				})
			}

			log.Printf("Upserting %v documents to collection \n", len(documents))
			log.Println()

			if len(documents) <= 0 || len(ids) <= 0 || len(metadatas) <= 0 {
				continue
			}

			_, upError := GetSlackInstance().slackCollection.Upsert(context.Background(), nil, metadatas, documents, ids)
			if upError != nil {
				log.Fatalf("Failed to upsert to chroma: %v \n", upError)
				http.Error(w, "Failed to upsert to chroma", http.StatusInternalServerError)
				return
			}

			/*
				Event:
					id: 1
					event: onProgress
					data: {"channel_id": 0.4}
			*/

			event := map[string]interface{}{
				"id":    fmt.Sprintf("%v - %v", channelId, idx),
				"event": "onProgress",
				"data":  map[string]interface{}{"channel_id": (idx + 1) / len(messageFiles)},
			}
			eventJson, _ := json.Marshal(event)

			io.Writer.Write(w, eventJson)

			flusher.Flush()
		}
	}

	event := map[string]interface{}{
		"id":    "100",
		"event": "done",
		"data":  map[string]interface{}{},
	}
	eventJson, _ := json.Marshal(event)

	io.Writer.Write(w, eventJson)

	flusher.Flush()
}

// extract details from ZIP file
func (slack *Slack) extractDetailsFromZip(zipFilePath string) error {
	fileDestinationFolder := "extracted_slack_data"

	// Opening the zip file
	openedZipFile, openError := zip.OpenReader(zipFilePath)

	log.Printf("Extracting zip file: %v \n", zipFilePath)

	if openError != nil {
		return fmt.Errorf("error while trying to open zip file: %v", openError)
	}
	defer openedZipFile.Close()

	for _, fileInZip := range openedZipFile.File {
		filePath := filepath.Join(fileDestinationFolder, fileInZip.Name)
		log.Println("unzipping file", filePath)

		// if the file is empty directory, create a directory
		if fileInZip.FileInfo().IsDir() {
			log.Printf("Directory: %s \n", fileInZip.Name)
			dirError := os.MkdirAll(filePath, os.ModePerm)
			if dirError != nil {
				return fmt.Errorf("error while trying to create directory: %v \n", dirError)
			}

			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			if err != nil {
				return fmt.Errorf("error while trying to create directory: %v", err)
			}
		} else {
			destinationFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fileInZip.Mode())
			if err != nil {
				return fmt.Errorf("error while trying to open file with flag: %v", err)
			}
			defer destinationFile.Close()

			//Opening the file and copy it's contents
			fileInArchive, err := fileInZip.Open()
			if err != nil {
				return fmt.Errorf("error while trying to open file: %v", err)
			}
			defer fileInArchive.Close()

			if _, err := io.Copy(destinationFile, fileInArchive); err != nil {
				return fmt.Errorf("error while trying to copy file to %v: %v", destinationFile, err)
			}
		}
	}

	return nil
}

func (slack *Slack) readExtractedData() error {
	file, err := os.Open("extracted_slack_data")
	if err != nil {
		return fmt.Errorf("error while trying to open extracted_slack_data folder: %v", err)
	}
	defer file.Close()

	list, err := file.Readdirnames(0) // read all file names
	if err != nil {
		return fmt.Errorf("error while trying to read file names from extracted_slack_data folder: %v", err)
	}

	for _, name := range list {
		if name == "channels.json" {
			return extractJsonContentFromFile(name, &slack.Channels)
		}
	}

	return nil
}

func extractJsonContentFromFile(fileName string, receiverPtr interface{}) error {
	jsonFile, jsonError := os.Open(filepath.Join("extracted_slack_data", fileName))
	if jsonError != nil {
		return fmt.Errorf("error while tying to open extracted file, channels.json: %v", jsonError)
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	unmarshalError := json.Unmarshal(byteValue, receiverPtr)
	if unmarshalError != nil {
		return fmt.Errorf("error while trying to decode JSON: %v", unmarshalError)
	}

	return nil
}
