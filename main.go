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
	url := "https://music.youtube.com/watch?v=b5E4Q9_DC5A"
	re, err := regexp.Compile(`watch\?v=(.*)`)
	if err != nil {
		fmt.Println("error compiling regex,", err)
		return ""
	}
	matches := re.FindStringSubmatch(url)
	if len(matches) < 2 {
		fmt.Println("No match found")
		return ""
	}
	return matches[1]
}

func getYoutubeUrl(spotifyTrackID string) string {
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
	youtubeUrl, ok := response.LinksByPlatform["youtubeMusic"]
	if !ok {
		fmt.Println("youtubeMusic URL not found")
		return ""
	}
	postURL := youtubeUrl.Url
	return postURL
}

func getSpotifyUrl(youtubeID string) string {
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
	youtubeUrl, ok := response.LinksByPlatform["spotify"]
	if !ok {
		fmt.Println("spotify URL not found")
		return ""
	}
	postURL := youtubeUrl.Url
	return postURL
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	var spotifyURL string
	var spotifyTrackID string
	var postURL string

	if strings.Contains(m.Content, "https://spotify.link") {
		spotifyURL = convertSpotifyLink2OpenSpotifyCom(m.Content)
		spotifyTrackID = getSpotifyTrackID(spotifyURL)
		postURL = getYoutubeUrl(spotifyTrackID)
	} else if strings.Contains(m.Content, "https://open.spotify.com") {
		spotifyURL = m.Content
		spotifyTrackID = getSpotifyTrackID(spotifyURL)
		postURL = getYoutubeUrl(spotifyTrackID)
	} else if strings.Contains(m.Content, "https://music.youtube.com/watch") {
		youtubeID := getYoutubeID(m.Content)
		postURL = getSpotifyUrl(youtubeID)
	}

	s.ChannelMessageSend(m.ChannelID, postURL)

}
