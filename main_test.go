package main

import (
	"flag"
	"testing"
)

var runAcceptanceTests bool

var _ = func() bool {
	testing.Init()
	return true
}()

func init() {
	flag.BoolVar(&runAcceptanceTests, "acceptance-tests", false, "set flag when running acceptance tests")
	flag.Parse()
}

func TestRunMain(t *testing.T) {
	if !runAcceptanceTests {
		t.Skip()
	}
	doMain()
}
