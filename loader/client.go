package loader

import "net/http"

func client(disableCompression, disableKeepAlive, skipVerify bool, timeoutms int, allowRedirects bool, clientCert, clientKey, caCert string, usehttp2 bool) (*http.Client, error) {
	panic("None")
}