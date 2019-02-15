package client

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/thingful/kuzu/pkg/logger"
	"github.com/thingful/kuzu/pkg/version"
)

// Client is our custom client type that ensures a timeout is used, and adds a
// user agent header to be polite.
type Client struct {
	client    *http.Client
	userAgent string
	verbose   bool
}

// NewClient returns a new client instance initialized with a user agent string
// and timeout
func NewClient(timeout int, verbose bool) *Client {
	c := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	return &Client{
		client:    c,
		userAgent: fmt.Sprintf("grow(%s)/%s", version.BinaryName, version.Version),
		verbose:   verbose,
	}
}

// Get attempts to fetch the given URL, setting the correct authorization and
// user agent header
// TODO: add context to pass in logger
func (c *Client) Get(ctx context.Context, requestURL, accessToken string) ([]byte, error) {
	log := logger.FromContext(ctx)

	if c.verbose {
		log.Log(
			"msg", "getting url",
			"url", requestURL,
		)
	}

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request object")
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		if err.(*url.Error).Timeout() {
			return nil, TimeoutError
		}

		return nil, UnexpectedError
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Log("msg", "unexpected response code", "code", resp.StatusCode)
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return nil, UnauthorizedError
		case http.StatusNotFound:
			return nil, NotFoundError
		default:
			return nil, fmt.Errorf("Unexpected response: %s", resp.Status)
		}
	}

	return ioutil.ReadAll(resp.Body)
}
