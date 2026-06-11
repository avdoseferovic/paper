package paper_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/avdoseferovic/paper/internal/fixture"
	"github.com/avdoseferovic/paper/internal/merror"
	"github.com/avdoseferovic/paper/internal/mocks"
	mock "github.com/avdoseferovic/paper/internal/mocktest"
	pdf "github.com/avdoseferovic/paper/internal/pdf"
	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/consts/protection"

	gofpdf "github.com/avdoseferovic/paper/internal/providers/paper"

	"github.com/avdoseferovic/paper/internal/assert"
	"github.com/avdoseferovic/paper/pkg/core"
	"github.com/avdoseferovic/paper/pkg/core/entity"
	"github.com/avdoseferovic/paper/pkg/props"
)

const (
	codeContent = "code"
)

func TestNew(t *testing.T) {
	t.Parallel()
	// Act
	sut := gofpdf.New(&gofpdf.Dependencies{})

	// Assert
	assert.NotNil(t, sut)
	assert.Equal(t, "*paper.provider", fmt.Sprintf("%T", sut))
}

func TestProvider_RenderIssuesAreRecordedAndCopied(t *testing.T) {
	t.Parallel()

	cell := &entity.Cell{}
	prop := fixture.RectProp()

	cache := mocks.NewCache(t)
	cache.EXPECT().GetImage("matrix-code-code", extension.Png).Return(nil, errors.New("cache miss"))

	code := mocks.NewCode(t)
	code.EXPECT().GenDataMatrix(codeContent).Return(nil, errors.New("generate failed"))

	text := mocks.NewText(t)
	text.EXPECT().Add("could not generate matrixcode", cell, merror.DefaultErrorText)

	sut := gofpdf.New(&gofpdf.Dependencies{
		Cache: cache,
		Code:  code,
		Text:  text,
	})

	sut.AddMatrixCode(codeContent, cell, &prop)

	issueProvider := sut.(core.RenderIssueProvider)
	issues := issueProvider.RenderIssues()
	assert.Len(t, issues, 1)
	assert.Equal(t, "matrixcode.generate", issues[0].Operation)
	assert.Equal(t, "could not generate matrixcode", issues[0].Message)
	assert.Contains(t, issues[0].Error, "generate failed")

	issues[0].Operation = "changed"
	assert.Equal(t, "matrixcode.generate", issueProvider.RenderIssues()[0].Operation)
}

func TestProvider_AddText(t *testing.T) {
	t.Parallel()
	// Arrange
	txtContent := "text"
	cell := &entity.Cell{}
	prop := fixture.TextProp()

	text := mocks.NewText(t)
	text.EXPECT().Add(txtContent, cell, &prop).Once()

	dep := &gofpdf.Dependencies{
		Text: text,
	}
	sut := gofpdf.New(dep)

	// Act
	sut.AddText(txtContent, cell, &prop)
}

func TestProvider_GetTextHeight(t *testing.T) {
	t.Parallel()
	// Arrange
	fontHeightToReturn := 10.0
	prop := fixture.FontProp()

	font := mocks.NewFont(t)
	font.EXPECT().GetHeight(prop.Family, prop.Style, prop.Size).Return(fontHeightToReturn).Once()

	dep := &gofpdf.Dependencies{
		Font: font,
	}
	sut := gofpdf.New(dep)

	// Act
	fontHeight := sut.GetFontHeight(&prop)

	// Assert
	assert.Equal(t, fontHeightToReturn, fontHeight)
}

func TestProvider_AddLine(t *testing.T) {
	t.Parallel()
	// Arrange
	cell := &entity.Cell{}
	prop := fixture.LineProp()

	line := mocks.NewLine(t)
	line.EXPECT().Add(cell, &prop).Once()

	dep := &gofpdf.Dependencies{
		Line: line,
	}
	sut := gofpdf.New(dep)

	// Act
	sut.AddLine(cell, &prop)
}

func TestProvider_AddCheckbox(t *testing.T) {
	t.Parallel()
	// Arrange
	label := "checkbox label"
	cell := &entity.Cell{}
	prop := fixture.CheckboxProp()

	checkbox := mocks.NewCheckbox(t)
	checkbox.EXPECT().Add(label, cell, &prop).Once()

	dep := &gofpdf.Dependencies{
		Checkbox: checkbox,
	}
	sut := gofpdf.New(dep)

	// Act
	sut.AddCheckbox(label, cell, &prop)
}

func TestProvider_AddMatrixCode(t *testing.T) {
	t.Parallel()
	t.Run("when cannot find image on cache and cannot generate data matrix, should apply error message", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{}
		prop := fixture.RectProp()

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("matrix-code-code", extension.Png).Return(nil, errors.New("anyError1")).Once()

		code := mocks.NewCode(t)
		code.EXPECT().GenDataMatrix(codeContent).Return(nil, errors.New("anyError2")).Once()

		text := mocks.NewText(t)
		text.EXPECT().Add("could not generate matrixcode", cell, merror.DefaultErrorText).Once()

		dep := &gofpdf.Dependencies{
			Cache: cache,
			Code:  code,
			Text:  text,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.AddMatrixCode(codeContent, cell, &prop)
	})

	t.Run("when can find image on cache but cannot add image, should apply error message", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{}
		prop := fixture.RectProp()

		img := &entity.Image{Bytes: []byte{1, 2, 3}}

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("matrix-code-code", extension.Png).Return(img, nil).Once()

		code := mocks.NewCode(t)

		text := mocks.NewText(t)
		text.EXPECT().Add("could not add matrixcode to document", cell, merror.DefaultErrorText).Once()

		cfg := &entity.Config{
			Margins: &entity.Margins{
				Left:   10,
				Top:    10,
				Right:  10,
				Bottom: 10,
			},
		}

		image := mocks.NewImage(t)
		image.EXPECT().Add(img, cell, cfg.Margins, &prop, extension.Png, false).Return(errors.New("anyError")).Once()

		fpdf := newPDF(t)
		fpdf.EXPECT().ClearError()

		dep := &gofpdf.Dependencies{
			Cache: cache,
			Text:  text,
			Image: image,
			PDF:   fpdf,
			Cfg:   cfg,
			Code:  code,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.AddMatrixCode("code", cell, &prop)
	})
	t.Run("when can find image on cache and can add image, should not apply error message", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{}
		prop := fixture.RectProp()

		img := &entity.Image{Bytes: []byte{1, 2, 3}}

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("matrix-code-code", extension.Png).Return(img, nil).Once()

		code := mocks.NewCode(t)

		cfg := &entity.Config{
			Margins: &entity.Margins{
				Left:   10,
				Top:    10,
				Right:  10,
				Bottom: 10,
			},
		}

		image := mocks.NewImage(t)
		image.EXPECT().Add(img, cell, cfg.Margins, &prop, extension.Png, false).Return(nil).Once()

		dep := &gofpdf.Dependencies{
			Cache: cache,
			Image: image,
			Cfg:   cfg,
			Code:  code,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.AddMatrixCode("code", cell, &prop)
	})
	t.Run("when matrx code is generated with the code sent, it should generate matrix code with the same code", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{}
		cfg := fixture.ConfigEntity()
		prop := fixture.RectProp()
		img := &entity.Image{Bytes: []byte{1, 2, 3}}

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("matrix-code-code", extension.Png).Return(nil, errors.New("anyError1")).Once()
		cache.EXPECT().AddImage("matrix-code-code", img).Return().Once()

		code := mocks.NewCode(t)
		code.EXPECT().GenDataMatrix(codeContent).Return(img, nil).Once()

		image := mocks.NewImage(t)
		image.EXPECT().Add(img, cell, cfg.Margins, &prop, extension.Png, false).Return(nil).Once()

		dep := &gofpdf.Dependencies{
			Cache: cache,
			Code:  code,
			Cfg:   &cfg,
			Image: image,
		}

		// Act
		gofpdf.New(dep).AddMatrixCode(codeContent, cell, &prop)
	})
}

func TestProvider_AddQrCode(t *testing.T) {
	t.Parallel()
	t.Run("when cannot find image on cache and cannot generate qr code, should apply error message", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{}
		prop := fixture.RectProp()

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("qr-code-code", extension.Png).Return(nil, errors.New("anyError1")).Once()

		code := mocks.NewCode(t)
		code.EXPECT().GenQr(codeContent).Return(nil, errors.New("anyError2")).Once()

		text := mocks.NewText(t)
		text.EXPECT().Add("could not generate qrcode", cell, merror.DefaultErrorText).Once()

		dep := &gofpdf.Dependencies{
			Cache: cache,
			Code:  code,
			Text:  text,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.AddQrCode("code", cell, &prop)
	})
	t.Run("when can find image on cache but cannot add image, should apply error message", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{}
		prop := fixture.RectProp()

		img := &entity.Image{Bytes: []byte{1, 2, 3}}

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("qr-code-code", extension.Png).Return(img, nil).Once()

		code := mocks.NewCode(t)

		text := mocks.NewText(t)
		text.EXPECT().Add("could not add qrcode to document", cell, merror.DefaultErrorText).Once()

		cfg := &entity.Config{
			Margins: &entity.Margins{
				Left:   10,
				Top:    10,
				Right:  10,
				Bottom: 10,
			},
		}

		image := mocks.NewImage(t)
		image.EXPECT().Add(img, cell, cfg.Margins, &prop, extension.Png, false).Return(errors.New("anyError")).Once()

		fpdf := newPDF(t)
		fpdf.EXPECT().ClearError()

		dep := &gofpdf.Dependencies{
			Cache: cache,
			Text:  text,
			Image: image,
			PDF:   fpdf,
			Cfg:   cfg,
			Code:  code,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.AddQrCode("code", cell, &prop)
	})
	t.Run("when can find image on cache and can add image, should not apply error message", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{}
		prop := fixture.RectProp()

		img := &entity.Image{Bytes: []byte{1, 2, 3}}

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("qr-code-code", extension.Png).Return(img, nil).Once()

		code := mocks.NewCode(t)

		cfg := &entity.Config{
			Margins: &entity.Margins{
				Left:   10,
				Top:    10,
				Right:  10,
				Bottom: 10,
			},
		}

		image := mocks.NewImage(t)
		image.EXPECT().Add(img, cell, cfg.Margins, &prop, extension.Png, false).Return(nil).Once()

		dep := &gofpdf.Dependencies{
			Cache: cache,
			Image: image,
			Cfg:   cfg,
			Code:  code,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.AddQrCode("code", cell, &prop)
	})
	t.Run("when qrcode is generated with the code sent, it should generate qr code with the same code", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{}
		prop := fixture.RectProp()
		img := &entity.Image{Bytes: []byte{1, 2, 3}}
		cfg := fixture.ConfigEntity()

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("qr-code-code", extension.Png).Return(nil, errors.New("anyError1")).Once()
		cache.EXPECT().AddImage("qr-code-code", img).Return().Once()

		code := mocks.NewCode(t)
		code.EXPECT().GenQr(codeContent).Return(img, nil).Once()

		image := mocks.NewImage(t)
		image.EXPECT().Add(img, cell, cfg.Margins, &prop, extension.Png, false).Return(nil).Once()

		dep := &gofpdf.Dependencies{
			Cache: cache,
			Code:  code,
			Cfg:   &cfg,
			Image: image,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.AddQrCode(codeContent, cell, &prop)
	})
}

func TestProvider_AddBarCode(t *testing.T) {
	t.Parallel()
	t.Run("when cannot find image on cache and cannot generate bar code, should apply error message", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{}
		prop := fixture.BarcodeProp()

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("bar-code-codecode128", extension.Png).Return(nil, errors.New("anyError1")).Once()

		code := mocks.NewCode(t)
		code.EXPECT().GenBar(codeContent, cell, &prop).Return(nil, errors.New("anyError2")).Once()

		text := mocks.NewText(t)
		text.EXPECT().Add("could not generate barcode", cell, merror.DefaultErrorText).Once()

		dep := &gofpdf.Dependencies{
			Cache: cache,
			Code:  code,
			Text:  text,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.AddBarCode(codeContent, cell, &prop)
	})
	t.Run("when can find image on cache but cannot add image, should apply error message", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{}
		prop := fixture.BarcodeProp()

		img := &entity.Image{Bytes: []byte{1, 2, 3}}

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("bar-code-codecode128", extension.Png).Return(img, nil).Once()
		cache.EXPECT().AddImage("bar-code-codecode128", img).Once()

		text := mocks.NewText(t)
		text.EXPECT().Add("could not add barcode to document", cell, merror.DefaultErrorText).Once()

		cfg := &entity.Config{
			Margins: &entity.Margins{
				Left:   10,
				Top:    10,
				Right:  10,
				Bottom: 10,
			},
		}

		image := mocks.NewImage(t)
		image.EXPECT().Add(img, cell, cfg.Margins, prop.ToRectProp(), extension.Png, false).Return(errors.New("anyError")).Once()

		fpdf := newPDF(t)
		fpdf.EXPECT().ClearError()

		dep := &gofpdf.Dependencies{
			Cache: cache,
			Text:  text,
			Image: image,
			PDF:   fpdf,
			Cfg:   cfg,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.AddBarCode(codeContent, cell, &prop)
	})
	t.Run("when can find image on cache and can add image, should not apply error message", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{}
		prop := fixture.BarcodeProp()

		img := &entity.Image{Bytes: []byte{1, 2, 3}}

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("bar-code-codecode128", extension.Png).Return(img, nil).Once()
		cache.EXPECT().AddImage("bar-code-codecode128", img).Once()

		cfg := &entity.Config{
			Margins: &entity.Margins{
				Left:   10,
				Top:    10,
				Right:  10,
				Bottom: 10,
			},
		}

		image := mocks.NewImage(t)
		image.EXPECT().Add(img, cell, cfg.Margins, prop.ToRectProp(), extension.Png, false).Return(nil).Once()

		dep := &gofpdf.Dependencies{
			Cache: cache,
			Image: image,
			Cfg:   cfg,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.AddBarCode(codeContent, cell, &prop)
	})
	t.Run("when barcode is ean and everything is correct, should not apply error message", func(t *testing.T) {
		t.Parallel()
		// Arrange
		cell := &entity.Cell{}
		prop := fixture.BarcodeProp()
		prop.Type = consts.BarcodeEAN

		img := &entity.Image{Bytes: []byte{1, 2, 3}}

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("bar-code-codeean", extension.Png).Return(img, nil).Once()
		cache.EXPECT().AddImage("bar-code-codeean", img).Once()

		cfg := &entity.Config{
			Margins: &entity.Margins{
				Left:   10,
				Top:    10,
				Right:  10,
				Bottom: 10,
			},
		}

		image := mocks.NewImage(t)
		image.EXPECT().Add(img, cell, cfg.Margins, prop.ToRectProp(), extension.Png, false).Return(nil).Once()

		dep := &gofpdf.Dependencies{
			Cache: cache,
			Image: image,
			Cfg:   cfg,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.AddBarCode(codeContent, cell, &prop)
	})
}

func TestProvider_CreateRow(t *testing.T) {
	t.Parallel()
	// Arrange
	height := 10.0

	fpdf := newPDF(t)
	fpdf.EXPECT().Ln(height).Once()

	dep := &gofpdf.Dependencies{
		PDF: fpdf,
	}

	sut := gofpdf.New(dep)

	// Act
	sut.CreateRow(height)
}

func TestProvider_EnsurePage(t *testing.T) {
	t.Parallel()

	t.Run("adds pages until requested page is current", func(t *testing.T) {
		t.Parallel()
		fpdf := newPDF(t)
		fpdf.EXPECT().PageNo().Return(1).Once()
		fpdf.EXPECT().AddPage().Once()
		fpdf.EXPECT().PageNo().Return(2).Once()

		sut := gofpdf.New(&gofpdf.Dependencies{PDF: fpdf})

		sut.(core.PageProvider).EnsurePage(2)
	})

	t.Run("does not add a page when already current", func(t *testing.T) {
		t.Parallel()
		fpdf := newPDF(t)
		fpdf.EXPECT().PageNo().Return(2).Once()

		sut := gofpdf.New(&gofpdf.Dependencies{PDF: fpdf})

		sut.(core.PageProvider).EnsurePage(2)
		fpdf.AssertNotCalled(t, "AddPage")
	})
}

func TestProvider_CreateCol(t *testing.T) {
	t.Parallel()
	// Arrange
	width := 10.0
	height := 20.0
	cfg := &entity.Config{}
	prop := fixture.CellProp()

	cellWriter := mocks.NewCellWriter(t)
	cellWriter.EXPECT().Apply(width, height, cfg, &prop).Once()

	dep := &gofpdf.Dependencies{
		CellWriter: cellWriter,
	}

	sut := gofpdf.New(dep)

	// Act
	sut.CreateCol(width, height, cfg, &prop)
}

func TestProvider_SetProtection(t *testing.T) {
	t.Parallel()
	t.Run("when protection is nil, should ignore protection", func(t *testing.T) {
		t.Parallel()
		// Arrange
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("code unexpectedly panicked: %v", r)
			}
		}()

		// Act
		dep := &gofpdf.Dependencies{}
		sut := gofpdf.New(dep)

		// Act
		sut.SetProtection(nil)
	})
	t.Run("when protection is valid, should apply protection", func(t *testing.T) {
		t.Parallel()
		// Arrange
		p := &entity.Protection{
			Type:          protection.Print,
			UserPassword:  "userPassword",
			OwnerPassword: "ownerPassword",
		}

		fpdf := newPDF(t)
		fpdf.EXPECT().SetProtectionAlgorithm(pdf.ProtectionRC4).Once()
		fpdf.EXPECT().SetProtection(byte(p.Type), p.UserPassword, p.OwnerPassword).Once()

		dep := &gofpdf.Dependencies{
			PDF: fpdf,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.SetProtection(p)
	})
	t.Run("when protection uses AES-128, should select AES backend algorithm", func(t *testing.T) {
		t.Parallel()
		// Arrange
		p := &entity.Protection{
			Type:          protection.Print,
			UserPassword:  "userPassword",
			OwnerPassword: "ownerPassword",
			Algorithm:     protection.AES128,
		}

		fpdf := newPDF(t)
		fpdf.EXPECT().SetProtectionAlgorithm(pdf.ProtectionAES128).Once()
		fpdf.EXPECT().SetProtection(byte(p.Type), p.UserPassword, p.OwnerPassword).Once()

		dep := &gofpdf.Dependencies{
			PDF: fpdf,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.SetProtection(p)
	})
}

func TestProvider_SetCompression(t *testing.T) {
	t.Parallel()
	// Arrange
	fpdf := newPDF(t)
	fpdf.EXPECT().SetCompression(true).Once()

	dep := &gofpdf.Dependencies{
		PDF: fpdf,
	}

	sut := gofpdf.New(dep)

	// Act
	sut.SetCompression(true)
}

func TestProvider_SetMetadata(t *testing.T) {
	t.Parallel()
	t.Run("when metadata is nil, should avoid process", func(t *testing.T) {
		t.Parallel()
		// Arrange
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("code unexpectedly panicked: %v", r)
			}
		}()

		dep := &gofpdf.Dependencies{}

		sut := gofpdf.New(dep)

		// Act
		sut.SetMetadata(nil)
	})
	t.Run("when metadata is filled, should apply", func(t *testing.T) {
		t.Parallel()
		// Arrange
		timeNow := time.Now()

		fpdf := newPDF(t)
		fpdf.EXPECT().SetAuthor("author", true).Once()
		fpdf.EXPECT().SetCreator("creator", true).Once()
		fpdf.EXPECT().SetSubject("subject", true).Once()
		fpdf.EXPECT().SetTitle("title", true).Once()
		fpdf.EXPECT().SetKeywords("keyword", true).Once()
		fpdf.EXPECT().SetCreationDate(timeNow).Once()

		dep := &gofpdf.Dependencies{
			PDF: fpdf,
		}
		sut := gofpdf.New(dep)

		// Act
		sut.SetMetadata(&entity.Metadata{
			Author: &entity.Utf8Text{
				Text: "author",
				UTF8: true,
			},
			Creator: &entity.Utf8Text{
				Text: "creator",
				UTF8: true,
			},
			Subject: &entity.Utf8Text{
				Text: "subject",
				UTF8: true,
			},
			Title: &entity.Utf8Text{
				Text: "title",
				UTF8: true,
			},
			KeywordsStr: &entity.Utf8Text{
				Text: "keyword",
				UTF8: true,
			},
			CreationDate: &timeNow,
		})
	})
}

func TestProvider_GenerateBytes(t *testing.T) {
	t.Parallel()
	// Arrange
	fpdf := newPDF(t)
	fpdf.EXPECT().Output(mock.Anything).Return(errors.New("anyError")).Once()

	dep := &gofpdf.Dependencies{
		PDF: fpdf,
	}
	sut := gofpdf.New(dep)

	// Act
	bytes, err := sut.GenerateBytes()

	// Assert
	assert.Nil(t, bytes)
	assert.NotNil(t, err)
}

func TestProvider_AddImageFromBytes(t *testing.T) {
	t.Parallel()
	t.Run("when image is invalid, should apply message error", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := fixture.RectProp()
		cell := &entity.Cell{}

		text := mocks.NewText(t)
		text.EXPECT().Add("could not parse image bytes", cell, merror.DefaultErrorText).Once()

		dep := &gofpdf.Dependencies{
			Text: text,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.AddImageFromBytes([]byte{1, 2, 3}, cell, &prop, "invalid")
	})
	t.Run("when image is valid but cannot add to document, should apply message error", func(t *testing.T) {
		t.Parallel()
		// Arrange
		img := &entity.Image{
			Bytes:     []byte{1, 2, 3},
			Extension: extension.Jpg,
		}
		prop := fixture.RectProp()
		cell := &entity.Cell{}

		cfg := &entity.Config{
			Margins: &entity.Margins{
				Left:   10,
				Top:    10,
				Right:  10,
				Bottom: 10,
			},
		}

		text := mocks.NewText(t)
		text.EXPECT().Add("could not add image to document", cell, merror.DefaultErrorText).Once()

		image := mocks.NewImage(t)
		image.EXPECT().Add(img, cell, cfg.Margins, &prop, img.Extension, false).Return(errors.New("anyError")).Once()

		fpdf := newPDF(t)
		fpdf.EXPECT().ClearError().Once()

		dep := &gofpdf.Dependencies{
			Text:  text,
			Image: image,
			PDF:   fpdf,
			Cfg:   cfg,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.AddImageFromBytes(img.Bytes, cell, &prop, img.Extension)
	})
	t.Run("when image is valid and can add to document, should not apply", func(t *testing.T) {
		t.Parallel()
		// Arrange
		img := &entity.Image{
			Bytes:     []byte{1, 2, 3},
			Extension: extension.Jpg,
		}
		prop := fixture.RectProp()
		cell := &entity.Cell{}

		cfg := &entity.Config{
			Margins: &entity.Margins{
				Left:   10,
				Top:    10,
				Right:  10,
				Bottom: 10,
			},
		}

		image := mocks.NewImage(t)
		image.EXPECT().Add(img, cell, cfg.Margins, &prop, img.Extension, false).Return(nil).Once()

		dep := &gofpdf.Dependencies{
			Image: image,
			Cfg:   cfg,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.AddImageFromBytes(img.Bytes, cell, &prop, img.Extension)
	})
}

func TestProvider_AddBackgroundImageFromBytes(t *testing.T) {
	t.Parallel()
	t.Run("when image is invalid, should apply message error", func(t *testing.T) {
		t.Parallel()
		// Arrange
		prop := fixture.RectProp()
		cell := &entity.Cell{}

		text := mocks.NewText(t)
		text.EXPECT().Add("could not parse image bytes", cell, merror.DefaultErrorText).Once()

		dep := &gofpdf.Dependencies{
			Text: text,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.AddBackgroundImageFromBytes([]byte{1, 2, 3}, cell, &prop, "invalid")
	})
	t.Run("when image is valid but cannot add to document, should apply message error", func(t *testing.T) {
		t.Parallel()
		// Arrange
		img := &entity.Image{
			Bytes:     []byte{1, 2, 3},
			Extension: extension.Jpg,
		}
		prop := fixture.RectProp()
		cell := &entity.Cell{}

		cfg := &entity.Config{
			Margins: &entity.Margins{
				Left:   10,
				Top:    10,
				Right:  10,
				Bottom: 10,
			},
		}

		text := mocks.NewText(t)
		text.EXPECT().Add("could not add image to document", cell, merror.DefaultErrorText).Once()

		image := mocks.NewImage(t)
		image.EXPECT().Add(img, cell, cfg.Margins, &prop, img.Extension, true).Return(errors.New("anyError")).Once()

		fpdf := newPDF(t)
		fpdf.EXPECT().ClearError().Once()
		fpdf.EXPECT().SetHomeXY().Once()

		dep := &gofpdf.Dependencies{
			Text:  text,
			Image: image,
			PDF:   fpdf,
			Cfg:   cfg,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.AddBackgroundImageFromBytes(img.Bytes, cell, &prop, img.Extension)
	})
	t.Run("when image is valid and can add to document, should not apply message error", func(t *testing.T) {
		t.Parallel()
		// Arrange
		img := &entity.Image{
			Bytes:     []byte{1, 2, 3},
			Extension: extension.Jpg,
		}
		prop := fixture.RectProp()
		cell := &entity.Cell{}

		cfg := &entity.Config{
			Margins: &entity.Margins{
				Left:   10,
				Top:    10,
				Right:  10,
				Bottom: 10,
			},
		}

		image := mocks.NewImage(t)
		image.EXPECT().Add(img, cell, cfg.Margins, &prop, img.Extension, true).Return(nil).Once()

		fpdf := newPDF(t)
		fpdf.EXPECT().SetHomeXY().Once()

		dep := &gofpdf.Dependencies{
			Image: image,
			PDF:   fpdf,
			Cfg:   cfg,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.AddBackgroundImageFromBytes(img.Bytes, cell, &prop, img.Extension)
	})
}

func TestProvider_GetDimensionsByMatrixCode(t *testing.T) {
	t.Parallel()
	t.Run("when cannot find image on cache and cannot generate data matrix, should return error", func(t *testing.T) {
		t.Parallel()
		// Arrange

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("matrix-code-code", extension.Png).Return(nil, errors.New("anyError1")).Once()

		code := mocks.NewCode(t)
		code.EXPECT().GenDataMatrix(codeContent).Return(nil, errors.New("anyError2")).Once()

		dep := &gofpdf.Dependencies{
			Cache: cache,
			Code:  code,
		}

		sut := gofpdf.New(dep)

		// Act
		dimensions, err := sut.GetDimensionsByMatrixCode(codeContent)

		// Assert
		assert.Nil(t, dimensions)
		assert.NotNil(t, err)
	})
	t.Run("when cannot find image on cache but can generate data matrix, should return dimension", func(t *testing.T) {
		t.Parallel()
		// Arrange
		img := &entity.Image{Bytes: []byte{1, 2, 3}}

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("matrix-code-code", extension.Png).Return(nil, errors.New("anyError1")).Once()
		cache.EXPECT().AddImage("matrix-code-code", img)

		code := mocks.NewCode(t)
		code.EXPECT().GenDataMatrix(codeContent).Return(img, nil).Once()

		cfg := &entity.Config{
			Margins: &entity.Margins{
				Left:   10,
				Top:    10,
				Right:  10,
				Bottom: 10,
			},
		}

		image := mocks.NewImage(t)
		image.EXPECT().GetImageDimensions(img, extension.Png).Return(&entity.Dimensions{Width: 1, Height: 1})

		dep := &gofpdf.Dependencies{
			Cache: cache,
			Image: image,
			Cfg:   cfg,
			Code:  code,
		}

		sut := gofpdf.New(dep)

		// Act
		dimensions, err := sut.GetDimensionsByMatrixCode(codeContent)

		// Assert
		assert.NotNil(t, dimensions)
		assert.Nil(t, err)
	})
	t.Run("when can find matrix on cache, should return dimension", func(t *testing.T) {
		t.Parallel()
		img := &entity.Image{Bytes: []byte{1, 2, 3}}

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("matrix-code-code", extension.Png).Return(img, nil).Once()

		code := mocks.NewCode(t)

		cfg := &entity.Config{Margins: &entity.Margins{Left: 10, Top: 10, Right: 10, Bottom: 10}}

		image := mocks.NewImage(t)
		image.EXPECT().GetImageDimensions(img, extension.Png).Return(&entity.Dimensions{Width: 1, Height: 1})

		dep := &gofpdf.Dependencies{
			Cache: cache,
			Image: image,
			Cfg:   cfg,
			Code:  code,
		}

		sut := gofpdf.New(dep)

		// Act
		dimensions, err := sut.GetDimensionsByMatrixCode(codeContent)

		// Assert
		assert.NotNil(t, dimensions)
		assert.Nil(t, err)
	})
}

func TestProvider_GetDimensionsByQrCode(t *testing.T) {
	t.Parallel()
	t.Run("when cannot find image on cache and cannot generate qrCode, should return error", func(t *testing.T) {
		t.Parallel()
		// Arrange

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("qr-code-code", extension.Png).Return(nil, errors.New("anyError1")).Once()

		code := mocks.NewCode(t)
		code.EXPECT().GenQr(codeContent).Return(nil, errors.New("anyError2")).Once()

		dep := &gofpdf.Dependencies{
			Cache: cache,
			Code:  code,
		}

		sut := gofpdf.New(dep)

		// Act
		dimensions, err := sut.GetDimensionsByQrCode(codeContent)

		// Assert
		assert.Nil(t, dimensions)
		assert.NotNil(t, err)
	})
	t.Run("when cannot find image on cache but can generate qrCode, should return dimension", func(t *testing.T) {
		t.Parallel()
		// Arrange
		img := &entity.Image{Bytes: []byte{1, 2, 3}}

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("qr-code-code", extension.Png).Return(nil, errors.New("anyError1")).Once()
		cache.EXPECT().AddImage("qr-code-code", img)

		code := mocks.NewCode(t)
		code.EXPECT().GenQr(codeContent).Return(img, nil).Once()

		cfg := &entity.Config{
			Margins: &entity.Margins{
				Left:   10,
				Top:    10,
				Right:  10,
				Bottom: 10,
			},
		}

		image := mocks.NewImage(t)
		image.EXPECT().GetImageDimensions(img, extension.Png).Return(&entity.Dimensions{Width: 1, Height: 1})

		dep := &gofpdf.Dependencies{
			Cache: cache,
			Image: image,
			Cfg:   cfg,
			Code:  code,
		}

		sut := gofpdf.New(dep)

		// Act
		dimensions, err := sut.GetDimensionsByQrCode(codeContent)

		// Assert
		assert.NotNil(t, dimensions)
		assert.Nil(t, err)
	})
	t.Run("when can find qrCode on cache, should return dimension", func(t *testing.T) {
		t.Parallel()
		img := &entity.Image{Bytes: []byte{1, 2, 3}}

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("qr-code-code", extension.Png).Return(img, nil).Once()

		code := mocks.NewCode(t)

		cfg := &entity.Config{
			Margins: &entity.Margins{
				Left:   10,
				Top:    10,
				Right:  10,
				Bottom: 10,
			},
		}

		image := mocks.NewImage(t)
		image.EXPECT().GetImageDimensions(img, extension.Png).Return(&entity.Dimensions{Width: 1, Height: 1})

		dep := &gofpdf.Dependencies{
			Cache: cache,
			Image: image,
			Cfg:   cfg,
			Code:  code,
		}

		sut := gofpdf.New(dep)

		// Act
		dimensions, err := sut.GetDimensionsByQrCode(codeContent)

		// Assert
		assert.NotNil(t, dimensions)
		assert.Nil(t, err)
	})
}

func TestProvider_GetDimensionsByImage(t *testing.T) {
	t.Parallel()
	t.Run("when cannot find image on cache and cannot load image, should return error", func(t *testing.T) {
		t.Parallel()
		// Arrange

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("test/assets/images/biplane.jpg", extension.Jpg).Return(nil, errors.New("anyError1")).Once()
		cache.EXPECT().LoadImage("test/assets/images/biplane.jpg", extension.Jpg).Return(errors.New("anyError1")).Once()

		dep := &gofpdf.Dependencies{
			Cache: cache,
		}

		sut := gofpdf.New(dep)

		// Act
		dimensions, err := sut.GetDimensionsByImage("test/assets/images/biplane.jpg")

		// Assert
		assert.Nil(t, dimensions)
		assert.NotNil(t, err)
	})

	t.Run("when can find image on cache, should return dimension", func(t *testing.T) {
		t.Parallel()
		img := &entity.Image{Bytes: []byte{1, 2, 3}}

		// Arrange

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage("test/assets/images/biplane.jpg", extension.Jpg).Return(img, nil).Once()

		image := mocks.NewImage(t)
		image.EXPECT().GetImageDimensions(img, extension.Jpg).Return(&entity.Dimensions{Width: 1, Height: 1})

		dep := &gofpdf.Dependencies{
			Cache: cache,
			Image: image,
		}

		sut := gofpdf.New(dep)

		// Act
		dimensions, err := sut.GetDimensionsByImage("test/assets/images/biplane.jpg")

		// Assert
		assert.Nil(t, err)
		assert.NotNil(t, dimensions)
	})
}

func TestProvider_GetDimensionsByImageByte(t *testing.T) {
	t.Parallel()
	t.Run("when invalid format is sent, should return an error", func(t *testing.T) {
		t.Parallel()
		// Arrange
		dep := &gofpdf.Dependencies{}

		sut := gofpdf.New(dep)

		// Act
		dimensions, err := sut.GetDimensionsByImageByte([]byte{1, 2, 3}, "jj")

		// Assert
		assert.Nil(t, dimensions)
		assert.NotNil(t, err)
	})

	t.Run("when bytes are sent, should return dimension", func(t *testing.T) {
		t.Parallel()
		img := fixture.ImageEntity()

		// Arrange

		image := mocks.NewImage(t)
		image.EXPECT().GetImageDimensions(&img, extension.Png).Return(&entity.Dimensions{Width: 1, Height: 1}).Once()

		dep := &gofpdf.Dependencies{
			Image: image,
		}

		sut := gofpdf.New(dep)

		// Act
		dimensions, err := sut.GetDimensionsByImageByte(img.Bytes, extension.Png)

		// Assert
		assert.Nil(t, err)
		assert.NotNil(t, dimensions)
	})
}

func TestProvider_SetCursor_AppliesPageMargins(t *testing.T) {
	t.Parallel()
	// Arrange
	fpdf := newPDF(t)
	fpdf.EXPECT().GetMargins().Return(5.0, 7.0, 0.0, 0.0)
	fpdf.EXPECT().SetXY(15.0, 27.0).Once()

	dep := &gofpdf.Dependencies{PDF: fpdf}
	sut := gofpdf.New(dep)

	// Act — cell coords are margin-relative; SetCursor must translate to absolute.
	sut.(core.PositionProvider).SetCursor(10.0, 20.0)

	// Assert: SetXY called with absolute coords (10+5, 20+7)
}

func TestProvider_DrawFilledCircle_AppliesPageMargins(t *testing.T) {
	t.Parallel()
	// Arrange
	fpdf := newPDF(t)
	fpdf.EXPECT().GetMargins().Return(5.0, 7.0, 0.0, 0.0)
	fpdf.EXPECT().GetFillColor().Return(0, 0, 0)
	fpdf.EXPECT().SetFillColor(1, 2, 3)
	// cx = cell.X + cell.Width/2 + left = 10 + 3 + 5 = 18
	// cy = cell.Y + cell.Height/2 + top = 20 + 3 + 7 = 30
	fpdf.EXPECT().Circle(18.0, 30.0, 3.0, "F").Once()
	fpdf.EXPECT().SetFillColor(0, 0, 0) // restore

	dep := &gofpdf.Dependencies{PDF: fpdf}
	sut := gofpdf.New(dep)

	// Act
	sut.(core.ShapeProvider).DrawFilledCircle(
		&entity.Cell{X: 10.0, Y: 20.0, Width: 6.0, Height: 6.0},
		&props.Color{Red: 1, Green: 2, Blue: 3},
	)
}

/*func TestProvider_AddImageFromFile(t *testing.T) {
	t.Run("when cannot find image in cache and cannot load image, should apply error message", func(t *testing.T) {
		t.Parallel()
		// Arrange
		file := "file.jpg"
		cell := &entity.Cell{}
		prop := fixture.RectProp()

		cache := mocks.NewCache(t)
		cache.EXPECT().GetImage(file, extension.Jpg)

		dep := &gofpdf.Dependencies{
			Cache: cache,
		}

		sut := gofpdf.New(dep)

		// Act
		sut.AddImageFromFile(file, cell, &prop)
	})
}
*/
