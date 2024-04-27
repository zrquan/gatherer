# Gatherer

Gatherer 是一个简易的爬虫工具，它可以从各种内容中收集资源链接和 API 然后进行访问

[![asciicast](https://asciinema.org/a/lv1EaQyBFkeOtI74DBjP7vFRs.svg)](https://asciinema.org/a/lv1EaQyBFkeOtI74DBjP7vFRs)

```
Gatherer v0.1.0

Usage of ./gatherer:
  -H value
        HTTP request headers (eg. -H 'Header1:value' -H 'Header2:value')
  -ch
        Run Javascript in headless Chrome
  -debug
        Debug mode
  -dep int
        Maximum path depth (default 1)
  -igq
        Ignore the query portion on the URL from a[href]
  -json
        Log as JSON format
  -limit int
        Maximum number of concurrent requests (default 100)
  -nr
        Disallow auto redirect
  -proxy string
        Proxy URL
  -sub
        Allow to visit sub-domains
  -t int
        Request timeout (second) (default 10)
  -tt int
        Total timeout (second)
  -u string
        Target URL
  -ua
        Use random User-Agent
  -w string
        Wordlist file path
```

## Features

- 从 JS 代码中收集资源链接
- 从 Webpack 打包的代码中收集动态生成的 JS 资源链接
- 从 Swagger 文档中解析 API 的完整路径、方法、参数
- 从 robots.txt 中收集资源链接
- 从 XML sitemap 中收集资源链接
- 执行 JS 完成页面渲染，比如 SPA

## Thanks

- [colly](https://github.com/gocolly/colly)
- [hakrawler](https://github.com/hakluke/hakrawler)
- [LinkFinder](https://github.com/GerbenJavado/LinkFinder)
- [Packer-Fuzzer](https://github.com/rtcatc/Packer-Fuzzer)
- more...