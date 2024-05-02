package filter

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
)

type StatusFilter struct {
	Values []int
}

func NewStatusFilter(input string) (IFilter, error) {
	var values []int
	for _, status := range strings.Split(input, ",") {
		code, err := strconv.Atoi(status)
		if err != nil {
			return nil, err
		}
		values = append(values, code)
	}
	return &StatusFilter{Values: values}, nil
}

func (sf *StatusFilter) Filter(response *colly.Response) (bool, error) {
	return slices.Contains(sf.Values, response.StatusCode), nil
}

func (sf *StatusFilter) Repr() string {
	tmp := make([]string, len(sf.Values))
	for _, v := range sf.Values {
		tmp = append(tmp, strconv.Itoa(v))
	}
	return strings.Join(tmp, ",")
}

func (sf *StatusFilter) ReprVerbose() string {
	return fmt.Sprintf("Response status: %s", sf.Repr())
}
