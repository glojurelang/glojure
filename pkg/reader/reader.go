package reader

import (
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/glojurelang/glojure/pkg/lang"
)

var (
	symQuote         = lang.NewSymbol("quote")
	symList          = lang.NewSymbol("clojure.core/list")
	symSeq           = lang.NewSymbol("clojure.core/seq")
	symConcat        = lang.NewSymbol("clojure.core/concat")
	symUnquote       = lang.NewSymbol("clojure.core/unquote")
	symSpliceUnquote = lang.NewSymbol("clojure.core/splice-unquote")

	specials = func() map[string]bool {
		specialStrs := []string{
			"def",
			"loop*",
			"recur",
			"if",
			"case*",
			"let*",
			"letfn*",
			"do",
			"fn*",
			"quote",
			"var",
			"clojure.core/import*",
			".",
			"set!",
			"deftype*",
			"reify*",
			"try",
			"catch",
			"throw",
			"finally",
			"monitor-enter",
			"monitor-exit",
			"new",
			"&",
		}
		ret := make(map[string]bool)
		for _, s := range specialStrs {
			ret[s] = true
		}
		return ret
	}()

	// ErrEOF is returned when the end of the input is reached after
	// all input has been read. Callers can check for this error to
	// determine if an error is due to malformed input or exhausted
	// input. ErrEOF will only be returned when a form could not be
	// read because the input was exhausted, not when a form was
	// malformed.
	ErrEOF = errors.New("EOF")

	readerCondSentinel = &struct{}{}
	stopRuneSentinel   = &struct{}{}
)

type (
	trackingRuneScanner struct {
		rs io.RuneScanner

		filename       string
		nextRuneLine   int
		nextRuneColumn int

		// keep track of the last two runes read, most recent last.
		history []pos
	}

	pos struct {
		Filename string
		Line     int
		Column   int
	}

	Error struct {
		wrapped error
		pos     pos
	}
)

func (e *Error) Error() string {
	prefix := fmt.Sprintf("%s:%d:%d: ", e.pos.Filename, e.pos.Line, e.pos.Column)
	return prefix + e.wrapped.Error()
}

func (e *Error) Unwrap() error {
	return e.wrapped
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
		history:        make([]pos, 0, 2),
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
	r.history = append(r.history, pos{
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
func (r *trackingRuneScanner) pos() pos {
	if len(r.history) == 0 {
		return pos{
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
		getCurrentNS   func() *lang.Namespace

		// map for function shorthand arguments.
		// non-nil only when reading a function shorthand.
		fnArgMap   map[int]*lang.Symbol
		argCounter int

		posStack []pos

		pendingForms []any
	}
)

type options struct {
	filename     string
	resolver     SymbolResolver
	getCurrentNS func() *lang.Namespace
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

// WithGetCurrentNS sets the function to be used to get the current namespace.
func WithGetCurrentNS(getCurrentNS func() *lang.Namespace) Option {
	return func(o *options) {
		o.getCurrentNS = getCurrentNS
	}
}

func New(r io.RuneScanner, opts ...Option) *Reader {
	o := options{}

	for _, opt := range opts {
		opt(&o)
	}
	getCurrentNS := func() *lang.Namespace {
		if lang.GlobalEnv != nil { // TODO: should be unnecessary
			return lang.GlobalEnv.CurrentNamespace()
		}
		return lang.FindOrCreateNamespace(lang.NewSymbol("user"))
	}
	if o.getCurrentNS != nil {
		getCurrentNS = o.getCurrentNS
	}
	return &Reader{
		rs:             newTrackingRuneScanner(r, o.filename),
		symbolResolver: o.resolver,
		getCurrentNS:   getCurrentNS,

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
		node, err := r.readExpr(true, 0)
		if err == ErrEOF {
			break
		}
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

// ReadOne reads the next expression from the input. If the input
// contains more than one expression, subsequent calls to ReadOne will
// return the next expression. If the input contains no expressions,
// ErrEOF will be returned.
func (r *Reader) ReadOne() (interface{}, error) {
	return r.readExpr(true, 0)
}

// error returns a formatted error that includes the current position
// of the scanner.
func (r *Reader) error(format string, args ...interface{}) error {
	return &Error{
		pos:     r.rs.pos(),
		wrapped: fmt.Errorf(format, args...),
	}
}

// popSection returns the last section read, ending at the current
// input, and pops it off the stack.
func (r *Reader) popSection() lang.IPersistentMap {
	top := r.posStack[len(r.posStack)-1]
	r.posStack = r.posStack[:len(r.posStack)-1]

	return lang.NewMap(
		lang.KWFile, r.rs.filename,
		lang.KWLine, top.Line,
		lang.KWColumn, top.Column,
		lang.KWEndLine, r.rs.pos().Line,
		lang.KWEndColumn, r.rs.pos().Column,
	)
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

func (r *Reader) readExpr(eofOK bool, stopRune rune) (expr any, err error) {
	for {
		form, err := r.read(eofOK, stopRune)
		if err != nil {
			return nil, err
		}
		// No-op reads return the rune scanner, so just continue.
		if form == r.rs {
			continue
		}

		return form, nil
	}
}

func (r *Reader) read(eofOK bool, stopRune rune) (expr any, err error) {
	if len(r.pendingForms) > 0 {
		form := r.pendingForms[0]
		r.pendingForms = r.pendingForms[1:]
		return form, nil
	}

	rune, err := r.next()
	if eofOK && errors.Is(err, io.EOF) {
		// return the EOF sentinel error
		return nil, ErrEOF
	}
	if err != nil {
		return nil, err
	}

	if rune == stopRune {
		return stopRuneSentinel, nil
	}

	r.pushSection()
	defer func() {
		s := r.popSection()
		obj, ok := expr.(lang.IObj)
		if !ok {
			return
		}
		meta := obj.Meta()
		for seq := lang.Seq(s); seq != nil; seq = seq.Next() {
			entry := seq.First().(lang.IMapEntry)
			meta = lang.Assoc(meta, entry.Key(), entry.Val()).(lang.IPersistentMap)
		}
		expr = obj.WithMeta(meta)
	}()

	switch rune {
	case ')':
		return nil, r.error("unexpected ')'")
	case '}':
		return nil, r.error("unexpected '}'")
	case ']':
		return nil, r.error("unexpected ']'")

	case '{':
		return r.readMap()
	case '(':
		return r.readList()
	case '[':
		return r.readVector()
	case '"':
		return r.readString()
	case '\\':
		return r.readChar()
	case ':':
		return r.readKeyword()
	case '%':
		return r.readArg()
	case '\'':
		return r.readQuote()
	case '`':
		return r.readSyntaxQuote()
	case '~':
		return r.readUnquote()
	case '@':
		return r.readDeref()
	case '#':
		return r.readDispatch(eofOK, stopRune)
	case '^':
		meta, err := r.readMeta()
		if err != nil {
			return nil, err
		}
		val, err := r.readExpr(eofOK, stopRune)
		if err != nil {
			return nil, err
		}
		return lang.WithMeta(val, meta)
	default:
		r.rs.UnreadRune()
		return r.readSymbol()
	}
}

func (r *Reader) read1ForColl(end rune) (result any, done bool, err error) {
	if len(r.pendingForms) > 0 {
		form := r.pendingForms[0]
		r.pendingForms = r.pendingForms[1:]
		return form, false, nil
	}

	for {
		rune, err := r.next()
		if err != nil {
			return nil, false, err
		}
		if isSpace(rune) {
			continue
		}
		if rune == end {
			return nil, true, nil
		}

		r.rs.UnreadRune()
		node, err := r.readExpr(false, end)
		if err != nil {
			return nil, false, err
		}
		return node, false, nil
	}
}

func (r *Reader) readForColl(end rune) ([]any, error) {
	var nodes []interface{}
	for {
		node, done, err := r.read1ForColl(end)
		if err != nil {
			return nil, err
		}
		if done || node == stopRuneSentinel {
			break
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (r *Reader) readList() (interface{}, error) {
	nodes, err := r.readForColl(')')
	if err != nil {
		return nil, err
	}
	return lang.NewList(nodes...), nil
}

func (r *Reader) readVector() (interface{}, error) {
	nodes, err := r.readForColl(']')
	if err != nil {
		return nil, err
	}
	return lang.NewVector(nodes...), nil
}

func (r *Reader) readMap() (interface{}, error) {
	keyVals, err := r.readForColl('}')
	if err != nil {
		return nil, err
	}
	if len(keyVals)%2 != 0 {
		return nil, r.error("map literal must contain an even number of forms")
	}
	return lang.NewMap(keyVals...), nil
}

func (r *Reader) readSet() (interface{}, error) {
	vals, err := r.readForColl('}')
	if err != nil {
		return nil, err
	}
	return lang.NewSet(vals...), nil
}

func (r *Reader) readString() (interface{}, error) {
	var str string
	for {
		rune, _, err := r.rs.ReadRune()
		if err != nil {
			return nil, r.error("error reading string: %w", err)
		}

		if rune == '\\' {
			str += string(rune)
			rune, _, err = r.rs.ReadRune()
			if err != nil {
				return nil, r.error("error reading string: %w", err)
			}
			str += string(rune)
			continue
		} else if rune == '"' {
			break
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

	return str, nil
}

func (r *Reader) nextID() int {
	id := r.argCounter
	r.argCounter++
	return id
}

func (r *Reader) genArg(i int) *lang.Symbol {
	prefix := "rest"
	if i != -1 {
		prefix = fmt.Sprintf("p%d", i)
	}
	return lang.NewSymbol(fmt.Sprintf("%s__%d#", prefix, r.nextID()))
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

	argSuffix := sym.(*lang.Symbol).Name()[1:]
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
			return nil, r.error("arg literal must be %%, %%& or %%integer")
		}
		if argIndex < 1 {
			return nil, r.error("arg literal must be %%, %%& or %%integer > 0")
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
	r.fnArgMap = make(map[int]*lang.Symbol)
	defer func() {
		r.fnArgMap = nil
	}()

	r.rs.UnreadRune()
	body, err := r.readExpr(false, 0)
	if err != nil {
		return nil, err
	}

	const maxArgIndex = 20

	args := make([]interface{}, 0, len(r.fnArgMap))
	var restSym *lang.Symbol
	// NB: arg keys are 1-indexed, -1 represents a "rest" arg
	for i, sym := range r.fnArgMap {
		for i > len(args) {
			if i > maxArgIndex {
				return nil, r.error("function shorthand cannot have more than %d args", maxArgIndex)
			}
			args = append(args, nil)
		}
		if i == -1 {
			restSym = sym
			continue
		}
		args[i-1] = sym
	}
	if restSym != nil {
		args = append(args, lang.NewSymbol("&"), restSym)
	}
	// fill in any missing args with generated args
	for i, arg := range args {
		if arg != nil {
			continue
		}
		args[i] = r.genArg(i + 1)
	}

	return lang.NewList(
		lang.NewSymbol("fn*"),
		lang.NewVector(args...),
		body,
	), nil
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

	rn, err := lang.RuneFromCharLiteral("\\" + char)
	if err != nil {
		return nil, r.error("invalid character literal: %w", err)
	}
	return lang.NewChar(rn), nil
}

func (r *Reader) readQuoteType(form string) (interface{}, error) {
	node, err := r.readExpr(false, 0)
	if err != nil {
		return nil, err
	}

	return lang.NewList(lang.NewSymbol(form), node), nil
}

func (r *Reader) readQuote() (interface{}, error) {
	return r.readQuoteType("quote")
}

func (r *Reader) readSyntaxQuote() (interface{}, error) {
	node, err := r.readExpr(false, 0)
	if err != nil {
		return nil, err
	}

	// symbolNameMap tracks the names of symbols that have been renamed.
	// symbols that end with a '#' have '#' replaced with a unique
	// suffix.
	symbolNameMap := make(map[string]*lang.Symbol)
	return r.syntaxQuote(symbolNameMap, node), nil
}

func (r *Reader) syntaxQuote(symbolNameMap map[string]*lang.Symbol, node interface{}) interface{} {
	switch node := node.(type) {
	case lang.Keyword, lang.Char, string:
		return node
	case *lang.Symbol:
		sym := node
		if specials[sym.String()] {
			return lang.NewList(symQuote, sym)
		}
		switch {
		case sym.Namespace() == "" && strings.HasSuffix(sym.Name(), "#"):
			gs, ok := symbolNameMap[sym.String()]
			if ok {
				sym = gs
				break
			}
			// TODO: use a global counter, not the length of this map
			newSym := lang.NewSymbol(strings.TrimSuffix(sym.Name(), "#") + "__" + strconv.Itoa(len(symbolNameMap)) + "__auto__")
			symbolNameMap[sym.String()] = newSym
			sym = newSym
		case sym.Namespace() == "" && strings.HasSuffix(sym.Name(), "."):
			// TODO: match clojure behavior!
		case sym.Namespace() == "" && strings.HasPrefix(sym.Name(), "."):
			// simply quote method names
		case sym.Namespace() != "" && strings.Contains(sym.Name(), "."):
			// special class-like go value
			// TODO: is this a good syntax?
		case r.symbolResolver != nil:
			var nsym *lang.Symbol
			if sym.Namespace() != "" {
				alias := lang.InternSymbol(nil, sym.Namespace())
				nsym = r.symbolResolver.ResolveStruct(alias)
				if nsym == nil {
					nsym = r.symbolResolver.ResolveAlias(alias)
				}
			}
			if nsym != nil {
				sym = lang.InternSymbol(nsym.Name(), sym.Name())
			} else if sym.Namespace() == "" {
				rsym := r.symbolResolver.ResolveStruct(sym)
				if rsym == nil {
					rsym = r.symbolResolver.ResolveVar(sym)
				}
				if rsym != nil {
					sym = rsym
				} else {
					sym = lang.InternSymbol(r.symbolResolver.CurrentNS().Name(), sym.Name())
				}
			}
		default:
			// HACK: handle well-known host forms
			if strings.Contains(sym.Name(), ".") {
				break
			}
			// TODO: need to do anything for equiv of clojure maybeClass?
			sym = r.resolveSymbol(sym)
		}
		// TODO: match actual LispReader.java behavior
		return lang.NewList(symQuote, sym)
	case lang.IPersistentMap:
		var keyvals []interface{}
		for seq := lang.Seq(node); seq != nil; seq = seq.Next() {
			entry := seq.First().(lang.IMapEntry)
			keyvals = append(keyvals, entry.Key(), entry.Val())
		}
		return lang.NewList(
			lang.NewSymbol("clojure.core/apply"),
			lang.NewSymbol("clojure.core/hash-map"),
			lang.NewList(
				lang.NewSymbol("clojure.core/seq"),
				lang.NewCons(
					lang.NewSymbol("clojure.core/concat"),
					r.sqExpandList(symbolNameMap, keyvals),
				),
			),
		)
	case lang.IPersistentList, lang.IPersistentVector:
		_, isVector := node.(lang.IPersistentVector)
		if lang.Count(node) == 0 {
			if isVector {
				//(clojure.core/apply clojure.core/vector (clojure.core/seq (clojure.core/concat)))
				return lang.NewList(
					lang.NewSymbol("clojure.core/apply"),
					lang.NewSymbol("clojure.core/vector"),
					lang.NewList(
						lang.NewSymbol("clojure.core/seq"),
						lang.NewList(
							lang.NewSymbol("clojure.core/concat"),
						),
					),
				)
			}
			return lang.NewList(symList)
		}
		if r.isUnquote(node) {
			return lang.First(lang.Rest(node))
		}

		elements := []interface{}{symConcat}
		for seq := lang.Seq(node); seq != nil; seq = seq.Next() {
			first := seq.First()
			if seq, ok := first.(lang.ISeq); ok && lang.Equals(lang.First(seq), symSpliceUnquote) {
				elements = append(elements, lang.First(lang.Rest(first)))
			} else {
				elements = append(elements, lang.NewList(symList, r.syntaxQuote(symbolNameMap, first)))
			}
		}

		ret := lang.NewList(symSeq,
			lang.NewList(elements...))
		if isVector {
			ret = lang.NewList(
				lang.NewSymbol("clojure.core/apply"),
				lang.NewSymbol("clojure.core/vector"),
				ret)
		}
		return ret
	}
	return lang.NewList(symQuote, node)
}

func (r *Reader) sqExpandList(symbolNameMap map[string]*lang.Symbol, els []interface{}) lang.ISeq {
	var ret lang.IPersistentVector = lang.NewVector()
	for _, v := range els {
		if r.isUnquote(v) {
			ret = ret.Cons(lang.NewList(lang.NewSymbol("clojure.core/list"), lang.First(lang.Rest(v)))).(lang.IPersistentVector)
		} else if r.isUnquoteSplicing(v) {
			ret = ret.Cons(lang.First(lang.Rest(v))).(lang.IPersistentVector)
		} else {
			ret = ret.Cons(lang.NewList(lang.NewSymbol("clojure.core/list"), r.syntaxQuote(symbolNameMap, v))).(lang.IPersistentVector)
		}
	}
	return lang.Seq(ret)
}

func (r *Reader) isUnquote(form interface{}) bool {
	seq, ok := form.(lang.ISeq)
	return ok && lang.Equals(seq.First(), symUnquote)
}

func (r *Reader) isUnquoteSplicing(form interface{}) bool {
	seq, ok := form.(lang.ISeq)
	return ok && lang.Equals(seq.First(), symSpliceUnquote)
}

func (r *Reader) readDeref() (interface{}, error) {
	// TODO: look up 'deref' with the symbol resolver
	// it should resolve to clojure.core/deref in the go case
	return r.readQuoteType("clojure.core/deref")
}

func (r *Reader) readUnquote() (interface{}, error) {
	rn, _, err := r.rs.ReadRune()
	if err != nil {
		return nil, r.error("error reading input: %w", err)
	}
	if rn == '@' {
		return r.readQuoteType("clojure.core/splice-unquote")
	}

	r.rs.UnreadRune()
	return r.readQuoteType("clojure.core/unquote")
}

func (r *Reader) readDispatch(eofOK bool, stopRune rune) (interface{}, error) {
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
		_, err := r.readExpr(eofOK, stopRune)
		if err != nil {
			return nil, err
		}
		// return the next one
		return r.readExpr(eofOK, stopRune)
	case '(':
		// function shorthand
		return r.readFunctionShorthand()
	case '\'':
		// var
		expr, err := r.readExpr(false, 0)
		if err != nil {
			return nil, err
		}
		return lang.NewList(lang.NewSymbol("var"), expr), nil
	case '"':
		return r.readRegex()
	case '^':
		r.rs.UnreadRune()
		// just read normally
		return r.readExpr(false, 0)
	case '#':
		return r.readSymbolicValue()
	case '?':
		return r.readConditional(eofOK, stopRune)
	case '!':
		// comment, discard until end of line
		for {
			rn, _, err := r.rs.ReadRune()
			if err != nil {
				return nil, r.error("error reading input: %w", err)
			}
			if rn == '\n' {
				break
			}
		}
		return r.readExpr(eofOK, stopRune)
	default:
		return nil, r.error("invalid dispatch character: %c", rn)
	}
}

func (r *Reader) readNamespacedMap() (interface{}, error) {
	nsKWVal, err := r.readKeyword()
	if err != nil {
		return nil, err
	}

	nsKW := nsKWVal.(lang.Keyword)
	if strings.Contains(nsKW.String(), "/") {
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

	newKeyVals := []interface{}{}
	for mp := lang.Seq(mapVal); mp != nil; mp = mp.Next() {
		kv := mp.First()

		key := kv.(*lang.MapEntry).Key()
		val := kv.(*lang.MapEntry).Val()

		keyKW, ok := key.(lang.Keyword)
		if !ok || keyKW.Namespace() != "" {
			newKeyVals = append(newKeyVals, key, val)
			continue
		}
		newKey := lang.NewKeyword(nsKW.Name() + "/" + keyKW.Name())
		newKeyVals = append(newKeyVals, newKey, val)
	}

	m, err := lang.WithMeta(lang.NewMap(newKeyVals...), mapVal.(lang.IMeta).Meta())
	if err != nil {
		// This should never happen. Maps can have metadata.
		panic(err)
	}
	return m, nil
}

func (r *Reader) readSymbolicValue() (interface{}, error) {
	v, err := r.readExpr(false, 0)
	if err != nil {
		return nil, err
	}
	sym, ok := v.(*lang.Symbol)
	if !ok {
		return nil, r.error("symbolic value must be a symbol")
	}
	switch sym.Name() {
	case "Inf":
		return math.Inf(1), nil
	case "-Inf":
		return math.Inf(-1), nil
	case "NaN":
		return math.NaN(), nil
	}
	return nil, r.error("unknown symbolic value: ##%s", sym.Name())
}

var (
	numPrefixRegex = regexp.MustCompile(`^[-+]?([0-9]+|[1-9]+r)`)
	radixRegex     = regexp.MustCompile(`^[-+]?([2-9]|[12][0-9]|3[0-6])r([0-9a-zA-Z]+N?)$`)
	intRegex       = regexp.MustCompile(`^[-+]?\d+N?$`)
	ratioRegex     = regexp.MustCompile(`^[-+]?\d+\/\d+$`)
	hexRegex       = regexp.MustCompile(`^[-+]?0[xX]([a-fA-F]|\d)*N?$`)
	floatRegex     = regexp.MustCompile(`^[-+]?(\d+\.\d*|\.\d+)([eE][-+]?\d+)?|[-+]?(\d+)([eE][-+]?\d+)$`)
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

	base := 0 // infer from prefix
	isRadixNumber := false
	if match := radixRegex.FindStringSubmatch(numStr); match != nil {
		sign := ""
		if numStr[0] == '-' || numStr[0] == '+' {
			sign = string(numStr[0])
		}
		radix, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, r.error("error parsing radix %s: %w", match[1], err)
		}
		if radix > 36 {
			return nil, r.error("radix out of range: %d", radix)
		}
		base = radix
		numStr = sign + match[2]
		isRadixNumber = true
	}

	if isRadixNumber || intRegex.MatchString(numStr) || hexRegex.MatchString(numStr) {
		if strings.HasSuffix(numStr, "N") {
			bi, err := lang.NewBigIntWithBase(numStr[:len(numStr)-1], base)
			if err != nil {
				return nil, r.error("invalid big int: %w", err)
			}

			return bi, nil
		}

		intVal, err := strconv.ParseInt(numStr, base, 64)
		if err != nil {
			if errors.Is(err, strconv.ErrRange) {
				bi, err := lang.NewBigIntWithBase(numStr, base)
				if err != nil {
					return nil, r.error("invalid big int: %w", err)
				}
				return bi, nil
			}
			return nil, r.error("invalid number: %s", numStr)
		}

		return int64(intVal), nil
	}

	if ratioRegex.MatchString(numStr) {
		parts := strings.Split(numStr, "/")

		numBig, err := lang.NewBigInt(parts[0])
		if err != nil {
			return nil, r.error("invalid ratio: %s", numStr)
		}
		denomBig, err := lang.NewBigInt(parts[1])
		if err != nil {
			return nil, r.error("invalid ratio: %s", numStr)
		}
		return lang.NewRatioBigInt(numBig, denomBig), nil
	}

	// else, it's a float
	// if the last character is M, it's a big decimal
	if strings.HasSuffix(numStr, "M") {
		bd, err := lang.NewBigDecimal(numStr[:len(numStr)-1])
		if err != nil {
			return nil, r.error("invalid big decimal: %w", err)
		}
		return bd, nil
	}

	if floatRegex.MatchString(numStr) {
		num, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return nil, r.error("invalid number: %s", numStr)
		}

		return num, nil
	}

	return nil, r.error("invalid number: %s", numStr)

}

func (r *Reader) readSymbol() (ret interface{}, retErr error) {
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

	// check if symbol is a keyword
	switch sym {
	case "nil":
		return nil, nil
	case "true":
		return true, nil
	case "false":
		return false, nil
	}

	defer func() {
		if r := recover(); r != nil {
			retErr = r.(error)
		}
	}()

	return lang.NewSymbol(sym), nil
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
	if sym == "" || sym == ":" || strings.Contains(sym[1:], ":") {
		return nil, r.error("invalid keyword: :" + sym)
	}
	if sym[0] == ':' {
		// TODO: handle auto-resolving keywords with namespaces
		ns := r.getCurrentNS().Name().Name()
		sym = ns + "/" + sym[1:]
	}
	return lang.NewKeyword(sym), nil
}

func (r *Reader) readMeta() (lang.IPersistentMap, error) {
	res, err := r.readExpr(false, 0)
	if err != nil {
		return nil, err
	}

	switch res := res.(type) {
	case *lang.Map:
		return res, nil
	case *lang.Symbol, string:
		return lang.NewMap(lang.KWTag, res), nil
	case lang.Keyword:
		return lang.NewMap(res, true), nil
	default:
		return nil, r.error("metadata must be a map, symbol, keyword, or string")
	}
}

func (r *Reader) readConditional(eofOK bool, stopRune rune) (any, error) {
	rn, _, err := r.rs.ReadRune()
	if err != nil {
		return nil, r.error("error reading conditional: %w", err)
	}

	var splicing bool
	if rn == '@' {
		splicing = true
	} else {
		r.rs.UnreadRune()
	}

	node, err := r.readExpr(false, 0)
	if err != nil {
		return nil, err
	}

	// must always be a list
	lst, ok := node.(lang.IPersistentList)
	if !ok {
		return nil, r.error("read-cond body must be a list")
	}

	var form any = readerCondSentinel

	seq := lang.Seq(lst)
	for seq != nil {
		feature := seq.First()
		hfeat, err := r.hasFeature(feature)
		if err != nil {
			return nil, err
		}
		seq = seq.Next()
		if seq == nil {
			return nil, r.error("read-cond requires an even number of forms")
		}
		if hfeat {
			form = seq.First()
			break
		}

		seq = seq.Next()
	}

	if form == readerCondSentinel {
		// return the next expression (not nil!)
		form, err := r.readExpr(eofOK, stopRune)
		if err != nil {
			return nil, err
		}
		return form, nil
	}

	if splicing {
		seqable, ok := form.(lang.Seqable)
		if !ok {
			return nil, r.error("splicing read-cond form must be seqable")
		}
		seq := seqable.Seq()
		first := seq.First()
		for seq = seq.Next(); seq != nil; seq = seq.Next() {
			r.pendingForms = append(r.pendingForms, seq.First())
		}
		return first, nil
	}

	return form, nil
}

// Test cases: topLevel splicing; multiple spliced forms; last form is
// non-matching conditional; odd number of forms; nested splice; conditional at end of collection;
// conditional at end of input;

// hasFeature reports whether the reader has the given reader
// conditional feature.
func (r *Reader) hasFeature(feat any) (bool, error) {
	kw, ok := feat.(lang.Keyword)
	if !ok {
		return false, r.error("reader conditional feature must be a keyword")
	}
	name := kw.Name()

	// err on reserved features: else, none
	if name == "else" || name == "none" {
		return false, r.error(fmt.Sprintf("feature name %q is reserved", name))
	}

	switch name {
	case "default":
		return true, nil
	case "glj":
		return true, nil
	default:
		return false, nil
	}
}

////////////////////////////////////////////////////////////////////////////////

// Translated from Clojure's Compiler.java
func (r *Reader) resolveSymbol(sym *lang.Symbol) *lang.Symbol {
	if strings.Contains(sym.Name(), ".") {
		return sym
	}
	if sym.Namespace() != "" {
		ns := lang.NamespaceFor(r.getCurrentNS(), sym)
		if ns == nil || (ns.Name().Name() == "" && sym.Namespace() == "") ||
			(ns.Name().Name() != "" && ns.Name().Name() == sym.Namespace()) {
			return sym
		}
		return lang.InternSymbol(ns.Name().Name(), sym.Name())
	}

	currentNS := r.getCurrentNS()
	o := currentNS.GetMapping(sym)
	switch o := o.(type) {
	case nil:
		return lang.InternSymbol(currentNS.Name().Name(), sym.Name())
	case *lang.Var:
		return lang.InternSymbol(o.Namespace().Name().Name(), o.Symbol().Name())
	}
	return nil
}

func isSpace(r rune) bool {
	return r == ',' || unicode.IsSpace(r)
}
