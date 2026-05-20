// html-demo generates a rich HTML capability PDF from an HTML body parsed via pkg/html.
package main

import (
	"fmt"
	"log"
	"os"

	maroto "github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontfamily"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/html"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

const body = `
<html>
<head>
  <link rel="stylesheet" href="extra.css">
  <style>
    h1 { color: #0f2c4a; font-size: 22pt; font-weight: bold; padding-bottom: 1mm }
    h2 { color: #0f2c4a; font-size: 11.5pt; font-weight: bold; padding: 1.8mm 3.2mm; background-color: #eef3f9; border-radius: 1.5mm }
    h3 { color: #1d3f5a; font-size: 9.5pt; font-weight: bold; padding-bottom: 0.4mm }
    h4 { color: #3c4350; font-size: 9pt; font-weight: bold }
    h5 { color: #5c4d2e; font-size: 8.5pt; font-weight: bold }
    h6 { color: #6a6a6a; font-size: 8pt; font-weight: bold }
    p { font-size: 8.5pt; line-height: 1.18; color: #2a323b }
    a { color: #1e5a86 }
    blockquote { color: #3d4a55; background-color: #f1f5f9; padding: 2.8mm 3.5mm; border-left: 0.9mm solid #5a7e9b; border-top-right-radius: 1.5mm; border-bottom-right-radius: 1.5mm }
    pre { color: #25415a; background-color: #f5f1e8; padding: 2.5mm 3mm; border: 0.25mm solid #d8c8aa; border-radius: 1.5mm; font-size: 7.8pt; line-height: 1.25 }

    .muted { color: #6a6a6a }
    .accent { color: #b63a26 }
    .ok { color: #217a4d }
    .small { font-size: 7.5pt }
    .tiny { font-size: 7pt }
    .subtitle { color: #5a6975; font-size: 9pt; line-height: 1.3 }
    .kicker { color: #8a5a18; font-size: 7.5pt; font-weight: bold }
    .gap { color: #ffffff; font-size: 5pt; line-height: 1 }
    .gap-lg { color: #ffffff; font-size: 10pt; line-height: 1 }

    .hero { display: flex; gap: 4mm; margin-bottom: 2mm }
    .hero-copy { flex: 5; background-color: #eef4fb; padding: 5mm 5.5mm; border-radius: 3mm; border: 0.3mm solid #c5d6e6 }
    .hero-side { flex: 3; background-color: #f8efd9; padding: 4mm 4mm; border-radius: 3mm; border: 0.3mm solid #dcc58e }
    .hero-side h3 { color: #6f4d10; font-size: 10pt }
    .hero-tag { background-color: #0f2c4a; color: #ffffff; padding: 1.4mm 2.6mm; border-radius: 1.4mm; font-size: 7.5pt; font-weight: bold }
    .hero-row { display: flex; gap: 2.4mm }

    .stats { display: flex; gap: 2.8mm }
    .stat { padding: 2.8mm 3mm; border-radius: 2mm; border: 0.25mm solid #d6dde6 }
    .stat h3 { color: #0f2c4a; font-size: 9pt }
    .stat p { font-size: 8pt; color: #404a55; line-height: 1.2 }
    .stat-a { background-color: #eaf4ee }
    .stat-b { background-color: #fbeeed }
    .stat-c { background-color: #ecf0f7 }
    .stat-d { background-color: #f6efe0 }

    .cards { display: flex; gap: 3.5mm }
    .card { padding: 3.2mm 3.5mm; border-radius: 2mm; border: 0.25mm solid #cdd6e1 }
    .card h3 { color: #0f2c4a }
    .card-blue { background-color: #e8f0fa; border-color: #b8cde4 }
    .card-teal { background-color: #e3f1ee; border-color: #afd1c9 }
    .card-amber { background-color: #fbf3e3; border-color: #e4cd99 }

    .column-stack { display: flex; flex-direction: column; row-gap: 1.6mm }
    .band { background-color: #f6f9fc; padding: 2.4mm 3mm; border: 0.2mm solid #d6dde8; border-radius: 1.8mm }
    .band-strong { background-color: #e7f1ec; padding: 2.4mm 3mm; border-left: 0.8mm solid #2d8a57; border-radius: 1.8mm }
    .band-amber { background-color: #f9efd9; padding: 2.4mm 3mm; border-left: 0.8mm solid #c08a1a; border-radius: 1.8mm }
    .offset-list { margin: 1mm 0 1mm 2mm }
    .circle-numbers { list-style-type: decimal-circle }
    .roman-list { list-style-type: upper-roman }

    .split { display: flex; gap: 4mm }
    .wide { flex: 2 }
    .narrow { flex: 1 }
    .basis-quarter { flex-basis: 25%; background-color: #eef4fa; padding: 2.4mm; border-radius: 1.8mm; border: 0.25mm solid #c8d8e6; font-size: 8pt; color: #1d3f5a; font-weight: bold }
    .basis-half { flex-basis: 50%; background-color: #f5efe5; padding: 2.4mm; border-radius: 1.8mm; border: 0.25mm solid #d9c8a7; font-size: 8pt; color: #6f4d10; font-weight: bold }
    .basis-rest { flex: 1; background-color: #edf6f0; padding: 2.4mm; border-radius: 1.8mm; border: 0.25mm solid #b9d7c8; font-size: 8pt; color: #1f5d3a; font-weight: bold }
    .centered { display: flex; justify-content: center; gap: 2.5mm }
    .endline { display: flex; justify-content: flex-end; gap: 2.5mm }
    .space-between { display: flex; justify-content: space-between }
    .pill { background-color: #1e3f5e; color: #ffffff; padding: 1.6mm 2.6mm; border-radius: 1.8mm; font-size: 8pt; font-weight: bold }
    .pill-light { background-color: #e3eaf2; color: #1e3f5e; padding: 1.6mm 2.6mm; border-radius: 1.8mm; font-size: 8pt; font-weight: bold }

    .table-wrap { background-color: #ffffff; padding: 1.5mm; border: 0.25mm solid #d2dae5; border-radius: 2mm }
    th, td { padding: 1.8mm 2.2mm; font-size: 8pt }
    th { font-size: 7.5pt }

    .kpi { display: flex; gap: 2.5mm }
    .kpi-item { flex: 1; background-color: #ffffff; padding: 2.4mm 2.5mm; border-radius: 1.8mm; border: 0.25mm solid #d2dae5 }
    .kpi-item h4 { color: #5a6975; font-size: 7pt }
    .kpi-item p { font-size: 14pt; color: #0f2c4a; font-weight: bold; line-height: 1 }
    .kpi-item .small { font-size: 7pt; color: #5a6975 }

    .footer-bar { background-color: #0f2c4a; color: #ffffff; padding: 2.5mm 3mm; border-radius: 2mm }
    .footer-bar p { color: #ffffff; font-size: 7.5pt }
  </style>
</head>
<body>
  <main>
    <section class="hero">
      <div class="hero-copy">
        <p class="kicker">HTML TO PDF CAPABILITY ATLAS</p>
        <h1>Maroto HTML Demo</h1>
        <p class="subtitle">One generated PDF exercising the supported HTML tags, CSS styling, flex layout, table spans, list markers, and document composition patterns.</p>
        <p class="gap"></p>
        <div class="hero-row">
          <div class="hero-tag">Pure Go</div>
          <div class="hero-tag">No browser</div>
          <div class="hero-tag">No Node</div>
        </div>
      </div>
      <aside class="hero-side">
        <h3>At a glance</h3>
        <p class="small">Drop styled HTML into Maroto and get a paginated PDF with native rows, tables, and lists — no external renderer.</p>
      </aside>
    </section>

    <p class="gap"></p>

    <div class="stats">
      <div class="stat stat-a"><h3>Blocks</h3><p>Sections, articles, asides, headers, paragraphs, rules, quotes, code.</p></div>
      <div class="stat stat-b"><h3>Styles</h3><p>Color, size, emphasis, underline, strike, background, padding, border, radius.</p></div>
      <div class="stat stat-c"><h3>Layout</h3><p>Flex rows, weighted columns, basis sizing, gaps, alignment, stacked columns.</p></div>
      <div class="stat stat-d"><h3>Data</h3><p>Tables with header rows, zebra rows, colspan, rowspan, rich text, nested lists.</p></div>
    </div>

    <p class="gap-lg"></p>

    <h2>Typography &amp; Inline Runs</h2>
    <p>This paragraph combines <b>bold</b>, <strong>strong</strong>, <i>italic</i>, <em>emphasis</em>, <u>underline</u>, <s>strike</s>, <sub>subscript</sub>, <sup>superscript</sup>, and a live <a href="https://github.com/johnfercher/maroto">project link</a>.</p>
    <p>Paragraph-level CSS controls color and size while inline tags add emphasis, links, line breaks, and vertical alignment.</p>
    <div class="band">
      <h3>Heading scale</h3>
      <p><b>h4:</b> Fourth level heading<br><b>h5:</b> Fifth level heading<br><b>h6:</b> Sixth level heading</p>
    </div>

    <p class="gap"></p>

    <blockquote>
      Blockquotes render as padded, bordered, colored rows. Useful for callouts, excerpts, warnings, or customer notes.
    </blockquote>

    <p class="gap"></p>

    <pre>package main

func main() {
    // Preformatted blocks preserve spacing and keep code samples readable.
    println("html demo")
}</pre>

    <p class="gap-lg"></p>

    <h2>Container Styling</h2>
    <div class="band">
      <p><b>Background + radius:</b> containers can carry background color, padding, borders, and rounded corners around multiple child rows. <b>Nested text:</b> headings, paragraphs, inline formatting, and links stay inside the styled container.</p>
    </div>

    <p class="gap"></p>

    <div class="split">
      <div class="band-strong">
        <h3>Per-side borders</h3>
        <p>Different border widths and colors apply per side for section dividers and highlighted panels.</p>
      </div>
      <div class="band-amber">
        <h3>Asymmetric radius</h3>
        <p>Individual corner radii are accepted, so the shape can be asymmetric while staying PDF-friendly.</p>
      </div>
    </div>

    <p class="gap-lg"></p>

    <h2>Flex Layout Options</h2>
    <section class="cards">
      <div class="card card-blue">
        <h3>Equal columns</h3>
        <p>Three children in a row share the 12-column Maroto grid.</p>
      </div>
      <div class="card card-teal">
        <h3>Gap support</h3>
        <p>The gap reserves space between columns or falls back to visual margins.</p>
      </div>
      <div class="card card-amber">
        <h3>Auto height</h3>
        <p>Each column can contain multiple rows and rich styled content.</p>
      </div>
    </section>

    <p class="gap"></p>

    <div class="split">
      <div class="wide band">
        <h3>Weighted flex: 2</h3>
        <p>This column gets more grid space than the sibling because it uses a larger flex grow value.</p>
      </div>
      <div class="narrow band-strong">
        <h3>Weighted flex: 1</h3>
        <p>The narrower column keeps its own background, padding, and border.</p>
      </div>
    </div>

    <p class="gap"></p>

    <div class="split">
      <div class="basis-quarter">flex-basis: 25%</div>
      <div class="basis-half">flex-basis: 50%</div>
      <div class="basis-rest">flex: 1 remainder</div>
    </div>

    <p class="gap"></p>

    <div class="centered">
      <div class="pill">center</div>
      <div class="pill-light">justify-content</div>
      <div class="pill">with gap</div>
    </div>

    <p class="gap"></p>

    <div class="space-between">
      <div class="pill">space-between left</div>
      <div class="pill-light">middle</div>
      <div class="pill">right</div>
    </div>

    <p class="gap"></p>

    <div class="column-stack">
      <div class="band"><b>Column flex row 1:</b> flex-direction: column emits stacked rows.</div>
      <div class="band-strong"><b>Column flex row 2:</b> row-gap inserts spacing between stacked children.</div>
      <div class="band"><b>Column flex row 3:</b> each item keeps its own block styling.</div>
    </div>

    <p class="gap-lg"></p>

    <h2>Lists &amp; Nested Markers</h2>
    <div class="split">
      <div class="wide">
        <h3>Unordered with nested ordered list</h3>
        <ul class="offset-list">
          <li>Bullet item with <b>rich text</b></li>
          <li>Nested plan
            <ol type="a">
              <li>Lower alpha markers</li>
              <li>Inline <i>formatting</i> remains available</li>
              <li>Links render as <a href="https://example.com">hyperlinked runs</a></li>
            </ol>
          </li>
          <li>Final bullet item</li>
        </ul>
      </div>
      <div class="narrow">
        <h3>Ordered marker styles</h3>
        <ol class="circle-numbers">
          <li>Decimal circles</li>
          <li>Useful for steps</li>
          <li>Compact and clear</li>
        </ol>
        <p class="gap"></p>
        <ol class="roman-list">
          <li>Upper roman</li>
          <li>Second roman</li>
        </ol>
      </div>
    </div>

    <p class="gap-lg"></p>

    <h2>Tables, Spans &amp; Cell Styles</h2>
    <div class="table-wrap">
      <table>
        <thead>
          <tr style="background-color:#0f2c4a;color:#ffffff">
            <th>FEATURE</th>
            <th>HTML</th>
            <th>CSS</th>
            <th>RESULT</th>
          </tr>
        </thead>
        <tbody>
          <tr><td rowspan="2"><b>Rich text</b></td><td>b, i, u, s</td><td>color, size</td><td>Runs in one cell</td></tr>
          <tr style="background-color:#f5f8fb"><td>a, sub, sup</td><td>links</td><td>Hyperlinks and align</td></tr>
          <tr><td><b>Containers</b></td><td>div, section</td><td>bg, border, padding</td><td>Styled blocks</td></tr>
          <tr style="background-color:#f5f8fb"><td><b>Flex</b></td><td>display:flex</td><td>gap, flex, basis</td><td>Grid columns</td></tr>
          <tr><td><b>Lists</b></td><td>ul, ol, li</td><td>margin</td><td>Nested markers</td></tr>
          <tr style="background-color:#f5f8fb"><td><b>Images</b></td><td>img</td><td>width, height</td><td>PNG, JPG, SVG</td></tr>
          <tr style="background-color:#e7f2ea"><td><b>Coverage</b></td><td><b>all of the above</b></td><td><b>in this document</b></td><td style="color:#1d6a3f"><b>visible</b></td></tr>
        </tbody>
      </table>
    </div>

    <p class="gap-lg"></p>

    <h2>Report-Style Composition</h2>
    <header class="band">
      <h3>Quarterly Operations Snapshot</h3>
      <p>Semantic containers such as header, nav, article, aside, section, and footer fall through to block layout while preserving their children and styles.</p>
    </header>

    <p class="gap"></p>

    <div class="kpi">
      <div class="kpi-item"><h4>ON-TIME DELIVERY</h4><p>95%</p><span class="small">target 92%</span></div>
      <div class="kpi-item"><h4>ACTIVE REGIONS</h4><p>18</p><span class="small">+2 vs Q3</span></div>
      <div class="kpi-item"><h4>OPEN EXCEPTIONS</h4><p>4</p><span class="small">all triaged</span></div>
      <div class="kpi-item"><h4>NPS</h4><p>62</p><span class="small">leading peers</span></div>
    </div>

    <p class="gap"></p>

    <section class="split">
      <article class="wide band">
        <h3>Operational narrative</h3>
        <p>The HTML translator is built for structured documents: invoices, reports, statements, quotes, packing slips, certificates, and compact dashboards.</p>
        <p>Long documents paginate naturally through Maroto while native rows can still be mixed with HTML-generated rows on the same page.</p>
      </article>
      <aside class="narrow band-strong">
        <h3>Highlights</h3>
        <p><b>4</b> regions improved<br><b>0</b> SLA breaches<br><b>1</b> new partner</p>
      </aside>
    </section>

    <p class="gap"></p>

    <div class="table-wrap">
      <table>
        <thead>
          <tr style="background-color:#28515f;color:#ffffff"><th>REGION</th><th>OWNER</th><th>STATUS</th><th>NEXT STEP</th></tr>
        </thead>
        <tbody>
          <tr><td>North</td><td>Avery</td><td style="color:#1d6a3f"><b>Green</b></td><td>Renew supplier schedule</td></tr>
          <tr style="background-color:#f5f8fb"><td>South</td><td>Blair</td><td style="color:#8a5a18"><b>Watch</b></td><td>Confirm warehouse capacity</td></tr>
          <tr><td>West</td><td>Casey</td><td style="color:#b63a26"><b>Red</b></td><td>Escalate shipping exception</td></tr>
          <tr style="background-color:#f5f8fb"><td>International</td><td>Drew</td><td style="color:#1d6a3f"><b>Green</b></td><td>Close customs checklist</td></tr>
        </tbody>
      </table>
    </div>

    <p class="gap-lg"></p>

    <h2 id="modern-features">Modern HTML + CSS</h2>
    <p>The features below were added in the engine + styling extension and are
       all parsed by maroto's pure-Go pipeline — no browser, no JS.</p>

    <h3>Modern colours and opacity</h3>
    <p>
      <span style="background-color: rgba(255, 0, 0, 0.3); padding: 0.5mm 2mm">rgba</span>
      <span style="background-color: hsl(210, 80%, 90%); padding: 0.5mm 2mm">hsl</span>
      <span style="background-color: #00ff8080; padding: 0.5mm 2mm">#rrggbbaa</span>
      <span style="background-color: tomato; color: white; padding: 0.5mm 2mm">tomato</span>
      <span style="background-color: rebeccapurple; color: white; padding: 0.5mm 2mm">rebeccapurple</span>
    </p>

    <h3>Typography polish</h3>
    <p style="text-transform: uppercase; letter-spacing: 0.4pt">letter-spaced uppercase</p>
    <p style="text-transform: capitalize">capitalized words for headings or labels</p>
    <p style="text-indent: 8mm">Text-indent shifts the leading edge of a
       paragraph (whole-paragraph in v1, first-line indent is deferred).</p>

    <h3>Expanded tag coverage</h3>
    <p>Highlight a <mark>marked phrase</mark>, press <kbd>Ctrl+Shift+P</kbd>,
       call <code>doSomething()</code>, refer to <abbr title="HyperText Markup Language">HTML</abbr>,
       quote <q>like this</q>, cite <cite>The Pragmatic Programmer</cite>,
       use a variable <var>x</var>, show <samp>sample output</samp>, and
       a <small>smaller aside</small>.</p>

    <dl>
      <dt>Definition list</dt><dd>Pairs of term and description.</dd>
      <dt>Details / summary</dt><dd>Always-expanded styled block.</dd>
    </dl>

    <details>
      <summary>Accordion-style summary</summary>
      <p>PDF has no toggle so &lt;details&gt; always renders open with a bold summary.</p>
    </details>

    <hr style="border-top: 1.5pt dashed #888">

    <h3>Selectors and zebra rows</h3>
    <style>
      .zebra tr:nth-child(even) td { background-color: #f4f6f8 }
      .zebra tr:first-child td { font-weight: bold }
    </style>
    <div class="table-wrap">
      <table class="zebra">
        <tr><td>Header A</td><td>Header B</td><td>Header C</td></tr>
        <tr><td>Row 1</td><td>data</td><td>data</td></tr>
        <tr><td>Row 2</td><td>data</td><td>data</td></tr>
        <tr><td>Row 3</td><td>data</td><td>data</td></tr>
        <tr><td>Row 4</td><td>data</td><td>data</td></tr>
      </table>
    </div>

    <h3>CSS variables and calc()</h3>
    <style>
      :root { --brand: #0f2c4a; --brand-soft: #eef3f9 }
      .var-demo { background-color: var(--brand-soft); color: var(--brand); padding: 2mm; border-radius: 1.5mm; width: calc(100% - 20mm) }
    </style>
    <div class="var-demo">
      <p>This box's background, text colour, and width are all driven by CSS
         variables and a calc(100% - 20mm) width.</p>
    </div>

    <h3>Internal anchors</h3>
    <p><a href="#modern-features">Jump back to the Modern features section</a>
       — clickable internal link backed by gofpdf's named destinations.</p>

    <p class="gap-lg"></p>

    <h2>Best Practices</h2>
    <section class="cards">
      <div class="card card-blue">
        <h3>Authoring</h3>
        <p>Use HTML for document bodies; keep recurring page furniture in Maroto headers or native rows.</p>
      </div>
      <div class="card card-teal">
        <h3>Assets</h3>
        <p>Resolve images via a scoped base directory or a custom resolver so output stays deterministic.</p>
      </div>
      <div class="card card-amber">
        <h3>Pagination</h3>
        <p>Keep highly styled containers compact; backgrounds and borders are kept together intentionally.</p>
      </div>
    </section>

    <p class="gap"></p>

    <div class="split">
      <div class="wide band">
        <h3>Recommended document recipe</h3>
        <ol class="circle-numbers">
          <li>Design the body as semantic HTML with small reusable classes</li>
          <li>Use flex for horizontal groups and tables for dense data</li>
          <li>Use native Maroto rows for fixed repeating chrome</li>
          <li>Generate and inspect the resulting PDF in tests or examples</li>
        </ol>
      </div>
      <div class="narrow band-amber">
        <h3>Good fit</h3>
        <ul>
          <li>Invoices</li>
          <li>Statements</li>
          <li>Reports</li>
          <li>Certificates</li>
          <li>Op summaries</li>
        </ul>
      </div>
    </div>

    <p class="gap"></p>

    <div class="footer-bar">
      <p><b>Maroto HTML —</b> a single PDF showcasing styled HTML, flex layout, lists, and tables. Generated entirely in pure Go.</p>
    </div>
  </main>
</body>
</html>`

func main() {
	cfg := config.NewBuilder().
		WithLeftMargin(20).
		WithTopMargin(15).
		WithRightMargin(20).
		WithBottomMargin(15).
		Build()

	m := maroto.New(cfg)
	m.AddRows(buildHeader()...)
	cfgWidth := cfg.Dimensions.Width - cfg.Margins.Left - cfg.Margins.Right
	rows, err := html.FromString(body,
		html.WithGridSize(cfg.MaxGridSize),
		html.WithContentWidth(cfgWidth),
		html.WithImageBaseDir("cmd/html-demo/assets"),
		html.WithStylesheetBaseDir("cmd/html-demo/assets"),
	)
	if err != nil {
		log.Fatalf("parse html: %v", err)
	}
	m.AddRows(rows...)

	doc, err := m.Generate()
	if err != nil {
		log.Fatalf("generate: %v", err)
	}

	out := "/Users/avdo/maroto/test/output/html-demo.pdf"
	if err := doc.Save(out); err != nil {
		log.Fatalf("save: %v", err)
	}
	info, _ := os.Stat(out)
	fmt.Printf("Wrote %s (%d bytes)\n", out, info.Size())
}

// buildHeader returns the rows that render on every page above the body.
func buildHeader() []core.Row {
	dark := &props.Color{Red: 26, Green: 62, Blue: 114}
	muted := &props.Color{Red: 120, Green: 120, Blue: 120}

	titleRow := row.New(10).Add(
		col.New(8).Add(text.New("MAROTO HTML PDF DEMO", props.Text{
			Family: fontfamily.Helvetica,
			Style:  fontstyle.Bold,
			Size:   14,
			Color:  dark,
		})),
		col.New(4).Add(text.New("html-demo@maroto.example", props.Text{
			Family: fontfamily.Helvetica,
			Size:   9,
			Align:  align.Right,
			Color:  muted,
		})),
	)

	dividerRow := row.New(2).Add(col.New(12).Add(line.New(props.Line{
		Color:     dark,
		Thickness: 0.6,
	})))

	return []core.Row{titleRow, dividerRow}
}
