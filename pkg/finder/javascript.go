package finder

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/go-rod/rod"
	"github.com/zrquan/gatherer/pkg/util"
)

var linkFinderRegex = regexp.MustCompile(`(?:"|')` + `(` +
	// Match a scheme [a-Z]*1-10 or //
	`((?:[a-zA-Z]{1,10}://|//)` +
	// Match a domainname (any character + dot)
	`[^"'/]{1,}\.` +
	// The domainextension and/or path
	`[a-zA-Z]{2,}[^"']{0,})` +

	`|` +

	// Start with /,../,./
	`((?:/|\.\./|\./)` +
	// Next character can't be...
	`[^"'><,;| *()(%%$^/\\\[\]]` +
	// Rest of the characters can't be
	`[^"'><,;|()]{1,})` +

	`|` +

	// Relative endpoint with /
	`([a-zA-Z0-9_\-/]{1,}/` +
	// Resource name
	`[a-zA-Z0-9_\-/]{1,}` +
	// Rest + extension (length 1-4 or action)
	`\.(?:[a-zA-Z]{1,4}|action)` +
	// ? or # mark with parameters
	`(?:[\?|#][^"|']{0,}|))` +

	`|` +

	// REST API (no extension) with /
	`([a-zA-Z0-9_\-/]{1,}/` +
	// Proper REST endpoints usually have 3+ chars
	`[a-zA-Z0-9_\-/]{3,}` +
	// ? or # mark with parameters
	`(?:[\?|#][^"|']{0,}|))` +

	`|` +

	// filename
	`([a-zA-Z0-9_\-]{1,}` +
	// . + extension
	`\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml)` +
	// ? or # mark with parameters
	`(?:[\?|#][^"|']{0,}|))` +

	`)` + `(?:"|')`,
)

// FindLinks 从 JS 文件中获取 URL
func FindLinksFromJS(source string) []string {
	var endpoints []string
	match := linkFinderRegex.FindAllStringSubmatch(source, -1)
	for _, m := range match {
		// 去重
		ep := m[1]
		if !slices.Contains(endpoints, ep) {
			endpoints = append(endpoints, ep)
		}
	}
	return endpoints
}

// FindDynamicLinks 通过动态生成并执行 JS 代码，获取 Webpack 打包的资源路径
func FindDynamicLinksFromJS(source string, browser *rod.Browser) []string {
	var endpoints []string
	jsRegex := regexp.MustCompile(`\w\.p\+"(.*?)\.js`)
	match := jsRegex.FindAllStringSubmatch(source, -1)
	for _, m := range match {
		if len(m[1]) < 30000 {
			code := fmt.Sprintf(`"%s.js"`, m[1])
			jsFunc := fmt.Sprintf(`function() {js_url=%s; return js_url}`, code)
			variables := regexp.MustCompile(`\[.*?\]`).FindAllString(code, -1)
			if len(variables) > 0 {
				if v0 := variables[0]; strings.Contains(v0, "[") && strings.Contains(v0, "]") {
					variable := strings.Replace(strings.Replace(v0, "[", "", -1), "]", "", -1)
					jsFunc = strings.Replace(jsFunc, "function()", "function("+variable+")", -1)
				}
			}
			flagCode := regexp.MustCompile(`\(\{\}\[(.*?)\]\|\|.\)`).FindAllString(jsFunc, -1)
			if flagCode != nil {
				jsFunc = strings.Replace(jsFunc, fmt.Sprintf("({}[%s]||%s)", flagCode[0], flagCode[0]), flagCode[0], -1)
			}
			var nameList1 []string
			for _, nm := range regexp.MustCompile(`\{(.*?)\:`).FindAllStringSubmatch(code, -1) {
				nameList1 = append(nameList1, nm[1])
			}
			var nameList2 []string
			for _, nm := range regexp.MustCompile(`\,(.*?)\:`).FindAllStringSubmatch(code, -1) {
				nameList2 = append(nameList2, nm[1])
			}
			nameList := util.Dedup(slices.Concat(nameList1, nameList2))
			for _, name := range nameList {
				returnValue, err := evalJavascript(jsFunc, name, browser)
				if err != nil {
					break
				}
				endpoints = append(endpoints, returnValue)
			}
		}
	}
	return endpoints
}

// TODO: 执行 Javascript，后续优化代码结构
func evalJavascript(jsFunc, name string, browser *rod.Browser) (string, error) {
	page := browser.MustPage("about:blank")
	defer page.MustClose()

	name = strings.ReplaceAll(name, `"`, "")
	result, err := page.Eval(jsFunc, name)
	if err != nil {
		fmt.Println("Failed to execute script:", err)
		return "", err
	}
	if result.Type != "" {
		return result.Value.String(), nil
	}
	return "", errors.New("the result is undefined or null")
}
