package http

import (
	"fmt"
	"io/ioutil"
	"net/http"
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
func NewClient(timeout time.Duration, logger kitlog.Logger) *Client {
	logger = kitlog.With(logger, "module", "client")

	c := &http.Client{
		Timeout: timeout,
	}

	return &Client{
		client:    c,
		userAgent: fmt.Sprintf("grow(%s)/%s", version.BinaryName, version.Version),
		logger:    logger,
	}
}

// Get attempts to fetch the given URL, setting the correct authorization and
// user agent header
func (c *Client) Get(url, accessToken string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request object")
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Log("msg", "unexpected response code", "code", resp.StatusCode)
		return nil, errors.New("Unexpected status code")
	}

	return ioutil.ReadAll(resp.Body)
}
