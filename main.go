package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

// Variables used for command line parameters
var (
	Token string
)

// @flow

type Response struct {
	EntityUniqueId     string                `json:"entityUniqueId"`
	UserCountry        string                `json:"userCountry"`
	PageUrl            string                `json:"pageUrl"`
	LinksByPlatform    map[Platform]LinkData `json:"linksByPlatform"`
	EntitiesByUniqueId map[string]EntityData `json:"entitiesByUniqueId"`
}

type LinkData struct {
	EntityUniqueId      string `json:"entityUniqueId"`
	Url                 string `json:"url"`
	NativeAppUriMobile  string `json:"nativeAppUriMobile,omitempty"`
	NativeAppUriDesktop string `json:"nativeAppUriDesktop,omitempty"`
}

type EntityData struct {
	Id              string      `json:"id"`
	Type            string      `json:"type"`
	Title           string      `json:"title,omitempty"`
	ArtistName      string      `json:"artistName,omitempty"`
	ThumbnailUrl    string      `json:"thumbnailUrl,omitempty"`
	ThumbnailWidth  int         `json:"thumbnailWidth,omitempty"`
	ThumbnailHeight int         `json:"thumbnailHeight,omitempty"`
	ApiProvider     APIProvider `json:"apiProvider"`
	Platforms       []Platform  `json:"platforms"`
}

type Platform string

const (
	Spotify      Platform = "spotify"
	Itunes       Platform = "itunes"
	AppleMusic   Platform = "appleMusic"
	Youtube      Platform = "youtube"
	YoutubeMusic Platform = "youtubeMusic"
	Google       Platform = "google"
	GoogleStore  Platform = "googleStore"
	Pandora      Platform = "pandora"
	Deezer       Platform = "deezer"
	Tidal        Platform = "tidal"
	AmazonStore  Platform = "amazonStore"
	AmazonMusic  Platform = "amazonMusic"
	Soundcloud   Platform = "soundcloud"
	Napster      Platform = "napster"
	Yandex       Platform = "yandex"
	Spinrilla    Platform = "spinrilla"
	Audius       Platform = "audius"
	Audiomack    Platform = "audiomack"
	Anghami      Platform = "anghami"
	Boomplay     Platform = "boomplay"
)

type APIProvider string

const (
	SpotifyAPI    APIProvider = "spotify"
	ItunesAPI     APIProvider = "itunes"
	YoutubeAPI    APIProvider = "youtube"
	GoogleAPI     APIProvider = "google"
	PandoraAPI    APIProvider = "pandora"
	DeezerAPI     APIProvider = "deezer"
	TidalAPI      APIProvider = "tidal"
	AmazonAPI     APIProvider = "amazon"
	SoundcloudAPI APIProvider = "soundcloud"
	NapsterAPI    APIProvider = "napster"
	YandexAPI     APIProvider = "yandex"
	SpinrillaAPI  APIProvider = "spinrilla"
	AudiusAPI     APIProvider = "audius"
	AudiomackAPI  APIProvider = "audiomack"
	AnghamiAPI    APIProvider = "anghami"
	BoomplayAPI   APIProvider = "boomplay"
)

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()
}

func main() {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

func multipleUrl2SingleUrl(m string) []string {
	reg := "\r\n|\n"

	arr1 := regexp.MustCompile(reg).Split(m, -1)

	for _, s := range arr1 {
		fmt.Printf("%s\n", s)
	}
	return arr1
}

func convertSpotifyLink2OpenSpotifyCom(m string) string {
	re, err := regexp.Compile(`http(.*)://(.*)`)
	if err != nil {
		fmt.Println("error compiling regex ,", err)
		return ""
	}
	url := re.FindString(m)
	req, _ := http.NewRequest("GET", url, nil)
	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("error getting response,", err)
		return ""
	}
	dumpResp, _ := httputil.DumpResponse(resp, true)
	getSpotifyURL, err := regexp.Compile(`https://open.spotify.com(.*)\?`)
	if err != nil {
		fmt.Println("error compiling regex,", err)
		return ""
	}
	spotifyURL := getSpotifyURL.FindString(string(dumpResp))
	spotifyURL = strings.Replace(spotifyURL, "?", "", -1)
	return spotifyURL
}

func getSpotifyTrackID(spotifyURL string) string {
	getIdFromUrl, err := regexp.Compile(`track/(\w+)`)
	if err != nil {
		fmt.Println("error compiling regex,", err)
		return ""
	}
	matches := getIdFromUrl.FindStringSubmatch(spotifyURL)
	if len(matches) < 2 {
		return ""
	}
	spotifyTrackID := matches[1]
	return spotifyTrackID
}

func getYoutubeID(m string) string {
	re, err := regexp.Compile(`watch\?v=(.*)`)
	if err != nil {
		fmt.Println("error compiling regex,", err)
		return ""
	}
	matches := re.FindStringSubmatch(m)
	if len(matches) < 2 {
		fmt.Println("No match found")
		return ""
	}
	return matches[1]
}

func getTrackASIN(m string) string {
	re, err := regexp.Compile(`trackAsin=([A-Z0-9]{10})`)
	if err != nil {
		fmt.Println("error compiling regex,", err)
		return ""
	}
	matches := re.FindStringSubmatch(m)
	if len(matches) < 2 {
		fmt.Println("No match found")
		return ""
	}
	return matches[1]
}

func getAppleMusicID(m string) string {
	re, err := regexp.Compile(`\?i=([0-9]{10})`)
	if err != nil {
		fmt.Println("error compiling regex,", err)
		return ""
	}
	matches := re.FindStringSubmatch(m)
	if len(matches) < 2 {
		fmt.Println("No match found")
		return ""
	}
	return matches[1]
}

func getYoutubeUrlFromSpotify(spotifyTrackID string) string {
	resp, err := http.Get("https://api.song.link/v1-alpha.1/links?platform=spotify&type=song&id=" + spotifyTrackID + "&userCountry=JP&songIfSingle=true")
	if err != nil {
		fmt.Println("error getting response,", err)
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading response,", err)
		return ""
	}
	var response Response
	json.Unmarshal(body, &response)
	songUrl, ok := response.LinksByPlatform["youtubeMusic"]
	if !ok {
		fmt.Println("youtubeMusic URL not found")
		return "error getting youtubeMusic URL"
	}
	postURL := songUrl.Url
	return postURL
}

func getYoutubeUrlFromAmazon(trackASIN string) string {
	resp, err := http.Get("https://api.song.link/v1-alpha.1/links?platform=amazonMusic&type=song&id=" + trackASIN + "&userCountry=JP&songIfSingle=true")
	if err != nil {
		fmt.Println("error getting response,", err)
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading response,", err)
		return ""
	}
	var response Response
	json.Unmarshal(body, &response)
	songUrl, ok := response.LinksByPlatform["youtubeMusic"]
	if !ok {
		fmt.Println("youtubeMusic URL not found")
		return "error getting youtubeMusic URL"
	}
	postURL := songUrl.Url
	return postURL
}

func getYoutubeUrlFromAppleMusic(appleMusicID string) string {
	resp, err := http.Get("https://api.song.link/v1-alpha.1/links?platform=appleMusic&type=song&id=" + appleMusicID + "&userCountry=JP&songIfSingle=true")
	if err != nil {
		fmt.Println("error getting response,", err)
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading response,", err)
		return ""
	}
	var response Response
	json.Unmarshal(body, &response)
	songUrl, ok := response.LinksByPlatform["youtubeMusic"]
	if !ok {
		fmt.Println("youtubeMusic URL not found")
		return "error getting youtubeMusic URL"
	}
	postURL := songUrl.Url
	return postURL
}

func getSpotifyUrlFromYoutube(youtubeID string) string {
	resp, err := http.Get("https://api.song.link/v1-alpha.1/links?platform=youtubeMusic&type=song&id=" + youtubeID + "&userCountry=JP&songIfSingle=true")
	if err != nil {
		fmt.Println("error getting response,", err)
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading response,", err)
		return ""
	}
	var response Response
	json.Unmarshal(body, &response)
	songUrl, ok := response.LinksByPlatform["spotify"]
	if !ok {
		fmt.Println("spotify URL not found")
		return "error getting spotify URL"
	}
	postURL := songUrl.Url
	return postURL
}

func getSpotifyUrlFromAmazon(trackASIN string) string {
	resp, err := http.Get("https://api.song.link/v1-alpha.1/links?platform=amazonMusic&type=song&id=" + trackASIN + "&userCountry=JP&songIfSingle=true")
	if err != nil {
		fmt.Println("error getting response,", err)
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading response,", err)
		return ""
	}
	var response Response
	json.Unmarshal(body, &response)
	songUrl, ok := response.LinksByPlatform["spotify"]
	if !ok {
		fmt.Println("spotify URL not found")
		return "error getting spotify URL"
	}
	postURL := songUrl.Url
	return postURL
}

func getSpotifyUrlFromAppleMusic(appleMusicID string) string {
	resp, err := http.Get("https://api.song.link/v1-alpha.1/links?platform=appleMusic&type=song&id=" + appleMusicID + "&userCountry=JP&songIfSingle=true")
	if err != nil {
		fmt.Println("error getting response,", err)
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading response,", err)
		return ""
	}
	var response Response
	json.Unmarshal(body, &response)
	songUrl, ok := response.LinksByPlatform["spotify"]
	if !ok {
		fmt.Println("spotify URL not found")
		return "error getting spotify URL"
	}
	postURL := songUrl.Url
	return postURL
}

func getAmazonUrlFromSpotify(spotifyTrackID string) string {
	resp, err := http.Get("https://api.song.link/v1-alpha.1/links?platform=spotify&type=song&id=" + spotifyTrackID + "&userCountry=JP&songIfSingle=true")
	if err != nil {
		fmt.Println("error getting response,", err)
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading response,", err)
		return ""
	}
	var response Response
	json.Unmarshal(body, &response)
	songUrl, ok := response.LinksByPlatform["amazonMusic"]
	if !ok {
		fmt.Println("amazonMusic URL not found")
		return "error getting amazonMusic URL"
	}
	postURL := strings.Replace(songUrl.Url, ".com", ".co.jp", 1)
	return postURL
}

func getAmazonUrlFromYoutube(youtubeID string) string {
	resp, err := http.Get("https://api.song.link/v1-alpha.1/links?platform=youtubeMusic&type=song&id=" + youtubeID + "&userCountry=JP&songIfSingle=true")
	if err != nil {
		fmt.Println("error getting response,", err)
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading response,", err)
		return ""
	}
	var response Response
	json.Unmarshal(body, &response)
	songUrl, ok := response.LinksByPlatform["amazonMusic"]
	if !ok {
		fmt.Println("amazonMusic URL not found")
		return "error getting amazonMusic URL"
	}
	postURL := strings.Replace(songUrl.Url, ".com", ".co.jp", 1)
	return postURL
}

func getAmazonUrlFromAppleMusic(appleMusicID string) string {
	resp, err := http.Get("https://api.song.link/v1-alpha.1/links?platform=appleMusic&type=song&id=" + appleMusicID + "&userCountry=JP&songIfSingle=true")
	if err != nil {
		fmt.Println("error getting response,", err)
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading response,", err)
		return ""
	}
	var response Response
	json.Unmarshal(body, &response)
	songUrl, ok := response.LinksByPlatform["amazonMusic"]
	if !ok {
		fmt.Println("amazonMusic URL not found")
		return "error getting amazonMusic URL"
	}
	postURL := strings.Replace(songUrl.Url, ".com", ".co.jp", 1)
	return postURL
}

func getAppleMusicUrlFromSpotify(spotifyTrackID string) string {
	resp, err := http.Get("https://api.song.link/v1-alpha.1/links?platform=spotify&type=song&id=" + spotifyTrackID + "&userCountry=JP&songIfSingle=true")
	if err != nil {
		fmt.Println("error getting response,", err)
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading response,", err)
		return ""
	}
	var response Response
	json.Unmarshal(body, &response)
	songUrl, ok := response.LinksByPlatform["appleMusic"]
	if !ok {
		fmt.Println("appleMusic URL not found")
		return "error getting appleMusic URL"
	}
	postURL := songUrl.Url
	return postURL
}

func getAppleMusicUrlFromYoutube(youtubeID string) string {
	resp, err := http.Get("https://api.song.link/v1-alpha.1/links?platform=youtubeMusic&type=song&id=" + youtubeID + "&userCountry=JP&songIfSingle=true")
	if err != nil {
		fmt.Println("error getting response,", err)
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading response,", err)
		return ""
	}
	var response Response
	json.Unmarshal(body, &response)
	songUrl, ok := response.LinksByPlatform["appleMusic"]
	if !ok {
		fmt.Println("appleMusic URL not found")
		return "error getting appleMusic URL"
	}
	postURL := songUrl.Url
	return postURL
}

func getAppleMusicUrlFromAmazon(trackASIN string) string {
	resp, err := http.Get("https://api.song.link/v1-alpha.1/links?platform=amazonMusic&type=song&id=" + trackASIN + "&userCountry=JP&songIfSingle=true")
	if err != nil {
		fmt.Println("error getting response,", err)
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading response,", err)
		return ""
	}
	var response Response
	json.Unmarshal(body, &response)
	songUrl, ok := response.LinksByPlatform["appleMusic"]
	if !ok {
		fmt.Println("appleMusic URL not found")
		return "error getting appleMusic URL"
	}
	postURL := songUrl.Url
	return postURL
}

// Function to retrieve URLs from various music services based on source type and ID
func getURLsFromService(sourceType string, sourceID string) map[string]string {
	urls := make(map[string]string)
	switch sourceType {
	case "spotify":
		urls["youtube"] = getYoutubeUrlFromSpotify(sourceID)
		urls["amazon"] = getAmazonUrlFromSpotify(sourceID)
		urls["apple"] = getAppleMusicUrlFromSpotify(sourceID)
	case "youtube":
		urls["spotify"] = getSpotifyUrlFromYoutube(sourceID)
		urls["amazon"] = getAmazonUrlFromYoutube(sourceID)
		urls["apple"] = getAppleMusicUrlFromYoutube(sourceID)
	case "amazon":
		urls["spotify"] = getSpotifyUrlFromAmazon(sourceID)
		urls["youtube"] = getYoutubeUrlFromAmazon(sourceID)
		urls["apple"] = getAppleMusicUrlFromAmazon(sourceID)
	case "apple":
		urls["spotify"] = getSpotifyUrlFromAppleMusic(sourceID)
		urls["youtube"] = getYoutubeUrlFromAppleMusic(sourceID)
		urls["amazon"] = getAmazonUrlFromAppleMusic(sourceID)
	}

	// Retry fetching URLs from alternative services when primary fetch fails
	retryFromOtherService(urls)

	return urls
}

// Function to retry URL retrieval through alternative services when the primary fetch fails
func retryFromOtherService(urls map[string]string) {
	// Retry YouTube URL
	if urls["youtube"] == "error getting youtubeMusic URL" {
		switch {
		case urls["amazon"] != "error getting amazonMusic URL":
			urls["youtube"] = getYoutubeUrlFromAmazon(getTrackASIN(urls["amazon"]))
		case urls["apple"] != "error getting appleMusic URL":
			urls["youtube"] = getYoutubeUrlFromAppleMusic(getAppleMusicID(urls["apple"]))
		case urls["spotify"] != "error getting spotify URL":
			urls["youtube"] = getYoutubeUrlFromSpotify(getSpotifyTrackID(urls["spotify"]))
		}
	}

	// Retry Amazon URL
	if urls["amazon"] == "error getting amazonMusic URL" {
		switch {
		case urls["youtube"] != "error getting youtubeMusic URL":
			urls["amazon"] = getAmazonUrlFromYoutube(getYoutubeID(urls["youtube"]))
		case urls["apple"] != "error getting appleMusic URL":
			urls["amazon"] = getAmazonUrlFromAppleMusic(getAppleMusicID(urls["apple"]))
		case urls["spotify"] != "error getting spotify URL":
			urls["amazon"] = getAmazonUrlFromSpotify(getSpotifyTrackID(urls["spotify"]))
		}
	}

	// Retry Spotify URL
	if urls["spotify"] == "error getting spotify URL" {
		switch {
		case urls["youtube"] != "error getting youtubeMusic URL":
			urls["spotify"] = getSpotifyUrlFromYoutube(getYoutubeID(urls["youtube"]))
		case urls["amazon"] != "error getting amazonMusic URL":
			urls["spotify"] = getSpotifyUrlFromAmazon(getTrackASIN(urls["amazon"]))
		case urls["apple"] != "error getting appleMusic URL":
			urls["spotify"] = getSpotifyUrlFromAppleMusic(getAppleMusicID(urls["apple"]))
		}
	}

	// Retry Apple Music URL
	if urls["apple"] == "error getting appleMusic URL" {
		switch {
		case urls["youtube"] != "error getting youtubeMusic URL":
			urls["apple"] = getAppleMusicUrlFromYoutube(getYoutubeID(urls["youtube"]))
		case urls["amazon"] != "error getting amazonMusic URL":
			urls["apple"] = getAppleMusicUrlFromAmazon(getTrackASIN(urls["amazon"]))
		case urls["spotify"] != "error getting spotify URL":
			urls["apple"] = getAppleMusicUrlFromSpotify(getSpotifyTrackID(urls["spotify"]))
		}
	}
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	msg := multipleUrl2SingleUrl(m.Content)
	var post []string

	for _, str := range msg {
		fmt.Println()
		var sourceType string
		var sourceID string

		switch {
		case strings.Contains(str, "https://spotify.link"):
			spotifyURL := convertSpotifyLink2OpenSpotifyCom(str)
			sourceType = "spotify"
			sourceID = getSpotifyTrackID(spotifyURL)
		case strings.Contains(str, "https://open.spotify.com"):
			sourceType = "spotify"
			sourceID = getSpotifyTrackID(str)
		case strings.Contains(str, "https://music.youtube.com/watch"):
			sourceType = "youtube"
			sourceID = getYoutubeID(str)
		case strings.Contains(str, "https://music.amazon"):
			sourceType = "amazon"
			sourceID = getTrackASIN(str)
		case strings.Contains(str, "music.apple.com"):
			sourceType = "apple"
			sourceID = getAppleMusicID(str)
		default:
			continue
		}

		if sourceID == "" {
			continue
		}

		// Retrieve URLs from each service
		urls := getURLsFromService(sourceType, sourceID)

		// Process to maintain the order of original URLs
		var urlsToPost []string
		switch sourceType {
		case "spotify":
			urlsToPost = []string{urls["youtube"], urls["amazon"], urls["apple"]}
		case "youtube":
			urlsToPost = []string{urls["spotify"], urls["amazon"], urls["apple"]}
		case "amazon":
			urlsToPost = []string{urls["spotify"], urls["youtube"], urls["apple"]}
		case "apple":
			urlsToPost = []string{urls["spotify"], urls["youtube"], urls["amazon"]}
		}

		// Add only non-error URLs
		for _, url := range urlsToPost {
			if !strings.Contains(url, "error getting") && url != "" {
				post = append(post, url)
			}
		}
	}

	if len(post) > 0 {
		postmsg := strings.Join(post, "\n")
		s.ChannelMessageSend(m.ChannelID, postmsg)
	}
}
