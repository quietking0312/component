package mhttp

import (
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	cli     *http.Client
	baseUri string
}

func NewClient(cli *http.Client, baseUri string) *Client {
	return &Client{
		cli:     cli,
		baseUri: strings.Trim(baseUri, "/"),
	}
}

type Request struct {
	Method string
	Path   string
	Args   map[string][]string
	Body   io.Reader
	Header http.Header
}

type RespFunc func(response *http.Response) error

func (cli *Client) Do(request Request, respFunc RespFunc) error {
	uriPath := strings.Join([]string{cli.baseUri, strings.Trim(request.Path, "/")}, "/")
	uri, err := url.Parse(uriPath)
	if err != nil {
		return err
	}
	values := url.Values(request.Args)
	argsData, err := url.QueryUnescape(values.Encode())
	if err != nil {
		return err
	}
	uri.RawQuery = argsData
	req, err := http.NewRequest(request.Method, uri.String(), request.Body)
	if err != nil {
		return err
	}
	if request.Header != nil {
		req.Header = request.Header
	}
	resp, err := cli.cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return respFunc(resp)
}
