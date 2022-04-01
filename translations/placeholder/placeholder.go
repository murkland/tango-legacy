package placeholder

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// HACK: This is required to support go1.18 for now.
func placeholder() {
	p := message.NewPrinter(language.AmericanEnglish)
	p.Printf("SELECT_ROM")
	p.Printf("ENTER_MATCHMAKING_CODE")
}
