package config

import "github.com/avdoseferovic/paper/pkg/consts"

// WithConcurrentMode defines concurrent generation, chunk workers define how mano chuncks
// will be executed concurrently.
func (b *CfgBuilder) WithConcurrentMode(chunkWorkers int) Builder {
	if chunkWorkers < 1 {
		return b
	}

	b.generationMode = consts.GenerationConcurrent
	b.chunkWorkers = chunkWorkers
	return b
}

// WithSequentialMode defines that paper will run in default mode.
func (b *CfgBuilder) WithSequentialMode() Builder {
	b.chunkWorkers = 1
	b.generationMode = consts.GenerationSequential
	return b
}

// WithSequentialLowMemoryMode defines that paper will run focusing in reduce memory consumption,
// chunk workers define how many divisions the work will have.
func (b *CfgBuilder) WithSequentialLowMemoryMode(chunkWorkers int) Builder {
	if chunkWorkers < 1 {
		return b
	}

	b.generationMode = consts.GenerationSequentialLowMemory
	b.chunkWorkers = chunkWorkers
	return b
}

// WithDebug defines a debug behaviour where paper will draw borders in everything.
func (b *CfgBuilder) WithDebug(on bool) Builder {
	b.debug = on
	return b
}

// WithMaxGridSize defines a custom max grid sum which it will change the sum of column sizes.
func (b *CfgBuilder) WithMaxGridSize(maxGridSize int) Builder {
	if maxGridSize <= 0 {
		return b
	}

	b.maxGridSize = maxGridSize
	return b
}
