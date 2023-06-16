package mhttp

import (
	"crypto/tls"
	"crypto/x509"
	"golang.org/x/net/http2"
	"net/http"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := &http.Client{}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(nil)
	c.Transport = &http2.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            caCertPool,
			InsecureSkipVerify: true,
		},
	}

}
