# gofencefmt

gofencefmt is a tool to reformat blocks of Go code (in the spirit of `gofmt`)
that appear in a Markdown code fence.

## Motivating Examples

Let's consider some examples:

In the simplest case, we have a code snippet that is not indented correctly.

```go
// Input:
package main

func songThatNeverEnds() {
itGoesOn()
andOn()
myFriends()
}
```

If you run gofencefmt on the fence's contents, it will produce this:

```go
// Output:
package main

func songThatNeverEnds() {
	itGoesOn()
	andOn()
	myFriends()
}
```

Note that the code is indented even with Go's native tabs.  Now, that isn't all
that interesting in and of itself, so let's consider some more interesting
cases:

1. **Indented Fences**: Below we have an indented code fence that is a child
   of this Markdown list item.  It's also malformatted.

   ```go
   // Input:
   package main

   func songThatNeverEnds() {
   itGoesOn()
   andOn()
   myFriends()
   }
   ```

   When gofencefmt is run on the literal text range of the indented fence, it
   ensures the output matches the same fundamental level of Markdown
   indentation.

   ```go
   // Output:
   package main

   func songThatNeverEnds() {
   	itGoesOn()
   	andOn()
   	myFriends()
   }
   ```

2. **Supports Snippet**: Sometimes when writing code snippets, we like to excerpt
   top-level identifiers without them being part of a whole program.

   ```go
   // Input:
   func songThatNeverEnds() {
   itGoesOn()
   andOn()
   myFriends()
   }
   ```

   gofencefmt handles these just fine:

   ```go
   // Output:
   func songThatNeverEnds() {
   	itGoesOn()
   	andOn()
   	myFriends()
   }
   ```

   **Note:** There is no `package` clause in the snippet above!

   gofencefmt can even do this for code snippets that would appear inline in a
   function (or method).

   ```go
   // Input:
   for isSongThatNeverEnds() {
   itGoesOn()
   andOn()
   myFriends()
   }
   ```

   And even output them in the most concise and indentation-correct way:

   ```go
   // Output:
   for isSongThatNeverEnds() {
   	itGoesOn()
   	andOn()
   	myFriends()
   }
   ```

## Installation

Use the standard `go install` workflow as follows:

```shell
% go install github.com/matttproud/gofencefmt@latest
```

## Usage

I have been using this with Vim and Neovim through the `!` program filter
directive.  I use visual line mode to select the range of interest, which is
exclusively the Go code in the code fence, and enter the command `:!
gofencefmt`.

Let's imagine Vim open as such:

~~~
    ```go {.good}
VL  func Test(t *testing.T) {
VL  // elided
VL  if diff := cmp.Diff(want, got); diff != "" {
VL    t.Errorf("f() = %v, want %v diff (-want, got):\n\n%v", got, want, diff)
VL  }
VL  // elided
VL  }
    ```
~~~

Here the lines prefixed with "VL" indicate these lines have been selected in
visual line mode (**Note:** "VL" is an annotation I am providing, nothing you
will see in Vim).  After the visual selection has been made, run
`:! gofencefmt`.

I have **not** tested this gofencefmt with [`conform.nvim`](https://github.com/stevearc/conform.nvim)'s [injected
language
formatting](https://github.com/stevearc/conform.nvim/blob/master/doc/advanced_topics.md#injected-language-formatting-code-blocks).  I presume it would work in some capacity.

