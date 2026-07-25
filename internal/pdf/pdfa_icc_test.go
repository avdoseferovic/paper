package pdf

import (
	"encoding/binary"
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
)

func TestSRGBICCProfile_ShouldContainValidHeaderAndRequiredTags(t *testing.T) {
	t.Parallel()

	profile := srgbICCProfile()

	assert.Greater(t, len(profile), 2000)
	assert.Equal(t, uint32(len(profile)&0x7FFFFFFF), binary.BigEndian.Uint32(profile[0:4]))
	assert.Equal(t, []byte("mntr"), profile[12:16])
	assert.Equal(t, []byte("RGB "), profile[16:20])
	assert.Equal(t, []byte("XYZ "), profile[20:24])
	assert.Equal(t, []byte("acsp"), profile[36:40])
	assert.Equal(t, uint32(9), binary.BigEndian.Uint32(profile[128:132]))

	tags := make(map[string]bool)
	for i := range 9 {
		offset := 132 + i*12
		tags[string(profile[offset:offset+4])] = true
	}
	for _, tag := range []string{"desc", "cprt", "wtpt", "rXYZ", "gXYZ", "bXYZ", "rTRC", "gTRC", "bTRC"} {
		assert.True(t, tags[tag], "missing ICC tag %s", tag)
	}
}
