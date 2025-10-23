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
		{
			input:  "testdata/oneline.go",
			output: "testdata/golden/oneline.go",
		},
		{
			input:  "testdata/trailing.go",
			output: "testdata/golden/trailing.go",
		},
		{
			input:  "testdata/regression.go",
			output: "testdata/golden/regression.go",
		},
	} {
		t.Run(test.input, func(t *testing.T) {
			f, err := os.Open(test.input)
			if err != nil {
				t.Fatalf("opening input: %v", err)
			}
			t.Cleanup(func() {
				if err := f.Close(); err != nil {
					t.Fatalf("closing file: %v", err)
				}
			})
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

func TestMinIndent(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{
			name: "empty",
			in:   "",
			want: 0,
		},
		{
			name: "whitespace_only",
			in:   "  \n\t \n",
			want: 2,
		},
		{
			name: "no_indent",
			in:   "hello\nworld",
			want: 0,
		},
		{
			name: "spaces",
			in:   "  hello\n  world",
			want: 2,
		},
		{
			name: "tabs",
			in:   "\thello\n\tworld",
			want: 1,
		},
		{
			name: "mixed_indent_spaces_and_tabs",
			in:   "  hello\n\tworld",
			want: 1,
		},
		{
			name: "varied_indent",
			in:   "   hello\n world\n  again",
			want: 1,
		},
		{
			name: "with_empty_line",
			in:   "  hello\n\n  world",
			want: 2,
		},
		{
			name: "leading_empty_line",
			in:   "\n  hello\n  world",
			want: 2,
		},
		{
			name: "no_indent_with_empty_line",
			in:   "hello\n\nworld",
			want: 0,
		},
		{
			name: "single_tab",
			in:   "\t// Hi",
			want: 1,
		},
		{
			name: "double_tab",
			in:   "\t\t // Hi",
			want: 3,
		},
		{
			name: "space_and_tab",
			in:   " \t // Hi",
			want: 3,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			want := test.want
			if got := minIndent(test.in); got != want {
				t.Errorf("minIndent() = %v, want %v", got, want)
			}
		})
	}
}

func init() {
	opts.RegisterFlags(flag.CommandLine)
}
