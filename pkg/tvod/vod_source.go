package tvod

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

type VodPart struct {
	Name    string
	URL     string
	BaseURL string
}

func (p VodPart) Download(ctx context.Context, writer io.Writer) (int64, error) {
	stream, err := p.Stream(ctx)
	if err != nil {
		return 0, err
	}
	defer stream.Close()
	return io.Copy(writer, stream)
}

func (p VodPart) Stream(ctx context.Context) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.URL, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == 200 {
		return res.Body, nil
	}
	return nil, fmt.Errorf("server did not respond with 200 (%d): %v", res.StatusCode, res)
}

type VodSource struct {
	Name        string
	playlistURL string
}

// GetParts get video parts from the playlist file (makes a request)
func (s VodSource) GetParts(ctx context.Context) ([]VodPart, error) {
	parts := strings.Split(s.playlistURL, "/")
	baseUrl := strings.Join(parts[:len(parts)-1], "/")

	vodParts := make([]VodPart, 0)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.playlistURL, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`[0-9]+\.ts`)
	partNames := re.FindAllString(string(body), -1)
	if partNames == nil {
		return nil, errors.New("returned no parts")
	}

	for _, name := range partNames {
		vodParts = append(vodParts, VodPart{
			Name:    name,
			BaseURL: baseUrl,
			URL:     fmt.Sprintf("%s/%s", baseUrl, name),
		})
	}

	return vodParts, nil
}
