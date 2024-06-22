package search

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	chroma_go "github.com/EyobAshenaki/chroma-go"
)

// TODO: implement an authentication method to replace the use of this token
const token = "Bearer wz1rgk853b8tpbg18aiux3cdae"

// TODO: replace this with a dynamic value from the .env file
const mmAPI = "http://localhost:8065/api/v4"

type UserDetail struct {
	Id        string `json:"id"`
	UserName  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type TeamDetail struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type ChannelDetail struct {
	Id          string `json:"id"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	TeamId      string `json:"team_id"`
}

type PostDetail struct {
	Id        string `json:"id"`
	Message   string `json:"message"`
	UserId    string `json:"user_id"`
	Type      string `json:"type"`
	UpdateAt  int64  `json:"update_at"`
	DeleteAt  int64  `json:"delete_at"`
	ChannelId string `json:"channel_id"`
}

type MetadataSchema struct {
	UserId      string `json:"user_id"` // (not necessary for slack)
	UserName    string `json:"user_name"`
	UserDmLink  string `json:"user_dm_link"` // (not necessary for slack)
	ChannelName string `json:"channel_name"`
	ChannelLink string `json:"channel_link"` // (not necessary for slack)
	Message     string `json:"message"`
	MessageLink string `json:"message_link"` // (not necessary for slack)
	Time        string `json:"time"`
	Source      string `json:"source"`
	Access      string `json:"access"`
	Score       string `json:"score"`
}

type SearchRespnse struct {
	Metadata    MetadataSchema `json:"metadata"`
	LLMResponse string         `json:"llm_response"`
}

func Search(query string) SearchRespnse {
	log.Println("Search started ...")

	// TODO: replace this with a dynamic value
	userId := "3fx4agxqnbd4pctix7h66cj6m"

	// get list of channels the user belongs to
	// TODO: implement this function
	mmChannelIds := getUserChannels(userId)

	log.Printf("number of channels: %v", len(mmChannelIds))

	client := chroma_go.GetChromaInstance()

	// search the chroma collection using the query provided while filtering the result by channel_id the user belongs to
	response := client.Query(query, mmChannelIds)

	// join the documents from the chroma result using "\n" and store it as a context to feed it to LLM
	llmContext := ""
	for _, documents := range response.Documents {
		for _, document := range documents {
			llmContext += document + "\n"
		}
	}

	formattedMetadatas := []map[string]interface{}{}
	for _, metadatas := range response.Metadatas {
		formattedMetadatas = append(formattedMetadatas, metadatas...)
	}

	formattedIds := []string{}
	for _, ids := range response.Ids {
		formattedIds = append(formattedIds, ids...)
	}

	formattedDistances := []float32{}
	for _, distances := range response.Distances {
		formattedDistances = append(formattedDistances, distances...)
	}

	mmMetadataDetails := MetadataSchema{}
	for idx, formattedMetadata := range formattedMetadatas {
		// format the metadata using the metadata schema for mattermost data
		if formattedMetadata["source"].(string) == "mm" {
			// get message details
			postDetail := getPostDetails(formattedIds[idx])

			//get user details
			userDetail := getUserDetails(formattedMetadata["user_id"].(string))

			//get channel details
			channelDetail := getChannelDetails(formattedMetadata["channel_id"].(string))

			// get team details
			teamDetail := getTeamDetails(channelDetail.TeamId)

			linkURL := "http://localhost:8065/" + teamDetail.Name

			// format the metadata
			mmMetadataDetails = MetadataSchema{
				UserId:      userDetail.Id,
				UserName:    userDetail.UserName,
				UserDmLink:  linkURL + "/messages/@" + userDetail.UserName,
				ChannelName: channelDetail.Name,
				ChannelLink: linkURL + "/channels/" + channelDetail.Name,
				Message:     postDetail.Message,
				MessageLink: linkURL + "/pl/" + postDetail.Id,
				Time:        time.Unix(postDetail.UpdateAt/1000, 0).Format(time.RFC822),
				Source:      formattedMetadata["access"].(string),
				Access:      formattedMetadata["source"].(string),
				Score:       fmt.Sprintf("%f", (1 - formattedDistances[idx])),
			}
		}
		// TODO: format the metadata using the metadata schema for slack data

	}

	// TODO: replace this with a dynamic value
	withLLM := false // a boolean used to check if the user wants an llm response
	llmResponse := ""
	if llmContext == "" && mmMetadataDetails == (MetadataSchema{}) {
		llmResponse = "Unable to find conversations related to your query."
	} else if withLLM {
		// TODO: implement this function
		llmResponse = getLLMResponse(llmContext, query)
	}

	return SearchRespnse{
		Metadata:    mmMetadataDetails,
		LLMResponse: llmResponse,
	}
}

func getLLMResponse(llmContext, query string) string {
	// TODO: implement this

	return "llm response"
}

func getDetails(reqUrl string, returnDetail interface{}) {
	req, err := http.NewRequest(http.MethodGet, reqUrl, nil)
	if err != nil {
		log.Fatalf("client: could not create request: %s\n", err)
	}

	// TODO: implement an authentication method to replace the use of this token
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	client := http.Client{
		Timeout: 10 * time.Second,
	}
	response, err := client.Do(req)
	if err != nil {
		log.Fatalf("client: error making http request: %s\n", err)
	}
	// defer response.Body.Close()

	if response.StatusCode != 200 {
		log.Fatalf("client: Failed to fetch from %v. Status code: %d", reqUrl, response.StatusCode)
	}

	// automatically filters the response body to only include the fields
	// specified in Channels struct by json tags
	err = json.NewDecoder(response.Body).Decode(returnDetail)
	if err != nil {
		log.Fatalf("client: could not decode json: %s\n", err)

	}
}

func getTeamDetails(teamId string) (teamDetail TeamDetail) {
	reqUrl := mmAPI + "/teams/" + teamId

	getDetails(reqUrl, &teamDetail)

	return teamDetail
}

func getChannelDetails(channelId string) (channelDetail ChannelDetail) {
	reqUrl := mmAPI + "/channels/" + channelId

	getDetails(reqUrl, &channelDetail)

	return channelDetail
}

func getUserDetails(userId string) (userDetail UserDetail) {
	reqUrl := mmAPI + "/users/" + userId

	getDetails(reqUrl, &userDetail)

	return userDetail
}

func getPostDetails(postId string) (postDetail PostDetail) {
	reqUrl := mmAPI + "/posts/" + postId

	getDetails(reqUrl, &postDetail)

	return postDetail
}

func getUserChannels(userId string) []interface{} {
	// TODO: implement this

	// handle the errors in here using log.fatal

	return []interface{}{
		// "9unddga5zin75goadjbwipz9tr",
		// "jdwhwtegcpb3tei871ohw9sz9y",
	}
}
