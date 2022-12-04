package reader

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
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

var (
	syntaxRunes = []rune{'\\', '(', ')', '[', ']', '{', '}', '"', ';', '`', '~', '^', '@', ','}
)

func isSyntaxRune(rn rune) bool {
	for _, s := range syntaxRunes {
		if rn == s {
			return true
		}
	}
	return false
}

type (
	Reader struct {
		rs *trackingRuneScanner

		symbolResolver SymbolResolver

		// map for function shorthand arguments.
		// non-nil only when reading a function shorthand.
		fnArgMap   map[int]*value.Symbol
		argCounter int

		posStack []value.Pos
	}
)

type options struct {
	filename string
	resolver SymbolResolver
}

// Option represents an option that can be passed to New.
type Option func(*options)

// WithFilename sets the filename to be associated with the input.
func WithFilename(filename string) Option {
	return func(o *options) {
		o.filename = filename
	}
}

// WithSymbolResolver sets the symbol resolver to be used when reading.
func WithSymbolResolver(resolver SymbolResolver) Option {
	return func(o *options) {
		o.resolver = resolver
	}
}

func New(r io.RuneScanner, opts ...Option) *Reader {
	o := options{
		resolver: defaultSymbolResolver,
	}

	for _, opt := range opts {
		opt(&o)
	}
	return &Reader{
		rs:             newTrackingRuneScanner(r, o.filename),
		symbolResolver: o.resolver,

		// TODO: attain through a configured autogen function.
		//
		// we're starting at 3 here to match Clojure's behavior, which is
		// likely determined by some internal behavior. improve this with a
		// better test harness.
		argCounter: 3,
	}
}

// Read reads all expressions from the input until a read error occurs
// or io.EOF is reached. A final io.EOF will not be returned if the
// input ends with a valid expression or if it contains no expressions
// at all.
func (r *Reader) ReadAll() ([]interface{}, error) {
	var nodes []interface{}
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
	if len(r.posStack) != 0 {
		panic(fmt.Sprintf("position stack not empty: %+v", r.posStack))
	}
	return nodes, nil
}

func (r *Reader) ReadOne() (interface{}, error) {
	_, err := r.next()
	if err != nil {
		return nil, err
	}
	r.rs.UnreadRune()
	return r.readExpr()
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
		if isSpace(rn) {
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

func (r *Reader) readExpr() (interface{}, error) {
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
	case '{':
		return r.readMap()
	case '}':
		return nil, r.error("unexpected '}'")
	case '[':
		return r.readVector()
	case ']':
		return nil, r.error("unexpected ']'")
	case '"':
		return r.readString()
	case '\\':
		return r.readChar()
	case ':':
		return r.readKeyword()
	case '%':
		return r.readArg()

		// TODO: implement as reader macros
	case '\'':
		return r.readQuote()
	case '`':
		return r.readSyntaxQuote()
	case '~':
		return r.readUnquote()
	case '@':
		return r.readDeref()
	case '#':
		return r.readDispatch()
	case '^':
		// TODO: attach to next form
		_, err := r.readMeta()
		if err != nil {
			return nil, err
		}
		return r.readExpr()
	default:
		r.rs.UnreadRune()
		return r.readSymbol()
	}
}

func (r *Reader) readList() (interface{}, error) {
	var nodes []interface{}
	for {
		rune, err := r.next()
		if err != nil {
			return nil, err
		}
		if isSpace(rune) {
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

func (r *Reader) readVector() (interface{}, error) {
	var nodes []interface{}
	for {
		rune, err := r.next()
		if err != nil {
			return nil, err
		}
		if isSpace(rune) {
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

func (r *Reader) readMap() (interface{}, error) {
	var keyVals []interface{}
	for {
		rune, err := r.next()
		if err != nil {
			return nil, err
		}
		if isSpace(rune) {
			continue
		}
		if rune == '}' {
			break
		}

		r.rs.UnreadRune()
		el, err := r.readExpr()
		if err != nil {
			return nil, err
		}
		keyVals = append(keyVals, el)
	}
	if len(keyVals)%2 != 0 {
		return nil, r.error("map literal must contain an even number of forms")
	}
	return value.NewMap(keyVals, value.WithSection(r.popSection())), nil
}

func (r *Reader) readSet() (interface{}, error) {
	var vals []interface{}
	for {
		rune, err := r.next()
		if err != nil {
			return nil, err
		}
		if isSpace(rune) {
			continue
		}
		if rune == '}' {
			break
		}

		r.rs.UnreadRune()
		el, err := r.readExpr()
		if err != nil {
			return nil, err
		}
		vals = append(vals, el)
	}
	return value.NewSet(vals, value.WithSection(r.popSection())), nil
}

func (r *Reader) readString() (interface{}, error) {
	var str string
	sawSlash := false
	for {
		rune, _, err := r.rs.ReadRune()
		if err != nil {
			return nil, r.error("error reading string: %w", err)
		}

		if rune == '\\' {
			sawSlash = true
		} else if rune == '"' && !sawSlash {
			break
		} else {
			sawSlash = false
		}

		if rune == '\n' {
			str += "\\n"
		} else {
			str += string(rune)
		}
	}

	str, err := strconv.Unquote(`"` + str + `"`)
	if err != nil {
		return nil, r.error("invalid string: %w", err)
	}

	r.popSection()
	return str, nil
}

func (r *Reader) nextID() int {
	id := r.argCounter
	r.argCounter++
	return id
}

func (r *Reader) genArg(i int) *value.Symbol {
	prefix := "rest"
	if i != -1 {
		prefix = fmt.Sprintf("p%d", i)
	}
	return value.NewSymbol(fmt.Sprintf("%s__%d#", prefix, r.nextID()))
}

func (r *Reader) readArg() (interface{}, error) {
	r.rs.UnreadRune()
	sym, err := r.readSymbol()
	if err != nil {
		return nil, err
	}
	// if we're not parsing function shorthand, just return the symbol
	if r.fnArgMap == nil {
		return sym, nil
	}

	argSuffix := sym.(*value.Symbol).Name()[1:]
	switch {
	case argSuffix == "&":
		if r.fnArgMap[-1] == nil {
			r.fnArgMap[-1] = r.genArg(-1)
		}
		return r.fnArgMap[-1], nil
	case argSuffix == "":
		if r.fnArgMap[1] == nil {
			r.fnArgMap[1] = r.genArg(1)
		}
		return r.fnArgMap[1], nil
	default:
		argIndex, err := strconv.Atoi(argSuffix)
		if err != nil {
			return nil, r.error("arg literal must be %, %& or %integer")
		}
		if r.fnArgMap[argIndex] == nil {
			r.fnArgMap[argIndex] = r.genArg(argIndex)
		}
		return r.fnArgMap[argIndex], nil
	}
}

func (r *Reader) readFunctionShorthand() (interface{}, error) {
	if r.fnArgMap != nil {
		return nil, r.error("nested #()s are not allowed")
	}
	r.fnArgMap = make(map[int]*value.Symbol)
	defer func() {
		r.fnArgMap = nil
	}()

	r.rs.UnreadRune()
	body, err := r.readExpr()
	if err != nil {
		return nil, err
	}

	args := make([]interface{}, 0, len(r.fnArgMap))
	var restSym *value.Symbol
	// NB: arg keys are 1-indexed, -1 represents a "rest" arg
	for i, sym := range r.fnArgMap {
		for i > len(args) {
			args = append(args, nil)
		}
		if i == -1 {
			restSym = sym
			continue
		}
		args[i-1] = sym
	}
	if restSym != nil {
		args = append(args, value.NewSymbol("&"), restSym)
	}
	// fill in any missing args with generated args
	for i, arg := range args {
		if arg != nil {
			continue
		}
		args[i] = r.genArg(i + 1)
	}

	return value.NewList([]interface{}{
		value.NewSymbol("fn*"),
		value.NewVector(args),
		body,
	}, value.WithSection(r.popSection())), nil
}

func (r *Reader) readRegex() (interface{}, error) {
	var str string
	sawSlash := false
	for {
		rune, _, err := r.rs.ReadRune()
		if err != nil {
			return nil, r.error("error reading regex: %w", err)
		}

		if rune == '\\' {
			sawSlash = true
		} else if rune == '"' && !sawSlash {
			break
		} else {
			sawSlash = false
		}

		if rune == '\n' {
			str += "\\n"
		} else {
			str += string(rune)
		}
	}

	r.popSection()
	re, err := regexp.Compile(str)
	if err != nil {
		return nil, r.error("invalid regex: %w", err)
	}
	return re, nil
}

func (r *Reader) readChar() (interface{}, error) {
	var char string
	for {
		rn, _, err := r.rs.ReadRune()
		if errors.Is(err, io.EOF) && char != "" {
			break
		}
		if err != nil {
			return nil, r.error("error reading character: %w", err)
		}

		// TODO: helper for non-char/non-symbol runes
		if unicode.IsSpace(rn) || (len(char) > 0 && isSyntaxRune(rn)) {
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

func (r *Reader) readQuoteType(form string) (interface{}, error) {
	node, err := r.readExpr()
	if err != nil {
		return nil, err
	}
	section := r.popSection()

	symbolEndPos := section.StartPos
	symbolEndPos.Column += len(form)
	items := []interface{}{
		value.NewSymbol(form, value.WithSection(value.Section{StartPos: section.StartPos, EndPos: symbolEndPos})),
		node,
	}
	return value.NewList(items, value.WithSection(section)), nil
}

func (r *Reader) readQuote() (interface{}, error) {
	return r.readQuoteType("quote")
}

func (r *Reader) readSyntaxQuote() (interface{}, error) {
	return r.readQuoteType("quasiquote")
}

func (r *Reader) readDeref() (interface{}, error) {
	// TODO: look up 'deref' with the symbol resolver
	// it should resolve to glojure.core/deref in the go case
	return r.readQuoteType("clojure.core/deref")
}

func (r *Reader) readUnquote() (interface{}, error) {
	rn, _, err := r.rs.ReadRune()
	if err != nil {
		return nil, r.error("error reading input: %w", err)
	}
	if rn == '@' {
		return r.readQuoteType("splice-unquote")
	}

	r.rs.UnreadRune()
	return r.readQuoteType("clojure.core/unquote")
}

func (r *Reader) readDispatch() (interface{}, error) {
	rn, _, err := r.rs.ReadRune()
	if err != nil {
		return nil, r.error("error reading input: %w", err)
	}

	switch rn {
	case ':':
		return r.readNamespacedMap()
	case '{':
		return r.readSet()
	case '_':
		// discard form
		_, err := r.readExpr()
		if err != nil {
			return nil, err
		}
		r.popSection()
		// return the next one
		return r.readExpr()
	case '(':
		// function shorthand
		return r.readFunctionShorthand()
	case '\'':
		// var
		expr, err := r.readExpr()
		if err != nil {
			return nil, err
		}
		return value.NewList([]interface{}{
			value.NewSymbol("var"),
			expr,
		}, value.WithSection(r.popSection())), nil
	case '"':
		return r.readRegex()
	default:
		return nil, r.error("invalid dispatch character: %c", rn)
	}
}

func (r *Reader) readNamespacedMap() (interface{}, error) {
	r.pushSection()
	nsKWVal, err := r.readKeyword()
	if err != nil {
		return nil, err
	}

	nsKW := nsKWVal.(*value.Keyword)
	if strings.Contains(nsKW.Value, "/") {
		return nil, r.error("namespaced map must specify a valid namespace: %s", nsKW)
	}

	rn, err := r.next()
	if err != nil {
		return nil, r.error("error reading input: %w", err)
	}

	if rn != '{' {
		return nil, r.error("Namespaced map must specify a map")
	}

	mapVal, err := r.readMap()
	if err != nil {
		return nil, r.error("error reading namespaced map: %w", err)
	}

	mp := mapVal.(value.ISeq)

	newKeyVals := []interface{}{}
	for !mp.IsEmpty() {
		kv := mp.First()
		mp = mp.Rest()

		key := kv.(*value.Vector).ValueAt(0)
		val := kv.(*value.Vector).ValueAt(1)

		keyKW, ok := key.(*value.Keyword)
		if !ok || keyKW.Namespace() != "" {
			newKeyVals = append(newKeyVals, key, val)
			continue
		}
		newKey := value.NewKeyword(nsKW.Value+"/"+keyKW.Name(), value.WithSection(keyKW.Section))
		newKeyVals = append(newKeyVals, newKey, val)
	}

	return value.NewMap(newKeyVals, value.WithSection(mapVal.(*value.Map).Section)), nil
}

var (
	numPrefixRegex = regexp.MustCompile(`^[-+]?[0-9]+`)
	intRegex       = regexp.MustCompile(`^[-+]?\d(\d|[a-fA-F])*N?$`)
	hexRegex       = regexp.MustCompile(`^[-+]?0[xX]([a-fA-F]|\d)*N?$`)
)

func isValidNumberCharacter(rn rune) bool {
	if isSpace(rn) || isSyntaxRune(rn) {
		return false
	}
	// TODO: look at clojure code to understand this. it seems likely
	// that these are reader macros, but I'm not sure.
	return rn != '#' && rn != '%' && rn != '\''
}

func (r *Reader) readNumber(numStr string) (interface{}, error) {
	for {
		rn, _, err := r.rs.ReadRune()
		if errors.Is(err, io.EOF) && numStr != "" {
			break
		}
		if err != nil {
			return nil, r.error("error reading symbol: %w", err)
		}
		if !isValidNumberCharacter(rn) {
			r.rs.UnreadRune()
			break
		}
		numStr += string(rn)
	}

	if intRegex.MatchString(numStr) || hexRegex.MatchString(numStr) {
		if strings.HasSuffix(numStr, "N") {
			bi, err := value.NewBigInt(numStr[:len(numStr)-1])
			if err != nil {
				return nil, r.error("invalid big int: %w", err)
			}

			r.popSection()
			return bi, nil
		}

		intVal, err := strconv.ParseInt(numStr, 0, 64)
		if err != nil {
			return nil, r.error("invalid number: %s", numStr)
		}

		r.popSection()
		return int64(intVal), nil
	}

	// else, it's a float
	// if the last character is M, it's a big decimal
	if strings.HasSuffix(numStr, "M") {
		bd, err := value.NewBigDecimal(numStr[:len(numStr)-1])
		if err != nil {
			return nil, r.error("invalid big decimal: %w", err)
		}
		r.popSection()
		return bd, nil
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return nil, r.error("invalid number: %s", numStr)
	}

	r.popSection()
	return num, nil
}

func (r *Reader) readSymbol() (interface{}, error) {
	// TODO: a cleaner way to do this. adding some hacks while trying to
	// match clojure's reader's behavior.

	var sym string
	for {
		rn, _, err := r.rs.ReadRune()
		if errors.Is(err, io.EOF) && sym != "" {
			break
		}
		if err != nil {
			return nil, r.error("error reading symbol: %w", err)
		}
		if isSpace(rn) || isSyntaxRune(rn) {
			r.rs.UnreadRune()
			break
		}
		sym += string(rn)

		if numPrefixRegex.MatchString(sym) {
			return r.readNumber(sym)
		}
	}
	if sym == "" {
		return nil, r.error("error reading symbol")
	}

	section := r.popSection()
	// check if symbol is a keyword
	switch sym {
	case "nil":
		return nil, nil
	case "true":
		return true, nil
	case "false":
		return false, nil
	}

	symVal := value.NewSymbol(sym, value.WithSection(section))
	if !symVal.IsValidFormat() {
		return nil, r.error("invalid symbol: %s", sym)
	}
	return symVal, nil
}

func (r *Reader) readKeyword() (interface{}, error) {
	var sym string
	for {
		rn, _, err := r.rs.ReadRune()
		if errors.Is(err, io.EOF) && sym != "" {
			break
		}
		if err != nil {
			return nil, r.error("error reading keyword: %w", err)
		}
		if isSpace(rn) || isSyntaxRune(rn) {
			r.rs.UnreadRune()
			break
		}
		sym += string(rn)
	}
	if sym == "" || sym == ":" {
		return nil, r.error("invalid keyword: :" + sym)
	}
	return value.NewKeyword(sym, value.WithSection(r.popSection())), nil
}

func (r *Reader) readMeta() (interface{}, error) {
	res, err := r.readExpr()
	if err != nil {
		return nil, err
	}
	r.popSection()
	return res, nil
}

func isSpace(r rune) bool {
	return r == ',' || unicode.IsSpace(r)
}
