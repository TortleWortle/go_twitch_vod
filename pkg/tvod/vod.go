package tvod

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"strings"
)

const clientID = "kimne78kx3ncx6brgo4mv6wki5h1ko"

type Vod struct {
	Sources []VodSource
	token   token
	videoID string
}

type token struct {
	Token     string `json:"token"`
	Signature string `json:"sig"`
}

// NewVod initializes an empty Vod use Load() to load info from twitch
func NewVod(videoID string) *Vod {
	vod := &Vod{
		token:   token{},
		videoID: videoID,
	}
	return vod
}

type loadTokenResponse struct {
	Data struct {
		VideoPlaybackAccessToken struct {
			Value     string `json:"value"`
			Signature string `json:"signature"`
			Typename  string `json:"__typename"`
		} `json:"videoPlaybackAccessToken"`
	} `json:"data"`
	Extensions struct {
		DurationMilliseconds int    `json:"durationMilliseconds"`
		OperationName        string `json:"operationName"`
		RequestID            string `json:"requestID"`
	} `json:"extensions"`
}

func getLoadTokenBody(id string) string {
	return fmt.Sprintf(`{"operationName":"PlaybackAccessToken_Template","query":"query PlaybackAccessToken_Template($login: String!, $isLive: Boolean!, $vodID: ID!, $isVod: Boolean!, $playerType: String!, $platform: String!) {  streamPlaybackAccessToken(channelName: $login, params: {platform: $platform, playerBackend: \"mediaplayer\", playerType: $playerType}) @include(if: $isLive) {    value    signature   authorization { isForbidden forbiddenReasonCode }   __typename  }  videoPlaybackAccessToken(id: $vodID, params: {platform: $platform, playerBackend: \"mediaplayer\", playerType: $playerType}) @include(if: $isVod) {    value    signature   __typename  }}","variables":{"isLive":false,"login":"","isVod":true,"vodID":"%s","playerType":"site","platform":"web"}}`, id)
}

func (v *Vod) loadToken(ctx context.Context) error {
	url := "https://gql.twitch.tv/gql"
	body := strings.NewReader(getLoadTokenBody(v.videoID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return errors.Join(errors.New("creating request"), err)
	}
	req.Header.Set("Client-ID", clientID)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Join(errors.New("performing"), err)
	}
	defer res.Body.Close()
	slog.Info("token response", slog.Int("status", res.StatusCode))
	bodyRes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.Join(errors.New("reading body"), err)
	}
	var tok loadTokenResponse
	err = json.Unmarshal(bodyRes, &tok)
	if err != nil {
		return errors.Join(errors.New("unmarshalling json"), err)
	}

	newToken := token{
		Token:     tok.Data.VideoPlaybackAccessToken.Value,
		Signature: tok.Data.VideoPlaybackAccessToken.Signature,
	}
	v.token = newToken
	return nil
}

// Load the info from twitch servers
func (v *Vod) Load(ctx context.Context) error {
	// grab token
	slog.Info("loading token")
	err := v.loadToken(ctx)
	if err != nil {
		return errors.Join(errors.New("loading token"), err)
	}
	// grab sources
	slog.Info("loading sources")
	err = v.loadSources(ctx)
	if err != nil {
		return errors.Join(errors.New("loading sources"), err)
	}
	return nil
}

func (v *Vod) loadSources(ctx context.Context) error {
	url := fmt.Sprintf("http://usher.ttvnw.net/vod/%s.m3u8?nauth=%s&nauthsig=%s&allow_source=true&player=twitchweb&allow_spectre=true", v.videoID, v.token.Token, v.token.Signature)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("could not get sources (statuscode: %d)", res.StatusCode)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.Join(errors.New("reading body"), err)
	}
	sources, err := parseSources(string(body))
	if err != nil {
		return errors.Join(errors.New("parsing sources"), err)
	}
	v.Sources = sources
	return nil
}

// TODO: parse actual info from body
func parseSources(body string) ([]VodSource, error) {
	sources := make([]VodSource, 0)
	lines := strings.Split(body, "\n")
	sourceLines := make([]string, 0)
	for _, line := range lines {
		if strings.HasSuffix(line, "/index-dvr.m3u8") {
			sourceLines = append(sourceLines, line)
		}
	}
	for _, line := range sourceLines {
		parts := strings.Split(line, "/")
		name := parts[len(parts)-2]
		source := VodSource{
			Name:        name,
			playlistURL: line,
		}
		sources = append(sources, source)
	}
	return sources, nil
}
