package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	histo "github.com/HdrHistogram/hdrhistogram-go"
	"github.com/PeterWang723/pong-bot/loader"
	"github.com/PeterWang723/pong-bot/util"
	"github.com/spf13/cobra"
)

const APP_VERSION = "0.10"

var (
//default that can be overridden from the command line
	versionFlag bool = false
	duration int = 10 //seconds
	goroutines int = 2
	testUrl string
	method string = "GET"
	host string
	headerFlags util.HeaderList
	header map[string]string
	statsAggregator chan *loader.RequesterStats
	sigChan chan os.Signal
	timeoutms int
	allowRedirectsFlag bool = false
	disableCompression bool
	disableKeepAlive bool
	skipVerify bool
	playbackFile string
	reqBody string
	clientCert string
	clientKey string
	caCert string
	http2 bool
	cpus int = 0

	rootCmd = &cobra.Command{
		Use:   "pbot <url>",
        Short: "pong-bot is a CLI tool for HTTP benchmarking",
	}
)

func init() {
	statsAggregator = make(chan *loader.RequesterStats, goroutines)
	sigChan = make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	// Binding flags to variables
	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Print version details")
	rootCmd.Flags().IntVarP(&duration, "duration", "d", 10, "Duration of test in seconds (Default 10)")
	rootCmd.Flags().IntVarP(&goroutines, "concurrency", "c", 2, "Number of goroutines to use (Default 2)")
	rootCmd.Flags().StringVarP(&method, "method", "M", "GET", "HTTP method to use (Default GET)")
	rootCmd.Flags().StringVarP(&host, "host", "", "", "Host header")
	rootCmd.Flags().VarP(&headerFlags, "header", "H", "Header to add to each request (you can define multiple -H flags)")
	rootCmd.Flags().IntVar(&timeoutms, "timeout", 1000, "Socket/request timeout in ms (Default 1000)")
	rootCmd.Flags().BoolVar(&allowRedirectsFlag, "redir", false, "Allow redirects (Default false)")
	rootCmd.Flags().BoolVar(&disableCompression, "no-c", false, "Disable compression (Default false)")
	rootCmd.Flags().BoolVar(&disableKeepAlive, "no-ka", false, "Disable keep-alive (Default false)")
	rootCmd.Flags().BoolVar(&skipVerify, "no-vr", false, "Skip SSL certificate verification (Default false)")
	rootCmd.Flags().StringVar(&playbackFile, "playback", "", "Playback file name")
	rootCmd.Flags().StringVar(&reqBody, "body", "", "Request body string or @filename")
	rootCmd.Flags().StringVar(&clientCert, "cert", "", "Client certificate file (for SSL/TLS)")
	rootCmd.Flags().StringVar(&clientKey, "key", "", "Client key file (for SSL/TLS)")
	rootCmd.Flags().StringVar(&caCert, "ca", "", "CA certificate file to verify peer against (SSL/TLS)")
	rootCmd.Flags().BoolVar(&http2, "http2", true, "Use HTTP/2 (Default true)")
	rootCmd.Flags().IntVar(&cpus, "cpus", 0, "Number of CPUs to use (0 to use all available CPUs)")
	header = make(map[string]string)

	rootCmd.Run = run
}

// Execute executes the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
 
func run(cmd *cobra.Command, args []string) {
	for _, hdr := range headerFlags {
		hp := strings.SplitN(hdr, ":", 2)
		header[hp[0]] = hp[1]
	}

	if versionFlag {
		fmt.Println("Version:", APP_VERSION)
		return
	}

	if playbackFile != "" {
		data, err := os.ReadFile(playbackFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		testUrl = string(data)
	} else {
		if args != nil {
			testUrl = args[0]
		} else {
			fmt.Println("No url added")
			return
		}
	}

	if cpus > 0 {
		runtime.GOMAXPROCS(cpus)
	}

	fmt.Printf("Running %vs test @ %v\n  %v goroutine(s) running concurrently\n", duration, testUrl, goroutines)

	if len(reqBody) > 0 && reqBody[0] == '@'{
		bodyFilename := reqBody[1:]
		data, err := os.ReadFile(bodyFilename)
		if err != nil {
			fmt.Println(fmt.Errorf("could not read file %q: %v", bodyFilename, err))
			os.Exit(1)
		}
		reqBody = string(data)
	}

	loadGen := loader.NewLoadConfig(duration, goroutines, testUrl, reqBody, method, host, header, statsAggregator, timeoutms,
		allowRedirectsFlag, disableCompression, disableKeepAlive, skipVerify, clientCert, clientKey, caCert, http2)
	
	start := time.Now()
	
	for i := 0; i < goroutines; i++ {
		go loadGen.RunSingleLoadSession()
	}

	responders := 0
	aggStats := loader.RequesterStats{ErrMap: make(map[string]int), Histogram: histo.New(1,int64(duration * 1000000),4)}

	for responders < goroutines {
		select{
			case <-sigChan:
				loadGen.Stop()
				fmt.Printf("stopping...\n")
			case stats := <-statsAggregator:
				aggStats.NumErrs += stats.NumErrs
				aggStats.NumRequests += stats.NumRequests
				aggStats.TotRespSize += stats.TotRespSize
				aggStats.TotDuration += stats.TotDuration
				responders++
				for k,v := range stats.ErrMap {
					aggStats.ErrMap[k] += v
				}
				aggStats.Histogram.Merge(stats.Histogram)
		}
	} 

	duration := time.Since(start)

	if aggStats.NumRequests == 0 {
		fmt.Println("Error: No statistics collected / no requests found")
		fmt.Printf("Number of Errors:\t%v\n", aggStats.NumErrs)
		if aggStats.NumErrs > 0 {
			fmt.Printf("Error Counts:\t\t%v\n", util.MapToString(aggStats.ErrMap))
		}
		return
	}

	avgThreadDur := aggStats.TotDuration / time.Duration(responders) //need to average the aggregated duration

	reqRate := float64(aggStats.NumRequests) / avgThreadDur.Seconds()
	bytesRate := float64(aggStats.TotRespSize) / avgThreadDur.Seconds()

	overallReqRate := float64(aggStats.NumRequests) / duration.Seconds()
	overallBytesRate := float64(aggStats.TotRespSize) / duration.Seconds()

	fmt.Printf("%v requests in %v, %v read\n", aggStats.NumRequests, avgThreadDur, util.ByteSize{Size: float64(aggStats.TotRespSize)})
	fmt.Printf("Requests/sec:\t\t%.2f\nTransfer/sec:\t\t%v\n", reqRate, util.ByteSize{Size: bytesRate})
	fmt.Printf("Overall Requests/sec:\t%.2f\nOverall Transfer/sec:\t%v\n", overallReqRate, util.ByteSize{Size: overallBytesRate})
	fmt.Printf("Fastest Request:\t%v\n", util.ToDuration(aggStats.Histogram.Min()))
	fmt.Printf("Avg Req Time:\t\t%v\n", util.ToDuration(int64(aggStats.Histogram.Mean())))
	fmt.Printf("Slowest Request:\t%v\n", util.ToDuration(aggStats.Histogram.Max()))
	fmt.Printf("Number of Errors:\t%v\n", aggStats.NumErrs)
	if aggStats.NumErrs > 0 {
		fmt.Printf("Error Counts:\t\t%v\n", util.MapToString(aggStats.ErrMap))
	}
	fmt.Printf("10%%:\t\t\t%v\n", util.ToDuration(aggStats.Histogram.ValueAtPercentile(.10)))
	fmt.Printf("50%%:\t\t\t%v\n", util.ToDuration(aggStats.Histogram.ValueAtPercentile(.50)))
	fmt.Printf("75%%:\t\t\t%v\n", util.ToDuration(aggStats.Histogram.ValueAtPercentile(.75)))
	fmt.Printf("99%%:\t\t\t%v\n", util.ToDuration(aggStats.Histogram.ValueAtPercentile(.99)))
	fmt.Printf("99.9%%:\t\t\t%v\n", util.ToDuration(aggStats.Histogram.ValueAtPercentile(.999)))
	fmt.Printf("99.9999%%:\t\t%v\n", util.ToDuration(aggStats.Histogram.ValueAtPercentile(.999999)))
	fmt.Printf("99.99999%%:\t\t%v\n", util.ToDuration(aggStats.Histogram.ValueAtPercentile(.9999999)))
	fmt.Printf("stddev:\t\t\t%v\n", util.ToDuration(int64(aggStats.Histogram.StdDev())))
	// aggStats.Histogram.PercentilesPrint(os.Stdout,1,1)
}




