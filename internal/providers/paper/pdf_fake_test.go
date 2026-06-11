package paper_test

import (
	"io"
	"reflect"
	"strconv"
	"time"

	mock "github.com/avdoseferovic/paper/internal/mocktest"
	pdf "github.com/avdoseferovic/paper/internal/pdf"
)

type pdfMock struct {
	mock.Mock
}

type pdfExpecter struct {
	mock *mock.Mock
}

type pdfCall struct {
	*mock.Call
}

func (c *pdfCall) Return(values ...any) *pdfCall {
	c.Call.Return(values...)
	return c
}

func (c *pdfCall) Run(fn any) *pdfCall {
	rv := reflect.ValueOf(fn)
	c.Call.Run(func(args mock.Arguments) {
		in := make([]reflect.Value, rv.Type().NumIn())
		for i := range in {
			arg := args.Get(i)
			value := reflect.ValueOf(arg)
			param := rv.Type().In(i)
			if value.Type().AssignableTo(param) {
				in[i] = value
			} else {
				in[i] = value.Convert(param)
			}
		}
		rv.Call(in)
	})
	return c
}

func (c *pdfCall) Once() *pdfCall {
	c.Call.Once()
	return c
}

func (c *pdfCall) Times(n int) *pdfCall {
	c.Call.Times(n)
	return c
}

func (c *pdfCall) Maybe() *pdfCall {
	c.Call.Maybe()
	return c
}

func newPDF(t interface {
	mock.TestingT
	Helper()
	Cleanup(func())
},
) *pdfMock {
	t.Helper()
	m := &pdfMock{}
	m.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *pdfMock) EXPECT() *pdfExpecter {
	return &pdfExpecter{mock: &m.Mock}
}

func (m *pdfMock) AssertNotCalled(t mock.TestingT, method string, args ...any) bool {
	return m.Mock.AssertNotCalled(t, method, args...)
}

func (e *pdfExpecter) AddPage(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("AddPage", args...)}
}

func (e *pdfExpecter) Bookmark(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("Bookmark", args...)}
}

func (e *pdfExpecter) Circle(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("Circle", args...)}
}

func (e *pdfExpecter) ClearError(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("ClearError", args...)}
}

func (e *pdfExpecter) ClipEnd(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("ClipEnd", args...)}
}

func (e *pdfExpecter) ClipRect(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("ClipRect", args...)}
}

func (e *pdfExpecter) GetFillColor(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("GetFillColor", args...)}
}

func (e *pdfExpecter) GetMargins(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("GetMargins", args...)}
}

func (e *pdfExpecter) GetStringWidth(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("GetStringWidth", args...)}
}

func (e *pdfExpecter) Image(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("Image", args...)}
}

func (e *pdfExpecter) Line(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("Line", args...)}
}

func (e *pdfExpecter) Link(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("Link", args...)}
}

func (e *pdfExpecter) LinkString(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("LinkString", args...)}
}

func (e *pdfExpecter) Ln(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("Ln", args...)}
}

func (e *pdfExpecter) Output(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("Output", args...)}
}

func (e *pdfExpecter) PageNo(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("PageNo", args...)}
}

func (e *pdfExpecter) Rect(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("Rect", args...)}
}

func (e *pdfExpecter) RegisterImageOptionsReader(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("RegisterImageOptionsReader", args...)}
}

func (e *pdfExpecter) SetAlpha(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetAlpha", args...)}
}

func (e *pdfExpecter) SetAuthor(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetAuthor", args...)}
}

func (e *pdfExpecter) SetCompression(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetCompression", args...)}
}

func (e *pdfExpecter) SetCreationDate(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetCreationDate", args...)}
}

func (e *pdfExpecter) SetCreator(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetCreator", args...)}
}

func (e *pdfExpecter) SetDashPattern(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetDashPattern", args...)}
}

func (e *pdfExpecter) SetDrawColor(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetDrawColor", args...)}
}

func (e *pdfExpecter) SetFillColor(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetFillColor", args...)}
}

func (e *pdfExpecter) SetFont(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetFont", args...)}
}

func (e *pdfExpecter) SetFontSize(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetFontSize", args...)}
}

func (e *pdfExpecter) SetFontStyle(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetFontStyle", args...)}
}

func (e *pdfExpecter) SetHomeXY(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetHomeXY", args...)}
}

func (e *pdfExpecter) SetKeywords(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetKeywords", args...)}
}

func (e *pdfExpecter) SetLineWidth(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetLineWidth", args...)}
}

func (e *pdfExpecter) SetLink(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetLink", args...)}
}

func (e *pdfExpecter) SetProtection(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetProtection", args...)}
}

func (e *pdfExpecter) SetProtectionAlgorithm(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetProtectionAlgorithm", args...)}
}

func (e *pdfExpecter) SetSubject(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetSubject", args...)}
}

func (e *pdfExpecter) SetTextColor(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetTextColor", args...)}
}

func (e *pdfExpecter) SetTitle(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetTitle", args...)}
}

func (e *pdfExpecter) SetXY(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("SetXY", args...)}
}

func (e *pdfExpecter) Text(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("Text", args...)}
}

func (e *pdfExpecter) TransformBegin(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("TransformBegin", args...)}
}

func (e *pdfExpecter) TransformEnd(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("TransformEnd", args...)}
}

func (e *pdfExpecter) TransformRotate(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("TransformRotate", args...)}
}

func (e *pdfExpecter) UnicodeTranslatorFromDescriptor(args ...any) *pdfCall {
	return &pdfCall{Call: e.mock.On("UnicodeTranslatorFromDescriptor", args...)}
}

func (m *pdfMock) AddLink() int {
	ret := m.Called()
	return ret.Get(0).(int)
}

func (m *pdfMock) AddPage() {
	m.Called()
}

func (m *pdfMock) AddUTF8FontFromBytes(familyStr, styleStr string, bytes []byte) {
	m.Called(familyStr, styleStr, bytes)
}

func (m *pdfMock) Bookmark(txtStr string, level int, y float64) {
	m.Called(txtStr, level, y)
}

func (m *pdfMock) Circle(x, y, r float64, styleStr string) {
	m.Called(x, y, r, styleStr)
}

func (m *pdfMock) ClearError() {
	m.Called()
}

func (m *pdfMock) ClipEnd() {
	m.Called()
}

func (m *pdfMock) ClipRect(x, y, w, h float64, outline bool) {
	m.Called(x, y, w, h, outline)
}

func (m *pdfMock) GetFillColor() (int, int, int) {
	ret := m.Called()
	return ret.Get(0).(int), ret.Get(1).(int), ret.Get(2).(int)
}

func (m *pdfMock) GetMargins() (float64, float64, float64, float64) {
	ret := m.Called()
	return ret.Get(0).(float64), ret.Get(1).(float64), ret.Get(2).(float64), ret.Get(3).(float64)
}

func (m *pdfMock) GetStringWidth(s string) float64 {
	ret := m.Called(s)
	return asFloat64(ret.Get(0))
}

func (m *pdfMock) GetXY() (float64, float64) {
	ret := m.Called()
	return ret.Get(0).(float64), ret.Get(1).(float64)
}

func (m *pdfMock) Image(imageNameStr string, x, y, w, h float64, flow bool, tp string, link int, linkStr string) {
	m.Called(imageNameStr, x, y, w, h, flow, tp, link, linkStr)
}

func (m *pdfMock) Line(x1, y1, x2, y2 float64) {
	m.Called(x1, y1, x2, y2)
}

func (m *pdfMock) Link(x, y, w, h float64, link int) {
	m.Called(x, y, w, h, link)
}

func (m *pdfMock) LinkString(x, y, w, h float64, linkStr string) {
	m.Called(x, y, w, h, linkStr)
}

func (m *pdfMock) Ln(h float64) {
	m.Called(h)
}

func (m *pdfMock) Output(w io.Writer) error {
	ret := m.Called(w)
	return ret.Error(0)
}

func (m *pdfMock) PageNo() int {
	ret := m.Called()
	return ret.Get(0).(int)
}

func (m *pdfMock) Rect(x, y, w, h float64, styleStr string) {
	m.Called(x, y, w, h, styleStr)
}

func (m *pdfMock) RegisterImageOptionsReader(imgName string, options pdf.ImageOptions, r io.Reader) *pdf.ImageInfoType {
	ret := m.Called(imgName, options, r)
	info, _ := ret.Get(0).(*pdf.ImageInfoType)
	return info
}

func (m *pdfMock) SetAlpha(alpha float64, blendModeStr string) {
	m.Called(alpha, blendModeStr)
}

func (m *pdfMock) SetAuthor(authorStr string, isUTF8 bool) {
	m.Called(authorStr, isUTF8)
}

func (m *pdfMock) SetCompression(compress bool) {
	m.Called(compress)
}

func (m *pdfMock) SetCreationDate(tm time.Time) {
	m.Called(tm)
}

func (m *pdfMock) SetCreator(creatorStr string, isUTF8 bool) {
	m.Called(creatorStr, isUTF8)
}

func (m *pdfMock) SetDashPattern(dashArray []float64, dashPhase float64) {
	m.Called(dashArray, dashPhase)
}

func (m *pdfMock) SetDrawColor(r, g, b int) {
	m.Called(r, g, b)
}

func (m *pdfMock) SetFillColor(r, g, b int) {
	m.Called(r, g, b)
}

func (m *pdfMock) SetFont(familyStr, styleStr string, size float64) {
	m.Called(familyStr, styleStr, size)
}

func (m *pdfMock) SetFontSize(size float64) {
	m.Called(size)
}

func (m *pdfMock) SetFontStyle(styleStr string) {
	m.Called(styleStr)
}

func (m *pdfMock) SetHomeXY() {
	m.Called()
}

func (m *pdfMock) SetKeywords(keywordsStr string, isUTF8 bool) {
	m.Called(keywordsStr, isUTF8)
}

func (m *pdfMock) SetLineWidth(width float64) {
	m.Called(width)
}

func (m *pdfMock) SetLink(link int, y float64, page int) {
	m.Called(link, y, page)
}

func (m *pdfMock) SetProtection(actionFlag byte, userPassStr, ownerPassStr string) {
	m.Called(actionFlag, userPassStr, ownerPassStr)
}

func (m *pdfMock) SetProtectionAlgorithm(algorithm pdf.ProtectionAlgorithm) {
	m.Called(algorithm)
}

func (m *pdfMock) SetSubject(subjectStr string, isUTF8 bool) {
	m.Called(subjectStr, isUTF8)
}

func (m *pdfMock) SetTextColor(r, g, b int) {
	m.Called(r, g, b)
}

func (m *pdfMock) SetTitle(titleStr string, isUTF8 bool) {
	m.Called(titleStr, isUTF8)
}

func (m *pdfMock) SetXY(x, y float64) {
	m.Called(x, y)
}

func (m *pdfMock) Text(x, y float64, txtStr string) {
	m.Called(x, y, txtStr)
}

func (m *pdfMock) TransformBegin() {
	m.Called()
}

func (m *pdfMock) TransformEnd() {
	m.Called()
}

func (m *pdfMock) TransformRotate(angle, x, y float64) {
	m.Called(angle, x, y)
}

func (m *pdfMock) UnicodeTranslatorFromDescriptor(cpStr string) func(string) string {
	ret := m.Called(cpStr)
	return ret.Get(0).(func(string) string)
}

func asFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int8:
		return float64(n)
	case int16:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case uint:
		return float64(n)
	case uint8:
		return float64(n)
	case uint16:
		return float64(n)
	case uint32:
		return float64(n)
	case uint64:
		return float64(n)
	case string:
		f, err := strconv.ParseFloat(n, 64)
		if err == nil {
			return f
		}
	}
	panic("pdfMock: return value is not numeric")
}
