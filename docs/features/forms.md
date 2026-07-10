# AcroForms

`pkg/forms` creates interactive PDF form fields in generated documents. These
are PDF AcroForm fields, not just visual components: viewers can edit text
fields, toggle buttons, choose list values, and prepare unsigned signature
fields.

```go
form := forms.NewAcroForm().
	Add(forms.NewTextField("name", [4]float64{72, 700, 300, 720}, 0).SetValue("Ada")).
	Add(forms.NewCheckbox("agree", [4]float64{72, 670, 92, 690}, 0, false)).
	Add(forms.NewDropdown("role", [4]float64{72, 640, 250, 660}, 0,
		[]string{"Developer", "Designer", "Manager"}))

doc := paper.New(config.NewBuilder().WithAcroForm(form).Build())
doc.AddAutoRow(col.New(12).Add(text.New("Interactive form")))
pdf, err := doc.Generate(context.Background())
```

Fields use PDF coordinates in points with page indexes starting at zero. For
runtime configuration, call `(*Paper).SetAcroForm(form)` before `Generate`.

Supported fields:

- `NewTextField`, `NewMultilineTextField`, `NewPasswordField`
- `NewCheckbox`
- `NewRadioGroup`
- `NewDropdown`, `NewListBox`
- `NewSignatureField` for unsigned signature widgets

## Filling Existing Forms

`NewFormFiller` works with a parsed `pkg/reader` PDF and appends incremental
updates for changed field values:

```go
r, err := reader.Load("form.pdf")
if err != nil {
	return err
}

filler := forms.NewFormFiller(r)
names, err := filler.FieldNames()
if err != nil {
	return err
}

err = filler.SetValue("name", "Grace Hopper")
if err != nil {
	return err
}
err = filler.SetCheckbox("agree", true)
if err != nil {
	return err
}

err = filler.SaveTo("filled.pdf")
```

Filled forms can also be flattened into ordinary page content for supported
field types:

```go
err = filler.Flatten()
if err != nil {
	return err
}
err = filler.SaveTo("flattened.pdf")
```

Current form filling scope:

- `FieldNames`
- `GetValue`
- `SetValue` for text and choice fields
- `SetCheckbox`
- `Flatten` for simple text, choice, and checkbox fields in classic
  Paper-style AcroForms
- `Bytes` and `SaveTo`
- incremental field-object updates that preserve the original PDF revision
- flattening rewrites a fresh classic-xref PDF without `/AcroForm` or widget
  annotations

Limitations: full appearance-stream flattening, radio appearance rendering,
signature widgets, and arbitrary third-party form appearance regeneration are
not implemented yet. Complex xref-stream/object-stream PDFs depend on future
reader support.
