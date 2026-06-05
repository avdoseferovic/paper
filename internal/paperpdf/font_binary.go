package paperpdf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
)

type fileReader struct {
	readerPosition int64
	array          []byte
	err            error
}

func (fr *fileReader) Read(s int) []byte {
	if s < 0 {
		fr.setErrorf("invalid font read length %d", s)
		return nil
	}

	out := make([]byte, s)
	if s == 0 || fr.err != nil {
		return out
	}

	start := fr.readerPosition
	end := start + int64(s)
	if start < 0 || start > int64(len(fr.array)) || end < start {
		fr.setErrorf("invalid font read offset %d", start)
		return out
	}

	if end > int64(len(fr.array)) {
		fr.setErrorf("unexpected EOF reading font data")
		end = int64(len(fr.array))
	}
	copy(out, fr.array[start:end])
	fr.readerPosition = end
	return out
}

func (fr *fileReader) seek(shift int64, flag int) (int64, error) {
	if fr.err != nil {
		return fr.readerPosition, fr.err
	}

	target := fr.readerPosition
	if flag == 0 {
		target = shift
	} else if flag == 1 {
		target += shift
	} else if flag == 2 {
		target = int64(len(fr.array)) - shift
	} else {
		fr.setErrorf("invalid font seek mode %d", flag)
		return fr.readerPosition, fr.err
	}

	if target < 0 || target > int64(len(fr.array)) {
		fr.setErrorf("invalid font seek offset %d", target)
		return fr.readerPosition, fr.err
	}

	fr.readerPosition = target
	return int64(fr.readerPosition), nil
}

func (fr *fileReader) setErrorf(format string, args ...any) {
	if fr.err == nil {
		fr.err = fmt.Errorf(format, args...)
	}
}

func unpackUint16Array(data []byte) []int {
	answer := make([]int, 1)
	r := bytes.NewReader(data)
	bs := make([]byte, 2)
	var e error
	var c int
	c, e = r.Read(bs)
	for e == nil && c > 0 {
		answer = append(answer, int(binary.BigEndian.Uint16(bs)))
		c, e = r.Read(bs)
	}
	return answer
}

func unpackUint32Array(data []byte) []int {
	answer := make([]int, 1)
	r := bytes.NewReader(data)
	bs := make([]byte, 4)
	var e error
	var c int
	c, e = r.Read(bs)
	for e == nil && c > 0 {
		answer = append(answer, int(binary.BigEndian.Uint32(bs)))
		c, e = r.Read(bs)
	}
	return answer
}

func unpackUint16(data []byte) int {
	return int(binary.BigEndian.Uint16(data))
}

func packHeader(N uint32, n1, n2, n3, n4 int) []byte {
	answer := make([]byte, 0)
	bs4 := make([]byte, 4)
	binary.BigEndian.PutUint32(bs4, N)
	answer = append(answer, bs4...)
	bs := make([]byte, 2)
	binary.BigEndian.PutUint16(bs, uint16(n1))
	answer = append(answer, bs...)
	binary.BigEndian.PutUint16(bs, uint16(n2))
	answer = append(answer, bs...)
	binary.BigEndian.PutUint16(bs, uint16(n3))
	answer = append(answer, bs...)
	binary.BigEndian.PutUint16(bs, uint16(n4))
	answer = append(answer, bs...)
	return answer
}

func pack2Uint16(n1, n2 int) []byte {
	answer := make([]byte, 0)
	bs := make([]byte, 2)
	binary.BigEndian.PutUint16(bs, uint16(n1))
	answer = append(answer, bs...)
	binary.BigEndian.PutUint16(bs, uint16(n2))
	answer = append(answer, bs...)
	return answer
}

func pack2Uint32(n1, n2 int) []byte {
	answer := make([]byte, 0)
	bs := make([]byte, 4)
	binary.BigEndian.PutUint32(bs, uint32(n1))
	answer = append(answer, bs...)
	binary.BigEndian.PutUint32(bs, uint32(n2))
	answer = append(answer, bs...)
	return answer
}

func packUint32(n1 int) []byte {
	bs := make([]byte, 4)
	binary.BigEndian.PutUint32(bs, uint32(n1))
	return bs
}

func packUint16(n1 int) []byte {
	bs := make([]byte, 2)
	binary.BigEndian.PutUint16(bs, uint16(n1))
	return bs
}

func keySortStrings(s map[string][]byte) []string {
	keys := make([]string, len(s))
	i := 0
	for key := range s {
		keys[i] = key
		i++
	}
	sort.Strings(keys)
	return keys
}

func keySortInt(s map[int]int) []int {
	keys := make([]int, len(s))
	i := 0
	for key := range s {
		keys[i] = key
		i++
	}
	sort.Ints(keys)
	return keys
}

func keySortArrayRangeMap(s map[int][]int) []int {
	keys := make([]int, len(s))
	i := 0
	for key := range s {
		keys[i] = key
		i++
	}
	sort.Ints(keys)
	return keys
}
