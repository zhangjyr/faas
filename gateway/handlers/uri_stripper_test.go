package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_UriStripper_RemovesPrefix_WithFlatPath(t *testing.T) {
	rr := httptest.NewRecorder()

	prefix := "/function/"

	foundPath := ""
	nextHandler := func(w http.ResponseWriter, r *http.Request) {
		foundPath = r.URL.Path
	}

	decorated := MakeURIPrefixStripper(nextHandler, prefix)

	request, _ := http.NewRequest(http.MethodGet, "http://localhost:31111/function/tester", nil)
	decorated.ServeHTTP(rr, request)

	want := "tester"
	if want != foundPath {
		t.Errorf("Path stripper failed. Want: %s, but got: %s", want, foundPath)
		t.Fail()
	}
}

func Test_UriStripper_RemovesPrefix_WithMultipleParams(t *testing.T) {
	rr := httptest.NewRecorder()

	prefix := "/function/"

	foundPath := ""
	nextHandler := func(w http.ResponseWriter, r *http.Request) {
		foundPath = r.URL.Path
	}

	decorated := MakeURIPrefixStripper(nextHandler, prefix)

	request, _ := http.NewRequest(http.MethodGet, "http://localhost:31111/function/tester/22", nil)
	decorated.ServeHTTP(rr, request)

	want := "tester/22"
	if want != foundPath {
		t.Errorf("Path stripper failed. Want: %s, but got: %s", want, foundPath)
		t.Fail()
	}
}
