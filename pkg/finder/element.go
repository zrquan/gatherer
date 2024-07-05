package finder

import (
	"net/url"

	"github.com/gocolly/colly/v2"
	"github.com/projectdiscovery/gologger"
	"github.com/zrquan/gatherer/pkg/util"
)

func FindLinksFromAttributes(ignoreQuery bool, e *colly.HTMLElement, attr ...string) (links []string) {
	for _, aname := range attr {
		if value := e.Attr(aname); value != "" {
			link := e.Request.AbsoluteURL(value)
			if ignoreQuery {
				u, err := url.Parse(link)
				if err != nil {
					gologger.Warning().Str("link", link).Msg("Parse URL error")
					continue
				}
				link = util.StripQueryParams(u)
			}
			links = append(links, link)
		}
	}
	return
}
