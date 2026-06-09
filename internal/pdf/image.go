package pdf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"sort"
	"strings"
)

// ImageTypeFromMime returns the image type used in various image-related
// functions (for example, Image()) that is associated with the specified MIME
// type. For example, "jpg" is returned if mimeStr is "image/jpeg". An error is
// set if the specified MIME type is not supported.
func (f *PDF) ImageTypeFromMime(mimeStr string) string {
	switch mimeStr {
	case "image/png":
		return imageTypePNG
	case "image/jpg":
		return imageTypeJPG
	case "image/jpeg":
		return imageTypeJPG
	case "image/gif":
		return "gif"
	default:
		f.SetErrorf("unsupported image type: %s", mimeStr)
		return ""
	}
}

func (f *PDF) imageOut(info *ImageInfoType, x, y, w, h float64, allowNegativeX, flow bool, link int, linkStr string) {
	if w == 0 && h == 0 {
		w = -96
		h = -96
	}
	if w == -1 {
		w = -info.dpi
	}
	if h == -1 {
		h = -info.dpi
	}
	if w < 0 {
		w = -info.w * 72.0 / w / f.k
	}
	if h < 0 {
		h = -info.h * 72.0 / h / f.k
	}
	if w == 0 {
		w = h * info.w / info.h
	}
	if h == 0 {
		h = w * info.h / info.w
	}

	if flow {
		if f.y+h > f.pageBreakTrigger && !f.inHeader && !f.inFooter && f.acceptPageBreak() {
			x2 := f.x
			f.AddPageFormat(f.curOrientation, f.curPageSize)
			if f.err != nil {
				return
			}
			f.x = x2
		}
		y = f.y
		f.y += h
	}
	if !allowNegativeX {
		if x < 0 {
			x = f.x
		}
	}

	f.outf("q %.5f 0 0 %.5f %.5f %.5f cm /I%s Do Q", w*f.k, h*f.k, x*f.k, (f.h-(y+h))*f.k, info.i)
	if link > 0 || len(linkStr) > 0 {
		f.newLink(x, y, w, h, link, linkStr)
	}
}

// Image puts a JPEG, PNG or GIF image in the current page.
//
// Deprecated: use ImageOptions for details on the behavior of arguments.
func (f *PDF) Image(imageNameStr string, x, y, w, h float64, flow bool, tp string, link int, linkStr string) {
	options := ImageOptions{
		ReadDpi:   false,
		ImageType: tp,
	}
	f.ImageOptions(imageNameStr, x, y, w, h, flow, options, link, linkStr)
}

// ImageOptions puts a JPEG, PNG or GIF image in the current page. The size it
// will take on the page can be specified in different ways. If both w and h
// are 0, the image is rendered at 96 dpi. If either w or h is zero, it will be
// calculated from the other dimension so that the aspect ratio is maintained.
// If w and/or h are -1, the dpi for that dimension will be read from the
// ImageInfoType object. PNG files can contain dpi information, and if present,
// this information will be populated in the ImageInfoType object and used in
// Width, Height, and Extent calculations. Otherwise, the SetDpi function can
// be used to change the dpi from the default of 72.
//
// If w and h are any other negative value, their absolute values
// indicate their dpi extents.
//
// Supported JPEG formats are 24 bit, 32 bit and gray scale. Supported PNG
// formats are 24 bit, indexed color, and 8 bit indexed gray scale. If a GIF
// image is animated, only the first frame is rendered. Transparency is
// supported. It is possible to put a link on the image.
//
// imageNameStr may be the name of an image as registered with a call to either
// RegisterImageReader() or RegisterImage(). In the first case, the image is
// loaded using an io.Reader. This is generally useful when the image is
// obtained from some other means than as a disk-based file. In the second
// case, the image is loaded as a file. Alternatively, imageNameStr may
// directly specify a sufficiently qualified filename.
//
// However the image is loaded, if it is used more than once only one copy is
// embedded in the file.
//
// If x is negative, the current abscissa is used.
//
// If flow is true, the current y value is advanced after placing the image and
// a page break may be made if necessary.
//
// If link refers to an internal page anchor (that is, it is non-zero; see
// AddLink()), the image will be a clickable internal link. Otherwise, if
// linkStr specifies a URL, the image will be a clickable external link.
func (f *PDF) ImageOptions(imageNameStr string, x, y, w, h float64, flow bool, options ImageOptions, link int, linkStr string) {
	if f.err != nil {
		return
	}
	info := f.RegisterImageOptions(imageNameStr, options)
	if f.err != nil {
		return
	}
	f.imageOut(info, x, y, w, h, options.AllowNegativePosition, flow, link, linkStr)
}

// RegisterImageReader registers an image, reading it from Reader r, adding it
// to the PDF file but not adding it to the page.
//
// This function is now deprecated in favor of RegisterImageOptionsReader
func (f *PDF) RegisterImageReader(imgName, tp string, r io.Reader) *ImageInfoType {
	options := ImageOptions{
		ReadDpi:   false,
		ImageType: tp,
	}
	return f.RegisterImageOptionsReader(imgName, options, r)
}

// ImageOptions provides a place to hang any options we want to use while
// parsing an image.
//
// ImageType's possible values are (case insensitive):
// "JPG", "JPEG", "PNG" and "GIF". If empty, the type is inferred from
// the file extension.
//
// ReadDpi defines whether to attempt to automatically read the image
// dpi information from the image file. Normally, this should be set
// to true (understanding that not all images will have this info
// available). However, for backwards compatibility with previous
// versions of the API, it defaults to false.
//
// AllowNegativePosition can be set to true in order to prevent the default
// coercion of negative x values to the current x position.
type ImageOptions struct {
	ImageType             string
	ReadDpi               bool
	AllowNegativePosition bool
}

// RegisterImageOptionsReader registers an image, reading it from Reader r, adding it
// to the PDF file but not adding it to the page. Use Image() with the same
// name to add the image to the page. Note that tp should be specified in this
// case.
//
// See Image() for restrictions on the image and the options parameters.
func (f *PDF) RegisterImageOptionsReader(imgName string, options ImageOptions, r io.Reader) *ImageInfoType {
	if f.err != nil {
		return nil
	}
	info, ok := f.images[imgName]
	if ok {
		return info
	}

	if options.ImageType == "" {
		f.err = errMissingImageType
		return nil
	}
	options.ImageType = strings.ToLower(options.ImageType)
	if options.ImageType == "jpeg" {
		options.ImageType = imageTypeJPG
	}
	switch options.ImageType {
	case imageTypeJPG:
		info = f.parsejpg(r)
	case imageTypePNG:
		info = f.parsepng(r, options.ReadDpi)
	case "gif":
		info = f.parsegif(r)
	default:
		f.err = fmt.Errorf("%w: %s", errUnsupportedImageType, options.ImageType)
	}
	if f.err != nil {
		return nil
	}

	if info.i, f.err = generateImageID(info); f.err != nil {
		return nil
	}
	f.images[imgName] = info

	return info
}

// RegisterImage registers an image, adding it to the PDF file but not adding
// it to the page. Use Image() with the same filename to add the image to the
// page. Note that Image() calls this function, so this function is only
// necessary if you need information about the image before placing it.
//
// This function is now deprecated in favor of RegisterImageOptions.
// See Image() for restrictions on the image and the "tp" parameters.
func (f *PDF) RegisterImage(fileStr, tp string) *ImageInfoType {
	options := ImageOptions{
		ReadDpi:   false,
		ImageType: tp,
	}
	return f.RegisterImageOptions(fileStr, options)
}

// RegisterImageOptions registers an image, adding it to the PDF file but not
// adding it to the page. Use Image() with the same filename to add the image
// to the page. Note that Image() calls this function, so this function is only
// necessary if you need information about the image before placing it. See
// Image() for restrictions on the image and the "tp" parameters.
func (f *PDF) RegisterImageOptions(fileStr string, options ImageOptions) *ImageInfoType {
	info, ok := f.images[fileStr]
	if ok {
		return info
	}

	file, err := os.Open(fileStr)
	if err != nil {
		f.err = err
		return nil
	}

	if options.ImageType == "" {
		pos := strings.LastIndex(fileStr, ".")
		if pos < 0 {
			f.err = fmt.Errorf("%w: %s", errUntypedImageFile, fileStr)
			_ = file.Close()
			return nil
		}
		options.ImageType = fileStr[pos+1:]
	}

	info = f.RegisterImageOptionsReader(fileStr, options, file)
	closeErr := file.Close()
	if closeErr != nil && f.err == nil {
		f.err = closeErr
		return nil
	}
	return info
}

// GetImageInfo returns information about the registered image specified by
// imageStr. If the image has not been registered, nil is returned. The
// internal error is not modified by this method.
func (f *PDF) GetImageInfo(imageStr string) *ImageInfoType {
	return f.images[imageStr]
}

func (f *PDF) newImageInfo() *ImageInfoType {
	return &ImageInfoType{scale: f.k, dpi: 72}
}

// parsejpg extracts info from io.Reader with JPEG data
// Thank you, Bruno Michel, for providing this code.
func (f *PDF) parsejpg(r io.Reader) *ImageInfoType {
	info := f.newImageInfo()
	var (
		data bytes.Buffer
		err  error
	)
	_, err = data.ReadFrom(r)
	if err != nil {
		f.err = fmt.Errorf("read JPEG data: %w", err)
		return info
	}
	info.data = data.Bytes()

	config, err := jpeg.DecodeConfig(bytes.NewReader(info.data))
	if err != nil {
		f.err = fmt.Errorf("decode JPEG config: %w", err)
		return info
	}
	info.w = float64(config.Width)
	info.h = float64(config.Height)
	info.f = "DCTDecode"
	info.bpc = 8
	switch config.ColorModel {
	case color.GrayModel:
		info.cs = "DeviceGray"
	case color.YCbCrModel:
		info.cs = "DeviceRGB"
	case color.CMYKModel:
		info.cs = "DeviceCMYK"
	default:
		f.err = fmt.Errorf("%w: %v", errUnsupportedJPEGColorSpace, config.ColorModel)
		return info
	}
	return info
}

// parsepng extracts info from a PNG data
func (f *PDF) parsepng(r io.Reader, readdpi bool) *ImageInfoType {
	buf, err := bufferFromReader(r)
	if err != nil {
		f.err = err
		return nil
	}
	return f.parsepngstream(buf, readdpi)
}

func (f *PDF) readBeInt32(r io.Reader) int32 {
	var val int32
	err := binary.Read(r, binary.BigEndian, &val)
	if err != nil && !errors.Is(err, io.EOF) {
		f.err = err
	}
	return val
}

func (f *PDF) readByte(r io.Reader) byte {
	var val byte
	err := binary.Read(r, binary.BigEndian, &val)
	if err != nil {
		f.err = err
	}
	return val
}

// parsegif extracts info from a GIF data (via PNG conversion)
func (f *PDF) parsegif(r io.Reader) *ImageInfoType {
	data, err := bufferFromReader(r)
	if err != nil {
		f.err = err
		return nil
	}
	var img image.Image
	img, err = gif.Decode(data)
	if err != nil {
		f.err = err
		return nil
	}
	pngBuf := new(bytes.Buffer)
	err = png.Encode(pngBuf, img)
	if err != nil {
		f.err = err
		return nil
	}
	return f.parsepngstream(pngBuf, false)
}

func (f *PDF) putimages() {
	keyList := make([]string, 0, len(f.images))
	var key string
	for key = range f.images {
		keyList = append(keyList, key)
	}

	if f.catalogSort {
		sort.SliceStable(keyList, func(i, j int) bool { return f.images[keyList[i]].w < f.images[keyList[j]].w })
	}

	insertedImages := map[string]int{}

	for _, key = range keyList {
		image := f.images[key]

		insertedImageObjN, isFound := insertedImages[image.i]

		if isFound {
			image.n = insertedImageObjN
		} else {
			f.putimage(image)
			insertedImages[image.i] = image.n
		}
	}
}

func (f *PDF) putimage(info *ImageInfoType) {
	f.newobj()
	info.n = f.n
	f.out("<</Type /XObject")
	f.out("/Subtype /Image")
	f.outf("/Width %d", int(info.w))
	f.outf("/Height %d", int(info.h))
	if info.cs == colorSpaceIndexed {
		f.outf("/ColorSpace [/Indexed /DeviceRGB %d %d 0 R]", len(info.pal)/3-1, f.n+1)
	} else {
		f.outf("/ColorSpace /%s", info.cs)
		if info.cs == "DeviceCMYK" {
			f.out("/Decode [1 0 1 0 1 0 1 0]")
		}
	}
	f.outf("/BitsPerComponent %d", info.bpc)
	if len(info.f) > 0 {
		f.outf("/Filter /%s", info.f)
	}
	if len(info.dp) > 0 {
		f.outf("/DecodeParms <<%s>>", info.dp)
	}
	if len(info.trns) > 0 {
		var trns fmtBuffer
		for _, v := range info.trns {
			trns.printf("%d %d ", v, v)
		}
		f.outf("/Mask [%s]", trns.String())
	}
	if info.smask != nil {
		f.outf("/SMask %d 0 R", f.n+1)
	}
	f.outf("/Length %d>>", len(info.data))
	f.putstream(info.data)
	f.out("endobj")

	if len(info.smask) > 0 {
		smask := &ImageInfoType{
			w:     info.w,
			h:     info.h,
			cs:    "DeviceGray",
			bpc:   8,
			f:     info.f,
			dp:    sprintf("/Predictor 15 /Colors 1 /BitsPerComponent 8 /Columns %d", int(info.w)),
			data:  info.smask,
			scale: f.k,
		}
		f.putimage(smask)
	}

	if info.cs == colorSpaceIndexed {
		f.newobj()
		if f.compress {
			pal := sliceCompress(info.pal)
			f.outf("<</Filter /FlateDecode /Length %d>>", len(pal))
			f.putstream(pal)
		} else {
			f.outf("<</Length %d>>", len(info.pal))
			f.putstream(info.pal)
		}
		f.out("endobj")
	}
}

func (f *PDF) pngColorSpace(ct byte) (string, int) {
	colorVal := 1
	switch ct {
	case 0, 4:
		return "DeviceGray", colorVal
	case 2, 6:
		colspace := "DeviceRGB"
		colorVal = 3
		return colspace, colorVal
	case 3:
		return colorSpaceIndexed, colorVal
	default:
		f.SetErrorf("unknown color type in PNG buffer: %d", ct)
		return "", colorVal
	}
}

func (f *PDF) parsepngstream(buf *bytes.Buffer, readdpi bool) *ImageInfoType {
	info := f.newImageInfo()
	if f.err != nil {
		return info
	}

	if string(buf.Next(8)) != "\x89PNG\x0d\x0a\x1a\x0a" {
		f.SetErrorf("not a PNG buffer")
		return info
	}

	_ = buf.Next(4)
	if string(buf.Next(4)) != "IHDR" {
		f.SetErrorf("incorrect PNG buffer")
		return info
	}
	w := f.readBeInt32(buf)
	h := f.readBeInt32(buf)
	bpc := f.readByte(buf)
	if bpc > 8 {
		f.SetErrorf("16-bit depth not supported in PNG file")
	}
	ct := f.readByte(buf)
	var colspace string
	var colorVal int
	colspace, colorVal = f.pngColorSpace(ct)
	if f.err != nil {
		return info
	}
	if f.readByte(buf) != 0 {
		f.SetErrorf("unknown compression method in PNG buffer")
		return info
	}
	if f.readByte(buf) != 0 {
		f.SetErrorf("unknown filter method in PNG buffer")
		return info
	}
	if f.readByte(buf) != 0 {
		f.SetErrorf("interlacing not supported in PNG buffer")
		return info
	}
	_ = buf.Next(4)
	dp := sprintf("/Predictor 15 /Colors %d /BitsPerComponent %d /Columns %d", colorVal, bpc, w)

	chunks := f.readPNGChunks(buf, ct, readdpi, info)
	if f.err != nil {
		return info
	}
	if colspace == colorSpaceIndexed && len(chunks.pal) == 0 {
		f.SetErrorf("missing palette in PNG buffer")
	}
	info.w = float64(w)
	info.h = float64(h)
	info.cs = colspace
	info.bpc = int(bpc)
	info.f = "FlateDecode"
	info.dp = dp
	info.pal = chunks.pal
	info.trns = chunks.trns

	data := chunks.data
	if ct >= 4 {
		data = f.splitPNGAlpha(data, ct, int(w), int(h), info)
	}
	info.data = data
	return info
}

type pngChunks struct {
	pal  []byte
	trns []int
	data []byte
}

func (f *PDF) readPNGChunks(buf *bytes.Buffer, colorType byte, readDPI bool, info *ImageInfoType) pngChunks {
	chunks := pngChunks{
		pal:  make([]byte, 0, 32),
		data: make([]byte, 0, 32),
	}
	for keepReading := true; keepReading; {
		n := int(f.readBeInt32(buf))
		if f.err != nil {
			return chunks
		}
		if n < 0 || buf.Len() < n+8 {
			f.SetErrorf("incorrect PNG chunk")
			return chunks
		}

		chunkType := string(buf.Next(4))
		chunkData := buf.Next(n)
		_ = buf.Next(4)
		keepReading = f.applyPNGChunk(&chunks, chunkType, chunkData, colorType, readDPI, info)
		if keepReading {
			keepReading = n > 0
		}
	}
	return chunks
}

func (f *PDF) applyPNGChunk(
	chunks *pngChunks,
	chunkType string,
	chunkData []byte,
	colorType byte,
	readDPI bool,
	info *ImageInfoType,
) bool {
	switch chunkType {
	case "PLTE":
		chunks.pal = chunkData
	case "tRNS":
		chunks.trns = f.pngTransparency(colorType, chunkData)
	case "IDAT":
		chunks.data = append(chunks.data, chunkData...)
	case "IEND":
		return false
	case "pHYs":
		f.applyPNGPhysicalDimensions(chunkData, readDPI, info)
	}
	return true
}

func (f *PDF) pngTransparency(colorType byte, chunkData []byte) []int {
	switch colorType {
	case 0:
		if len(chunkData) < 2 {
			f.SetErrorf("incorrect PNG transparency chunk")
			return nil
		}
		return []int{int(chunkData[1])}
	case 2:
		if len(chunkData) < 6 {
			f.SetErrorf("incorrect PNG transparency chunk")
			return nil
		}
		return []int{int(chunkData[1]), int(chunkData[3]), int(chunkData[5])}
	default:
		pos := strings.Index(string(chunkData), "\x00")
		if pos >= 0 {
			return []int{pos}
		}
		return nil
	}
}

func (f *PDF) applyPNGPhysicalDimensions(chunkData []byte, readDPI bool, info *ImageInfoType) {
	if len(chunkData) < 9 {
		f.SetErrorf("incorrect PNG physical pixel dimensions chunk")
		return
	}
	x := int(binary.BigEndian.Uint32(chunkData[0:4]))
	y := int(binary.BigEndian.Uint32(chunkData[4:8]))
	if x != y || !readDPI {
		return
	}
	switch units := chunkData[8]; units {
	case 1:
		info.dpi = float64(x) / 39.3701
	default:
		info.dpi = float64(x)
	}
}

func (f *PDF) splitPNGAlpha(data []byte, colorType byte, width, height int, info *ImageInfoType) []byte {
	data, err := sliceUncompress(data)
	if err != nil {
		f.SetError(err)
		return data
	}

	color, alpha, ok := splitPNGAlphaBytes(data, colorType, width, height)
	if !ok {
		f.SetErrorf("PNG alpha channel data is truncated")
		return data
	}
	info.smask = sliceCompress(alpha)
	if f.pdfVersion < "1.4" {
		f.pdfVersion = "1.4"
	}
	return sliceCompress(color)
}

func splitPNGAlphaBytes(data []byte, colorType byte, width, height int) ([]byte, []byte, bool) {
	if colorType == 4 {
		return splitPNGGrayAlphaBytes(data, width, height)
	}
	return splitPNGRGBABytes(data, width, height)
}

func splitPNGGrayAlphaBytes(data []byte, width, height int) ([]byte, []byte, bool) {
	length := 2 * width
	if len(data) < height*(1+length) {
		return nil, nil, false
	}
	var color, alpha bytes.Buffer
	for i := range height {
		pos := (1 + length) * i
		color.WriteByte(data[pos])
		alpha.WriteByte(data[pos])
		elPos := pos + 1
		for range width {
			color.WriteByte(data[elPos])
			alpha.WriteByte(data[elPos+1])
			elPos += 2
		}
	}
	return color.Bytes(), alpha.Bytes(), true
}

func splitPNGRGBABytes(data []byte, width, height int) ([]byte, []byte, bool) {
	length := 4 * width
	if len(data) < height*(1+length) {
		return nil, nil, false
	}
	var color, alpha bytes.Buffer
	for i := range height {
		pos := (1 + length) * i
		color.WriteByte(data[pos])
		alpha.WriteByte(data[pos])
		elPos := pos + 1
		for range width {
			color.Write(data[elPos : elPos+3])
			alpha.WriteByte(data[elPos+3])
			elPos += 4
		}
	}
	return color.Bytes(), alpha.Bytes(), true
}
