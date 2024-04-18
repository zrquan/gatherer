package core

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
	"github.com/gocolly/colly/v2/extensions"
	log "github.com/sirupsen/logrus"
	"github.com/zrquan/gatherer/pkg/finder"
	"github.com/zrquan/gatherer/pkg/util"
)

type Runner struct {
	options       *Options
	coreCollector *colly.Collector
	jsCollector   *colly.Collector
	errorCounter  int64

	urlSet mapset.Set[string]
	lenSet mapset.Set[int]
}

func NewRunner(opts *Options) (*Runner, error) {
	collector, err := initCollector(opts)
	if err != nil {
		return nil, err
	}

	runner := &Runner{
		options:       opts,
		coreCollector: collector,
		jsCollector:   collector.Clone(),
		errorCounter:  0,
		urlSet:        mapset.NewSet[string](opts.Target),
		lenSet:        mapset.NewSet[int](0),
	}
	runner.prepare()
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
	runner.coreCollector.Visit(opts.Target)
	if opts.WordlistPath != "" {
		for opts.wordlist.Next() {
			path := string(opts.wordlist.Value())
			link, err := url.JoinPath(opts.Target, path)
			if err != nil {
				log.Warn("invalid path from wordlist:", path)
				continue
			}
			switch runner.dispatch(link) {
			case 0:
				runner.coreCollector.Visit(link)
			case 1:
				// Javascript 链接由 jsCollector 处理
				runner.jsCollector.Visit(link)
			case -1:
				continue
			}
		}
	}
	runner.Wait()
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

	// 禁止自动重定向
	if opts.NoRedirect {
		c.SetRedirectHandler(func(req *http.Request, via []*http.Request) error {
			var locations string
			for _, v := range via {
				locations += fmt.Sprintf(" <- %s", v.URL.String())
			}
			log.Warn("skip redirection: " + req.URL.String() + locations)
			return http.ErrUseLastResponse
		})
	}

	c.WithTransport(tp)

	return c, nil
}

// prepare 方法用于设置回调函数
func (runner *Runner) prepare() {
	opts := runner.options
	cc := runner.coreCollector
	jc := runner.jsCollector

	// 设置请求头
	for _, h := range opts.Headers {
		headerArgs := strings.SplitN(h, ":", 2)
		headerKey := strings.TrimSpace(headerArgs[0])
		headerValue := strings.TrimSpace(headerArgs[1])
		cc.OnRequest(func(r *colly.Request) {
			r.Headers.Set(headerKey, headerValue)
		})
		jc.OnRequest(func(r *colly.Request) {
			r.Headers.Set(headerKey, headerValue)
		})
	}

	// 统计错误次数
	reqErr := func(r *colly.Response, err error) {
		atomic.AddInt64(&runner.errorCounter, 1)
	}
	cc.OnError(reqErr)
	jc.OnError(reqErr)

	cc.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("href"))
		if opts.IgnoreQuery {
			u, err := url.Parse(link)
			if err != nil {
				log.WithField("link", link).Error("Parse URL error")
			}
			link = util.StripQueryParams(u)
		}
		switch runner.dispatch(link) {
		case 0:
			e.Request.Visit(link)
		case 1:
			jc.Visit(link)
		}
	})

	cc.OnHTML("script[src]", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("src"))
		if !runner.urlSet.Contains(link) {
			// Javascript 链接由 jsCollector 处理
			jc.Visit(link)
		}
	})

	cc.OnHTML("form[action]", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("action"))
		if !runner.urlSet.Contains(link) {
			// TODO: 处理表单提交和文件上传
			e.Request.Visit(link)
		}
	})

	cc.OnHTML("title", func(e *colly.HTMLElement) {
		title := util.FilterNewLines(e.Text)
		if title != "" {
			e.Request.Ctx.Put("title", title)
		}

		if title == "Swagger UI" && strings.HasSuffix(e.Request.URL.Path, "swagger-ui.html") {
			jc.Visit(e.Request.AbsoluteURL("swagger-resources"))
		}
	})

	// sitemap
	cc.OnXML("//urlset/url/loc", func(e *colly.XMLElement) {
		link := e.Text
		switch runner.dispatch(link) {
		case 0:
			e.Request.Visit(link)
		case 1:
			jc.Visit(link)
		}
	})

	cc.OnResponse(func(r *colly.Response) {
		if runner.filterResp(r) {
			return
		}

		// TODO: support openapi 3.x
		ct := r.Headers.Get("Content-Type")
		if strings.HasPrefix(ct, "application/json") && bytes.ContainsAny(r.Body, `"swagger": "2.0"`) {
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

					cc.Request(method, url, dataReader, r.Ctx, headers)
				}
			}
		}

		if r.StatusCode == 200 && r.Request.URL.Path == "/robots.txt" {
			endpoints := finder.FindLinksFromRobots(string(r.Body))
			for _, e := range endpoints {
				link := r.Request.AbsoluteURL(e)
				switch runner.dispatch(link) {
				case 0:
					r.Request.Visit(link)
				case 1:
					jc.Visit(link)
				case -1:
					continue
				}
			}
		}

		if opts.UseChrome {
			if err := renderPage(r, opts.Timeout, opts.Proxy); err != nil {
				log.Warn("chromep error: ", err)
				return
			}
		}
	})

	jc.OnResponse(func(r *colly.Response) {
		if runner.filterResp(r) {
			return
		}

		endpoints := mapset.NewSet[string]()

		content := string(r.Body)
		if strings.Contains(content, `document.createElement("script");`) {
			dynamicLinks := finder.FindDynamicLinksFromJS(content)
			for _, dl := range dynamicLinks {
				log.Printf("Found dynamic script \"%s\" from JS file: %s", dl, r.Request.URL.String())
				endpoints.Add(dl)
			}
		}

		endpoints.Append(finder.FindLinksFromJS(content)...)
		log.Printf("Found %d links from JS file: %s", endpoints.Cardinality(), r.Request.URL.String())

		for ep := range endpoints.Iterator().C {
			link := util.FixURL(r.Request.URL, ep)
			switch runner.dispatch(link) {
			case 0:
				cc.Visit(link)
			case 1:
				r.Request.Visit(link)
			}
		}
	})

	// FIXME: 如果没有禁止重定向，可能会输出重复的 URL
	cc.OnScraped(func(r *colly.Response) {
		url := r.Request.URL.String()
		runner.urlSet.Add(url)

		var fields log.Fields
		if t := r.Ctx.Get("title"); t != "" {
			fields = log.Fields{"code": r.StatusCode, "length": len(r.Body), "error": runner.errorCounter, "title": t}
		} else {
			fields = log.Fields{"code": r.StatusCode, "length": len(r.Body), "error": runner.errorCounter}
		}

		log.WithFields(fields).Info(url)
	})
	jc.OnScraped(func(r *colly.Response) {
		url := r.Request.URL.String()
		runner.urlSet.Add(url)

		log.WithFields(log.Fields{"code": r.StatusCode, "length": len(r.Body), "error": runner.errorCounter}).Info(url)
	})
}

// 根据传入的 url 决定如何访问，返回值为如下 int 类型
// 0: 使用 core collector 访问
// 1: 使用 js collector 访问
// -1: 跳过对该 url 的访问
func (runner *Runner) dispatch(url string) int {
	ext := util.GetExtension(url)
	if !runner.urlSet.Contains(url) {
		if ext == ".js" || ext == ".json" {
			return 1
		} else if strings.HasSuffix(url, "swagger-resources") {
			return 1
		} else {
			return 0
		}
	} else {
		return -1
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

func (runner *Runner) Wait() {
	runner.coreCollector.Wait()
	runner.jsCollector.Wait()
}
