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
	Itunes       Platform = "itunes" // Note: song.link API uses "itunes" for Apple Music links sometimes
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

// Service identifiers (used as keys in maps)
const (
	serviceSpotify = "spotify"
	serviceYoutube = "youtube"
	serviceAmazon  = "amazon"
	serviceApple   = "apple"
)

// Content type identifiers
const (
	contentTypeTrack = "song" // song.link APIはtrackではなくsongを使用
	contentTypeAlbum = "album"
)

// Map service identifiers to their corresponding Platform type for song.link API
var serviceToPlatform = map[string]Platform{
	serviceSpotify: Spotify,
	serviceYoutube: YoutubeMusic, // song.link uses youtubeMusic for YouTube Music links
	serviceAmazon:  AmazonMusic,
	serviceApple:   AppleMusic, // or Itunes, depending on song.link's mood
}

// ContentInfo holds information about the content type and ID
type ContentInfo struct {
	contentType string
	id          string
}

// serviceIDGetters maps a service identifier to a function that extracts the content type and ID from a URL.
var serviceIDGetters = map[string]func(string) ContentInfo{
	serviceSpotify: getSpotifyID,
	serviceYoutube: getYoutubeID,
	serviceAmazon:  getAmazonID,
	serviceApple:   getAppleMusicID,
}

// serviceRelationship defines which services can be converted from a source service.
// The order can imply preference if needed, but here it's just a list of targets.
var serviceRelationship = map[string][]string{
	serviceSpotify: {serviceYoutube, serviceAmazon, serviceApple},
	serviceYoutube: {serviceSpotify, serviceAmazon, serviceApple},
	serviceAmazon:  {serviceSpotify, serviceYoutube, serviceApple},
	serviceApple:   {serviceSpotify, serviceYoutube, serviceAmazon},
}

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()
}

func main() {
	if Token == "" {
		fmt.Println("Bot token (-t) is required.")
		os.Exit(1)
	}

	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("Error creating Discord session:", err)
		return
	}

	dg.AddHandler(messageCreate)
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening connection:", err)
		return
	}

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	dg.Close()
}

// multipleUrl2SingleUrl splits a message content by newlines.
func multipleUrl2SingleUrl(m string) []string {
	return regexp.MustCompile(`\r\n|\n`).Split(m, -1)
}

// fetchAllMusicURLs fetches URLs for all related services in a single API call.
// Returns a map of targetServiceType -> URL. Error values start with "error getting".
func fetchAllMusicURLs(sourceType string, contentInfo ContentInfo) map[string]string {
	results := make(map[string]string)

	sourcePlatform, ok := serviceToPlatform[sourceType]
	if !ok {
		fmt.Printf("error: unknown source service type for platform mapping: %s\n", sourceType)
		return results
	}

	targetServiceTypes, ok := serviceRelationship[sourceType]
	if !ok {
		return results
	}

	// Use "JP" as userCountry, can be parameterized if needed.
	apiURL := fmt.Sprintf("https://api.song.link/v1-alpha.1/links?platform=%s&type=%s&id=%s&userCountry=JP&songIfSingle=true",
		sourcePlatform, contentInfo.contentType, contentInfo.id)
	fmt.Printf(apiURL + "\n") // Console output for debugging
	resp, err := http.Get(apiURL)
	if err != nil {
		for _, targetType := range targetServiceTypes {
			errMsg := fmt.Sprintf("error getting %s URL (http error for source %s, type %s, id %s): %v",
				targetType, sourceType, contentInfo.contentType, contentInfo.id, err)
			fmt.Println(errMsg) // Console output
			results[targetType] = errMsg
		}
		return results
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		for _, targetType := range targetServiceTypes {
			errMsg := fmt.Sprintf("error getting %s URL (status %d for source %s, type %s, id %s)",
				targetType, resp.StatusCode, sourceType, contentInfo.contentType, contentInfo.id)
			fmt.Println(errMsg) // Console output
			results[targetType] = errMsg
		}
		return results
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		for _, targetType := range targetServiceTypes {
			errMsg := fmt.Sprintf("error getting %s URL (read body error for source %s, type %s, id %s): %v",
				targetType, sourceType, contentInfo.contentType, contentInfo.id, err)
			fmt.Println(errMsg) // Console output
			results[targetType] = errMsg
		}
		return results
	}

	var response Response
	if err = json.Unmarshal(body, &response); err != nil {
		var errorResponse struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		apiErr := json.Unmarshal(body, &errorResponse) == nil && errorResponse.Message != ""
		for _, targetType := range targetServiceTypes {
			var errMsg string
			if apiErr {
				errMsg = fmt.Sprintf("error getting %s URL (api error %d for source %s, type %s, id %s: %s)",
					targetType, errorResponse.Code, sourceType, contentInfo.contentType, contentInfo.id, errorResponse.Message)
			} else {
				errMsg = fmt.Sprintf("error getting %s URL (unmarshal error for source %s, type %s, id %s): %v. Body: %s",
					targetType, sourceType, contentInfo.contentType, contentInfo.id, err, string(body))
			}
			fmt.Println(errMsg) // Console output
			results[targetType] = errMsg
		}
		return results
	}

	for _, targetServiceType := range targetServiceTypes {
		targetPlatform, ok := serviceToPlatform[targetServiceType]
		if !ok {
			errMsg := fmt.Sprintf("error: unknown target service type for platform mapping: %s", targetServiceType)
			fmt.Println(errMsg) // Console output
			results[targetServiceType] = errMsg
			continue
		}

		linkData, exists := response.LinksByPlatform[targetPlatform]
		if !exists {
			if targetPlatform == AppleMusic {
				linkData, exists = response.LinksByPlatform[Itunes]
			}
			if !exists {
				errMsg := fmt.Sprintf("error getting %s URL (%s link not found in API response for source %s, type %s, id %s)",
					targetServiceType, targetPlatform, sourceType, contentInfo.contentType, contentInfo.id)
				fmt.Println(errMsg) // Console output
				results[targetServiceType] = errMsg
				continue
			}
		}

		finalURL := linkData.Url
		if targetServiceType == serviceAmazon {
			finalURL = strings.Replace(finalURL, ".com", ".co.jp", 1)
		}
		results[targetServiceType] = finalURL
	}

	return results
}

// retryFromOtherService attempts to fetch URLs for services that failed in the initial attempt,
// by using services that were successfully fetched as new sources.
// Each successful source makes at most one API call per retry attempt.
func retryFromOtherService(urls map[string]string, initialSourceType string) {
	maxRetries := len(serviceRelationship[initialSourceType])
	if maxRetries == 0 {
		maxRetries = 3
	}

	for retryAttempt := 0; retryAttempt < maxRetries; retryAttempt++ {
		servicesThatFailed := []string{}
		successfulServices := make(map[string]string)

		for service, url := range urls {
			if strings.HasPrefix(url, "error getting") {
				servicesThatFailed = append(servicesThatFailed, service)
			} else if url != "" {
				successfulServices[service] = url
			}
		}

		if len(servicesThatFailed) == 0 {
			break
		}

		madeProgressInThisAttempt := false
		for potentialNewSourceType, potentialNewSourceURL := range successfulServices {
			idExtractor, ok := serviceIDGetters[potentialNewSourceType]
			if !ok {
				continue
			}

			newContentInfo := idExtractor(potentialNewSourceURL)
			if newContentInfo.id == "" {
				continue
			}

			// 1 API call per source — covers all failed targets at once
			fetchedURLs := fetchAllMusicURLs(potentialNewSourceType, newContentInfo)

			for _, targetServiceToFix := range servicesThatFailed {
				if !strings.HasPrefix(urls[targetServiceToFix], "error getting") {
					continue // already fixed by a previous source
				}
				fetchedURL, exists := fetchedURLs[targetServiceToFix]
				if exists && !strings.HasPrefix(fetchedURL, "error getting") && fetchedURL != "" {
					urls[targetServiceToFix] = fetchedURL
					madeProgressInThisAttempt = true
				}
			}
		}

		if !madeProgressInThisAttempt {
			break
		}
	}
}

// getURLsFromService orchestrates fetching URLs for related services.
func getURLsFromService(sourceType string, contentInfo ContentInfo) map[string]string {
	if _, ok := serviceRelationship[sourceType]; !ok {
		fmt.Printf("Error: No defined relationship for source service type '%s'\n", sourceType)
		return make(map[string]string)
	}

	urls := fetchAllMusicURLs(sourceType, contentInfo)
	retryFromOtherService(urls, sourceType)
	return urls
}

// convertSpotifyLink2OpenSpotifyCom handles spotify.link short URLs
func convertSpotifyLink2OpenSpotifyCom(shortURL string) string {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Head(shortURL)
	if err != nil {
		getResp, getErr := http.Get(shortURL)
		if getErr != nil {
			fmt.Printf("Error resolving spotify.link (GET): %v\n", getErr)
			return ""
		}
		defer getResp.Body.Close()
		finalURL := getResp.Request.URL.String()
		if strings.Contains(finalURL, "open.spotify.com") {
			return finalURL
		}
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode <= 399 {
		location := resp.Header.Get("Location")
		if strings.Contains(location, "open.spotify.com") {
			return location
		}
	} else if strings.Contains(resp.Request.URL.String(), "open.spotify.com") {
		return resp.Request.URL.String()
	}

	req, _ := http.NewRequest("GET", shortURL, nil)
	httpClient := new(http.Client)
	httpResp, err := httpClient.Do(req)
	if err != nil {
		fmt.Println("Error resolving spotify.link (fallback GET):", err)
		return ""
	}
	defer httpResp.Body.Close()

	finalURL := httpResp.Request.URL.String()
	if strings.Contains(finalURL, "open.spotify.com") {
		if qIndex := strings.Index(finalURL, "?"); qIndex != -1 {
			return finalURL[:qIndex]
		}
		return finalURL
	}

	dumpResp, _ := httputil.DumpResponse(httpResp, true)
	getSpotifyURLRegex, _ := regexp.Compile(`https://open\.spotify\.com[^?\s"]+`)
	spotifyURLMatch := getSpotifyURLRegex.FindString(string(dumpResp))
	if spotifyURLMatch != "" {
		return spotifyURLMatch
	}

	fmt.Printf("Could not resolve spotify.link %s to an open.spotify.com URL\n", shortURL)
	return ""
}

func getSpotifyID(spotifyURL string) ContentInfo {
	trackRe := regexp.MustCompile(`track/(\w+)`)
	albumRe := regexp.MustCompile(`album/(\w+)`)

	if matches := trackRe.FindStringSubmatch(spotifyURL); len(matches) >= 2 {
		return ContentInfo{contentType: contentTypeTrack, id: matches[1]}
	}
	if matches := albumRe.FindStringSubmatch(spotifyURL); len(matches) >= 2 {
		return ContentInfo{contentType: contentTypeAlbum, id: matches[1]}
	}
	return ContentInfo{}
}

func getYoutubeID(youtubeURL string) ContentInfo {
	// 曲のID抽出
	trackRe := regexp.MustCompile(`(?:v=|youtu\.be/|embed/|shorts/)([a-zA-Z0-9_-]{11})`)
	if matches := trackRe.FindStringSubmatch(youtubeURL); len(matches) >= 2 {
		return ContentInfo{contentType: contentTypeTrack, id: matches[1]}
	}

	// プレイリスト（アルバム）のID抽出
	playlistRe := regexp.MustCompile(`playlist\?list=([A-Za-z0-9_-]+)`)
	if matches := playlistRe.FindStringSubmatch(youtubeURL); len(matches) >= 2 {
		return ContentInfo{contentType: contentTypeAlbum, id: matches[1]}
	}

	return ContentInfo{}
}

func getAmazonID(amazonURL string) ContentInfo {
	trackRe := regexp.MustCompile(`trackAsin=([A-Z0-9]{10})`)
	albumRe := regexp.MustCompile(`(?:albums|dp)/([A-Z0-9]{10})`)

	if matches := trackRe.FindStringSubmatch(amazonURL); len(matches) >= 2 {
		return ContentInfo{contentType: contentTypeTrack, id: matches[1]}
	}
	if matches := albumRe.FindStringSubmatch(amazonURL); len(matches) >= 2 {
		return ContentInfo{contentType: contentTypeAlbum, id: matches[1]}
	}
	return ContentInfo{}
}

func getAppleMusicID(appleMusicURL string) ContentInfo {
	trackRe := regexp.MustCompile(`\?i=(\d+)`)
	albumRe := regexp.MustCompile(`album/[^/]+/(\d+)`)

	if matches := trackRe.FindStringSubmatch(appleMusicURL); len(matches) >= 2 {
		return ContentInfo{contentType: contentTypeTrack, id: matches[1]}
	}
	if matches := albumRe.FindStringSubmatch(appleMusicURL); len(matches) >= 2 {
		return ContentInfo{contentType: contentTypeAlbum, id: matches[1]}
	}
	return ContentInfo{}
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	inputURLs := multipleUrl2SingleUrl(m.Content)
	var successfulUrlsToPost []string
	var failedServicesMessages []string
	processedSourceURLs := make(map[string]bool)

	for _, currentInputURL := range inputURLs {
		if strings.TrimSpace(currentInputURL) == "" {
			continue
		}

		var effectiveURL = currentInputURL
		var sourceType string
		var contentInfo ContentInfo

		if strings.Contains(currentInputURL, "spotify.link") {
			resolvedSpotifyURL := convertSpotifyLink2OpenSpotifyCom(currentInputURL)
			if resolvedSpotifyURL != "" {
				effectiveURL = resolvedSpotifyURL
				sourceType = serviceSpotify
				contentInfo = getSpotifyID(effectiveURL)
			} else {
				fmt.Printf("Console: Could not resolve spotify.link: %s\n", currentInputURL)
				failedServicesMessages = append(failedServicesMessages, fmt.Sprintf("error getting Spotify Link (%s): Failed to resolve shortened URL.", currentInputURL))
				continue
			}
		} else if strings.Contains(currentInputURL, "open.spotify.com") {
			sourceType = serviceSpotify
			contentInfo = getSpotifyID(currentInputURL)
			effectiveURL = currentInputURL
		} else if strings.Contains(currentInputURL, "music.youtube.com/") {
			sourceType = serviceYoutube
			contentInfo = getYoutubeID(currentInputURL)
			effectiveURL = currentInputURL
		} else if strings.Contains(currentInputURL, "music.amazon.") {
			sourceType = serviceAmazon
			contentInfo = getAmazonID(currentInputURL)
			effectiveURL = currentInputURL
		} else if strings.Contains(currentInputURL, "music.apple.com/") {
			sourceType = serviceApple
			contentInfo = getAppleMusicID(currentInputURL)
			effectiveURL = currentInputURL
		} else {
			continue
		}

		if contentInfo.id == "" {
			fmt.Printf("Console: Could not extract ID from URL: %s (Service Type: %s)\n", currentInputURL, sourceType)
			failedServicesMessages = append(failedServicesMessages, fmt.Sprintf("error getting %s ID from URL (%s): ID抽出失敗", strings.Title(sourceType), currentInputURL))
			continue
		}

		if processedSourceURLs[effectiveURL] {
			continue
		}
		processedSourceURLs[effectiveURL] = true

		retrievedServiceURLs := getURLsFromService(sourceType, contentInfo)

		targetServiceTypes := serviceRelationship[sourceType]
		for _, targetType := range targetServiceTypes {
			url, ok := retrievedServiceURLs[targetType]
			if ok && url != "" {
				if !strings.HasPrefix(url, "error getting") {
					isNew := true
					for _, existing := range successfulUrlsToPost {
						if existing == url {
							isNew = false
							break
						}
					}
					if isNew {
						successfulUrlsToPost = append(successfulUrlsToPost, url)
					}
				} else {
					rawError := url
					isNewError := true
					for _, existingMsg := range failedServicesMessages {
						if strings.HasPrefix(existingMsg, fmt.Sprintf("error getting %s", strings.Title(targetType))) {
							isNewError = false
							break
						}
					}
					if isNewError {
						failedServicesMessages = append(failedServicesMessages, rawError)
					}
					fmt.Printf("Console: Failed to get URL for %s from source %s (ID: %s): %s\n", targetType, sourceType, contentInfo.id, rawError)
				}
			} else if !ok {
				rawError := fmt.Sprintf("error getting %s: Failed to retrieve information (internal error, target not found).", strings.Title(targetType))
				failedServicesMessages = append(failedServicesMessages, rawError)
				fmt.Printf("Console: Target service %s not found in retrieved URLs map for source %s (ID: %s)\n", targetType, sourceType, contentInfo.id)
			}
		}
	}

	var messageParts []string

	if len(successfulUrlsToPost) > 0 {
		finalUniqueSuccessfulUrls := []string{}
		seenFinalUrls := make(map[string]bool)
		for _, url := range successfulUrlsToPost {
			if !seenFinalUrls[url] {
				finalUniqueSuccessfulUrls = append(finalUniqueSuccessfulUrls, url)
				seenFinalUrls[url] = true
			}
		}
		if len(finalUniqueSuccessfulUrls) > 0 {
			messageParts = append(messageParts, strings.Join(finalUniqueSuccessfulUrls, "\n"))
		}
	}

	if len(failedServicesMessages) > 0 {
		uniqueFailedMessages := []string{}
		seenFailed := make(map[string]bool)
		for _, msg := range failedServicesMessages {
			if !seenFailed[msg] {
				uniqueFailedMessages = append(uniqueFailedMessages, msg)
				seenFailed[msg] = true
			}
		}
		if len(uniqueFailedMessages) > 0 {
			if len(messageParts) > 0 {
				messageParts = append(messageParts, "\n") // Add a separator if there were successful URLs
			}
			messageParts = append(messageParts, "The following services could not be retrieved:")
			messageParts = append(messageParts, strings.Join(uniqueFailedMessages, "\n"))
		}
	}

	if len(messageParts) > 0 {
		responseMessage := strings.Join(messageParts, "\n")
		// Ensure message is not empty or just whitespace before sending
		if strings.TrimSpace(responseMessage) != "" {
			_, err := s.ChannelMessageSend(m.ChannelID, responseMessage)
			if err != nil {
				fmt.Println("Error sending message:", err)
			}
		} else {
			fmt.Println("Console: No valid information to send to Discord.")
		}
	} else {
		fmt.Println("Console: No URLs or error messages to post.")
	}
}
