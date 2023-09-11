package lang

import (
	"fmt"
	"strconv"
	"unicode/utf8"
)

// Char is a character value.
type Char rune

// NewChar creates a new character value.
func NewChar(value rune) Char {
	return Char(value)
}

func (c Char) Equals(v interface{}) bool {
	switch v := v.(type) {
	case Char:
		return rune(c) == rune(v)
	default:
		return false
	}
}

func (c Char) Hash() uint32 {
	return uint32(rune(c))
}

// RuneFromCharLiteral returns the rune value from a character
// literal.
func RuneFromCharLiteral(lit string) (rune, error) {
	if len(lit) < 2 || lit[0] != '\\' {
		return 0, fmt.Errorf("too short or not a char literal: %s", lit)
	}

	char := lit[1:]

	// Handle special characters
	// \newline, \space, \tab, \formfeed, \backspace, and \return
	switch char {
	case "newline":
		char = "\n"
	case "space":
		char = " "
	case "tab":
		char = "\t"
	case "formfeed":
		char = "\f"
	case "backspace":
		char = "\b"
	case "return":
		char = "\r"
	}
	// Handle unicode characters
	if len(char) > 1 && char[0] == 'u' {
		unquoted, _, _, err := strconv.UnquoteChar("\\"+char, 0)
		if err != nil {
			return 0, fmt.Errorf("invalid unicode character: %s", char)
		}
		char = string(unquoted)
	}

	// if the character is more than one rune, it's invalid
	if utf8.RuneCountInString(char) != 1 {
		return 0, fmt.Errorf("unexpected rune count")
	}

	rn, _ := utf8.DecodeRuneInString(char)
	return rn, nil
}

// CharLiteralFromRune returns a character literal from a rune.
func CharLiteralFromRune(rn rune) string {
	switch rn {
	case '\n':
		return `\newline`
	case ' ':
		return `\space`
	case '\t':
		return `\tab`
	case '\f':
		return `\formfeed`
	case '\b':
		return `\backspace`
	case '\r':
		return `\return`
	}

	return fmt.Sprintf("\\%c", rn)
}

func CharAt(s string, idx int) Char {
	return NewChar([]rune(s)[idx])
}
