# Protection

`WithProtection` applies PDF password protection and permission restrictions. You can require a password to open the document, restrict printing, copying, or modifying, or combine multiple restrictions.

## Permission flags (`protection.Type`)

| Constant | Restricted action |
|----------|-------------------|
| `protection.None` | No restrictions |
| `protection.Print` | Printing |
| `protection.Modify` | Document modification |
| `protection.Copy` | Copying text and graphics |
| `protection.AnnotForms` | Annotating and filling forms |

Flags can be combined with `|`: `protection.Print | protection.Copy` restricts both printing and copying.

## Encryption algorithms

| Constant | Behavior |
|----------|----------|
| `protection.RC4128` | Legacy RC4 protection. This is the compatibility default. |
| `protection.AES128` | AESV2 / PDF standard security handler revision 4 with 128-bit keys. |

Select AES-128 with:

```go
cfg := config.NewBuilder().
	WithProtection(protection.Copy, "user", "owner").
	WithProtectionAlgorithm(protection.AES128).
	Build()
```

## Usage notes

- An empty string for either password disables that password. A document with only an owner password can be opened without a password but restrictions apply to regular users.
- Protection uses the PDF standard security handler. It deters casual copying or printing, but is not confidentiality-grade encryption.
- RC4 remains the default for compatibility. Prefer `protection.AES128` for newly generated protected documents.
- AES-128 protected output uses random IVs, so encrypted PDF bytes are not deterministic across runs.
- AES-128 protected output is verified with `pdfcpu validate` during release checks.
- Paper's built-in PDF merger accepts unencrypted PDFs only. `Document.Merge`
  returns a merge error for RC4- or AES-protected input.
- For confidentiality, encrypt the file at rest or in transit.

## GoDoc
* [builder : WithProtection](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/config#CfgBuilder.WithProtection)
* [builder : WithProtectionAlgorithm](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/config#CfgBuilder.WithProtectionAlgorithm)
* [protection : Type](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/consts/protection)
* [protection : Encryption](https://pkg.go.dev/github.com/avdoseferovic/paper/pkg/consts/protection#Encryption)

## Code Example
[filename](../assets/examples/protection/main.go ':include :type=code')

## PDF Generated
```pdf
	assets/pdf/protection.pdf
```
## Time Execution
[filename](../assets/text/protection.txt  ':include :type=code')

## Test File
[filename](https://raw.githubusercontent.com/avdoseferovic/paper/master/test/paper/examples/protection.json  ':include :type=code')
