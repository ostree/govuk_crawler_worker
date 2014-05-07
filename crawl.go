package govuk_crawler_worker

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

var (
	CannotCrawlNonLocalHosts error = errors.New("Cannot crawl URLs that don't match the provided host")
	RetryRequestError        error = errors.New("Retry request: 429 or 5XX HTTP Response returned")

	statusCodes []int
	once        sync.Once
)

type Crawler struct {
	RootURL *url.URL
}

func RetryStatusCodes() []int {
	// This is go's equivalent of memoization/macro expansion. It's
	// being used here because we have a fixed array we're generating
	// with known values.
	once.Do(func() {
		statusCodes = []int{429}

		for i := 500; i <= 599; i++ {
			statusCodes = append(statusCodes, i)
		}
	})

	return statusCodes
}

func NewCrawler(rootURL string) (*Crawler, error) {
	if rootURL == "" {
		return nil, errors.New("Cannot provide an empty root URL")
	}

	u, err := url.Parse(rootURL)
	if err != nil {
		return nil, err
	}

	return &Crawler{
		RootURL: u,
	}, nil
}

func (c *Crawler) Crawl(crawlURL string) ([]byte, error) {
	u, err := url.Parse(crawlURL)
	if err != nil {
		return []byte{}, err
	}

	if !strings.HasPrefix(u.Host, c.RootURL.Host) {
		return []byte{}, CannotCrawlNonLocalHosts
	}

	resp, err := http.Get(crawlURL)
	if err != nil {
		return []byte{}, err
	}

	if contains(RetryStatusCodes(), resp.StatusCode) {
		return []byte{}, RetryRequestError
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return body, nil
}

func contains(haystack []int, needle int) bool {
	for _, hay := range haystack {
		if hay == needle {
			return true
		}
	}

	return false
}
