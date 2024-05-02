package filter

import (
	"fmt"
	"slices"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/zrquan/gatherer/pkg/util"
)

type ExtensionFilter struct {
	Values []string
}

func NewExtensionFilter(input string) (IFilter, error) {
	var values []string
	for _, ext := range strings.Split(input, ",") {
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		values = append(values, ext)
	}
	return &ExtensionFilter{Values: values}, nil
}

func (ef *ExtensionFilter) Filter(response *colly.Response) (bool, error) {
	ext := util.GetExtension(response.Request.URL.String())
	return slices.Contains(ef.Values, ext), nil
}

func (ef *ExtensionFilter) Repr() string {
	tmp := make([]string, len(ef.Values))
	for _, v := range ef.Values {
		tmp = append(tmp, strings.TrimPrefix(v, "."))
	}
	return strings.Join(tmp, ",")
}

func (sf *ExtensionFilter) ReprVerbose() string {
	return fmt.Sprintf("Response extension: %s", sf.Repr())
}
