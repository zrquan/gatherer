package finder

import (
	"fmt"
	"os"
	"testing"
)

func TestFindLinksFromSwagger(t *testing.T) {
	testFile := "schema.json"
	source, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("cannot open file: %s\n", testFile)
	}

	endpoints, err := FindLinksFromSwagger(source)
	if err != nil {
		t.Error(err)
	}

	for _, api := range endpoints {
		fmt.Printf("[%s] %s\n", api.Method, api.URL)
		for k, v := range api.Headers {
			fmt.Printf("  | %s: %s\n", k, v)
		}
		fmt.Printf("  > %s\n\n", api.Content)
	}
}
