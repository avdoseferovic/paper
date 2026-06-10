package core_test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/avdoseferovic/paper/pkg/core"
)

func ExampleDocument_Write() {
	var doc core.Document = core.NewPDF([]byte("%PDF-1.7\n"), nil)

	recorder := httptest.NewRecorder()
	recorder.Header().Set("Content-Type", "application/pdf")

	if _, err := doc.Write(recorder); err != nil {
		log.Fatal(err)
	}

	fmt.Println(recorder.Code == http.StatusOK)
	fmt.Println(recorder.Header().Get("Content-Type"))
	fmt.Print(recorder.Body.String())

	// Output:
	// true
	// application/pdf
	// %PDF-1.7
}
