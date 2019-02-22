package client

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/thingful/kuzu/pkg/logger"
	"github.com/thingful/kuzu/pkg/version"
)

var (
	durationHist = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "grow",
			Name:      "client_request_duration_seconds",
			Help:      "A histogram of the latency in seconds for requests made by the client",
		}, []string{"code", "method", "host"},
	)
)

func init() {
	prometheus.MustRegister(durationHist)
}

// Client is our custom client type that ensures a timeout is used, and adds a
// user agent header to be polite.
type Client struct {
	*http.Client
	userAgent string
	verbose   bool
}

// NewClient returns a new client instance initialized with a user agent string
// and timeout
func NewClient(timeout int, verbose bool) *Client {
	roundTripper := InstrumentRoundTripperDuration(durationHist, http.DefaultTransport)

	c := &http.Client{
		Timeout:   time.Duration(timeout) * time.Second,
		Transport: roundTripper,
	}

	return &Client{
		Client:    c,
		userAgent: fmt.Sprintf("grow(%s)/%s", version.BinaryName, version.Version),
		verbose:   verbose,
	}
}

// Get attempts to fetch the given URL, setting the correct authorization and
// user agent header
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

	resp, err := c.Do(req)
	if err != nil {
		log.Log(
			"msg", "error making request",
			"error", err,
		)
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

// Post attempts to send a JSON POST request to the given URL, setting the
// correct authorization header and content type. Returns an error on any
// failure, or a slice of bytes.
func (c *Client) Post(ctx context.Context, requestURL, accessToken string, body io.Reader) ([]byte, error) {
	return c.postOrPatch(ctx, http.MethodPost, requestURL, accessToken, body)
}

// Patch attempts to send a JSON PATCH request to the given URL, setting the
// correct authorization header and content type. Returns an error on any
// failure, or a slice of bytes.
func (c *Client) Patch(ctx context.Context, requestURL, accessToken string, body io.Reader) ([]byte, error) {
	return c.postOrPatch(ctx, http.MethodPatch, requestURL, accessToken, body)
}

func (c *Client) postOrPatch(ctx context.Context, method, requestURL, accessToken string, body io.Reader) ([]byte, error) {
	log := logger.FromContext(ctx)

	if c.verbose {
		log.Log(
			"msg", "posting to url",
			"url", requestURL,
		)
	}

	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		log.Log(
			"msg", "error making request",
			"error", err,
		)
		if err.(*url.Error).Timeout() {
			return nil, TimeoutError
		}

		return nil, UnexpectedError
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Log(
			"msg", "unexpected response code",
			"code", resp.StatusCode,
		)
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
