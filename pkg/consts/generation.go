package consts

// GenerationMode is the representation of a document generation mode.
type GenerationMode string

const (
	// GenerationSequential renders pages one at a time on a single goroutine.
	GenerationSequential GenerationMode = "sequential"
	// GenerationConcurrent renders page chunks in parallel worker goroutines
	// and merges the resulting documents.
	GenerationConcurrent GenerationMode = "concurrent"
	// GenerationSequentialLowMemory renders page chunks sequentially,
	// releasing memory between chunks, and merges the resulting documents.
	GenerationSequentialLowMemory GenerationMode = "sequential_low_memory"
)
