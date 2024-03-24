package finder

import (
	"regexp"
	"strings"

	"github.com/zrquan/gatherer/pkg/util"
)

var pathRegex = regexp.MustCompile(".*llow: ")

// FindLinksFromRobots 从 robots.txt 文件中获取相对路径
func FindLinksFromRobots(text string) []string {
	var endpoints []string
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "llow: ") {
			if path := pathRegex.ReplaceAllString(line, ""); path != "" {
				endpoints = append(endpoints, util.FilterNewLines(path))
			}
		}
	}
	return endpoints
}
