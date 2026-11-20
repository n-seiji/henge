package sqlguard

import (
	"errors"
	"strings"
	"unicode"
)

// An allowlist of leading keywords, not a denylist of statements: a dialect's
// unknown write statement must fail closed.
var readOnlyHeads = map[string]bool{
	"SELECT": true, "WITH": true, "SHOW": true, "DESCRIBE": true, "DESC": true,
	"EXPLAIN": true, "VALUES": true, "TABLE": true,
}

// Words that write or lock wherever they appear, such as a data-modifying CTE,
// SELECT ... INTO, SELECT ... FOR UPDATE, or EXPLAIN ANALYZE, which runs the
// statement. As a function name (followed by "(") they are allowed, so
// REPLACE(text, ...) still reads.
var writingWords = map[string]bool{
	"INSERT": true, "UPDATE": true, "DELETE": true, "MERGE": true, "UPSERT": true,
	"REPLACE": true, "CREATE": true, "ALTER": true, "DROP": true, "TRUNCATE": true,
	"GRANT": true, "REVOKE": true, "LOCK": true, "CALL": true, "EXEC": true,
	"EXECUTE": true, "COPY": true, "INTO": true, "ANALYZE": true, "VACUUM": true,
	"REFRESH": true, "LOAD": true, "EXPORT": true, "DECLARE": true, "SET": true,
	"BEGIN": true, "COMMIT": true, "ROLLBACK": true, "SHARE": true, "NOWAIT": true,
}

var (
	errMultiple = errors.New("複数の文はまとめて実行できません")
	errEmpty    = errors.New("SQL が空です")
	errUnclosed = errors.New("閉じていない文字列かコメントがあります")
)

type token struct {
	word     string
	call     bool
	semi     bool
	position int
}

func CheckReadOnly(sql string) error {
	tokens, err := tokenize(sql)
	if err != nil {
		return err
	}
	var words []token
	for i, tok := range tokens {
		if tok.semi {
			if i != len(tokens)-1 {
				return errMultiple
			}
			continue
		}
		words = append(words, tok)
	}
	if len(words) == 0 {
		return errEmpty
	}
	if head := words[0].word; !readOnlyHeads[head] {
		return &blockedError{word: head}
	}
	for _, tok := range words {
		if writingWords[tok.word] && !tok.call {
			return &blockedError{word: tok.word}
		}
	}
	return nil
}

type blockedError struct{ word string }

func (e *blockedError) Error() string {
	return e.word + " を含む SQL は、更新やロックをする可能性があるため実行できません"
}

func tokenize(sql string) ([]token, error) {
	var tokens []token
	runes := []rune(sql)
	for i := 0; i < len(runes); {
		r := runes[i]
		switch {
		case unicode.IsSpace(r), r == '(' || r == ')' || r == ',':
			i++
		case r == '-' && at(runes, i+1) == '-', r == '#':
			for i < len(runes) && runes[i] != '\n' {
				i++
			}
		case r == '/' && at(runes, i+1) == '*':
			end := indexFrom(runes, i+2, "*/")
			if end < 0 {
				return nil, errUnclosed
			}
			i = end + 2
		case r == '\'' || r == '"' || r == '`':
			end, ok := closeQuote(runes, i)
			if !ok {
				return nil, errUnclosed
			}
			i = end
		case r == '$':
			end, ok := closeDollarQuote(runes, i)
			if !ok {
				return nil, errUnclosed
			}
			i = end
		case r == ';':
			tokens = append(tokens, token{semi: true, position: i})
			i++
		case isWordStart(r):
			start := i
			for i < len(runes) && isWordPart(runes[i]) {
				i++
			}
			next := i
			for next < len(runes) && unicode.IsSpace(runes[next]) {
				next++
			}
			tokens = append(tokens, token{word: strings.ToUpper(string(runes[start:i])), call: at(runes, next) == '(', position: start})
		default:
			i++
		}
	}
	return tokens, nil
}

func at(runes []rune, i int) rune {
	if i < len(runes) {
		return runes[i]
	}
	return 0
}

func indexFrom(runes []rune, from int, needle string) int {
	index := strings.Index(string(runes[from:]), needle)
	if index < 0 {
		return -1
	}
	return from + len([]rune(string(runes[from:])[:index]))
}

// Doubled quotes and backslash escapes are both skipped, so neither dialect's
// escaping can end the literal early and expose its contents as keywords.
func closeQuote(runes []rune, start int) (int, bool) {
	quote := runes[start]
	for i := start + 1; i < len(runes); i++ {
		switch runes[i] {
		case '\\':
			i++
		case quote:
			if at(runes, i+1) == quote {
				i++
				continue
			}
			return i + 1, true
		}
	}
	return 0, false
}

func closeDollarQuote(runes []rune, start int) (int, bool) {
	end := start + 1
	for end < len(runes) && isWordPart(runes[end]) && runes[end] != '$' {
		end++
	}
	if at(runes, end) != '$' {
		// A bare "$1" placeholder, not a dollar-quoted string.
		return end, true
	}
	tag := string(runes[start : end+1])
	close := indexFrom(runes, end+1, tag)
	if close < 0 {
		return 0, false
	}
	return close + len([]rune(tag)), true
}

func isWordStart(r rune) bool { return r == '_' || unicode.IsLetter(r) }

func isWordPart(r rune) bool {
	return r == '_' || r == '$' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
