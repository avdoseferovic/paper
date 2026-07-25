package pdf

import (
	"encoding/binary"
	"math"
)

// srgbICCProfile returns a complete sRGB IEC61966-2.1 ICC v2 profile.
func srgbICCProfile() []byte {
	const (
		headerSize  = 128
		tagCount    = 9
		tagTableOff = headerSize
		tagTableSz  = 4 + tagCount*12
	)
	dataOff := tagTableOff + tagTableSz

	descData := iccTextDescriptionTag("sRGB IEC61966-2.1")
	cprtData := iccTextTag("Public Domain")
	wtptData := iccXYZTag(0.9504559, 1.0000000, 1.0890577)
	rXYZData := iccXYZTag(0.4360747, 0.2225045, 0.0139322)
	gXYZData := iccXYZTag(0.3850649, 0.7168786, 0.0971045)
	bXYZData := iccXYZTag(0.1430804, 0.0606169, 0.7141733)
	trcData := iccSRGBCurveTag()

	tags := []iccTagLayout{
		{"desc", descData},
		{"cprt", cprtData},
		{"wtpt", wtptData},
		{"rXYZ", rXYZData},
		{"gXYZ", gXYZData},
		{"bXYZ", bXYZData},
		{"rTRC", trcData},
	}

	offsets, profileSize := iccTagOffsets(tags, dataOff)
	trcOff := offsets[6]
	trcSize := len(trcData)
	profile := make([]byte, profileSize)

	iccPutUint32(profile[0:4], profileSize)
	profile[8] = 2
	profile[9] = 0x10
	copy(profile[12:16], "mntr")
	copy(profile[16:20], "RGB ")
	copy(profile[20:24], "XYZ ")
	binary.BigEndian.PutUint16(profile[24:26], 2024)
	binary.BigEndian.PutUint16(profile[26:28], 1)
	binary.BigEndian.PutUint16(profile[28:30], 1)
	copy(profile[36:40], "acsp")
	copy(profile[40:44], "APPL")
	iccPutS15Fixed16(profile[68:72], 0.9504559)
	iccPutS15Fixed16(profile[72:76], 1.0000000)
	iccPutS15Fixed16(profile[76:80], 1.0890577)

	iccPutUint32(profile[tagTableOff:], tagCount)
	entries := []iccTagEntry{
		{"desc", offsets[0], len(tags[0].data)},
		{"cprt", offsets[1], len(tags[1].data)},
		{"wtpt", offsets[2], len(tags[2].data)},
		{"rXYZ", offsets[3], len(tags[3].data)},
		{"gXYZ", offsets[4], len(tags[4].data)},
		{"bXYZ", offsets[5], len(tags[5].data)},
		{"rTRC", trcOff, trcSize},
		{"gTRC", trcOff, trcSize},
		{"bTRC", trcOff, trcSize},
	}
	for i, entry := range entries {
		p := tagTableOff + 4 + i*12
		copy(profile[p:p+4], entry.sig)
		iccPutUint32(profile[p+4:p+8], entry.offset)
		iccPutUint32(profile[p+8:p+12], entry.size)
	}
	for i, tag := range tags {
		copy(profile[offsets[i]:], tag.data)
	}
	return profile
}

type iccTagLayout struct {
	sig  string
	data []byte
}

type iccTagEntry struct {
	sig    string
	offset int
	size   int
}

func iccTagOffsets(tags []iccTagLayout, dataOff int) ([]int, int) {
	offsets := make([]int, len(tags))
	offset := dataOff
	for i, tag := range tags {
		offsets[i] = offset
		offset += len(tag.data)
		if offset%4 != 0 {
			offset += 4 - offset%4
		}
	}
	return offsets, offset
}

func iccPutUint32(b []byte, value int) {
	checked, _ := checkedUint32(value)
	binary.BigEndian.PutUint32(b, checked)
}

func iccPutS15Fixed16(b []byte, value float64) {
	fixed := int32(math.Round(value * 65536))
	binary.BigEndian.PutUint32(b, uint32FromInt32Bits(fixed))
}

// uint32FromInt32Bits reinterprets a signed 32-bit value as the raw bits an ICC
// s15Fixed16 field stores in a uint32. The magnitude is masked so the
// conversion is provably in range, then the sign bit is reapplied.
func uint32FromInt32Bits(v int32) uint32 {
	bits := uint32(v & 0x7FFFFFFF)
	if v < 0 {
		bits |= 0x80000000
	}
	return bits
}

func iccXYZTag(x, y, z float64) []byte {
	data := make([]byte, 20)
	copy(data[0:4], "XYZ ")
	iccPutS15Fixed16(data[8:12], x)
	iccPutS15Fixed16(data[12:16], y)
	iccPutS15Fixed16(data[16:20], z)
	return data
}

func iccTextDescriptionTag(value string) []byte {
	ascii := []byte(value)
	asciiLen := len(ascii) + 1
	size := 4 + 4 + 4 + asciiLen + 4 + 4 + 2 + 1 + 67
	data := make([]byte, size)
	copy(data[0:4], "desc")
	iccPutUint32(data[8:12], asciiLen)
	copy(data[12:12+len(ascii)], ascii)
	return data
}

func iccTextTag(value string) []byte {
	ascii := []byte(value)
	data := make([]byte, 4+4+len(ascii)+1)
	copy(data[0:4], "text")
	copy(data[8:], ascii)
	return data
}

func iccSRGBCurveTag() []byte {
	const entries = 1024
	data := make([]byte, 4+4+4+entries*2)
	copy(data[0:4], "curv")
	iccPutUint32(data[8:12], entries)
	for i := range entries {
		t := float64(i) / float64(entries-1)
		linear := iccSRGBLinearValue(t)
		rounded := int(math.Round(linear * 65535))
		value, _ := checkedUint16(rounded)
		offset := 12 + i*2
		binary.BigEndian.PutUint16(data[offset:offset+2], value)
	}
	return data
}

func iccSRGBLinearValue(encoded float64) float64 {
	if encoded <= 0.04045 {
		return encoded / 12.92
	}
	return math.Pow((encoded+0.055)/1.055, 2.4)
}
