/*
 * Copyright 2016-2023 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package schema

import (
	"github.com/dgraph-io/dgraph/lex"
)

// Constants representing type of different graphql lexed items.
const (
	itemText            lex.ItemType = 5 + iota // plain text
	itemNumber                                  // number
	itemLeftCurl                                // left curly bracket
	itemRightCurl                               // right curly bracket
	itemColon                                   // colon
	itemLeftRound                               // left round bracket
	itemRightRound                              // right round bracket
	itemAt                                      // '@'
	itemComma                                   // ','
	itemNewLine                                 // carriage-return or line-feed.
	itemDot                                     // '.'
	itemLeftSquare                              // '['
	itemRightSquare                             // ']'
	itemExclamationMark                         // '!'
	itemQuote                                   // double quote char: '"'
	itemQuotedText                              // See Lexer.LexQuotedString()
)

func lexText(l *lex.Lexer) lex.StateFn {
Loop:
	for {
		switch r := l.Next(); {
		case r == lex.EOF:
			break Loop
		case isNameBegin(r):
			l.Backup()
			return lexWord
		case isSpace(r):
			l.Ignore()
		case lex.IsEndOfLine(r):
			l.Emit(itemNewLine)
		case r == '.':
			l.Emit(itemDot)
		case r == '#':
			return lexTextComment
		case r == ',':
			l.Emit(itemComma)
		case r == '<':
			if err := lex.IRIRef(l, itemText); err != nil {
				return l.Errorf("Invalid schema: %v", err)
			}
		case r == '{':
			l.Emit(itemLeftCurl)
		case r == '}':
			l.Emit(itemRightCurl)
		case r == '(':
			l.Emit(itemLeftRound)
		case r == ')':
			l.Emit(itemRightRound)
		case r == ':':
			l.Emit(itemColon)
		case r == '@':
			l.Emit(itemAt)
		case r == '[':
			l.Emit(itemLeftSquare)
		case r == ']':
			l.Emit(itemRightSquare)
		case r == '!':
			l.Emit(itemExclamationMark)
		case r == '_':
			// Predicates can start with _.
			return lexWord
		case isDigit(r):
			nextRunes := l.PeekTwo()
			if r == '0' && isHexseparator(nextRunes[0]) && isHexadecimal(nextRunes[1]) {
				l.Backup()
				return lexHexNumber
			} else {
				l.Backup()
				return lexNumber
			}
		case r == '"':
			if err := l.LexQuotedString(); err != nil {
				return l.Errorf("Invalid schema: %v", err)
			}
			l.Emit(itemQuotedText)
		default:
			return l.Errorf("Invalid schema. Unexpected %s", l.Input[l.Start:l.Pos])
		}
	}
	if l.Pos > l.Start {
		l.Emit(itemText)
	}
	l.Emit(lex.ItemEOF)
	return nil
}

func lexWord(l *lex.Lexer) lex.StateFn {
	for {
		// The caller already checked isNameBegin, and absorbed one rune.
		r := l.Next()
		if isNameSuffix(r) {
			continue
		}
		l.Backup()
		l.Emit(itemText)
		break
	}
	return lexText
}

func lexNumber(l *lex.Lexer) lex.StateFn {
	for {
		// The caller already checked isNumber, and absorbed one rune.
		r := l.Next()
		if isDigit(r) {
			continue
		}
		l.Backup()
		l.Emit(itemNumber)
		break
	}
	return lexText
}

func lexHexNumber(l *lex.Lexer) lex.StateFn {
	// It satisfies 0[xX] then process the input as hexadecimal.
	l.Next()
	l.Next() // Absorb 0[xX]
	for {
		// The caller already checked isHexadecimal, and absorbed one rune.
		r := l.Next()
		if isHexadecimal(r) {
			continue
		}
		l.Backup()
		l.Emit(itemNumber)
		break
	}
	return lexText
}

// lexTextComment lexes a comment text inside a schema.
func lexTextComment(l *lex.Lexer) lex.StateFn {
	for {
		r := l.Next()
		if r == lex.EOF {
			l.Ignore()
			l.Emit(lex.ItemEOF)
			break
		}
		if !lex.IsEndOfLine(r) {
			continue
		}
		l.Ignore()
		l.Emit(itemNewLine)
		break
	}
	return lexText
}

// isNameBegin returns true if the rune is an alphabet.
func isNameBegin(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	default:
		return false
	}
}

func isNameSuffix(r rune) bool {
	if isNameBegin(r) {
		return true
	}
	if r >= '0' && r <= '9' {
		return true
	}
	if r == '_' || r == '.' || r == '-' { // Use by freebase.
		return true
	}
	return false
}

// isHexadecimal returns true if the rune is hexadecimal.
func isHexadecimal(r rune) bool {
	switch {
	case r >= 'a' && r <= 'f':
		return true
	case r >= 'A' && r <= 'F':
		return true
	case isDigit(r):
		return true
	default:
		return false
	}
}

func isHexseparator(r rune) bool {
	return r == 'x' || r == 'X'
}

// isDigit returns true if the rune is digit.
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// isSpace returns true if the rune is a tab or space.
func isSpace(r rune) bool {
	return r == '\u0009' || r == '\u0020'
}
