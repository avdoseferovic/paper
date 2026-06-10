package protection

// Encryption selects the PDF standard security handler encryption algorithm.
type Encryption byte

const (
	// RC4128 keeps the legacy RC4 protection behavior and remains the default
	// for compatibility until a future major release changes it.
	RC4128 Encryption = iota
	// AES128 selects AESV2, the PDF standard security handler revision 4
	// algorithm using AES-128-CBC for strings and streams.
	AES128
)
