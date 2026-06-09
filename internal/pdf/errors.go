package pdf

import (
	"errors"
	"fmt"
)

const (
	displayModeDefault = "default"

	blendModeNormal = "Normal"

	fontTypeTrueType   = "TrueType"
	fontTypeUTF8       = "UTF8"
	fontTypeUTF8Bitmap = "UTF8Bitmap"

	imageTypeJPG = "jpg"
	imageTypePNG = "png"

	colorSpaceIndexed = "Indexed"
)

var (
	errAlphaOutOfRange             = errors.New("alpha value (0.0 - 1.0) is out of range")
	errClipEndSequence             = errors.New("error attempting to end clip operation out of sequence")
	errClipProcedureOpen           = errors.New("clip procedure must be explicitly ended")
	errDynamicPDF                  = errors.New("")
	errFontNotSet                  = errors.New("font has not been set; unable to render text")
	errFontReader                  = errors.New("font reader error")
	errIncorrectLayoutDisplayMode  = errors.New("incorrect layout display mode")
	errIncorrectOrientation        = errors.New("incorrect orientation")
	errIncorrectUnit               = errors.New("incorrect unit")
	errIncorrectZoomDisplayMode    = errors.New("incorrect zoom display mode")
	errInvalidPageBoxType          = errors.New("invalid page box type")
	errMissingImageType            = errors.New("image type should be specified if reading from custom reader")
	errMissingTemplatePageZero     = errors.New("pages start at 1 No template will have a page 0")
	errObjectNumberOutOfRange      = errors.New("pdf object number is out of range")
	errTemplatePageMissing         = errors.New("template does not have page")
	errTransformationProcedureOpen = errors.New("transformation procedure must be explicitly ended")
	errTrueTypeCollectionEmpty     = errors.New("TrueType collection has no fonts")
	errTrueTypeCollectionOffset    = errors.New("TrueType collection font offset is invalid")
	errUnsupportedCFFFont          = errors.New("unsupported OpenType CFF font")
	errUnsupportedFontType         = errors.New("unsupported font type")
	errUnsupportedImageType        = errors.New("unsupported image type")
	errUnsupportedJPEGColorSpace   = errors.New("image JPEG buffer has unsupported color space")
	errUndefinedFont               = errors.New("undefined font")
	errUnrecognizedBlendMode       = errors.New("unrecognized blend mode")
	errUnknownPageSize             = errors.New("unknown page size")
	errUntypedImageFile            = errors.New("image file has no extension and no type was specified")
	errUTF8Font                    = errors.New("utf8 font error")
	errUnexpectedTrueTypeCodeType  = errors.New("not a TrueType font")
)

func staticErrorf(base error, format string, args ...any) error {
	return fmt.Errorf("%w: %s", base, fmt.Sprintf(format, args...))
}

func dynamicErrorf(format string, args ...any) error {
	return fmt.Errorf("%w%s", errDynamicPDF, fmt.Sprintf(format, args...))
}

func checkedUint16(n int) (uint16, bool) {
	if n < 0 || n > 0xffff {
		return 0, false
	}
	return uint16(n), true // #nosec G115 -- guarded by the bounds check above.
}

func checkedUint32(n int) (uint32, bool) {
	if n < 0 || uint64(n) > uint64(^uint32(0)) {
		return 0, false
	}
	return uint32(n), true // #nosec G115 -- guarded by the bounds check above.
}

func checkedByte(n int) (byte, bool) {
	if n < 0 || n > 0xff {
		return 0, false
	}
	return byte(n), true // #nosec G115 -- guarded by the bounds check above.
}
