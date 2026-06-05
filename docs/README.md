# Paper V2

[![GoDoc](https://godoc.org/github.com/avdoseferovic/paper?status.svg)](https://pkg.go.dev/github.com/avdoseferovic/paper)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go#template-engines)
[![Go Report Card](https://goreportcard.com/badge/github.com/avdoseferovic/paper)](https://goreportcard.com/report/github.com/avdoseferovic/paper)
[![CI](https://github.com/avdoseferovic/paper/actions/workflows/goci.yml/badge.svg)](https://github.com/avdoseferovic/paper/actions/workflows/goci.yml)
[![Lint](https://github.com/avdoseferovic/paper/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/avdoseferovic/paper/actions/workflows/golangci-lint.yml)
[![Codecov](https://img.shields.io/codecov/c/github/avdoseferovic/paper)](https://codecov.io/gh/avdoseferovic/paper)
[![Visits Badge](https://badges.pufler.dev/visits/avdoseferovic/paper)](https://badges.pufler.dev)
[![Stars Badge](https://img.shields.io/github/stars/avdoseferovic/paper.svg?style=social&label=Stars)](https://github.com/avdoseferovic/paper/stargazers)

### News :new:

#### 1. Paper`v2.4.0`is here! Try out:

* Installation with`go get`:

```bash
go get github.com/avdoseferovic/paper@v2.4.0
```

The public API was completely redesigned with the aim of enhancing the
library in various aspects. The main objectives of`v2.0.0`are:

1. [Improve usability](README.md?id=improve-usability);
2. [Allow unit testing](README.md?id=unit-testing);
3. [Add built-in metrics](README.md?id=built-in-metrics);
4. [Improve execution time](README.md?id=execution-time-improvement);
5. Allow recursive Row/Col; **(on roadmap)**
6. Allow generation based on [serialized data](https://github.com/avdoseferovic/paper/discussions/390).

## HTML to PDF

For HTML-only documents, the public API is one call:

```go
doc, err := paper.FromHTML(`<h1>Hello</h1><p>World</p>`)
if err != nil {
    log.Fatal(err)
}

_ = doc.Save("out.pdf")
```

Use the row/column API when you need to mix HTML with manual components, headers, footers, page numbers, or lower-level layout control.

## Code Example
This is part of the [simplest example](v2/examples/simplest?id=simplest).

[filename](assets/examples/simplest/v2/main.go ':include :type=code')

## PDF Example
This is part of the [billing example](v2/examples/billing?id=billing).

```pdf
	assets/pdf/billingv2.pdf
```

## Paper Columns and Rows

**Paper** employs a flexible grid system to structure content in a PDF document, consisting of rows and columns. This system is designed to provide both simplicity and versatility in layout management.

### Columns

- **Grid System**: Paper's layout is based on a 12-unit grid system. This means the width of each page is effectively divided into 12 equal parts (or "grid spaces").
- **Column Width**: When creating a column (using col.New(colSize)), the colSize parameter specifies how many of these 12 grid spaces the column should occupy. For example, col.New(1) creates a column that spans 1/12 of the page width, while col.New(6) spans half the page width.
- **Content Placement**: Columns are the primary containers for content such as text, images, and other components. The width of a column determines how much horizontal space its content occupies.
- **Total Width Constraint**: The sum of the widths of all columns within a single row should not exceed 12 grid spaces, aligning with the full width of the page.
- **Grid Space Customization**: Is possible to [customize](https://paper.tech/#/v2/features/maxgridsum?id=max-grid-sum) the max grid sum.

### Rows

- **Vertical Structuring**: Rows in Paper are used to organize content vertically. Each row acts as a horizontal container for columns.
- **Row Height**: The height of a row is defined when it is created (e.g., row.New(20)). This height determines the vertical space allocated for the row. Unlike columns, row height is not based on a grid system but is a relative unit of `mm`.
- **Sequential Layout**: Rows are added to the document in the order they are defined, creating a top-to-bottom flow of content. Each new row is placed immediately below the preceding row.
- **Layout Flexibility**: Rows offer flexibility in the layout design, allowing for various configurations of columns within them. From single full-width columns to multiple columns of different widths, rows accommodate diverse layout patterns.


## Conventions

Paper `v2` keeps the component model from the previous version while making the runtime boundaries stricter. Components are
declared first, then rendered through Paper-owned provider interfaces and the internal PDF backend. This keeps application code
independent from a third-party PDF engine dependency and gives Paper room to support other document outputs in the future.

### Structure
In paper`v2`, everything is a **component**. When you add a **row** to the document, you are essentially adding a
**component**. Similarly, when you add an **image** to a **col**, you are also introducing a **component**. The
functioning of paper`v2`is straightforward: it involves constructing a **components tree** and processing the
tree by incorporating the **components** into the document.

<iframe frameborder="0" style="width:100%;height:600px;" src="https://viewer.diagrams.net/?tags=%7B%7D&highlight=0000ff&edit=_blank&layers=1&nav=1&title=paperv2-structure.drawio#Uhttps%3A%2F%2Fdrive.google.com%2Fuc%3Fid%3D1H-xFq-6DNg-V6aUWsFxM0VthUvA5ptWZ%26export%3Ddownload"></iframe>
<div style="text-align: center;">Components tree</div>

### Runtime

Paper`v2`now features a distinct separation into two runtime phases: **Declaration Phase** and **Generation Phase**.

1. **Declaration Phase:** During this phase, you will create the paper instance, add various elements such as pages, rows, cols, images, and so on.
   - When calling the add methods, paper will not make any changes to the document. Instead, it will solely construct the **components tree**.
2. **Generation Phase:** This phase is triggered when the `Generate() (Document, error)` method is called.
   - In this phase, paper will traverse the **components tree** structure, compute the grid dimensions, and add components to the document.

## Improve Usability
The recursive design of paper`v1`has proven to be a limiting factor in enhancing project features. This
[issue][old_row_issue] was identified years ago. To enhance code quality and streamline the addition of new features,
a refactor of the [paper interface][old_paper_interface] is necessary. This involves extracting interfaces for each
feature within paper, such as (Row, Col, Text, QRCode, Image, etc). With the new interfaces, usability will be greatly
improved, enabling paper to reach a whole new level in this fresh new step.

### New Interfaces
[filename](https://raw.githubusercontent.com/avdoseferovic/paper/master/pkg/core/core.go ':include :type=code')

## Unit Testing
In paper`v2`, it is possible to write unit tests by analyzing the **components tree**. To facilitate the
writing of unit tests, we created a dedicated test package.

For an example, refer to [this link](v2/features/unittests?id=unit-testing).

## Built-in Metrics
This new version of paper introduces an **optional decorator** that provides metrics for nearly all operations
performed by the library. When the decorator is enabled, paper will populate the **report** struct within
the **document** response.

The **report** struct contains the following information. For a complete example, refer
to [this link](v2/basics?id=using-metrics-decorator).

[filename](../assets/text/report.txt ':include :type=code')

## Execution Time Improvement
In Paper`v2`, numerous performance enhancements have been implemented. The core algorithm is now **at least
twice as fast as V1**. This disparity becomes even more remarkable when parallel generation is enabled. The
subsequent results were achieved by generating a PDF with **100 pages** encompassing **all components supported**
by Paper`v2`.

[filename](../assets/text/parallel.txt ':include :type=code')

The PDF generated was a custom version of ([billing example](v2/examples/billing?id=billing)), with **100 pages**.
The pages are merged using Paper's in-repo PDF merger. For a complete example, please refer to
[this link](v2/features/parallelism?id=parallelism).


[old_paper_interface]: https://github.com/avdoseferovic/paper/blob/master/pkg/core/core.go
[old_row_issue]: https://github.com/avdoseferovic/paper/issues/55
