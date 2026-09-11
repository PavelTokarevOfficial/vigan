package twitch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Clip struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnail_url"`
	Duration     float64   `json:"duration"`
	CreatedAt    time.Time `json:"created_at"`
	Saved        bool      `json:"saved"`
}
type Client struct {
	id, secret, token string
	expires           time.Time
	mu                sync.Mutex
	http              *http.Client
}
type User struct {
	ID          string `json:"id"`
	Login       string `json:"login"`
	DisplayName string `json:"display_name"`
}

func (c *Client) User(ctx context.Context, login string) (User, error) {
	t, e := c.tokenFor(ctx)
	if e != nil {
		return User{}, e
	}
	r, e := http.NewRequestWithContext(ctx, "GET", "https://api.twitch.tv/helix/users?login="+url.QueryEscape(login), nil)
	if e != nil {
		return User{}, e
	}
	r.Header.Set("Client-Id", c.id)
	r.Header.Set("Authorization", "Bearer "+t)
	res, e := c.http.Do(r)
	if e != nil {
		return User{}, e
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return User{}, fmt.Errorf("twitch users status %d", res.StatusCode)
	}
	var x struct {
		Data []User `json:"data"`
	}
	if e = json.NewDecoder(res.Body).Decode(&x); e != nil {
		return User{}, e
	}
	if len(x.Data) == 0 {
		return User{}, fmt.Errorf("twitch user not found")
	}
	return x.Data[0], nil
}

func New(id, secret string) *Client {
	return &Client{id: id, secret: secret, http: &http.Client{Timeout: 15 * time.Second}}
}
func (c *Client) tokenFor(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.expires) {
		return c.token, nil
	}
	v := url.Values{"client_id": {c.id}, "client_secret": {c.secret}, "grant_type": {"client_credentials"}}
	r, e := http.NewRequestWithContext(ctx, "POST", "https://id.twitch.tv/oauth2/token", strings.NewReader(v.Encode()))
	if e != nil {
		return "", e
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, e := c.http.Do(r)
	if e != nil {
		return "", e
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return "", fmt.Errorf("twitch token status %d", res.StatusCode)
	}
	var x struct {
		AccessToken string `json:"access_token"`
		Expires     int    `json:"expires_in"`
	}
	if e = json.NewDecoder(res.Body).Decode(&x); e != nil {
		return "", e
	}
	c.token = x.AccessToken
	c.expires = time.Now().Add(time.Duration(x.Expires-60) * time.Second)
	return c.token, nil
}
func (c *Client) Clips(ctx context.Context, broadcasterID string, startedAt, endedAt time.Time) ([]Clip, error) {
	t, e := c.tokenFor(ctx)
	if e != nil {
		return nil, e
	}
	query := url.Values{
		"broadcaster_id": {broadcasterID},
		"ended_at":       {endedAt.UTC().Format(time.RFC3339)},
		"first":          {"100"},
		"started_at":     {startedAt.UTC().Format(time.RFC3339)},
	}
	u := "https://api.twitch.tv/helix/clips?" + query.Encode()
	r, e := http.NewRequestWithContext(ctx, "GET", u, nil)
	if e != nil {
		return nil, e
	}
	r.Header.Set("Client-Id", c.id)
	r.Header.Set("Authorization", "Bearer "+t)
	res, e := c.http.Do(r)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return nil, fmt.Errorf("twitch clips status %d", res.StatusCode)
	}
	var out struct {
		Data []Clip `json:"data"`
	}
	e = json.NewDecoder(res.Body).Decode(&out)
	return out.Data, e
}
