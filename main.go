package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	spotifyauth "github.com/zmb3/spotify/v2/auth"

	"github.com/bwmarrin/discordgo"
	"github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
)

// Variables used for command line parameters
var (
	Token string
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

func convertSpotifyLink2OpenSpotifyCom(m *discordgo.MessageCreate) string {
	re, err := regexp.Compile(`http(.*)://(.*)`)
	if err != nil {
		fmt.Println("error compiling regex ,", err)
		return ""
	}
	url := re.FindString(m.Content)
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

func getSpotifyTrackData(spotifyTrackID string) (string, string) {
	// Set your client credentials and callback URI.
	clientID := "b1448db923bd4a2c9cae7dccc9271cce"
	clientSecret := "f565e9c262cd49e09aea71fffecc2178"

	ctx := context.Background()
	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     spotifyauth.TokenURL,
	}
	token, err := config.Token(ctx)
	if err != nil {
		fmt.Println("error getting token,", err)
		return "", ""
	}

	httpClient := spotifyauth.New().Client(ctx, token)
	client := spotify.New(httpClient)

	results, err := client.GetTrack(context.Background(), spotify.ID(spotifyTrackID))
	if err != nil {
		fmt.Println("error getting track,", err)
		return "", ""
	}
	name := results.Name
	artist := results.Artists[0].Name
	return name, artist
}

func searchYoutube(name string, artist string) string {

	const developerKey = "AIzaSyADgBzQTFag1rJFnEb19giEcXS7s8ffpSg"
	flag.Parse()

	client := &http.Client{
		Transport: &transport.APIKey{Key: developerKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		fmt.Println("Error creating new YouTube client: %v", err)
	}

	// Make the API call to YouTube.
	call := service.Search.List([]string{"id", "snippet"}).
		Q(name + " " + artist).
		MaxResults(1)
	response, err := call.Do()
	if err != nil {
		fmt.Println("Error making search API call: %v", err)
	}

	var videoId string
	// If there is at least one search result, get the ID of the first video.
	if len(response.Items) > 0 {
		videoId = response.Items[0].Id.VideoId
	}
	URL := "https://music.youtube.com/watch?v=" + videoId
	return URL
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
	var name string
	var artist string
	var postURL string

	if strings.Contains(m.Content, "https://spotify.link") {
		spotifyURL = convertSpotifyLink2OpenSpotifyCom(m)
		spotifyTrackID = getSpotifyTrackID(spotifyURL)
		name, artist = getSpotifyTrackData(spotifyTrackID)
		postURL = searchYoutube(name, artist)
	} else if strings.Contains(m.Content, "https://open.spotify.com") {
		spotifyURL = m.Content
		spotifyTrackID = getSpotifyTrackID(spotifyURL)
		name, artist = getSpotifyTrackData(spotifyTrackID)
		postURL = searchYoutube(name, artist)
	}

	s.ChannelMessageSend(m.ChannelID, postURL)

}
