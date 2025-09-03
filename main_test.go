package main

import (
	"bytes"
	"flag"
	"os"
	"testing"

	"github.com/matttproud/goldentest"
)

var opts goldentest.Options

func Test(t *testing.T) {
	for _, test := range []struct {
		input  string
		output string
	}{
		{
			input:  "testdata/wholeprg.go",
			output: "testdata/golden/wholeprg.go",
		},
		{
			input:  "testdata/toplevel.go",
			output: "testdata/golden/toplevel.go",
		},
		{
			input:  "testdata/inline.go",
			output: "testdata/golden/inline.go",
		},
	} {
		t.Run(test.input, func(t *testing.T) {
			f, err := os.Open(test.input)
			if err != nil {
				t.Fatalf("opening input: %v", err)
			}
			t.Cleanup(func() { f.Close() })
			var out bytes.Buffer
			if err := run(f, &out); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := goldentest.DiffReader(t, &out, test.output, &opts); diff != "" {
				t.Errorf("run(...) yielded unexpected diff:\n\n%v", diff)
			}
		})
	}
}

func init() {
	opts.RegisterFlags(flag.CommandLine)
}
