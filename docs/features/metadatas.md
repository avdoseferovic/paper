# Metadatas

PDF metadata fields are stored in the document's information dictionary and are visible in PDF viewer properties dialogs, search indexes, and archiving systems. paper exposes them through builder methods.

## Usage notes

- Metadata does not appear in the rendered PDF content — it is only stored in the document's information dictionary.
- All fields are optional; omit any method for which you do not have a value.
- `WithCreationDate` accepts a `time.Time` value; paper formats it according to the PDF specification.

## GoDoc
* [builder : WithAuthor](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/config#CfgBuilder.WithAuthor)
* [builder : WithCreationDate](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/config#CfgBuilder.WithCreationDate)
* [builder : WithCreator](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/config#CfgBuilder.WithCreator)
* [builder : WithSubject](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/config#CfgBuilder.WithSubject)
* [builder : WithTitle](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/config#CfgBuilder.WithTitle)
* [builder : WithKeywords](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/config#CfgBuilder.WithKeywords)

## Code Example
[filename](../../assets/examples/metadatas/main.go ':include :type=code')

## PDF Generated
```pdf
	assets/pdf/metadatas.pdf
```

## Time Execution
[filename](../../assets/text/metadatas.txt  ':include :type=code')

## Test File
[filename](https://raw.githubusercontent.com/avdoseferovic/paper/master/test/paper/examples/metadatas.json  ':include :type=code')