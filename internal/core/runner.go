package core

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
	"github.com/gocolly/colly/v2/extensions"
	log "github.com/sirupsen/logrus"
	"github.com/zrquan/gatherer/pkg/finder"
	"github.com/zrquan/gatherer/pkg/util"
)

type Runner struct {
	mutex        sync.Mutex
	options      *Options
	collector    *colly.Collector
	errorCounter int64

	urlSet  mapset.Set[string]
	lenSet  mapset.Set[int]
	browser *rod.Browser
}

func NewRunner(opts *Options) (*Runner, error) {
	collector, err := initCollector(opts)
	if err != nil {
		return nil, err
	}

	l := launcher.New().
		Headless(true).
		Set("ignore-certificate-errors", "1").
		Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3830.0 Safari/537.36").
		Proxy(opts.Proxy).
		MustLaunch()

	runner := &Runner{
		options:      opts,
		collector:    collector,
		errorCounter: 0,
		urlSet:       mapset.NewSet[string](opts.Target),
		lenSet:       mapset.NewSet[int](0),
		browser:      rod.New().ControlURL(l).MustConnect(),
	}
	runner.prepareHooks()
	return runner, nil
}

func (runner *Runner) Execute() {
	if tt := runner.options.TotalTimeout; tt <= 0 {
		runner.startCollect()
	} else {
		finished := make(chan int, 1)
		go func() {
			runner.startCollect()
			finished <- 1
		}()

		select {
		case <-finished:
			close(finished)
			log.Info("All done.")
		case <-time.After(time.Duration(tt) * time.Second):
			log.Error("Gatherer timeout.")
		}
	}
}

func (runner *Runner) startCollect() {
	opts := runner.options
	runner.collector.Visit(opts.Target)
	if opts.WordlistPath != "" {
		for opts.wordlist.Next() {
			path := string(opts.wordlist.Value())
			link, err := url.JoinPath(opts.Target, path)
			if err != nil {
				log.Warn("invalid path from wordlist:", path)
				continue
			}
			runner.collector.Visit(link)
		}
	}
	runner.collector.Wait()
	runner.browser.MustClose()
	log.
		WithFields(log.Fields{"visited": runner.urlSet.Cardinality(), "error": runner.errorCounter}).
		Info("Gathering finished.")
}

// 根据命令选项初始化 colly.Collector
func initCollector(opts *Options) (*colly.Collector, error) {
	// init Transport
	tp := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(opts.Timeout) * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	if opts.Proxy != "" {
		proxyURL, _ := url.Parse(opts.Proxy)
		tp.Proxy = http.ProxyURL(proxyURL)
	}

	hostname, err := util.ExtractHostname(opts.Target)
	if err != nil {
		return nil, err
	}
	c := colly.NewCollector(
		// TODO: 在不同 collector 传递链接时继承请求深度
		colly.MaxDepth(opts.Depth),
		colly.Async(true),
		colly.AllowedDomains(hostname),
	)

	if opts.Debug {
		c.SetDebugger(&debug.LogDebugger{
			// 打印时间
			// Flag: log.LstdFlags,
		})
	}

	if opts.RandomUA {
		extensions.RandomUserAgent(c)
	}

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: opts.Parallel,
	})

	// filter static files
	excludeExtensions := `(?i)\.(png|apng|bmp|gif|ico|cur|jpg|jpeg|jfif|pjp|pjpeg|svg|tif|tiff|webp|xbm|3gp|aac|flac|mpg|mpeg|mp3|mp4|m4a|m4v|m4p|oga|ogg|ogv|mov|wav|webm|eot|woff|woff2|ttf|otf)(?:\?|#|$)`
	c.DisallowedURLFilters = append(c.DisallowedURLFilters, regexp.MustCompile(excludeExtensions))

	if opts.VisitSubdomains {
		filter, err := util.FilterSubdomains(hostname)
		if err != nil {
			return nil, err
		}
		c.AllowedDomains = nil
		c.URLFilters = []*regexp.Regexp{filter}
	}

	c.WithTransport(tp)

	return c, nil
}

// prepareHooks 方法用于设置回调函数
func (runner *Runner) prepareHooks() {
	opts := runner.options
	c := runner.collector

	// 禁止自动重定向
	if opts.NoRedirect {
		c.SetRedirectHandler(func(req *http.Request, via []*http.Request) error {
			var locations string
			for _, v := range via {
				locations += fmt.Sprintf(" <- %s", v.URL.String())
			}
			log.Warn("Skip redirection: " + req.URL.String() + locations)
			return http.ErrUseLastResponse
		})
	} else {
		c.SetRedirectHandler(func(req *http.Request, via []*http.Request) error {
			runner.mutex.Lock()
			defer runner.mutex.Unlock()
			// 避免多次重定向到同一位置
			if runner.urlSet.Contains(req.URL.String()) {
				return http.ErrUseLastResponse
			}
			runner.urlSet.Add(req.URL.String())
			return nil
		})
	}

	// 设置请求头
	for _, h := range opts.Headers {
		headerArgs := strings.SplitN(h, ":", 2)
		headerKey := strings.TrimSpace(headerArgs[0])
		headerValue := strings.TrimSpace(headerArgs[1])
		c.OnRequest(func(r *colly.Request) {
			r.Headers.Set(headerKey, headerValue)
		})
	}

	c.OnError(func(r *colly.Response, err error) {
		status := r.StatusCode
		link := r.Request.URL.String()

		if status >= 300 && status < 400 {
			location := r.Headers.Get("Location")
			if location == link+"/" {
				runner.visitLink(location, r.Request)
				return
			}
		}

		for _, f := range opts.filters {
			result, err := f.Filter(r)
			if err != nil {
				continue
			}
			if result {
				return
			}
		}

		log.WithFields(log.Fields{
			"code":   status,
			"length": len(r.Body),
		}).Warn(r.Request.URL.String())

		atomic.AddInt64(&runner.errorCounter, 1)
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("href"))
		if opts.IgnoreQuery {
			u, err := url.Parse(link)
			if err != nil {
				log.WithField("link", link).Error("Parse URL error")
			}
			link = util.StripQueryParams(u)
		}
		runner.visitLink(link, e.Request)
	})

	c.OnHTML("script[src]", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("src"))
		runner.visitLink(link, e.Request)
	})

	c.OnHTML("form[action]", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("action"))
		runner.visitLink(link, e.Request)
	})

	c.OnHTML("title", func(e *colly.HTMLElement) {
		title := util.FilterNewLines(e.Text)
		if title != "" {
			e.Request.Ctx.Put("title", title)
		}

		if title == "Swagger UI" && strings.HasSuffix(e.Request.URL.Path, "swagger-ui.html") {
			runner.visitLink(e.Request.AbsoluteURL("swagger-resources"), e.Request)
		}
	})

	// sitemap
	c.OnXML("//urlset/url/loc", func(e *colly.XMLElement) {
		link := e.Text
		runner.visitLink(link, e.Request)
	})

	c.OnResponse(func(r *colly.Response) {
		if runner.filterResp(r) {
			return
		}

		if opts.UseChrome {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(opts.Timeout)*time.Second)
			defer cancel()

			page := runner.browser.Context(ctx).MustPage(r.Request.URL.String())
			defer page.MustClose()

			err := page.WaitLoad()
			if err != nil {
				log.Warn("browser error: ", err)
			} else {
				content, err := page.HTML()
				if err != nil {
					log.Warn("browser error: ", err)
				} else {
					r.Body = []byte(content)
				}
			}
		}

		// TODO: support openapi 3.x
		if util.IsSwaggerSchema(r) {
			endpoints, err := finder.FindLinksFromSwagger(r.Body)
			if err != nil {
				log.Errorf("Parse swagger error: %s", err)
			} else {
				log.Printf("Found %d APIs from Swagger document: %s", len(endpoints), r.Request.URL.String())
				for _, api := range endpoints {
					url := util.FixURL(r.Request.URL, api.URL)
					method := strings.ToUpper(api.Method)
					if method == "DELETE" {
						continue
					}

					dataReader := bytes.NewReader([]byte(api.Content))
					headers := r.Request.Headers.Clone()
					for k, v := range api.Headers {
						headers.Set(k, v)
					}

					c.Request(method, url, dataReader, r.Ctx, headers)
				}
			}
		}

		if util.IsScriptOrJSON(r.Request.URL.String()) {
			endpoints := mapset.NewSet[string]()

			content := string(r.Body)
			if strings.Contains(content, `document.createElement("script");`) {
				dynamicLinks := finder.FindDynamicLinksFromJS(content, runner.browser)
				for _, dl := range dynamicLinks {
					log.Debugf("Found dynamic script \"%s\" from JS file: %s", dl, r.Request.URL.String())
					endpoints.Add(dl)
				}
			}

			endpoints.Append(finder.FindLinksFromJS(content)...)
			log.Debugf("Found %d links from JS file: %s", endpoints.Cardinality(), r.Request.URL.String())

			for ep := range endpoints.Iterator().C {
				var link string
				if strings.HasPrefix(ep, "./") {
					link = util.FixURL(r.Request.URL, ep)
				} else {
					u, _ := url.Parse(opts.targetRoot)
					link = util.FixURL(u, ep)
				}
				runner.visitLink(link, r.Request)
			}
		}

		if r.StatusCode == 200 && r.Request.URL.Path == "/robots.txt" {
			endpoints := finder.FindLinksFromRobots(string(r.Body))
			for _, e := range endpoints {
				link := r.Request.AbsoluteURL(e)
				runner.visitLink(link, r.Request)
			}
		}
	})

	c.OnScraped(func(r *colly.Response) {
		runner.mutex.Lock()
		defer runner.mutex.Unlock()

		url := r.Request.URL.String()
		runner.urlSet.Add(url)

		for _, f := range opts.filters {
			result, err := f.Filter(r)
			if err != nil {
				continue
			}
			if result {
				return
			}
		}

		var fields log.Fields
		if t := r.Ctx.Get("title"); t != "" {
			fields = log.Fields{"code": r.StatusCode, "length": len(r.Body), "title": t}
			// reset page title
			r.Ctx.Put("title", "")
		} else {
			fields = log.Fields{"code": r.StatusCode, "length": len(r.Body)}
		}

		log.WithFields(fields).Info(url)
	})
}

func (runner *Runner) visitLink(link string, request *colly.Request) {
	if !runner.urlSet.Contains(link) {
		request.Visit(link)
	}
}

func (runner *Runner) filterResp(resp *colly.Response) bool {
	if resp.StatusCode == 404 || resp.StatusCode == 429 || resp.StatusCode < 100 {
		return true
	}

	// 长度一样的话视为相同的响应报文，只处理一次（应该用哈希值？）
	if runner.lenSet.Contains(len(resp.Body)) {
		return true
	} else {
		runner.lenSet.Add(len(resp.Body))
		return false
	}
}
