package filter

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
)

type LengthFilter struct {
	Values []int
}

func NewLengthFilter(input string) (IFilter, error) {
	var values []int
	for _, l := range strings.Split(input, ",") {
		length, err := strconv.Atoi(l)
		if err != nil {
			return nil, err
		}
		values = append(values, length)
	}
	return &LengthFilter{Values: values}, nil
}

func (lf *LengthFilter) Filter(response *colly.Response) (bool, error) {
	return slices.Contains(lf.Values, len(response.Body)), nil
}

func (lf *LengthFilter) Repr() string {
	tmp := make([]string, len(lf.Values))
	for _, v := range lf.Values {
		tmp = append(tmp, strconv.Itoa(v))
	}
	return strings.Join(tmp, ",")
}

func (lf *LengthFilter) ReprVerbose() string {
	return fmt.Sprintf("Response length: %s", lf.Repr())
}
