package recipe

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/DasDarki/nwm/internal/server"
)

type token struct {
	Text   string
	Quoted bool
	Raw    bool
}

func (t token) value() any {
	if t.Quoted {
		return t.Text
	}
	return server.ParseValue(t.Text)
}

type tokenized struct {
	tokens        []token
	trailingSpace bool
	unterminated  bool
}

func tokenize(line string) (tokenized, error) {
	var result tokenized
	runes := []rune(line)
	index := 0
	for index < len(runes) {
		for index < len(runes) && unicode.IsSpace(runes[index]) {
			index++
		}
		if index >= len(runes) {
			result.trailingSpace = true
			break
		}
		switch runes[index] {
		case '#':
			return result, nil
		case '"':
			end := index + 1
			for end < len(runes) && runes[end] != '"' {
				if runes[end] == '\\' {
					end++
				}
				end++
			}
			if end >= len(runes) {
				result.unterminated = true
				result.tokens = append(result.tokens, token{Text: string(runes[index+1:]), Quoted: true})
				return result, fmt.Errorf("unterminated double quote")
			}
			text, err := strconv.Unquote(string(runes[index : end+1]))
			if err != nil {
				return result, fmt.Errorf("invalid string %s", string(runes[index:end+1]))
			}
			result.tokens = append(result.tokens, token{Text: text, Quoted: true})
			index = end + 1
		case '\'':
			end := index + 1
			for end < len(runes) && runes[end] != '\'' {
				end++
			}
			if end >= len(runes) {
				result.unterminated = true
				result.tokens = append(result.tokens, token{Text: string(runes[index+1:]), Quoted: true})
				return result, fmt.Errorf("unterminated single quote")
			}
			result.tokens = append(result.tokens, token{Text: string(runes[index+1 : end]), Quoted: true})
			index = end + 1
		case '[', '{':
			end, err := matchBracket(runes, index)
			if err != nil {
				result.unterminated = true
				result.tokens = append(result.tokens, token{Text: string(runes[index:]), Raw: true})
				return result, err
			}
			result.tokens = append(result.tokens, token{Text: string(runes[index : end+1]), Raw: true})
			index = end + 1
		default:
			end := index
			for end < len(runes) && !unicode.IsSpace(runes[end]) {
				end++
			}
			result.tokens = append(result.tokens, token{Text: string(runes[index:end])})
			index = end
		}
		if index < len(runes) && !unicode.IsSpace(runes[index]) {
			return result, fmt.Errorf("expected a space after %q", result.tokens[len(result.tokens)-1].Text)
		}
	}
	return result, nil
}

func matchBracket(runes []rune, start int) (int, error) {
	depth := 0
	var quote rune
	for index := start; index < len(runes); index++ {
		character := runes[index]
		switch {
		case quote != 0:
			if character == '\\' && quote == '"' {
				index++
			} else if character == quote {
				quote = 0
			}
		case character == '"' || character == '\'':
			quote = character
		case character == '[' || character == '{':
			depth++
		case character == ']' || character == '}':
			depth--
			if depth == 0 {
				return index, nil
			}
		}
	}
	return 0, fmt.Errorf("unterminated %c", runes[start])
}

func QuoteWord(text string) string {
	if text == "" {
		return `""`
	}
	for _, character := range text {
		if unicode.IsSpace(character) || character == '"' || character == '\'' || character == '\\' {
			return strconv.Quote(text)
		}
	}
	if strings.ContainsAny(text[:1], "#[{") {
		return strconv.Quote(text)
	}
	return text
}

func formatValue(value any) string {
	if text, isString := value.(string); isString {
		if parsed, stillString := server.ParseValue(text).(string); stillString && parsed == text && QuoteWord(text) == text {
			return text
		}
		return strconv.Quote(text)
	}
	return server.FormatValue(value)
}
