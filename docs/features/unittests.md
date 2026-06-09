# Unit Testing

paper provides a dedicated `pkg/test` package that lets you write deterministic unit tests for your PDF-generating code. Instead of comparing binary PDF output, the test package serialises the document's **component tree** to JSON and compares it against a saved fixture. This makes tests fast, readable, and diff-friendly.

## How it works

1. Build your paper document normally.
2. In your test, call `test.New(t)` to create the test helper.
3. Call `.Assert(m.GetStructure()).Equals("fixture-name")` to compare the component tree against the JSON file under `test/paper/` at the module root.
4. Keep expected fixtures as reviewed JSON files under `test/paper/`.

## GoDoc
* [constructor : New](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/test#New)
* [method : Assert](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/test#PaperTest.Assert)
* [method : Equals](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/test#PaperTest.Equals)

## Fixture Path
The test helper stores JSON fixtures in [`test/paper/`](https://github.com/avdoseferovic/paper/tree/master/test/paper) relative to the module root.

## Code
[filename](../assets/examples/unittests/main_test.go ':include :type=code')

## Test file
[filename](https://raw.githubusercontent.com/avdoseferovic/paper/master/test/paper/example_unit_test.json ':include :type=code')
