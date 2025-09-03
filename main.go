package main

// Binary gofencefmt reformats Go code that appears in Markdown code fences.
// It is meant to be invoked in situ in a text editor on range of text, whereby
// the output is taken and replaces the original text in the buffer.
//
// Input:
//
//	``` if true { fmt.Println("I am invincible!") } ```
//
// Output:
//
//	``` if true { fmt.Println("I am invincible!") } ```
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
	return n
}

func toAST(in string) (*dst.File, error) {
	f, err := func() (f *dst.File, err error) {
		defer func() {
			if data := recover(); data != nil {
				err = fmt.Errorf("failed: %v", err)
			}
		}()
		f, err = decorator.Parse(in)
		return f, err
	}()
	if err != nil {
		return nil, err
	}
	return f, nil
}

var errGaveUp = errors.New("could not build AST")

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
	astIndent = minIndent(string(in))
	{
		// Try as whole program.
		f, err := toAST(string(in))
		if err == nil {
			return f, astIndent, 0, nil
		}
	}
	var buf bytes.Buffer
	{
		// Try as top-level identifiers.
		fmt.Fprintln(&buf, "package main")
		fmt.Fprintln(&buf, "")
		fmt.Fprintln(&buf, "// BEGIN")
		buf.Write(in)
		fmt.Fprintln(&buf, "// END")
		f, err := toAST(buf.String())
		if err == nil {
			return f, astIndent, 0, nil
		}
	}
	// Try as in-function.
	buf.Reset()
	fmt.Fprintln(&buf, "package main")
	fmt.Fprintln(&buf, "")
	fmt.Fprintln(&buf, "func init() {")
	fmt.Fprintln(&buf, "// BEGIN")
	buf.Write(in)
	fmt.Fprintln(&buf, "// END")
	fmt.Fprintln(&buf, "}")
	f, err := toAST(buf.String())
	if err != nil {
		return nil, 0, 0, errGaveUp
	}
	return f, astIndent, 1, nil
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
	var trimmed bytes.Buffer
	scanner := bufio.NewScanner(&formatted)
	indent := strings.Repeat(" ", c)
	for scanner.Scan() {
		txt := scanner.Text()
		trimmedTxt := strings.TrimSpace(txt)
		if trimmedTxt == "// BEGIN" {
			trimmed.Reset()
			continue
		}
		if trimmedTxt == "// END" {
			break
		}
		if txt != "" {
			fmt.Fprintf(&trimmed, "%s%s\n", indent, txt[n:])
		} else {
			fmt.Fprintf(&trimmed, "%s\n", indent)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanning: %v", err)
	}
	if _, err := io.Copy(w, &trimmed); err != nil {
		return fmt.Errorf("copying reformatted text: %v", err)
	}
	return nil
}

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		log.Fatalln(err)
	}
}
