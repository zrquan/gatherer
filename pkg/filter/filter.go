package filter

import (
	"fmt"

	"github.com/gocolly/colly/v2"
)

type IFilter interface {
	Filter(response *colly.Response) (bool, error)
	Repr() string
	ReprVerbose() string
}

func NewFilterByName(name, input string) (IFilter, error) {
	switch name {
	case "status":
		return NewStatusFilter(input)
	case "extension":
		return NewExtensionFilter(input)
	case "length":
		return NewLengthFilter(input)
	default:
		return nil, fmt.Errorf("could not create filter with name %s", name)
	}
}
