package core

import (
	"errors"
	"flag"
	"fmt"
	"net/url"

	"github.com/zrquan/gatherer/pkg/filter"
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
	TotalTimeout    int
	Headers         headerFlag
	WordlistPath    string
	Parallel        int
	Debug           bool
	RandomUA        bool
	Proxy           string
	VisitSubdomains bool
	NoRedirect      bool
	UseChrome       bool
	IgnoreQuery     bool
	JSONFormat      bool
	StatusFilter    string
	ExtensionFilter string
	LengthFilter    string

	wordlist   *input.Wordlist
	targetRoot string
	filters    []filter.IFilter
}

func ParseOptions() (*Options, error) {
	opts := &Options{}

	flag.StringVar(&opts.Target, "u", "", "Target URL")
	flag.IntVar(&opts.Depth, "dep", 1, "Maximum path depth")
	flag.IntVar(&opts.Timeout, "t", 10, "Request timeout (second)")
	flag.IntVar(&opts.TotalTimeout, "tt", 0, "Total timeout (second)")
	flag.Var(&opts.Headers, "H", "HTTP request headers (eg. -H 'Header1:value' -H 'Header2:value')")
	flag.StringVar(&opts.WordlistPath, "w", "", "Wordlist file path")
	flag.IntVar(&opts.Parallel, "limit", 100, "Maximum number of concurrent requests")
	flag.BoolVar(&opts.Debug, "debug", false, "Debug mode")
	flag.BoolVar(&opts.RandomUA, "ua", false, "Use random User-Agent")
	flag.StringVar(&opts.Proxy, "proxy", "", "Proxy URL")
	flag.BoolVar(&opts.VisitSubdomains, "sub", false, "Allow to visit sub-domains")
	flag.BoolVar(&opts.NoRedirect, "nr", false, "Disallow auto redirect")
	flag.BoolVar(&opts.UseChrome, "ch", false, "Run Javascript in headless Chrome")
	flag.BoolVar(&opts.IgnoreQuery, "igq", false, "Ignore the query portion on the URL from a[href]")
	flag.BoolVar(&opts.JSONFormat, "json", false, "Log as JSON format")
	flag.StringVar(&opts.StatusFilter, "sf", "", "Filter by status codes (separated by commas)")
	flag.StringVar(&opts.ExtensionFilter, "ef", "", "Filter by extensions (separated by commas)")
	flag.StringVar(&opts.LengthFilter, "lf", "", "Filter by response length (separated by commas)")

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
	u, _ := url.Parse(opts.Target)
	opts.targetRoot = fmt.Sprintf("%s://%s/", u.Scheme, u.Host)

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

	if sf := opts.StatusFilter; sf != "" {
		f, err := filter.NewFilterByName("status", sf)
		if err != nil {
			return err
		}
		opts.filters = append(opts.filters, f)
	}
	if ef := opts.ExtensionFilter; ef != "" {
		f, err := filter.NewFilterByName("extension", ef)
		if err != nil {
			return err
		}
		opts.filters = append(opts.filters, f)
	}
	if lf := opts.LengthFilter; lf != "" {
		f, err := filter.NewFilterByName("length", lf)
		if err != nil {
			return err
		}
		opts.filters = append(opts.filters, f)
	}

	return nil
}
