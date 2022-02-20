package cmd

import (
	"net/http"
	"strings"

	"github.com/m-mizutani/ghaudit/pkg/domain/types"
	"github.com/m-mizutani/goerr"
)

type httpHeader struct {
	key   string
	value string
}

type httpClient struct {
	headers []*httpHeader
}

func newHTTPClient(headers []string) (*httpClient, error) {
	client := &httpClient{}

	for _, hdr := range headers {
		parts := strings.Split(hdr, ":")
		if len(parts) < 2 {
			return nil, goerr.Wrap(types.ErrInvalidConfig, "invalid HTTP header format").With("header", hdr)
		}

		client.headers = append(client.headers, &httpHeader{
			key:   strings.TrimSpace(parts[0]),
			value: strings.TrimSpace(strings.Join(parts[1:], ":")),
		})
	}

	return client, nil
}

func (x *httpClient) Do(req *http.Request) (*http.Response, error) {
	for _, hdr := range x.headers {
		req.Header.Add(hdr.key, hdr.value)
	}

	return http.DefaultClient.Do(req)
}
