package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const BaseURL = "https://api.planningcenteronline.com"

// HTTPDoer abstracts http.Client for testing.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is an authenticated HTTP client for the PCO API.
type Client struct {
	BaseURL    string
	ClientID   string
	Secret     string
	HTTPClient HTTPDoer
}

// New creates a Client with the given credentials.
func New(clientID, secret string) *Client {
	return &Client{
		BaseURL:    BaseURL,
		ClientID:   clientID,
		Secret:     secret,
		HTTPClient: &http.Client{},
	}
}

func (c *Client) authHeader() string {
	creds := base64.StdEncoding.EncodeToString([]byte(c.ClientID + ":" + c.Secret))
	return "Basic " + creds
}

func (c *Client) do(ctx context.Context, method, path string, body string) ([]byte, error) {
	var fullURL string
	if strings.HasPrefix(path, "http") {
		fullURL = path
	} else {
		fullURL = c.BaseURL + path
	}

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(data),
			Method:     method,
			Path:       path,
		}
	}

	return data, nil
}

// Get performs a GET request. Params are appended as query string.
func (c *Client) Get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	return c.do(ctx, http.MethodGet, path, "")
}

// Post performs a POST request with a JSON body string.
func (c *Client) Post(ctx context.Context, path string, body string) ([]byte, error) {
	return c.do(ctx, http.MethodPost, path, body)
}

// Patch performs a PATCH request with a JSON body string.
func (c *Client) Patch(ctx context.Context, path string, body string) ([]byte, error) {
	return c.do(ctx, http.MethodPatch, path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	_, err := c.do(ctx, http.MethodDelete, path, "")
	return err
}

// APIError represents a non-2xx response from the PCO API.
type APIError struct {
	StatusCode int
	Body       string
	Method     string
	Path       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("PCO API error: %s %s returned %d: %s", e.Method, e.Path, e.StatusCode, e.Body)
}
