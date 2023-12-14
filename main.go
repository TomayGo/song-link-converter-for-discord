package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
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

func spotify(input string) string {
	output, err := exec.Command("./venv/bin/odesli-cli", input, "--provider", "youtubeMusic", "link").CombinedOutput()
	if err != nil {
		output, err = exec.Command("./venv/bin/odesli-cli", input, "--provider", "youtube", "link").CombinedOutput()
		if err != nil {
			fmt.Println("error getting any youtube link,", err)
			return "error getting youtube link"
		}
	}
	outputStr := strings.Replace(string(output), "www", "music", 1)
	return outputStr
}

func youtube(input string) string {
	output, err := exec.Command("./venv/bin/odesli-cli", input, "--provider", "spotify", "link").Output()
	if err != nil {
		fmt.Println("error getting spotify link,", err)
		return "error getting spotify link"
	}
	return string(output)
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	var postURL string
	if strings.Contains(m.Content, "https://spotify.link") {
		postURL = spotify(m.Content)
	} else if strings.Contains(m.Content, "https://open.spotify.com") {
		postURL = spotify(m.Content)
	} else if strings.Contains(m.Content, "https://music.youtube.com") {
		postURL = youtube(m.Content)
	}

	s.ChannelMessageSend(m.ChannelID, postURL)

}
