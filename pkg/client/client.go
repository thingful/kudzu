package client

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	"github.com/thingful/kuzu/pkg/version"
)

// Client is our custom client type that ensures a timeout is used, and adds a
// user agent header to be polite.
type Client struct {
	client    *http.Client
	userAgent string
	logger    kitlog.Logger
}

// NewClient returns a new client instance initialized with a user agent string
// and timeout
func NewClient(timeout int, logger kitlog.Logger) *Client {
	logger = kitlog.With(logger, "module", "client")

	logger.Log(
		"msg", "configuring http client",
		"timeout", timeout,
	)

	c := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	return &Client{
		client:    c,
		userAgent: fmt.Sprintf("grow(%s)/%s", version.BinaryName, version.Version),
		logger:    logger,
	}
}

// Get attempts to fetch the given URL, setting the correct authorization and
// user agent header
func (c *Client) Get(requestURL, accessToken string) ([]byte, error) {
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
		c.logger.Log("msg", "unexpected response code", "code", resp.StatusCode)
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
