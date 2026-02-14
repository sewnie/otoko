package bandcamp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

// Client embeds an [http.Client] to make currently implemented Bandcamp
// API calls that are undocumented.
type Client struct {
	BaseURL *url.URL
	*http.Client
}

// New returns a new Client. To make authenticated API calls,
// an authenticated auoted Bandcamp login 'identity' cookie is required.
func New(identity string) *Client {
	url := url.URL{Scheme: "https", Host: "bandcamp.com"}

	// Cookiejar preferred for Bandcamp to get the best BACKENDID
	jar, _ := cookiejar.New(nil)

	jar.SetCookies(&url, []*http.Cookie{
		{Name: "identity", Value: identity, Quoted: false, Domain: url.Host},
	})

	return &Client{
		BaseURL: url.JoinPath("api"),
		Client:  &http.Client{Jar: jar},
	}
}

func (c *Client) Request(method, endpoint string, body, v any) error {
	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return err
		}
	}

	req, err := http.NewRequest(
		method, c.BaseURL.JoinPath(endpoint).String(), buf)
	if err != nil {
		return err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	content := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(content, "application/json") {
		return &StatusError{StatusCode: resp.StatusCode}
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	ret := new(Error)
	if err := json.Unmarshal(b, &ret); err == nil && ret.IsError {
		return ret
	}

	if v != nil {
		return json.Unmarshal(b, &v)
	}

	return nil
}

type StatusError struct {
	StatusCode int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("bad response: %s", http.StatusText(e.StatusCode))
}

type Error struct {
	IsError bool   `json:"error"`
	Message string `json:"error_message"`
}

func (e Error) Error() string {
	return e.Message
}
