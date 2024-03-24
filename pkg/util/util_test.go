package util

import (
	"fmt"
	"net/url"
	"testing"
)

func TestFixURL(t *testing.T) {
	target := "http://example.com/api/v2/"
	path := "/v2/something/get"

	baseURL, _ := url.Parse(target)
	result := FixURL(baseURL, path)
	if result != "http://example.com/api/v2/something/get" {
		t.Errorf("wrong URL: %s\n", result)
	}
}

func TestDedup(t *testing.T) {
	input := []string{"test", "test", "xxx"}
	result := Dedup(input)
	if len(result) != 2 {
		t.Errorf("len(result) should be 2, not %d", len(result))
	}
	if result[0] != "test" {
		t.Errorf(`result[0] should be "test", not %s`, result[0])
	}
	if result[1] != "xxx" {
		t.Errorf(`result[1] should be "xxx", not %s`, result[1])
	}
}

func TestFilterSubdomains(t *testing.T) {
	input := "http://www.example.com"
	filterRegex, _ := FilterSubdomains(input)
	if sub := "sub.example.com"; !filterRegex.MatchString(sub) {
		t.Errorf(`"%s" should be a subdomain of "%s"`, sub, input)
	}
	if sub := "test.sub.example.com"; !filterRegex.MatchString(sub) {
		t.Errorf(`"%s" should be a subdomain of "%s"`, sub, input)
	}
	if sub := "example.com"; filterRegex.MatchString(sub) {
		t.Errorf(`"%s" should not be a subdomain of "%s"`, sub, input)
	}
}

func TestExtractHostname(t *testing.T) {
	input := "http://www.example.com"
	expected := "www.example.com"
	if hostname, err := ExtractHostname(input); err != nil || hostname != expected {
		t.Errorf(`hostname of "%s" should be "%s"`, input, expected)
	}
}

func ExampleGetExtension() {
	fmt.Println(GetExtension("http://www.example.com/path/test.jsp"))
	fmt.Println(GetExtension("http://www.example.com/path/test.php"))
	// Output:
	// .jsp
	// .php
}
