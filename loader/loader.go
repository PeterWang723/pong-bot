package loader

import (
	"errors"
	slog "log/slog"
	"net/http"
	"sync/atomic"
	"time"

	histo "github.com/HdrHistogram/hdrhistogram-go"
)

const (
	USER_AGENT = "pong-bot"
)

type LoadConfig struct {
	duration           int // seconds
	goroutines         int
	testUrl            string
	reqBody            string
	method             string
	host               string
	header             map[string]string
	statsAggregator    chan *RequesterStats
	timeoutms          int
	allowRedirects     bool
	disableCompression bool
	disableKeepAlive   bool
	skipVerify         bool
	interrupted        int32
	clientCert         string
	clientKey          string
	caCert             string
	http2              bool
}
// RequesterStats used for collecting aggregate statistics
type RequesterStats struct {
	TotRespSize    int64
	TotDuration    time.Duration
	NumRequests    int
	NumErrs        int
	ErrMap		   map[string]int
	Histogram	   *histo.Histogram
}

func NewLoadConfig(duration int, // seconds
	goroutines int,
	testUrl string,
	reqBody string,
	method string,
	host string,
	header map[string]string,
	statsAggregator chan *RequesterStats,
	timeoutms int,
	allowRedirects bool,
	disableCompression bool,
	disableKeepAlive bool,
	skipVerify bool,
	clientCert string,
	clientKey string,
	caCert string,
	http2 bool) (rt *LoadConfig) {
	rt = &LoadConfig{duration, goroutines, testUrl, reqBody, method, host, header, statsAggregator, timeoutms,
		allowRedirects, disableCompression, disableKeepAlive, skipVerify, 0, clientCert, clientKey, caCert, http2}
	return
}

func (cfg *LoadConfig) Stop() {
	atomic.StoreInt32(&cfg.interrupted, 1)
}

func unwrap(err error) error {
	for errors.Unwrap(err)!=nil {
		err = errors.Unwrap(err);
	}
	return err
}

// Requester a go function for repeatedly making requests and aggregating statistics as long as required
// When it is done, it sends the results using the statsAggregator channel
func (cfg *LoadConfig) RunSingleLoadSession() {
	stats := &RequesterStats{ErrMap: make(map[string]int), Histogram: histo.New(1,int64(cfg.duration * 1000000),4)}
	start := time.Now()

	httpClient, err := client(cfg.disableCompression, cfg.disableKeepAlive, cfg.skipVerify,
		cfg.timeoutms, cfg.allowRedirects, cfg.clientCert, cfg.clientKey, cfg.caCert, cfg.http2)
	
	if err != nil {
		slog.Error(unwrap(err).Error())
	}

	for time.Since(start).Seconds() <= float64(cfg.duration) && atomic.LoadInt32(&cfg.interrupted) == 0 {
		respSize, reqDur, err := DoRequest(httpClient, cfg.header, cfg.method, cfg.host, cfg.testUrl, cfg.reqBody)
		if err != nil {
			stats.ErrMap[unwrap(err).Error()] += 1
			stats.NumErrs ++
		} else if respSize > 0{
			stats.TotRespSize += int64(respSize)
			stats.TotDuration += reqDur
			stats.Histogram.RecordValue(reqDur.Microseconds())
			stats.NumRequests ++
		} else {
			stats.NumErrs ++
		}
	}
	cfg.statsAggregator <- stats
}

// DoRequest single request implementation. Returns the size of the response and its duration
// On error - returns -1 on both
func DoRequest(httpClient *http.Client, header map[string]string, method, host, loadUrl, reqBody string) (respSize int, duration time.Duration, err error) {
	panic("None")
}