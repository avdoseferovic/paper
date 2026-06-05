package paperpdf

type coreFontSet map[string]bool

var standardPageSizes = map[string]SizeType{
	"a3":      {841.89, 1190.55},
	"a4":      {595.28, 841.89},
	"a5":      {420.94, 595.28},
	"a6":      {297.64, 420.94},
	"a2":      {1190.55, 1683.78},
	"a1":      {1683.78, 2383.94},
	"letter":  {612, 792},
	"legal":   {612, 1008},
	"tabloid": {792, 1224},
}

var coreFontNames = []string{
	"courier",
	"helvetica",
	"times",
	"symbol",
	"zapfdingbats",
}

func cloneStandardPageSizes() map[string]SizeType {
	pageSizes := make(map[string]SizeType, len(standardPageSizes))
	for name, size := range standardPageSizes {
		pageSizes[name] = size
	}
	return pageSizes
}

func cloneCoreFontSet() coreFontSet {
	fonts := make(coreFontSet, len(coreFontNames))
	for _, name := range coreFontNames {
		fonts[name] = true
	}
	return fonts
}
