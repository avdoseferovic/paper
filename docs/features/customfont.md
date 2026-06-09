# Custom Font

Paper ships with a standard set of built-in fonts. `WithCustomFonts` lets you register additional TrueType (UTF-8) fonts so that text components can use them by name. This is essential for non-Latin scripts such as Arabic, Chinese, Japanese, Korean, or any language requiring characters outside the Latin-1 set.

## Usage notes

- Each style variant (`Normal`, `Bold`, `Italic`, `BoldItalic`) must be registered separately.
- If a style is not registered, paper falls back to the `Normal` variant of the same family.
- Font files are embedded in the PDF, so the output is self-contained but larger.

## GoDoc
* [builder : WithCustomFonts](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/config#CfgBuilder.WithCustomFonts)
* [repository : AddUTF8Font](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/repository#FontRepository.AddUTF8Font)
* [repository : Load](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/repository#FontRepository.Load)
* [entity : CustomFont](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/core/entity#CustomFont)

## Code Example
[filename](../assets/examples/customfont/main.go ':include :type=code')

## PDF Generated
```pdf
	assets/pdf/customfont.pdf
```
## Time Execution
[filename](../assets/text/customfont.txt  ':include :type=code')

## Test File
[filename](https://raw.githubusercontent.com/avdoseferovic/paper/master/test/paper/examples/customfont.json  ':include :type=code')