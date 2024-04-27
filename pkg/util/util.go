package util

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/gocolly/colly/v2"
)

func FixURL(base *url.URL, path string) string {
	// 处理 base 和 path 中重叠的路径
	// TODO: 当前实现不够完善，待优化
	basePathSlice := strings.Split(strings.TrimLeft(base.Path, "/"), "/")
	pathSlice := strings.Split(strings.TrimLeft(path, "/"), "/")
	cleaned := make([]string, 0, len(pathSlice))
	for _, p := range pathSlice {
		if !slices.Contains(basePathSlice, p) {
			cleaned = append(cleaned, p)
		}
	}
	path = strings.Join(cleaned, "/")

	nextLoc, err := url.Parse(path)
	if err != nil {
		return ""
	}
	return base.ResolveReference(nextLoc).String()
}

func IsAbsoluteURL(rawUrl string) bool {
	u, err := url.Parse(rawUrl)
	if err != nil || !u.IsAbs() {
		return false
	}
	return true
}

// ExtractHostname() extracts the hostname from a URL
func ExtractHostname(rawUrl string) (string, error) {
	u, err := url.Parse(rawUrl)
	if err != nil || !u.IsAbs() {
		return "", errors.New("input must be a valid absolute URL")
	}
	return u.Hostname(), nil
}

func GetExtension(rawUrl string) string {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return ""
	}
	return path.Ext(u.Path)
}

func FilterNewLines(s string) string {
	return regexp.MustCompile(`[\t\r\n]+`).ReplaceAllString(strings.TrimSpace(s), " ")
}

// FIXME: 目前要求输入的 hostname 必须为二级子域名
// 比如，输入 www.example.com 可以匹配 123.example.com 和 123.xxx.example.com 等下级子域名
// 但输入 123.xxx.example.com 或 example.com 都无法匹配 www.example.com
func FilterSubdomains(hostname string) (*regexp.Regexp, error) {
	if strings.Contains(hostname, "://") {
		h, err := ExtractHostname(hostname)
		if err != nil {
			return nil, err
		}
		hostname = h
	}
	dot := strings.Index(hostname, ".")
	domain := hostname[dot+1:]
	regex := regexp.MustCompile(".*(\\.|\\/\\/)" + strings.ReplaceAll(domain, ".", "\\.") + "((#|\\/|\\?).*)?")
	return regex, nil
}

// 字符串切片去重
func Dedup(original []string) []string {
	result := make([]string, 0, len(original))
	temp := map[string]struct{}{}
	for _, item := range original {
		if _, ok := temp[item]; !ok {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

func StripQueryParams(u *url.URL) string {
	var stripped = fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, u.Path)
	stripped = strings.TrimRight(stripped, "/")
	return stripped
}

func IsSwaggerSchema(resp *colly.Response) bool {
	ct := resp.Headers.Get("Content-Type")
	flag := []byte(`"swagger":`)
	return strings.HasPrefix(ct, "application/json") && bytes.Contains(resp.Body, flag)
}

func IsScriptOrJSON(link string) bool {
	ext := GetExtension(link)
	return ext == ".js" || ext == ".ts" || ext == ".json" || strings.HasSuffix(link, "swagger-resources")
}
