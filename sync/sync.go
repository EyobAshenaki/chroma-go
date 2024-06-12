package sync

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/EyobAshenaki/chroma-go/db"
)

const token = "Bearer yokb6gwehpfhmn64f9k63r4xiw"
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
	Ticker *time.Ticker
	Store  *db.DataStore
}

func New() *Sync {
	return &Sync{
		Store:  &db.DataStore{},
		Ticker: &time.Ticker{},
	}
}

func (sync *Sync) InitializeStore() {
	store := db.GetDataStore("mm-sync")

	// set fetch_interval
	err := store.Put("sync", "fetch_interval", []byte("6"))
	if err != nil {
		fmt.Println(err)
	}

	// set is_fetch_in_progress
	err = store.Put("sync", "is_fetch_in_progress", []byte(strconv.FormatBool(false)))
	if err != nil {
		fmt.Println(err)
	}

	// set is_sync_in_progress
	err = store.Put("sync", "is_sync_in_progress", []byte(strconv.FormatBool(false)))
	if err != nil {
		fmt.Println(err)
	}

	// set total_fetched_posts
	err = store.Put("sync", "total_fetched_posts", []byte(strconv.Itoa(0)))
	if err != nil {
		fmt.Println(err)
	}

	// set last_fetched_at
	err = store.Put("sync", "last_fetched_at", []byte(strconv.Itoa(0)))
	if err != nil {
		fmt.Println(err)
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

	sync.Store = store
}

func (sync *Sync) StopSync() error {
	if sync.Ticker == nil {
		return fmt.Errorf("sync is not running")
	}
	sync.Ticker.Stop()
	return nil
}

func (sync *Sync) CloseStore() {
	sync.Store.Close()
}

func (sync *Sync) StartFetch() error {
	fmt.Println("Start syncing...")

	// if fetching is in progress return nothing
	if isFetchInProgress, err := sync.getIsFetchInProgress(); isFetchInProgress {
		if err != nil {
			return err
		}

		return fmt.Errorf("fetch is in progress")
	}

	// get last synced time from db
	lastFetchedAt, err := sync.getLastFetchedAt()
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
	fetchErr := sync.setIsFetchInProgress(true)
	if fetchErr != nil {
		return fetchErr
	}

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
	} else {
		err := sync.setIsSyncInProgress(true)
		if err != nil {
			return err
		}
	}

	// Get all channels' data
	channels, err := GetAllChannels()
	if err != nil {
		return err
	}

	totalPosts := calcTotalPosts(channels)

	// Get the total number of posts since last sync
	totalPostsSinceLastSync := totalPosts - totalFetchedPosts

	var posts []Post
	loadedPosts := 0
	var syncPercentage float64

	for _, channel := range channels {
		// 200 is the max number of posts per page
		postParams.Set("per_page", "10")
		postParams.Set("page", "0")

		// loop through all pages in a channel
		for {
			// Fetch posts for the current page
			postsRes, err := FetchPostsForPage(channel.Id, postParams)
			if err != nil {
				return err
			}

			// add posts while keeping order
			for _, postId := range postsRes.Order {
				posts = append(posts, postsRes.Posts[postId])
			}

			// Increment the number of fetched posts
			loadedPosts = len(posts)

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
				return err
			}

			// upsert the filtered channel posts to chroma
			if err := upsertPostsToChroma(filteredPosts, access); err != nil {
				return err
			}

			// Set the total fetched posts in db
			sync.setTotalFetchedPosts(loadedPosts)

			// if the previous post id is empty, we have reached the end of the posts for this channel
			if postsRes.PreviousPostId == "" {
				break
			}

			// Increment the page number
			page := postParams.Get("page")
			nxtPage, err := strconv.Atoi(page)
			if err != nil {
				fmt.Println("Error converting string to int:", err)
				return err
			}
			nxtPage += 1

			postParams.Set("page", strconv.Itoa(nxtPage))

			// Calculate sync percentage
			if totalPostsSinceLastSync != 0 {
				syncPercentage = (float64(loadedPosts) / float64(totalPostsSinceLastSync)) * 100
			}
			// Print sync progress
			fmt.Printf("Syncing posts... %.2f%% complete\n", syncPercentage)
			time.Sleep(1 * time.Second)
		}
	}

	syncPercentage = (float64(loadedPosts) / float64(totalPostsSinceLastSync)) * 100

	fmt.Println("Total posts:", totalPostsSinceLastSync)
	fmt.Println("Total posts fetched:", totalFetchedPosts)

	// Set the last synced time in db
	sync.setLastFetchedAt(startSyncTime)

	// Print sync completion message
	fmt.Printf("Fetching posts... %.2f%% complete\n", syncPercentage)

	// Set syncing to false
	fetchErr = sync.setIsFetchInProgress(false)
	if fetchErr != nil {
		return fetchErr
	}

	return nil
}

func (sync *Sync) updateFetchInterval(newInterval time.Duration) error {
	if sync.Ticker == nil {
		return fmt.Errorf("sync is not running")
	}

	if newInterval <= 0 {
		sync.Ticker.Stop()
		return fmt.Errorf("fetch interval much me greater than 0")
	}

	// reset stops a ticker and resets its period to the specified
	// duration. The next tick will arrive after the new period elapses.
	sync.Ticker.Reset(newInterval)

	return sync.setFetchInterval(newInterval)
}

// ----------------------------- Is Sync In Progress --------------------
func (sync *Sync) setFetchInterval(interval time.Duration) error {
	if sync.Store == nil {
		return fmt.Errorf("store is not initialized")
	}

	return sync.Store.Put("sync", "fetch_interval", []byte(strconv.Itoa(int(interval.Hours()))))
}

func (sync *Sync) getFetchInterval() (int, error) {
	if sync.Store == nil {
		return 0, fmt.Errorf("store is not initialized")
	}

	b, err := sync.Store.Get("sync", "fetch_interval")

	if err != nil {
		return 0, nil
	}

	return strconv.Atoi(string(b))
}

// ----------------------------- Is Sync In Progress --------------------
func (sync *Sync) setIsSyncInProgress(truthVal bool) error {
	if sync.Store == nil {
		return fmt.Errorf("store is not initialized")
	}

	return sync.Store.Put("sync", "is_sync_in_progress", []byte(strconv.FormatBool(truthVal)))
}

func (sync *Sync) getIsSyncInProgress() (bool, error) {
	if sync.Store == nil {
		return false, fmt.Errorf("store is not initialized")
	}

	b, err := sync.Store.Get("sync", "is_sync_in_progress")

	if err != nil {
		return false, nil
	}

	return strconv.ParseBool(string(b))
}

// ----------------------------- Is Fetch In Progress --------------------
func (sync *Sync) setIsFetchInProgress(truthVal bool) error {
	if sync.Store == nil {
		return fmt.Errorf("store is not initialized")
	}

	return sync.Store.Put("sync", "is_fetch_in_progress", []byte(strconv.FormatBool(truthVal)))
}

func (sync *Sync) getIsFetchInProgress() (bool, error) {
	if sync.Store == nil {
		return false, fmt.Errorf("store is not initialized")
	}

	b, err := sync.Store.Get("sync", "is_fetch_in_progress")

	if err != nil {
		return false, nil
	}

	return strconv.ParseBool(string(b))
}

// ----------------------------- Is Total Fetched Posts --------------------
func (sync *Sync) setTotalFetchedPosts(totalPosts int) error {
	if sync.Store == nil {
		return fmt.Errorf("store is not initialized")
	}

	return sync.Store.Put("sync", "total_fetched_posts", []byte(strconv.Itoa(totalPosts)))
}

func (sync *Sync) getTotalFetchedPosts() (int, error) {
	if sync.Store == nil {
		return 0, fmt.Errorf("store is not initialized")
	}

	b, err := sync.Store.Get("sync", "total_fetched_posts")

	if err != nil {
		return 0, err
	}

	return strconv.Atoi(string(b))
}

// ----------------------------- Is Last Fetched At --------------------

func (sync *Sync) setLastFetchedAt(startSyncTime time.Time) error {
	if sync.Store == nil {
		return fmt.Errorf("store is not initialized")
	}

	return sync.Store.Put("sync", "last_fetched_at", []byte(strconv.FormatInt(startSyncTime.UnixMilli(), 10)))
}

func (sync *Sync) getLastFetchedAt() (time.Time, error) {
	if sync.Store == nil {
		return time.Time{}, fmt.Errorf("store is not initialized")
	}

	b, err := sync.Store.Get("sync", "last_fetched_at")

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

	fmt.Println("Request URL:", reqUrl)

	req, err := http.NewRequest(http.MethodGet, reqUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("client: could not create request: %s\n", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	client := http.Client{
		Timeout: 10 * time.Second,
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client: error making http request: %s\n", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("client: Failed to fetch new posts. Status code: %d", response.StatusCode)
	}

	// automatically filters the response body to only include the fields
	// specified in Channels struct by json tags
	err = json.NewDecoder(response.Body).Decode(&channels)
	if err != nil {
		return nil, fmt.Errorf("client: could not decode json: %s\n", err)

	}

	return channels, nil
}

// *** DONE *** //
func FetchPostsForPage(channelId string, params url.Values) (postRes PostResponse, err error) {
	reqUrl := mmAPI + "/channels/" + channelId + "/posts"

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return PostResponse{}, fmt.Errorf("client: could not create request: %s\n", err)
	}

	req.URL.RawQuery = params.Encode()

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	response, err := client.Do(req)
	if err != nil {
		return PostResponse{}, fmt.Errorf("client: error making http request: %s\n", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return PostResponse{}, fmt.Errorf("client: Failed to fetch new posts. Status code: %d", response.StatusCode)
	}

	err = json.NewDecoder(response.Body).Decode(&postRes)
	if err != nil {
		return PostResponse{}, fmt.Errorf("client: could not decode json: %s\n", err)
	}

	return postRes, nil
}

func deleteAndFilterPost(posts []Post) (filteredPosts []Post, err error) {
	// TODO: filter out any stickers / emojis
	// TODO: replace user handles with their real names

	for _, post := range posts {
		// TODO: delete posts from chroma if it's been deleted from mattermost
		// delete any posts that have been deleted
		if post.DeleteAt > 0 {
			deleteFromChroma("collection_name", post.Id)
		}

		// TODO: filter posts that are not of type text and empty messages
		// filter out any irrelevant posts
		if post.Type == "" && post.Message != "" {
			filteredPosts = append(filteredPosts, post)
		}
	}

	return nil, nil
}

func deleteFromChroma(collectionName, postId string) {
	// TODO: implement this

	fmt.Println("Deleting post from chroma...", collectionName, postId)
}

func upsertPostsToChroma(filteredPosts []Post, access string) (err error) {
	// TODO: implement this

	fmt.Println("Upserting posts to chroma...", filteredPosts, access)

	return nil
}
