// Package bandcamp provides required Web API access to undocumented user API.
package bandcamp

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"
)

type FanID int64

type Fan struct {
	Username string `json:"username"`
	URL      string `json:"url"`
	ID       FanID  `json:"fan_id"`
	// Tralbum lookup/following ignored
}

// Report the current authenticated user.
func (c *Client) GetFan() (*Fan, error) {
	var data struct {
		Summary Fan `json:"collection_summary"`
	}

	err := c.Request("GET", "fan/2/collection_summary", nil, &data)
	if err != nil {
		return nil, err
	}

	return &data.Summary, nil
}

type Time struct {
	time.Time
}

func (t *Time) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == 'n' {
		*t = Time{}
		return nil
	}
	date, err := time.Parse(`"02 Jan 2006 15:04:05 GMT"`, string(b))
	if err != nil {
		return err
	}
	*t = Time{Time: date}
	return nil
}

type Track struct {
	// Artist, duration excluded
	Title  string
	Number int64  // 0 if item is track
	URL    string // empty if not from wishlist
}

func (t *Track) UnmarshalJSON(b []byte) error {
	// Bandcamp stores metadata differently if it is from the
	// Mobile API (track_num vs track_number ??? etc)
	var data map[string]any

	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	title, ok := data["title"].(string)
	if !ok {
		return errors.New("bandcamp: expected title")
	}
	t.Title = title

	num, ok := data["track_number"].(int64)
	if !ok {
		num, ok = data["track_num"].(int64)
	}
	if ok {
		t.Number = int64(num)
	}

	s, ok := data["streaming_url"].(map[string]string)
	if ok {
		t.URL = s["mp3-128"]
	}

	return nil
}

// Bandcamp store this behind the fancollection/1/collection_items endpoint, and
// also keep it under the page-data if visited the user's profile.
type Collection []Item

func (c *Collection) UnmarshalJSON(b []byte) error {
	var data struct {
		Items          []Item             `json:"items"`
		RedownloadURLs map[string]string  `json:"redownload_urls"`
		Tracklist      map[string][]Track `json:"tracklists"`
	}

	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	for _, i := range data.Items {
		var ok bool

		i.Tracks, ok = data.Tracklist[i.String()]
		if !ok {
			return fmt.Errorf("item %s missing tracklist", i)
		}

		i.Download, ok = data.RedownloadURLs[i.Sale.String()]
		if !ok {
			return fmt.Errorf("item %s missing redownload", i)
		}

		*c = append(*c, i)
	}

	return nil
}

func (c *Client) GetCollection(id FanID) (Collection, error) {
	var ci Collection

	body := map[string]any{
		"fan_id": id,
		// UNIX time:item ID:a|d|t:count:
		"older_than_token": fmt.Sprintf("%d::a::", time.Now().Unix()),
		"count":            math.MaxInt64, // Entire collection
	}
	err := c.Request("POST", "fancollection/1/collection_items", body, &ci)
	if err != nil {
		return nil, err
	}

	return ci, nil
}

func (c *Client) GetWishlist(id FanID) ([]Item, error) {
	var data struct {
		Items     []Item             `json:"items"`
		Tracklist map[string][]Track `json:"tracklists"`
	}

	body := map[string]any{
		"fan_id": id,
		// UNIX time:item ID:a|d|t:count:
		"older_than_token": fmt.Sprintf("%d::a::", time.Now().Unix()),
		"count":            math.MaxInt64, // Entire collection
	}

	err := c.Request("POST", "fancollection/1/wishlist_items", body, &data)
	if err != nil {
		return nil, err
	}

	for i := range data.Items {
		var ok bool

		data.Items[i].Tracks, ok = data.Tracklist[data.Items[i].String()]
		if !ok {
			return nil, fmt.Errorf("item %d missing tracklist", i)
		}

	}
	return data.Items, nil
}

// Value uses the currency data in the HTML metadata using the given fan's
// URL page to report the total cost of the given items converted to the given
// target currency.
func (c *Client) Value(f *Fan, items Collection, target string) (float64, error) {
	var data struct {
		Currencies struct {
			Rates map[string]float64
		} `json:"currency_data"`
	}

	req, err := http.NewRequest("GET", f.URL, nil)
	if err != nil {
		return -1, err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	if err := decodeBlob(resp.Body, &data); err != nil {
		return -1, err
	}

	rate, ok := data.Currencies.Rates[target]
	if !ok {
		return -1, fmt.Errorf("unknown currency: %s", target)
	}

	var total float64
	for _, item := range items {
		rate, _ := data.Currencies.Rates[item.Currency]
		total += item.Price * rate
	}

	return total / rate, nil
}
