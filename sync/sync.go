package sync

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

// var token string
// var mmAPI string
const token = "Bearer yokb6gwehpfhmn64f9k63r4xiw"
const mmAPI = "http://localhost:8065/api/v4"

type ChannelProp string

type PostResponse struct {
	Order          []string
	Posts          map[string]interface{}
	PreviousPostId string
}

func Start() error {
	fmt.Println("Start syncing...")
	// token = "Bearer yokb6gwehpfhmn64f9k63r4xiw"
	// mmAPI = "http://localhost:8065/api/v4"

	isSyncInProgress := false

	// if syncing is in progress return nothing
	if isSyncInProgress {
		return nil
	}

	// set syncing to true so no other sync can start
	isSyncInProgress = true

	//save the time where syncing started
	startSyncTime := time.Now()

	// get last synced time from db
	lastSyncedTimeInMilliseconds := getLastSyncedTime().UnixMilli()

	// get total fetched posts from db
	totalFetchedPosts := getTotalFetchedPosts()

	// declare a dict to store request parameters
	var params map[string]interface{}

	// Assign the since property in the request param dict to get all posts since that time.
	// if the since property is not defined all posts will be fetched from MM db
	if lastSyncedTimeInMilliseconds != 0 && totalFetchedPosts != 0 {
		params["since"] = lastSyncedTimeInMilliseconds
	}

	// TODO: remove this afa
	channelProps := []string{"id", "type", "display_name", "total_msg_count"}

	// Get all channels' data
	channels, err := getAllChannels(channelProps...)

	if err != nil {
		return err
	}

	// Calculate the total posts
	totalPosts := 0

	for _, channel := range channels {
		totalPosts += channel["total_msg_count"].(int)
	}

	// Get the total number of posts since last sync
	totalPostsSinceLastSync := totalPosts - totalFetchedPosts

	var posts []map[string]interface{}
	loadedPosts := 0
	var syncPercentage float64

	for _, channel := range channels {
		// 200 is the max number of posts per page

		// reset page to 0 for each channel
		params["per_page"] = 10
		params["page"] = 0

		// Used to check if there are any more pages of posts to fetch
		previousPostId := "~"

		// loop through all pages in a channel
		for {
			// Fetch posts for the current page
			postsRes, err := fetchPostsForPage(channel["id"].(string), params)
			if err != nil {
				return err
			}

			// Get the ids for all posts in the 'order' field and filter out each post_detail_fields we want for each post
			/*
				This is the schema for the response:
						{
								"order": [ ...list of post_ids... ],
								"posts": {
										...."post_id_1": { ...1st post details... },
												"post_id_2": { ...2nd post details... }...  }
						}
			*/

			// filteredPostFields := []string{"id", "message", "user_id", "type", "update_at", "delete_at", "channel_id"}

			// Loop through all posts in a page
			for _, postId := range postsRes.Order {
				var post map[string]interface{}

				// filter out the fields we want for each post
				post["id"] = postsRes.Posts[postId].(map[string]interface{})["id"]
				post["message"] = postsRes.Posts[postId].(map[string]interface{})["message"]
				post["user_id"] = postsRes.Posts[postId].(map[string]interface{})["user_id"]
				post["type"] = postsRes.Posts[postId].(map[string]interface{})["type"]
				post["update_at"] = postsRes.Posts[postId].(map[string]interface{})["update_at"]
				post["delete_at"] = postsRes.Posts[postId].(map[string]interface{})["delete_at"]

				// add the filtered post to the list of posts
				posts = append(posts, post)
			}

			// Increment the number of fetched posts
			loadedPosts += len(posts)

			// get the channel's access restriction (private/ public)
			access := ""
			switch channel["type"].(string) {
			case "O":
				// public channel
				access = "pub"
			case "P":
				// private channel
				access = "pri"
			}

			// remove deleted posts from chroma and filter out any irrelevant posts
			filteredPosts, err := deleteAndFilterPost(posts, access)
			if err != nil {
				return err
			}

			// upsert the filtered channel posts to chroma
			if err := upsertPostsToChroma(filteredPosts, access); err != nil {
				return err
			}

			// Update the previous post id
			previousPostId = postsRes.PreviousPostId

			// Increment the page number
			params["page"] = params["page"].(int) + 1

			// Calculate sync percentage
			if totalFetchedPosts != 0 {
				syncPercentage = (float64(loadedPosts) / float64(totalPostsSinceLastSync)) * 100
			}
			// Print sync progress
			fmt.Printf("Syncing posts... %.2f%% complete\n", syncPercentage)

			// Check if there are any more pages of posts
			// if previousPostId == posts[len(posts)-1]["id"] {
			// 	break
			// }
			if previousPostId == "" {
				break
			}
		}
	}

	// Set to 100 manually to indicate completion
	syncPercentage = 100

	fmt.Println("Total posts:", totalPostsSinceLastSync)
	fmt.Println("Total posts fetched:", totalFetchedPosts)

	// Update the last synced time in db
	updateLastSyncedTime(startSyncTime)

	// Update the total fetched posts in db
	updateTotalFetchedPosts(totalPosts)

	// Print sync completion message

	// Set syncing to false
	isSyncInProgress = false

	// Calculate the total time taken for syncing

	// -----------------------------------------------------------

	// make a request to fetch new posts
	// response, err := makeRequest("GET", "https://example.com/api/posts", params)
	// if err != nil {
	// 	return err
	// }

	// if response.StatusCode != 200 {
	// 	return fmt.Errorf("Failed to fetch new posts. Status code: %d", response.StatusCode)
	// }

	// // parse the response body
	// var posts []map[string]interface{}
	// err = json.Unmarshal(response.Body, &posts)
	// if err != nil {
	// 	return err
	// }

	// // save the fetched posts to db
	// for _, post := range posts {
	// 	// do something
	// }

	// // update the last synced time in db
	// // do something
	return nil
}

func updateTotalFetchedPosts(totalPosts int) (err error) {
	// TODO: implement this

	// do something
	return nil
}

func updateLastSyncedTime(startSyncTime time.Time) (err error) {
	// TODO: implement this

	// do something
	return nil
}

func getLastSyncedTime() time.Time {
	// TODO: implement this

	// do something
	return time.Now()
}

func getTotalFetchedPosts() int {
	// TODO: implement this

	// do something
	return 0
}

func getAllChannels(props ...string) (channels []map[string]interface{}, err error) {
	fmt.Println("Getting all channels...", props)

	req, err := http.NewRequest(http.MethodGet, mmAPI, nil)
	if err != nil {
		fmt.Printf("client: could not create request: %s\n", err)
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("client: error making http request: %s\n", err)
		return nil, err
	}

	fmt.Printf("client: got response!\n")
	fmt.Printf("client: status code: %d\n", res.StatusCode)

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("client: could not read response body: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("client: response body: %s\n", resBody)

	return nil, nil
}

func fetchPostsForPage(s string, params map[string]interface{}) (posts PostResponse, err error) {
	// TODO: implement this

	// do something
	return PostResponse{}, nil
}

func deleteAndFilterPost(post []map[string]interface{}, access string) (filteredPosts map[string]interface{}, err error) {
	// TODO: implement this

	// do something
	return nil, nil
}

func upsertPostsToChroma(filteredPosts map[string]interface{}, access string) (err error) {
	// TODO: implement this

	// do something
	return nil
}
