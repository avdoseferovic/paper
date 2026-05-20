// html-demo generates a polished invoice PDF that combines Maroto's native
// header/footer/page-number support with an HTML body parsed via pkg/html.
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
  <style>
    h1 { color: #0f2c5e; font-size: 26pt; font-weight: bold }
    h2 { color: #1a3e72; font-size: 12pt; font-weight: bold }
    h3 { color: #1a3e72; font-size: 10pt; font-weight: bold }
    .muted   { color: #6a6a6a }
    .accent  { color: #c0392b }
    .small   { font-size: 9pt }
    .subtitle { color: #6a6a6a; font-size: 10pt }

    .parties { display: flex; gap: 5mm }
    .card-blue  { background-color: #eaf1fb; padding: 5mm; border-radius: 3mm }
    .card-teal  { background-color: #e6f4f1; padding: 5mm; border-radius: 3mm }
    .card-amber { background-color: #fbf3e3; padding: 5mm; border-radius: 3mm }

    .actions { display: flex; gap: 4mm }
    .footer-band {
      background-color: #f4f6fa;
      padding: 4mm;
      border-radius: 2mm;
    }
  </style>
</head>
<body>
  <img src="icon.svg" width="14mm" height="14mm" alt="logo">
  <h1>INVOICE #2026-0042</h1>
  <p class="subtitle">Issued 19 May 2026 &nbsp;&bull;&nbsp; Payment due within 30 days</p>

  <div class="parties">
    <div class="card-blue">
      <h3>BILL TO</h3>
      <p><b>Acme Corporation</b><br>
      123 Industrial Way<br>
      Springfield, IL 62701<br>
      <span class="muted">accounts@acme.example</span></p>
    </div>
    <div class="card-teal">
      <h3>SHIP TO</h3>
      <p><b>Acme Warehouse #4</b><br>
      890 Logistics Blvd<br>
      Springfield, IL 62702<br>
      <span class="muted">receiving@acme.example</span></p>
    </div>
    <div class="card-amber">
      <h3>PAYMENT</h3>
      <p>Net 30 days<br>
      <b>First National Bank</b><br>
      Account 0123-4567<br>
      <span class="muted">SWIFT FNBKUS33</span></p>
    </div>
  </div>

  <p>&nbsp;</p>

  <h2 class="title-band">SUMMARY</h2>
  <p>Thank you for your <i>continued</i> business this quarter. Below is the itemised
  breakdown of services and goods provided.</p>

  <table>
    <thead>
      <tr style="background-color:#1a3e72;color:#ffffff">
        <th><b>ITEM</b></th>
        <th><b>QUANTITY</b></th>
        <th><b>UNIT PRICE</b></th>
        <th><b>TOTAL</b></th>
      </tr>
    </thead>
    <tbody>
      <tr><td>Widget Pro (premium edition)</td><td>3</td><td>$10.00</td><td>$30.00</td></tr>
      <tr style="background-color:#f7f9fc"><td>Gadget Plus (extended warranty)</td><td>2</td><td>$25.00</td><td>$50.00</td></tr>
      <tr><td>Onboarding services</td><td>4 hours</td><td>$75.00</td><td>$300.00</td></tr>
      <tr style="background-color:#f7f9fc"><td>Travel reimbursement</td><td>1</td><td>$120.00</td><td>$120.00</td></tr>
      <tr><td colspan="3">Subtotal</td><td>$500.00</td></tr>
      <tr><td colspan="3">Tax (8%)</td><td>$40.00</td></tr>
      <tr><td colspan="3"><b>Amount due</b></td><td><b>$540.00</b></td></tr>
      <tr style="background-color:#fbecec"><td colspan="3"><b class="accent">Amount due (final)</b></td><td><b class="accent">$540.00 USD</b></td></tr>
    </tbody>
  </table>

  <p>&nbsp;</p>

  <div class="actions">
    <div>
      <h2 class="title-band">PAYMENT INSTRUCTIONS</h2>
      <ol class="circle-numbers">
        <li>Wire transfer to <b>Account 0123-4567</b> at First National Bank</li>
        <li>Reference your invoice number <b>#2026-0042</b> in the memo</li>
        <li>Confirm via email to <a href="mailto:billing@maroto.example">billing@maroto.example</a></li>
      </ol>
    </div>
    <div>
      <h2 class="title-band">NOTES</h2>
      <ul>
        <li>All prices are quoted in USD</li>
        <li>Late payments incur a <b>1.5% monthly</b> service charge</li>
        <li>For questions, reach out to <a href="https://example.com/support">our support team</a></li>
      </ul>
    </div>
  </div>

  <p>&nbsp;</p>

  <div class="footer-band">
    <p class="small" style="text-align:center;color:#1a3e72"><i>Thank you for your business!</i></p>
  </div>
</body>
</html>`

func main() {
	cfg := config.NewBuilder().
		WithPageNumber(props.PageNumber{
			Pattern: "Page {current} of {total}",
			Place:   props.RightBottom,
			Family:  fontfamily.Helvetica,
			Size:    9,
			Color:   &props.Color{Red: 120, Green: 120, Blue: 120},
		}).
		WithLeftMargin(20).
		WithTopMargin(15).
		WithRightMargin(20).
		WithBottomMargin(15).
		Build()

	m := maroto.New(cfg)

	if err := m.RegisterHeader(buildHeader()...); err != nil {
		log.Fatalf("register header: %v", err)
	}
	if err := m.RegisterFooter(buildFooter()...); err != nil {
		log.Fatalf("register footer: %v", err)
	}

	cfgWidth := cfg.Dimensions.Width - cfg.Margins.Left - cfg.Margins.Right
	rows, err := html.FromString(body,
		html.WithGridSize(cfg.MaxGridSize),
		html.WithContentWidth(cfgWidth),
		html.WithImageBaseDir("cmd/html-demo/assets"),
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
		col.New(8).Add(text.New("MAROTO INVOICE SYSTEM", props.Text{
			Family: fontfamily.Helvetica,
			Style:  fontstyle.Bold,
			Size:   14,
			Color:  dark,
		})),
		col.New(4).Add(text.New("invoices@maroto.example", props.Text{
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

// buildFooter returns the rows that render on every page below the body.
func buildFooter() []core.Row {
	muted := &props.Color{Red: 120, Green: 120, Blue: 120}

	dividerRow := row.New(2).Add(col.New(12).Add(line.New(props.Line{
		Color:     muted,
		Thickness: 0.3,
	})))

	infoRow := row.New(6).Add(
		col.New(8).Add(text.New("Maroto Invoice System — confidential", props.Text{
			Family: fontfamily.Helvetica,
			Size:   8,
			Color:  muted,
		})),
		// the page-number slot is filled by config.WithPageNumber on the right
		col.New(4).Add(text.New("", props.Text{})),
	)

	return []core.Row{dividerRow, infoRow}
}
