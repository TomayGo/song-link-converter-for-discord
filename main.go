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
		var spotifyURL string
		var spotifyTrackID string
		var youtubeID string
		var trackASIN string
		var fromspotify bool
		var fromyoutube bool
		var fromamazon bool
		switch {
		case strings.Contains(str, "https://spotify.link"):
			fromspotify = true
			spotifyURL = convertSpotifyLink2OpenSpotifyCom(str)
			spotifyTrackID = getSpotifyTrackID(spotifyURL)
		case strings.Contains(str, "https://open.spotify.com"):
			fromspotify = true
			spotifyURL = str
			spotifyTrackID = getSpotifyTrackID(spotifyURL)
		case strings.Contains(str, "https://music.youtube.com/watch"):
			fromyoutube = true
			youtubeID = getYoutubeID(str)
		case strings.Contains(str, "https://music.amazon"):
			fromamazon = true
			trackASIN = getTrackASIN(str)
		}

		if fromspotify {
			youtubeURL := getYoutubeUrlFromSpotify(spotifyTrackID)
			amazonURL := getAmazonUrlFromSpotify(spotifyTrackID)
			if youtubeURL == "error getting youtubeMusic URL" && amazonURL != "error getting amazonMusic URL" {
				youtubeURL = getYoutubeUrlFromAmazon(getTrackASIN(amazonURL))
			}
			if amazonURL == "error getting amazonMusic URL" && youtubeURL != "error getting youtubeMusic URL" {
				amazonURL = getAmazonUrlFromYoutube(getYoutubeID(youtubeURL))
			}
			post = append(post, youtubeURL, amazonURL)
		} else if fromyoutube {
			spotifyURL := getSpotifyUrlFromYoutube(youtubeID)
			amazonURL := getAmazonUrlFromYoutube(youtubeID)
			if spotifyURL == "error getting spotify URL" && amazonURL != "error getting amazonMusic URL" {
				spotifyURL = getSpotifyUrlFromAmazon(getTrackASIN(amazonURL))
			}
			if amazonURL == "error getting amazonMusic URL" && spotifyURL != "error getting spotify URL" {
				amazonURL = getAmazonUrlFromSpotify(getSpotifyTrackID(spotifyURL))
			}
			post = append(post, spotifyURL, amazonURL)
		} else if fromamazon {
			spotifyURL := getSpotifyUrlFromAmazon(trackASIN)
			youtubeURL := getYoutubeUrlFromAmazon(trackASIN)
			if spotifyURL == "error getting spotify URL" && youtubeURL != "error getting youtubeMusic URL" {
				spotifyURL = getSpotifyUrlFromYoutube(getYoutubeID(youtubeURL))
			}
			if youtubeURL == "error getting youtubeMusic URL" && spotifyURL != "error getting spotify URL" {
				youtubeURL = getYoutubeUrlFromSpotify(getSpotifyTrackID(spotifyURL))
			}
			post = append(post, spotifyURL, youtubeURL)
		}
	}

	postmsg := strings.Join(post, "\n")
	s.ChannelMessageSend(m.ChannelID, postmsg)

}
