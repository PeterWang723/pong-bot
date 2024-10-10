package util

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

type HeaderList []string

// Type implements pflag.Value.
func (i *HeaderList) Type() string {
	return "The slice of header"
}

func (i *HeaderList) String() string {
	out := []string{}
	for _, s := range *i {
		out = append(out, s)
	}
	return strings.Join(out, ", ")
}

func (i *HeaderList) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// ByteSize a helper struct that implements the String() method and returns a human readable result. Very useful for %v formatting.
type ByteSize struct {
	Size float64
}

func (size ByteSize) String() string {
	var rt float64
	var suffix string
	const (
		Byte  = 1
		KByte = Byte * 1024
		MByte = KByte * 1024
		GByte = MByte * 1024
	)

	if size.Size > GByte {
		rt = size.Size / GByte
		suffix = "GB"
	} else if size.Size > MByte {
		rt = size.Size / MByte
		suffix = "MB"
	} else if size.Size > KByte {
		rt = size.Size / KByte
		suffix = "KB"
	} else {
		rt = size.Size
		suffix = "bytes"
	}

	srt := fmt.Sprintf("%.2f%v", rt, suffix)

	return srt
}

func ToDuration(usecs int64) time.Duration {
	return time.Duration(usecs*1000)
}

func MapToString(m map[string]int) string {
	s := make([]string, 0, len(m))
	for k,v := range m {
		s = append(s, fmt.Sprint(k,"=",v))
	}

	return strings.Join(s,",")
}

// RedirectError specific error type that happens on redirection
type RedirectError struct {
	msg string
}

func (error *RedirectError) Error() string {
	return error.msg
}

func NewRedirectError(message string) *RedirectError {
	rt := RedirectError{msg: message}
	return &rt
}

//EstimateHttpHeadersSize had to create this because headers size was not counted
func EstimateHttpHeadersSize(headers http.Header) (result int64) {
	result = 0

	for k, v := range headers {
		result += int64(len(k) + len(": \r\n"))
		for _, s := range v {
			result += int64(len(s))
		}
	}

	result += int64(len("\r\n"))

	return result
}