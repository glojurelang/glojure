package reader

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"unicode"

	"github.com/glojurelang/glojure/value"
)

type trackingRuneScanner struct {
	rs io.RuneScanner

	filename       string
	nextRuneLine   int
	nextRuneColumn int

	// keep track of the last two runes read, most recent last.
	history []value.Pos
}

func newTrackingRuneScanner(rs io.RuneScanner, filename string) *trackingRuneScanner {
	if filename == "" {
		filename = "<unknown-file>"
	}
	return &trackingRuneScanner{
		rs:             rs,
		filename:       filename,
		nextRuneLine:   1,
		nextRuneColumn: 1,
		history:        make([]value.Pos, 0, 2),
	}
}

func (r *trackingRuneScanner) ReadRune() (rune, int, error) {
	rn, size, err := r.rs.ReadRune()
	if err != nil {
		return rn, size, err
	}
	if len(r.history) == 2 {
		r.history[0] = r.history[1]
		r.history = r.history[:1]
	}
	r.history = append(r.history, value.Pos{
		Filename: r.filename,
		Line:     r.nextRuneLine,
		Column:   r.nextRuneColumn,
	})
	if rn == '\n' {
		r.nextRuneLine++
		r.nextRuneColumn = 1
	} else {
		r.nextRuneColumn++
	}
	return rn, size, nil
}

func (r *trackingRuneScanner) UnreadRune() error {
	err := r.rs.UnreadRune()
	if err != nil {
		return err
	}
	if len(r.history) == 0 {
		panic("UnreadRune called when history is empty")
	}
	lastPos := r.history[len(r.history)-1]
	r.history = r.history[:len(r.history)-1]
	r.nextRuneLine = lastPos.Line
	r.nextRuneColumn = lastPos.Column
	return nil
}

// pos returns the position of the next rune that will be read.
func (r *trackingRuneScanner) pos() value.Pos {
	if len(r.history) == 0 {
		return value.Pos{
			Filename: r.filename,
			Line:     r.nextRuneLine,
			Column:   r.nextRuneColumn,
		}
	}
	return r.history[len(r.history)-1]
}

type Reader struct {
	rs *trackingRuneScanner

	posStack []value.Pos
}

type options struct {
	filename string
}

// Option represents an option that can be passed to New.
type Option func(*options)

// WithFilename sets the filename to be associated with the input.
func WithFilename(filename string) Option {
	return func(o *options) {
		o.filename = filename
	}
}

func New(r io.RuneScanner, opts ...Option) *Reader {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return &Reader{
		rs: newTrackingRuneScanner(r, o.filename),
	}
}

// Read reads all expressions from the input until a read error occurs
// or io.EOF is reached. A final io.EOF will not be returned if the
// input ends with a valid expression or if it contains no expressions
// at all.
func (r *Reader) ReadAll() ([]value.Value, error) {
	var nodes []value.Value
	for {
		_, err := r.next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, r.error("error reading input: %w", err)
		}
		r.rs.UnreadRune()
		node, err := r.readExpr()
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// error returns a formatted error that includes the current position
// of the scanner.
func (r *Reader) error(format string, args ...interface{}) error {
	pos := r.rs.pos()
	prefix := fmt.Sprintf("%s:%d:%d: ", pos.Filename, pos.Line, pos.Column)
	return fmt.Errorf(prefix+format, args...)
}

// popSection returns the last section read, ending at the current
// input, and pops it off the stack.
func (r *Reader) popSection() value.Section {
	sec := value.Section{
		StartPos: r.posStack[len(r.posStack)-1],
		EndPos:   r.rs.pos(),
	}
	r.posStack = r.posStack[:len(r.posStack)-1]
	return sec
}

// pushSection pushes a new section onto the stack, starting at the
// current input.
func (r *Reader) pushSection() {
	r.posStack = append(r.posStack, r.rs.pos())
}

// next returns the next rune that is not whitespace or a comment.
func (r *Reader) next() (rune, error) {
	for {
		rn, _, err := r.rs.ReadRune()
		if err != nil {
			return 0, r.error("error reading input: %w", err)
		}
		if unicode.IsSpace(rn) {
			continue
		}
		if rn == ';' {
			for {
				rn, _, err := r.rs.ReadRune()
				if err != nil {
					return 0, r.error("error reading input: %w", err)
				}
				if rn == '\n' {
					break
				}
			}
			continue
		}
		return rn, nil
	}
}

func (r *Reader) readExpr() (value.Value, error) {
	rune, err := r.next()
	if err != nil {
		return nil, err
	}

	r.pushSection()
	switch rune {
	case '(':
		return r.readList()
	case ')':
		return nil, r.error("unexpected ')'")
	case '[':
		return r.readVector()
	case ']':
		return nil, r.error("unexpected ']'")
	case '"':
		return r.readString()
	case '\\':
		return r.readChar()
	case '\'':
		return r.readQuote()
	case '`':
		return r.readQuasiquote()
	case '~':
		return r.readUnquote()
	case ',': // TODO: treat as whitespace, as in Clojure
		return nil, r.error("unquote not implemented")
	case '#':
		return nil, r.error("reader macros not implemented")
	case ':':
		return r.readKeyword()
	default:
		r.rs.UnreadRune()
		return r.readSymbol()
	}
}

func (r *Reader) readList() (value.Value, error) {
	var nodes []value.Value
	for {
		rune, err := r.next()
		if err != nil {
			return nil, err
		}
		if unicode.IsSpace(rune) {
			continue
		}
		if rune == ')' {
			break
		}

		r.rs.UnreadRune()
		node, err := r.readExpr()
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return value.NewList(nodes, value.WithSection(r.popSection())), nil
}

func (r *Reader) readVector() (value.Value, error) {
	var nodes []value.Value
	for {
		rune, err := r.next()
		if err != nil {
			return nil, err
		}
		if unicode.IsSpace(rune) {
			continue
		}
		if rune == ']' {
			break
		}

		r.rs.UnreadRune()
		node, err := r.readExpr()
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return value.NewVector(nodes, value.WithSection(r.popSection())), nil
}

func (r *Reader) readString() (value.Value, error) {
	var str string
	for {
		rune, _, err := r.rs.ReadRune()
		if err != nil {
			return nil, r.error("error reading string: %w", err)
		}
		// handle escape sequences
		if rune == '\\' {
			rune, _, err = r.rs.ReadRune()
			if err != nil {
				return nil, r.error("error reading string: %w", err)
			}
			switch rune {
			case 'n':
				rune = '\n'
			case 't':
				rune = '\t'
			case 'r':
				rune = '\r'
			case '"':
				rune = '"'
			case '\\':
				rune = '\\'
			default:
				return nil, r.error("invalid escape sequence: \\%c", rune)
			}
		} else if rune == '"' {
			break
		}
		str += string(rune)
	}
	return value.NewStr(str, value.WithSection(r.popSection())), nil
}

func (r *Reader) readChar() (value.Value, error) {
	var char string
	for {
		rn, _, err := r.rs.ReadRune()
		if err != nil {
			return nil, r.error("error reading character: %w", err)
		}
		if unicode.IsSpace(rn) || rn == '(' || rn == ')' || rn == '[' || rn == ']' || rn == '\\' {
			r.rs.UnreadRune()
			break
		}
		char += string(rn)
	}

	rn, err := value.RuneFromCharLiteral("\\" + char)
	if err != nil {
		return nil, r.error("invalid character literal: %w", err)
	}
	return value.NewChar(rn, value.WithSection(r.popSection())), nil
}

func (r *Reader) readQuoteType(form string) (value.Value, error) {
	node, err := r.readExpr()
	if err != nil {
		return nil, err
	}
	section := r.popSection()
	items := []value.Value{
		value.NewSymbol(form, value.WithSection(value.Section{StartPos: section.StartPos, EndPos: node.Pos()})),
		node,
	}
	return value.NewList(items, value.WithSection(section)), nil
}

func (r *Reader) readQuote() (value.Value, error) {
	return r.readQuoteType("quote")
}

func (r *Reader) readQuasiquote() (value.Value, error) {
	return r.readQuoteType("quasiquote")
}

func (r *Reader) readUnquote() (value.Value, error) {
	rn, _, err := r.rs.ReadRune()
	if err != nil {
		return nil, r.error("error reading input: %w", err)
	}
	if rn == '@' {
		return r.readQuoteType("splice-unquote")
	}

	r.rs.UnreadRune()
	return r.readQuoteType("unquote")
}

func (r *Reader) readSymbol() (value.Value, error) {
	var sym string
	for {
		rn, _, err := r.rs.ReadRune()
		if err != nil {
			return nil, r.error("error reading symbol: %w", err)
		}
		if unicode.IsSpace(rn) || rn == '(' || rn == ')' || rn == '[' || rn == ']' {
			r.rs.UnreadRune()
			break
		}
		sym += string(rn)
	}
	// check if symbol is a number
	if num, err := strconv.ParseFloat(sym, 64); err == nil {
		return value.NewNum(num, value.WithSection(r.popSection())), nil
	}

	// check if symbol is a keyword
	switch sym {
	case "nil":
		return value.NewNil(value.WithSection(r.popSection())), nil
	case "true":
		return value.NewBool(true, value.WithSection(r.popSection())), nil
	case "false":
		return value.NewBool(false, value.WithSection(r.popSection())), nil
	}

	return value.NewSymbol(sym, value.WithSection(r.popSection())), nil
}

func (r *Reader) readKeyword() (value.Value, error) {
	var sym string
	for {
		rn, _, err := r.rs.ReadRune()
		if err != nil {
			return nil, r.error("error reading keyword: %w", err)
		}
		if unicode.IsSpace(rn) || rn == ')' || rn == ']' {
			r.rs.UnreadRune()
			break
		}
		sym += string(rn)
	}
	return value.NewKeyword(sym, value.WithSection(r.popSection())), nil
}
