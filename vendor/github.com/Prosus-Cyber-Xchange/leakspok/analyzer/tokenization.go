package analyzer

import (
	"unicode"
	"unicode/utf8"
)

// TokenIterator returns an iterator function for range-over-func that yields tokens with their positions
// It uses the same delimiters as the tokenize function and returns Token structs with Start and End positions
// Note: positions are relative to the original input (before escape removal)
// Making `/` a token delimiter, breaks CNPJ detection since CNPJ have the format `11.444.777/0001-61`.
// But it's a tradeoff with detecting other entities inside URLs and File Paths.
//
//nolint:gocognit
func TokenIterator(s []byte) func(yield func(Token) bool) {
	delimiters := map[rune]struct{}{
		',': {}, ';': {}, '!': {}, '?': {}, '(': {}, ')': {},
		'[': {}, ']': {}, '{': {}, '}': {}, '"': {}, '\'': {},
		'/': {}, '\\': {}, '\n': {}, '\t': {}, '\r': {},
	}

	escapeChar := '\\'
	escapeCharLen := utf8.RuneLen(escapeChar)

	control := map[rune]struct{}{
		'n': {},
		't': {},
		'r': {},
	}

	isDelimiter := func(r rune) bool {
		if unicode.IsSpace(r) {
			return true
		}
		_, ok := delimiters[r]
		return ok
	}

	skipDelimiters := func(start int) int {
		isEscaping := false
		for start < len(s) {
			r, size := utf8.DecodeRune(s[start:])
			if r == escapeChar {
				start += size
				isEscaping = true
				continue
			}
			if isEscaping {
				if _, found := control[r]; found {
					isEscaping = false
					start += size
					continue
				}
			}
			if !isDelimiter(r) {
				break
			}
			start += size
		}
		return start
	}

	findTokenEnd := func(start int) int {
		isEscaping := false
		end := start
		for end < len(s) {
			r, size := utf8.DecodeRune(s[end:])
			if isDelimiter(r) {
				break
			}
			if isEscaping {
				if _, found := control[r]; found {
					end -= escapeCharLen // backtrack
					break
				}
			}
			isEscaping = r == escapeChar
			end += size
		}
		return end
	}

	return func(yield func(Token) bool) {
		start := 0
		for start < len(s) {
			start = skipDelimiters(start)
			if start >= len(s) {
				break
			}
			end := findTokenEnd(start)
			if !yield(Token{Start: start, End: end, Content: s[start:end]}) {
				return
			}
			start = end
		}
	}
}
