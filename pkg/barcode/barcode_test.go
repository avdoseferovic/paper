package barcode_test

import (
	"testing"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/internal/fixture"
	"github.com/avdoseferovic/paper/internal/mocks"
	mock "github.com/avdoseferovic/paper/internal/mocktest"
	"github.com/avdoseferovic/paper/pkg/barcode"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/props"
)

func TestNewCode128(t *testing.T) {
	t.Parallel()

	cell := fixture.CellEntity()
	component := barcode.NewCode128("ABC123")

	provider := mocks.NewProvider(t)
	provider.EXPECT().AddBarCode("ABC123", &cell, propsWithType(t, consts.BarcodeCode128)).Once()

	component.Render(provider, &cell)
}

func TestNewEAN13(t *testing.T) {
	t.Parallel()

	cell := fixture.CellEntity()
	component := barcode.NewEAN13("123456789123")

	provider := mocks.NewProvider(t)
	provider.EXPECT().AddBarCode("123456789123", &cell, propsWithType(t, consts.BarcodeEAN)).Once()

	component.Render(provider, &cell)
}

func TestNewQR(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "qrcode", barcode.NewQR("payload").GetStructure().GetData().Type)
}

func TestNewDataMatrix(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "matrixcode", barcode.NewDataMatrix("payload").GetStructure().GetData().Type)
}

func TestRowAndColConstructors(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, barcode.NewCode128Col(6, "ABC123"))
	assert.NotNil(t, barcode.NewCode128Row(12, "ABC123"))
	assert.NotNil(t, barcode.NewAutoCode128Row("ABC123"))
	assert.NotNil(t, barcode.NewEAN13Col(6, "123456789123"))
	assert.NotNil(t, barcode.NewEAN13Row(12, "123456789123"))
	assert.NotNil(t, barcode.NewAutoEAN13Row("123456789123"))
	assert.NotNil(t, barcode.NewQRCol(6, "payload"))
	assert.NotNil(t, barcode.NewQRRow(12, "payload"))
	assert.NotNil(t, barcode.NewAutoQRRow("payload"))
	assert.NotNil(t, barcode.NewDataMatrixCol(6, "payload"))
	assert.NotNil(t, barcode.NewDataMatrixRow(12, "payload"))
	assert.NotNil(t, barcode.NewAutoDataMatrixRow("payload"))
}

func propsWithType(t *testing.T, want consts.BarcodeType) any {
	t.Helper()
	return mock.MatchedBy(func(prop *props.Barcode) bool {
		return prop != nil && prop.Type == want
	})
}
