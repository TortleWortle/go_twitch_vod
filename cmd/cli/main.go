package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
)

const ClientID = "jzkbprff40iqj646a697cyrvl0zt2m6"

func main() {
	if len(os.Args[1]) < 2 {
		log.Fatal("Please specify video id")
	}
	videoID := os.Args[1]

	err := os.Mkdir("./tmp", 0777)
	if err != nil {
		if !errors.Is(err, os.ErrExist) {
			log.Fatalf("could not create tmp folder: %v", err)
		}
	}

	err = downloadVod("./tmp", videoID)
	if err != nil {
		log.Fatal(err)
	}
}

type token struct {
	Token     string `json:"token"`
	Signature string `json:"sig"`
}

func getToken(id string) (tok token, err error) {
	url := fmt.Sprintf("https://api.twitch.tv/api/vods/%s/access_token?as3=t", id)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Client-ID", ClientID)
	req.Header.Set("Accept", "application/vnd.twitchtv.v5+json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(body, &tok)
	return
}

func findChunkedPlaylistUrl(body string) (string, error) {
	for _, line := range strings.Split(body, "\n") {
		if strings.HasSuffix(line, "/chunked/index-dvr.m3u8") {
			return line, nil
		}
	}
	return "", errors.New("could not find playlist")
}

func getPlaylistUrl(tok token, videoID string) (string, error) {
	url := fmt.Sprintf("http://usher.twitch.tv/vod/%s?nauth=%s&nauthsig=%s&allow_source=true&player=twitchweb&allow_spectre=true", videoID, tok.Token, tok.Signature)
	res, err := http.Get(url)
	if err != nil {
		return "nil", err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return findChunkedPlaylistUrl(string(body))
}

func downloadVod(tmpdir, videoID string) error {
	dir := path.Join(tmpdir, videoID)
	// create temp dir
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		if !errors.Is(err, os.ErrExist) {
			return err
		}
	}
	token, err := getToken(videoID)
	if err != nil {
		return err
	}

	playlistUrl, err := getPlaylistUrl(token, videoID)
	if err != nil {
		return err
	}

	baseUrl, parts, err := getParts(playlistUrl)
	if err != nil {
		return err
	}

	err = downloadParts(dir, baseUrl, parts)
	if err != nil {
		return err
	}

	err = mergeParts(dir, parts)
	if err != nil {
		return err
	}

	err = os.Rename(path.Join(dir, "merged.ts"), fmt.Sprintf("%s.ts", videoID))
	if err != nil {
		return err
	}

	err = os.RemoveAll(dir)
	if err != nil {
		return errors.New(fmt.Sprintf("cleanup err: %v", err))
	}
	return nil
}

func mergeParts(dir string, parts []string) error {
	merged, err := os.OpenFile(path.Join(dir, "merged.ts"), os.O_WRONLY | os.O_APPEND | os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer merged.Close()

	for _, p := range parts {
		f, err := os.OpenFile(path.Join(dir, p), os.O_RDONLY, 0666)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(merged, f)
		if err != nil {
			return errors.New(fmt.Sprintf("could not copy: %v", err))
		}
		f.Close()
	}
	return nil
}

func downloadParts(dir, baseUrl string, parts []string) error {
	for _, part := range parts {
		pUrl := baseUrl + part
		req, err := http.NewRequest(http.MethodGet, pUrl, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Host", req.URL.Host)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return errors.New(fmt.Sprintf("could not download part(%s): %v", part, err))
		}
		defer res.Body.Close()
		f, err := os.OpenFile(path.Join(dir, part), os.O_CREATE | os.O_RDWR, 0666)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(f, res.Body)
		if err != nil {
			return err
		}

		f.Close()
		res.Body.Close()
	}
	return nil
}

func getPartNames(playlistUrl string) ([]string, error) {
	res, err := http.Get(playlistUrl)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`[0-9]+\.ts`)
	parts := re.FindAllString(string(body), -1)
	if parts == nil {
		return nil, errors.New("returned no parts")
	}
	return parts, nil
}

func getParts(playlistUrl string) (string, []string, error) {
	re := regexp.MustCompile(`index-.*\.m3u8`)
	baseUrl := re.ReplaceAllString(playlistUrl, "")
	parts, err := getPartNames(playlistUrl)
	if err != nil {
		return "", nil, err
	}
	return baseUrl, parts, nil
}
