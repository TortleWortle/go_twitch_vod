package tvod

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

const clientID = "jzkbprff40iqj646a697cyrvl0zt2m6"

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

func (v *Vod) loadToken(ctx context.Context) error {
	url := fmt.Sprintf("https://api.twitch.tv/api/vods/%s/access_token?as3=t", v.videoID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Client-ID", clientID)
	req.Header.Set("Accept", "application/vnd.twitchtv.v5+json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var tok token
	err = json.Unmarshal(body, &tok)
	if err != nil {
		return err
	}
	v.token = tok
	return nil
}

// Load the info from twitch servers
func (v *Vod) Load(ctx context.Context) error {
	// grab token
	err := v.loadToken(ctx)
	if err != nil {
		return err
	}
	// grab sources
	err = v.loadSources(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (v *Vod) loadSources(ctx context.Context) error {
	url := fmt.Sprintf("http://usher.twitch.tv/vod/%s?nauth=%s&nauthsig=%s&allow_source=true&player=twitchweb&allow_spectre=true", v.videoID, v.token.Token, v.token.Signature)
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
		return err
	}
	sources, err := parseSources(string(body))
	if err != nil {
		return err
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
