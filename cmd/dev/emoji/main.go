package main

import (
	"fmt"
	"os"

	paperpdf "github.com/avdoseferovic/paper/internal/paperpdf"
)

func main() {
	textFontBytes, err := os.ReadFile("docs/assets/fonts/arial-unicode-ms.ttf")
	if err != nil {
		panic(err)
	}

	emojiFontPath := os.Getenv("PAPER_EMOJI_FONT")
	if emojiFontPath == "" {
		emojiFontPath = "docs/assets/fonts/arial-unicode-ms.ttf"
	}

	emojiFontBytes, err := os.ReadFile(emojiFontPath)
	if err != nil {
		panic(err)
	}

	pdf := paperpdf.NewCustom(&paperpdf.InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		SizeStr:        "A4",
	})
	pdf.SetCompression(false)
	pdf.SetColorEmojiEnabled(true)
	pdf.AddUTF8FontFromBytes("Text", "", textFontBytes)
	pdf.AddUTF8FontFromBytes("Emoji", "", emojiFontBytes)
	if err := pdf.Error(); err != nil {
		panic(err)
	}

	pdf.AddPage()
	pdf.SetFont("Text", "", 20)
	pdf.Text(20, 30, "Paper emoji smoke test")
	pdf.Text(20, 45, "Emoji font: "+emojiFontPath)

	pdf.SetFont("Emoji", "", 20)
	colorEmojiEnabled := pdf.HasColorEmoji()

	pdf.SetFont("Text", "", 24)
	x, y := 20.0, 70.0
	before := "Here is some text "
	pdf.Text(x, y, before)
	x += pdf.GetStringWidth(before)

	pdf.SetFont("Emoji", "", 24)
	emoji := "😂"
	pdf.Text(x, y, emoji)
	x += pdf.GetStringWidth(emoji)

	pdf.SetFont("Text", "", 24)
	pdf.Text(x, y, " and back to text!")

	label := "Supplementary emoji:"
	pdf.SetFont("Text", "", 18)
	pdf.Text(20, 95, label)
	emojiX := 20 + pdf.GetStringWidth(label) + 5
	pdf.SetFont("Emoji", "", 18)
	pdf.Text(emojiX, 95, "😀 😃 😄 🚀")
	pdf.SetFont("Text", "", 18)
	pdf.Text(20, 110, fmt.Sprintf("Color emoji enabled: %v", colorEmojiEnabled))

	if err := pdf.OutputFileAndClose("/tmp/paper-emoji-smoke.pdf"); err != nil {
		panic(err)
	}
	fmt.Println("/tmp/paper-emoji-smoke.pdf")
}
