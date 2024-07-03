package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	chroma_go "github.com/EyobAshenaki/chroma-go"
	"github.com/EyobAshenaki/chroma-go/db"
	chroma "github.com/amikos-tech/chroma-go"
)

const token = "Bearer cg78izqitif87m8aacqq4bw4oc"
const mmAPI = "http://localhost:8065/api/v4"

type Post struct {
	Id        string `json:"id"`
	Message   string `json:"message"`
	UserId    string `json:"user_id"`
	Type      string `json:"type"`
	UpdateAt  int64  `json:"update_at"`
	DeleteAt  int64  `json:"delete_at"`
	ChannelId string `json:"channel_id"`
}

type PostResponse struct {
	Order          []string        `json:"order"`
	Posts          map[string]Post `json:"posts"`
	PreviousPostId string          `json:"prev_post_id"`
}

type Channel struct {
	Id            string `json:"id"`
	Type          string `json:"type"`
	DisplayName   string `json:"display_name"`
	TotalMsgCount int    `json:"total_msg_count"`
	// LastFetchedUpdate int64  `json:"last_fetched_update"`
}

type Sync struct {
	ticker               *time.Ticker
	store                *db.DataStore
	mattermostCollection *chroma.Collection
}

var instance *Sync
var once sync.Once

func GetSyncInstance() *Sync {
	once.Do(func() {
		chromaClient := chroma_go.GetChromaInstance()

		newCollection, colError := chromaClient.GetOrCreateCollection("mattermost")
		if colError != nil {
			log.Fatalf("Error while creating / getting collection: %v \n", colError)
		}

		initializedStore := initializeStore()

		instance = &Sync{
			store:                initializedStore,
			ticker:               nil,
			mattermostCollection: newCollection,
		}
	})

	return instance
}

func ResetVectorStore() (bool, error) {
	chromaClient := chroma_go.GetChromaInstance()
	return chromaClient.ResetData(context.Background())
}

// initialize the store in sync. if store is has values do nothing
func initializeStore() *db.DataStore {
	store := db.GetDataStore("mm-sync")

	// set fetch_interval
	if _, err := store.Get("sync", "fetch_interval"); err != nil {
		putError := store.Put("sync", "fetch_interval", []byte("15"))
		if putError != nil {
			fmt.Println(putError)
		}
	}

	// set is_fetch_in_progress
	if _, err := store.Get("sync", "is_fetch_in_progress"); err != nil {
		putError := store.Put("sync", "is_fetch_in_progress", []byte(strconv.FormatBool(false)))
		if putError != nil {
			fmt.Println(putError)
		}
	}

	// set is_sync_in_progress
	if _, err := store.Get("sync", "is_sync_in_progress"); err != nil {
		putError := store.Put("sync", "is_sync_in_progress", []byte(strconv.FormatBool(false)))
		if putError != nil {
			fmt.Println(putError)
		}
	}

	// set total_fetched_posts
	if _, err := store.Get("sync", "total_fetched_posts"); err != nil {
		putError := store.Put("sync", "total_fetched_posts", []byte(strconv.Itoa(0)))
		if putError != nil {
			fmt.Println(putError)
		}
	}

	// set last_fetched_at
	if _, err := store.Get("sync", "last_fetched_at"); err != nil {
		putError := store.Put("sync", "last_fetched_at", []byte(strconv.Itoa(0)))
		if putError != nil {
			fmt.Println(putError)
		}
	}

	// // set chroma_returned_results
	// err = store.Put("chroma", "chroma_returned_results", []byte("10"))
	// if err != nil {
	// 	fmt.Println(err)
	// }

	// // set max_chroma_distance
	// err = store.Put("chroma", "max_chroma_distance", []byte("10"))
	// if err != nil {
	// 	fmt.Println(err)
	// }

	// if *sync.store == (db.DataStore{}) {
	// 	sync.store = store
	// }

	return store
}

func (sync *Sync) StopSync() error {
	err := sync.setIsSyncInProgress(false)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("-------------------------------------")
	fmt.Println("*********** Stop syncing ***********")
	fmt.Println("-------------------------------------")
	fmt.Println()

	return nil
}

func (sync *Sync) StopTicker() {
	sync.ticker.Stop()
}

func (sync *Sync) SetTickerNil() {
	sync.ticker = nil
}

func (sync *Sync) IsTickerNil() bool {
	return sync.ticker == nil
}

func (sync *Sync) CloseStore() {
	sync.store.Close()
}

func (sync *Sync) StartFetch(percentageChan chan [][]byte, ctx context.Context) error {
	fmt.Println()
	fmt.Println("*********** Start fetching... ***********")
	fmt.Println()

	// if fetching is in progress return nothing
	if isFetchInProgress, err := sync.GetIsFetchInProgress(); isFetchInProgress {
		if err != nil {
			return err
		}

		return fmt.Errorf("fetch is in progress")
	}

	// get last synced time from db
	lastFetchedAt, err := sync.GetLastFetchedAt()
	if err != nil {
		return err
	}
	lastFetchedAtInMilliseconds := lastFetchedAt.UnixMilli()

	// get total fetched posts from db
	totalFetchedPosts, err := sync.getTotalFetchedPosts()
	if err != nil {
		return err
	}

	// set fetching to true so no other sync can start
	err = sync.setIsFetchInProgress(true)
	if err != nil {
		return err
	}

	// // Set fetching to false before returning
	defer sync.stopFetch()

	//save the time where syncing started
	startSyncTime := time.Now()

	// declare a dict to store request parameters
	// var params map[string]interface{}
	postParams := url.Values{
		"since":    {""},
		"per_page": {"200"},
		"page":     {"0"},
	}

	// Assign the since property in the request param to get all posts since that time.
	// if the since property is not defined all posts will be fetched from MM db
	if lastFetchedAtInMilliseconds != 0 && totalFetchedPosts != 0 {
		postParams.Set("since", fmt.Sprintf("%d", lastFetchedAtInMilliseconds))
	}

	// Get all channels' data
	channels, err := GetAllChannels()
	if err != nil {
		return err
	}

	totalPosts := calcTotalPosts(channels)
	log.Println("Total MM posts: ", totalPosts)
	log.Println("Embedded posts so far: ", totalFetchedPosts)

	// Get the total number of posts since last sync
	totalPostsSinceLastSync := totalPosts - totalFetchedPosts

	var posts []Post
	loadedPosts := 0
	var syncPercentage float64
	previousTotalFetchedPosts := totalFetchedPosts

	for _, channel := range channels {
		// 200 is the max number of posts per page
		postParams.Set("per_page", "10")
		postParams.Set("page", "0")

		// loop through all pages in a channel
		for {
			// Fetch posts for the current page
			postsRes, err := FetchPostsForPage(channel.Id, postParams)
			if err != nil {
				sync.setTotalFetchedPosts(previousTotalFetchedPosts)
				return err
			}

			if len(postsRes.Order) <= 0 {
				// log.Println("No posts found for channel: ", channel.Id)
				break
			}

			// add posts while keeping order
			for _, postId := range postsRes.Order {
				posts = append(posts, postsRes.Posts[postId])
			}

			// get the channel's access restriction (private/ public)
			access := ""
			switch channel.Type {
			case "O":
				// public channel
				access = "pub"
			case "P":
				// private channel
				access = "pri"
			}

			// remove deleted posts from chroma and filter out any irrelevant posts
			filteredPosts, err := deleteAndFilterPost(posts)
			if err != nil {
				sync.setTotalFetchedPosts(previousTotalFetchedPosts)
				return err
			}

			// upsert the filtered channel posts to chroma
			if len(filteredPosts) > 0 {
				// Increment the number of fetched posts
				loadedPosts = len(filteredPosts)

				if err := upsertPostsToChroma(filteredPosts, access); err != nil {
					sync.setTotalFetchedPosts(previousTotalFetchedPosts)
					return err
				}
			}

			// clean up posts slice
			posts = nil

			// Set the total fetched posts in db
			sync.setTotalFetchedPosts(totalFetchedPosts + loadedPosts)

			// if the previous post id is empty, we have reached the end of the posts for this channel
			if postsRes.PreviousPostId == "" {
				break
			}

			// Increment the page number
			page := postParams.Get("page")
			nxtPage, err := strconv.Atoi(page)
			if err != nil {
				sync.setTotalFetchedPosts(previousTotalFetchedPosts)
				return fmt.Errorf("error converting string to int: %v", err)
			}
			nxtPage += 1

			postParams.Set("page", strconv.Itoa(nxtPage))

			// Calculate sync percentage
			if totalPostsSinceLastSync != 0 {
				syncPercentage = (float64(loadedPosts) / float64(totalPostsSinceLastSync)) * 100
			}

			select {
			case <-ctx.Done():
				log.Println("fetch interrupted")
				sync.setTotalFetchedPosts(previousTotalFetchedPosts)
				close(percentageChan)
				return ctx.Err()
			default:
				log.Println("Send percentage from fetch: ", syncPercentage)
				var response [][]byte

				response = append(response, []byte("event: onProgress\n"))
				response = append(response, []byte(fmt.Sprintf("data: %.2f\n", syncPercentage)))
				response = append(response, []byte("\n"))

				percentageChan <- response
			}

		}
	}
	fmt.Println("Total posts:", totalPostsSinceLastSync)
	fmt.Println("Total posts fetched:", totalFetchedPosts)

	var response [][]byte

	response = append(response, []byte("event: onDone\n"))
	response = append(response, []byte(fmt.Sprintf("data: %.2f\n", syncPercentage)))
	response = append(response, []byte("\n"))

	percentageChan <- response

	// Set the last synced time in db
	sync.setLastFetchedAt(startSyncTime)

	return nil
}

func (sync *Sync) stopFetch() error {
	err := sync.setIsFetchInProgress(false)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("*********** Stop fetching ***********")
	fmt.Println()
	return nil
}

func (sync *Sync) StartSync(ctxWithCancel context.Context, percentageChan chan [][]byte) error {
	fmt.Println()
	fmt.Println("-------------------------------------")
	fmt.Println("*********** Start syncing ***********")
	fmt.Println("-------------------------------------")
	fmt.Println()

	// if syncing is in progress return nothing
	if isSyncInProgress, err := sync.GetIsSyncInProgress(); isSyncInProgress {
		if err != nil {
			return err
		}

		return fmt.Errorf("sync is in progress")
	}

	// get fetch interval from db
	fetchInterval, err := sync.GetFetchInterval()
	if err != nil {
		return err
	}

	// set syncing to true so no other sync can start
	err = sync.setIsSyncInProgress(true)
	if err != nil {
		return err
	}

	// start the ticker
	sync.ticker = time.NewTicker(time.Duration(fetchInterval) * time.Second)

	// Set isFetchInProgress to false before returning
	defer sync.stopFetch()

	// // stop ticker and ser isSyncInProgress to false before returning
	// defer sync.StopSync()
	alreadyRun := false
	for {
		select {
		case <-ctxWithCancel.Done():
			log.Println("sync interrupted")
			return fmt.Errorf("syncing error: %v", ctxWithCancel.Err())
		case tickerChan, ok := <-sync.ticker.C:
			if !ok {
				log.Printf("ticker channel closed: %v\n", ok)
				return nil
			}

			log.Printf("Fetch started at: %v \n", tickerChan)

			// start the fetch
			err := sync.StartFetch(percentageChan, ctxWithCancel)
			if err != nil {
				return fmt.Errorf("error while fetching: %v", err)
			}
		case currentTime := <-time.After(0): // This case runs immediately
			if !alreadyRun {
				log.Println("Immediate fetch started: ", currentTime)

				err := sync.StartFetch(percentageChan, ctxWithCancel)
				if err != nil {
					return fmt.Errorf("error while fetching: %v", err)
				}

				alreadyRun = true
			}
		}
	}
}

func (sync *Sync) UpdateFetchInterval(newInterval time.Duration) error {
	if newInterval <= 0 {
		// sync.ticker.Stop()
		return fmt.Errorf("fetch interval must me greater than 0")
	}

	// reset stops a ticker and resets its period to the specified
	// duration. The next tick will arrive after the new period elapses.
	if sync.ticker != nil {
		sync.ticker.Reset(newInterval)
	}

	return sync.setFetchInterval(newInterval)
}

// ----------------------------- Is Sync In Progress --------------------
func (sync *Sync) setFetchInterval(interval time.Duration) error {
	if *sync.store == (db.DataStore{}) {
		return fmt.Errorf("store is not initialized")
	}

	return sync.store.Put("sync", "fetch_interval", []byte(strconv.Itoa(int(interval.Hours()))))
}

func (sync *Sync) GetFetchInterval() (int, error) {
	if *sync.store == (db.DataStore{}) {
		return 0, fmt.Errorf("store is not initialized")
	}

	b, err := sync.store.Get("sync", "fetch_interval")

	if err != nil {
		return 0, err
	}

	return strconv.Atoi(string(b))
}

// ----------------------------- Is Sync In Progress --------------------
func (sync *Sync) setIsSyncInProgress(truthVal bool) error {
	if *sync.store == (db.DataStore{}) {
		return fmt.Errorf("store is not initialized")
	}

	return sync.store.Put("sync", "is_sync_in_progress", []byte(strconv.FormatBool(truthVal)))
}

func (sync *Sync) GetIsSyncInProgress() (bool, error) {
	if *sync.store == (db.DataStore{}) {
		return false, fmt.Errorf("store is not initialized")
	}

	b, err := sync.store.Get("sync", "is_sync_in_progress")

	if err != nil {
		return false, nil
	}

	return strconv.ParseBool(string(b))
}

// ----------------------------- Is Fetch In Progress --------------------
func (sync *Sync) setIsFetchInProgress(truthVal bool) error {
	if *sync.store == (db.DataStore{}) {
		return fmt.Errorf("store is not initialized")
	}

	return sync.store.Put("sync", "is_fetch_in_progress", []byte(strconv.FormatBool(truthVal)))
}

func (sync *Sync) GetIsFetchInProgress() (bool, error) {
	if *sync.store == (db.DataStore{}) {
		return false, fmt.Errorf("store is not initialized")
	}

	b, err := sync.store.Get("sync", "is_fetch_in_progress")

	if err != nil {
		return false, nil
	}

	return strconv.ParseBool(string(b))
}

// ----------------------------- Is Total Fetched Posts --------------------
func (sync *Sync) setTotalFetchedPosts(totalPosts int) error {
	if *sync.store == (db.DataStore{}) {
		return fmt.Errorf("store is not initialized")
	}

	return sync.store.Put("sync", "total_fetched_posts", []byte(strconv.Itoa(totalPosts)))
}

func (sync *Sync) getTotalFetchedPosts() (int, error) {
	if *sync.store == (db.DataStore{}) {
		return 0, fmt.Errorf("store is not initialized")
	}

	b, err := sync.store.Get("sync", "total_fetched_posts")

	if err != nil {
		return 0, err
	}

	return strconv.Atoi(string(b))

	// documentCount, err := sync.mattermostCollection.Count(context.Background())
	// if err != nil {
	// 	return 0, err
	// }

	// return int(documentCount), nil
}

// ----------------------------- Is Last Fetched At --------------------

func (sync *Sync) setLastFetchedAt(startSyncTime time.Time) error {
	if *sync.store == (db.DataStore{}) {
		return fmt.Errorf("store is not initialized")
	}

	return sync.store.Put("sync", "last_fetched_at", []byte(strconv.FormatInt(startSyncTime.UnixMilli(), 10)))
}

func (sync *Sync) GetLastFetchedAt() (time.Time, error) {
	if *sync.store == (db.DataStore{}) {
		return time.Time{}, fmt.Errorf("store is not initialized")
	}

	b, err := sync.store.Get("sync", "last_fetched_at")

	if err != nil {
		return time.Time{}, err
	}

	lastFetchedAt, err := strconv.ParseInt(string(b), 10, 64)

	if err != nil {
		fmt.Println("error while parsing int: ", err)
		return time.Time{}, err
	}

	return time.UnixMilli(lastFetchedAt), nil
}

// ---------------- Utility Functions ----------------

func calcTotalPosts(channels []Channel) int {
	total := 0
	for _, channel := range channels {
		total += channel.TotalMsgCount
	}
	return total
}

func GetAllChannels() (channels []Channel, err error) {
	reqUrl := mmAPI + "/channels"

	req, err := http.NewRequest(http.MethodGet, reqUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("client: could not create request: %s", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	client := http.Client{
		Timeout: 10 * time.Second,
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client: error making http request: %s", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("client: Failed to fetch channels. Status code: %d", response.StatusCode)
	}

	// automatically filters the response body to only include the fields
	// specified in Channels struct by json tags
	err = json.NewDecoder(response.Body).Decode(&channels)
	if err != nil {
		return nil, fmt.Errorf("client: could not decode json: %s", err)
	}

	return channels, nil
}

// Get all posts per page in a channel
func FetchPostsForPage(channelId string, params url.Values) (postRes PostResponse, err error) {
	reqUrl := mmAPI + "/channels/" + channelId + "/posts"

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return PostResponse{}, fmt.Errorf("client: could not create request: %s", err)
	}

	req.URL.RawQuery = params.Encode()

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	response, err := client.Do(req)
	if err != nil {
		return PostResponse{}, fmt.Errorf("client: error making http request: %s", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return PostResponse{}, fmt.Errorf("client: Failed to fetch new posts. Status code: %d", response.StatusCode)
	}

	err = json.NewDecoder(response.Body).Decode(&postRes)
	if err != nil {
		return PostResponse{}, fmt.Errorf("client: could not decode json: %s", err)
	}

	return postRes, nil
}

// Deletes posts that have been deleted from mattermost
// from chroma database and filters system and non-text
// messages
func deleteAndFilterPost(posts []Post) (filteredPosts []Post, err error) {
	// TODO: filter out any stickers / emojis
	// TODO: replace user handles with their real names

	for _, post := range posts {
		// delete posts from chroma if it's been deleted from mattermost
		if post.DeleteAt > 0 {
			err := deleteFromChroma(post.Id)
			if err != nil {
				return nil, err
			}

			// remove any posts that have been deleted
			removePost(posts, post.Id)
			continue
		}

		// filter out posts that are not of type text and empty messages
		// filter out any irrelevant posts
		if post.Type == "" && post.Message != "" {
			// TODO: format the post in this form "(date) user-name: message_text" before append
			filteredPosts = append(filteredPosts, post)
		}
	}

	return filteredPosts, nil
}

func removePost(posts []Post, postId string) {
	for idx, post := range posts {
		if post.Id == postId {
			posts = append(posts[:idx], posts[idx+1:]...)
		}
	}
}

func deleteFromChroma(postId string) error {
	fmt.Println("Deleting post from chroma...", postId)

	ids := []string{postId}

	_, delError := GetSyncInstance().mattermostCollection.Delete(context.Background(), ids, nil, nil)
	if delError != nil {
		return fmt.Errorf("error while deleting from chroma: %v", delError)
	}

	return nil
}

func upsertPostsToChroma(filteredPosts []Post, access string) (err error) {
	// log.Println("Upserting...", len(filteredPosts), access)

	metadatas := []map[string]interface{}{}
	documents := []string{}
	ids := []string{}

	/*
		{
			"id" : "message_id",
			"document" : "(date) User: message_text",
			"metadata" : {
					"source": "mm",
					"access" : "pri / pub",
					"channel_id" : "ch_0000",
					"user_id" : "usr_0000",
			}
		}
	*/

	// extract the mettadatas, ids and documents from the filtered posts in the above format
	for _, post := range filteredPosts {
		ids = append(ids, post.Id)
		documents = append(documents, post.Message)
		metadatas = append(metadatas, map[string]interface{}{
			"source":     "mm",
			"access":     access,
			"channel_id": post.ChannelId,
			"user_id":    post.UserId,
		})
	}

	// log.Println("Data: ", documents, ids, metadatas)

	cxtWithTimeout, cancelCtxWithTimeout := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelCtxWithTimeout()

	log.Println("Upserting to chroma...")

	// Even tho the embeddings are not provided, this function call will embed the documents using the embedding function defined in the collection
	_, upError := GetSyncInstance().mattermostCollection.Upsert(cxtWithTimeout, nil, metadatas, documents, ids)
	if upError != nil {
		return fmt.Errorf("failed to upsert to chroma: %v", upError)
	}

	return nil
}
