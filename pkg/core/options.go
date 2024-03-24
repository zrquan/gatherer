package core

import (
	"errors"
	"flag"

	"github.com/zrquan/gatherer/pkg/input"
	"github.com/zrquan/gatherer/pkg/output"
	"github.com/zrquan/gatherer/pkg/util"
)

type headerFlag []string

func (h *headerFlag) Set(value string) error {
	*h = append(*h, value)
	return nil
}

func (h *headerFlag) String() string {
	return "HTTP Headers"
}

type Options struct {
	Target          string
	Depth           int
	Timeout         int
	Headers         headerFlag
	WordlistPath    string
	Parallel        int
	Debug           bool
	RandomUA        bool
	Proxy           string
	FollowExternal  bool
	FollowSubdomain bool
	NoRedirect      bool
	UseChrome       bool
	JSONFormat      bool

	wordlist *input.Wordlist
}

func ParseOptions() (*Options, error) {
	opts := &Options{}

	flag.StringVar(&opts.Target, "u", "", "Target URL")
	flag.IntVar(&opts.Depth, "dep", 1, "Maximum path depth")
	flag.IntVar(&opts.Timeout, "t", 10, "Request timeout (second)")
	flag.Var(&opts.Headers, "H", "HTTP request headers")
	flag.StringVar(&opts.WordlistPath, "w", "", "Wordlist file path")
	flag.IntVar(&opts.Parallel, "limit", 100, "")
	flag.BoolVar(&opts.Debug, "debug", false, "Debug mode")
	flag.BoolVar(&opts.RandomUA, "ua", false, "Use random User-Agent")
	flag.StringVar(&opts.Proxy, "proxy", "", "Proxy URL")
	flag.BoolVar(&opts.FollowExternal, "fe", false, "Allow to visit third-party domains")
	flag.BoolVar(&opts.FollowSubdomain, "fs", true, "Allow to visit sub-domains")
	flag.BoolVar(&opts.NoRedirect, "nr", false, "Disallow auto redirect")
	flag.BoolVar(&opts.UseChrome, "ch", false, "Run Javascript in headless Chrome")
	flag.BoolVar(&opts.JSONFormat, "json", false, "Log as JSON format")

	flag.Parse()

	if err := validateOptions(opts); err != nil {
		return nil, err
	} else {
		return opts, nil
	}
}

// validateOptions 检查命令选项是否正确
func validateOptions(opts *Options) error {
	if opts.Target == "" {
		return errors.New("target URL is required")
	}
	if !util.IsAbsoluteURL(opts.Target) {
		return errors.New("invalid target URL")
	}
	if opts.Proxy != "" && !util.IsAbsoluteURL(opts.Proxy) {
		return errors.New("invalid proxy URL")
	}
	if opts.WordlistPath != "" {
		wl, err := input.NewWordlist(opts.WordlistPath)
		if err != nil {
			return err
		}
		opts.wordlist = wl
	}
	output.SetFormatter(opts.JSONFormat)
	return nil
}
