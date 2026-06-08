# Header

`RegisterHeader` registers a row (or a set of rows) that is automatically printed at the top of **every** page. The header appears just below the top margin and is drawn before any body content on each page. Use it for logos, report titles, column labels, or any content that should repeat on every page.

## Usage notes

- `RegisterHeader` accepts one or more `core.Row` values; they are stacked top-to-bottom in the order given.
- Header rows consume vertical space — paper deducts their total height from the useful area on every page so body content starts below the header. The useful area is the page height after top and bottom margins are removed.
- Call `RegisterHeader` once before generating any content; calling it again replaces the previous header.
- Returns an error if the header rows, together with any registered footer, exceed the page's useful height.

## GoDoc
* [paper : RegisterHeader](https://pkg.go.dev/github.com/avdoseferovic/paper#Paper.RegisterHeader)

## Code Example
[filename](../../assets/examples/header/main.go ':include :type=code')

## PDF Generated
```pdf
	assets/pdf/header.pdf
```

## Time Execution
[filename](../../assets/text/header.txt  ':include :type=code')

## Test File
[filename](https://raw.githubusercontent.com/avdoseferovic/paper/master/test/paper/examples/header.json  ':include :type=code')
