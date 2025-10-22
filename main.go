package main

// Binary gofencefmt reformats Go code that appears in Markdown code fences.
// It is meant to be invoked in situ in a text editor on range of text, whereby
// the output is taken and replaces the original text in the buffer.
//
// Input:
//
//	```
//	if true {
//	fmt.Println("I am invincible!")
//	}
//	```
//
// Output:
//
//	```
//	if true {
//	 	fmt.Println("I am invincible!")
// 	}
//	```
//
// If using Vim (or similar) and you visually select the range of text inside
// of the Markdown fences (```...```), run :!gofencefmt, Vim invokes gofencefmt
// with this selection as input and replaces the original text with the
// reformatted output.
//
// gofencefmt is implemented very naively, but it does do some ergonomic
// things:
//
//  1. Reformatting respects the minimum level of indentation that the text
//  inside the Markdown code fence appears in and re-aligns the reformatted
//  text along that indentation level.  This is to support code fences that
//  occur at deeper levels of indentation inside of a Markdown document (e.g.,
//  inside a list, block quote, etc).
//
//  2. It is capable of reformatting whole programs, fragments of top-level
//  identifiers, and excerpted segments of function blocks. package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"iter"
	"log"
	"os"
	"strings"
	"unicode"

	// package dst and friends were less impervious to panicking when
	// operating on fragments of programs instead of whole bodies of code.
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

func minIndent(in string) (n int) {
	r := strings.NewReader(in)
	scanner := bufio.NewScanner(r)
	n = -1
	for scanner.Scan() {
		txt := scanner.Text()
		if txt == "" {
			continue
		}
		var i int
		for _, c := range txt {
			if !unicode.IsSpace(c) {
				break
			}
			i++
		}
		if n == -1 {
			n = i
		}
		if i == 0 {
			return 0 // No point in scanning further.
		}
		if i < n {
			n = i
		}
	}
	if n < 0 {
		return 0 // As a failsafe for empty lines.
	}
	return n
}

func toAST(in string) (*dst.File, error) {
	f, err := func() (_ *dst.File, err error) {
		defer func() {
			if data := recover(); data != nil {
				err = fmt.Errorf("failed: %v", data)
			}
		}()
		f, err := decorator.Parse(in)
		return f, err
	}()
	if err != nil {
		return nil, err
	}
	return f, nil
}

var errGaveUp = errors.New("could not build AST")

func parseAsWholeProgram(prg string, buf *bytes.Buffer) (*dst.File, error) {
	defer buf.Reset()
	fmt.Fprintln(buf, "// BEGIN")
	buf.WriteString(prg)
	fmt.Fprintln(buf, "// END")
	f, err := toAST(buf.String())
	if err != nil {
		return nil, err
	}
	return f, nil
}

func parseAsTopLevelIdentifiers(prg string, buf *bytes.Buffer) (*dst.File, error) {
	defer buf.Reset()
	fmt.Fprintln(buf, "package main")
	fmt.Fprintln(buf, "")
	fmt.Fprintln(buf, "// BEGIN")
	buf.WriteString(prg)
	fmt.Fprintln(buf, "// END")
	f, err := toAST(buf.String())
	if err != nil {
		return nil, err
	}
	return f, nil
}

func parseAsFunction(prg string, buf *bytes.Buffer) (*dst.File, error) {
	defer buf.Reset()
	fmt.Fprintln(buf, "package main")
	fmt.Fprintln(buf, "")
	fmt.Fprintln(buf, "func init() {") // Just an arbitrary function to place things in.
	fmt.Fprintln(buf, "// BEGIN")
	buf.WriteString(prg)
	fmt.Fprintln(buf, "// END")
	fmt.Fprintln(buf, "}")
	f, err := toAST(buf.String())
	if err != nil {
		return nil, err
	}
	return f, nil
}

// parse attempts to generate an AST from the provided source in the reader.
// It re-represents the source in various forms in case it cannot be converted
// into an AST readily.  It returns the AST, the degree to which the Markdown
// fence content is indented, and whether the AST representation further
// indents the content unintentionally.
func parse(r io.Reader) (ast *dst.File, mdIndent int, astIndent int, err error) {
	in, err := io.ReadAll(r)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("reading: %v", err)
	}
	prg := string(in)
	astIndent = minIndent(prg)
	var buf bytes.Buffer
	if f, err := parseAsWholeProgram(prg, &buf); err == nil {
		return f, astIndent, 0, nil
	}
	if f, err := parseAsTopLevelIdentifiers(prg, &buf); err == nil {
		return f, astIndent, 0, nil
	}
	if f, err := parseAsFunction(prg, &buf); err == nil {
		return f, astIndent, 1, nil
	}
	return nil, 0, 0, errGaveUp
}

func trimTrailingSpace(buf *bytes.Buffer) {
	n := len(bytes.TrimRightFunc(buf.Bytes(), unicode.IsSpace))
	buf.Truncate(n)
}

var errNoBeginning = errors.New("could not find beginning")

func seekToBeginning(s *bufio.Scanner) error {
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "// BEGIN" {
			return s.Err()
		}
	}
	return errNoBeginning
}

var errNoEnd = errors.New("could not find end")

func readLinesUntilEnd(s *bufio.Scanner) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		for s.Scan() {
			line := s.Text()
			switch {
			case strings.TrimSpace(line) == "// END":
				yield("", s.Err())
				return
			case strings.HasSuffix(line, "// END"):
				yield(strings.TrimSuffix(line, "// END"), s.Err())
				return
			default:
				if !yield(line, nil) {
					return
				}
			}
		}
		yield("", errNoEnd)
	}
}

func isExclusivelyWhitespace(s string) bool {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func run(r io.Reader, w io.Writer) error {
	f, c, n, err := parse(r)
	if err != nil {
		return fmt.Errorf("parsing input: %v", err)
	}
	var formatted bytes.Buffer
	if err := decorator.Fprint(&formatted, f); err != nil {
		return fmt.Errorf("formatting AST: %v", err)
	}
	scanner := bufio.NewScanner(&formatted)
	if err := seekToBeginning(scanner); err != nil {
		return fmt.Errorf("seeking to beginning: %v", err)
	}
	indent := strings.Repeat(" ", c)
	var buf bytes.Buffer
	for line, err := range readLinesUntilEnd(scanner) {
		if err != nil {
			return fmt.Errorf("reading until end: %v", err)
		}
		switch {
		case isExclusivelyWhitespace(line):
			if _, err := fmt.Fprintf(&buf, "%s\n", indent); err != nil {
				return fmt.Errorf("writing empty line: %v", err)
			}
		default:
			if _, err := fmt.Fprintf(&buf, "%s%s\n", indent, line[n:]); err != nil {
				return fmt.Errorf("writing line: %v", err)
			}
		}
	}
	trimTrailingSpace(&buf)
	if _, err := io.Copy(w, &buf); err != nil {
		return err
	}
	return nil
}

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		log.Fatalln(err)
	}
}
