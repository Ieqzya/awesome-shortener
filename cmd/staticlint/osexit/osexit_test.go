package osexit_test

import (
	"testing"

	"awesome-shortener/cmd/staticlint/osexit"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestOsExit(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, osexit.Analyzer, "a")
}
