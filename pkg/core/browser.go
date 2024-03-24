package core

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/gocolly/colly/v2"
)

func renderPage(response *colly.Response, timeout int, proxyURL string) error {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3830.0 Safari/537.36"),
		chromedp.ProxyServer(proxyURL),
		chromedp.NoFirstRun,
		chromedp.Headless,
		chromedp.DisableGPU,
	}

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
	}

	var res string
	if err := chromedp.Run(ctx, chromedp.Navigate(response.Request.URL.String()),
		chromedp.OuterHTML("html", &res), // Scrape whole rendered page
	); err != nil {
		return fmt.Errorf("executing chromedp: %w", err)
	}

	response.Body = []byte(res)

	return nil
}
