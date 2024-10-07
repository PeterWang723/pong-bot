package loader

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/PeterWang723/pong-bot/util"
	"golang.org/x/net/http2"
)

func client(disableCompression, disableKeepAlive, skipVerify bool, timeoutms int, allowRedirects bool, clientCert, clientKey, caCert string, usehttp2 bool) (*http.Client, error) {
	client := &http.Client{}
	client.Transport = &http.Transport{
		DisableCompression: disableCompression,
		DisableKeepAlives: disableKeepAlive,
		ResponseHeaderTimeout: time.Millisecond * time.Duration(timeoutms),
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipVerify},
	}

	if !allowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return util.NewRedirectError("redirection not allowed")
		}
	}

	if clientCert == "" && clientKey == "" && caCert == "" {
		return client, nil
	}

	if clientCert == "" {
		return nil, fmt.Errorf("client certificate cannot be empty")
	}

	if clientKey == "" {
		return nil, fmt.Errorf("client key cannot be empty")
	}

	cert, err := tls.LoadX509KeyPair(clientCert, clientKey)

	if err != nil {
		return nil, fmt.Errorf("unable to load cert tried to load %v and %v but got %v", clientCert, clientKey, err)
	}

	clientCAcert, err := os.ReadFile(caCert)

	if err != nil {
		return nil, fmt.Errorf("unable to open cert %v", err)
	}

	clientCertPool := x509.NewCertPool()
	clientCertPool.AppendCertsFromPEM(clientCAcert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs: clientCertPool,
		InsecureSkipVerify: skipVerify,
	}

	t := &http.Transport{
		DisableCompression: disableCompression,
		DisableKeepAlives: disableKeepAlive,
		ResponseHeaderTimeout: time.Millisecond * time.Duration(timeoutms),
		TLSClientConfig: tlsConfig,
	}

	if usehttp2 {
		http2.ConfigureTransport(t)
	}

	client.Transport = t

	return client, nil
}