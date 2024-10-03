package util

import (
	"fmt"
	"strings"
	"time"
)

type HeaderList []string

// Type implements pflag.Value.
func (i *HeaderList) Type() string {
	panic("unimplemented")
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

func (self ByteSize) String() string {
	var rt float64
	var suffix string
	const (
		Byte  = 1
		KByte = Byte * 1024
		MByte = KByte * 1024
		GByte = MByte * 1024
	)

	if self.Size > GByte {
		rt = self.Size / GByte
		suffix = "GB"
	} else if self.Size > MByte {
		rt = self.Size / MByte
		suffix = "MB"
	} else if self.Size > KByte {
		rt = self.Size / KByte
		suffix = "KB"
	} else {
		rt = self.Size
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