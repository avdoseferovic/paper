package consts

// BarcodeType is the representation of a barcode symbology.
type BarcodeType string

const (
	// BarcodeCode128 represents the Code 128 symbology.
	BarcodeCode128 BarcodeType = "code128"
	// BarcodeEAN represents the EAN symbology.
	BarcodeEAN BarcodeType = "ean"
)
