package cellwriter_test

import mock "github.com/avdoseferovic/paper/internal/mocktest"

type pdfMock struct {
	mock.Mock
}

type pdfExpecter struct {
	mock *mock.Mock
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

func (e *pdfExpecter) CellFormat(args ...any) *mock.Call {
	return e.mock.On("CellFormat", args...)
}

func (e *pdfExpecter) ClosePath(args ...any) *mock.Call {
	return e.mock.On("ClosePath", args...)
}

func (e *pdfExpecter) CurveBezierCubicTo(args ...any) *mock.Call {
	return e.mock.On("CurveBezierCubicTo", args...)
}

func (e *pdfExpecter) DrawPath(args ...any) *mock.Call {
	return e.mock.On("DrawPath", args...)
}

func (e *pdfExpecter) GetDrawColor(args ...any) *mock.Call {
	return e.mock.On("GetDrawColor", args...)
}

func (e *pdfExpecter) GetFillColor(args ...any) *mock.Call {
	return e.mock.On("GetFillColor", args...)
}

func (e *pdfExpecter) GetLineWidth(args ...any) *mock.Call {
	return e.mock.On("GetLineWidth", args...)
}

func (e *pdfExpecter) GetXY(args ...any) *mock.Call {
	return e.mock.On("GetXY", args...)
}

func (e *pdfExpecter) Line(args ...any) *mock.Call {
	return e.mock.On("Line", args...)
}

func (e *pdfExpecter) LineTo(args ...any) *mock.Call {
	return e.mock.On("LineTo", args...)
}

func (e *pdfExpecter) MoveTo(args ...any) *mock.Call {
	return e.mock.On("MoveTo", args...)
}

func (e *pdfExpecter) Rect(args ...any) *mock.Call {
	return e.mock.On("Rect", args...)
}

func (e *pdfExpecter) SetAlpha(args ...any) *mock.Call {
	return e.mock.On("SetAlpha", args...)
}

func (e *pdfExpecter) SetDashPattern(args ...any) *mock.Call {
	return e.mock.On("SetDashPattern", args...)
}

func (e *pdfExpecter) SetDrawColor(args ...any) *mock.Call {
	return e.mock.On("SetDrawColor", args...)
}

func (e *pdfExpecter) SetFillColor(args ...any) *mock.Call {
	return e.mock.On("SetFillColor", args...)
}

func (e *pdfExpecter) SetLineWidth(args ...any) *mock.Call {
	return e.mock.On("SetLineWidth", args...)
}

func (m *pdfMock) CellFormat(w, h float64, txtStr, borderStr string, ln int, alignStr string, fill bool, link int, linkStr string) {
	m.Called(w, h, txtStr, borderStr, ln, alignStr, fill, link, linkStr)
}

func (m *pdfMock) ClosePath() {
	m.Called()
}

func (m *pdfMock) CurveBezierCubicTo(cx0, cy0, cx1, cy1, x, y float64) {
	m.Called(cx0, cy0, cx1, cy1, x, y)
}

func (m *pdfMock) DrawPath(styleStr string) {
	m.Called(styleStr)
}

func (m *pdfMock) GetDrawColor() (int, int, int) {
	ret := m.Called()
	return ret.Get(0).(int), ret.Get(1).(int), ret.Get(2).(int)
}

func (m *pdfMock) GetFillColor() (int, int, int) {
	ret := m.Called()
	return ret.Get(0).(int), ret.Get(1).(int), ret.Get(2).(int)
}

func (m *pdfMock) GetLineWidth() float64 {
	ret := m.Called()
	return ret.Get(0).(float64)
}

func (m *pdfMock) GetXY() (float64, float64) {
	ret := m.Called()
	return ret.Get(0).(float64), ret.Get(1).(float64)
}

func (m *pdfMock) Line(x1, y1, x2, y2 float64) {
	m.Called(x1, y1, x2, y2)
}

func (m *pdfMock) LineTo(x, y float64) {
	m.Called(x, y)
}

func (m *pdfMock) MoveTo(x, y float64) {
	m.Called(x, y)
}

func (m *pdfMock) Rect(x, y, w, h float64, styleStr string) {
	m.Called(x, y, w, h, styleStr)
}

func (m *pdfMock) SetAlpha(alpha float64, blendModeStr string) {
	m.Called(alpha, blendModeStr)
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

func (m *pdfMock) SetLineWidth(width float64) {
	m.Called(width)
}
