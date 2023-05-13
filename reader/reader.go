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

var (
	symQuote         = value.NewSymbol("quote")
	symList          = value.NewSymbol("glojure.core/list")
	symSeq           = value.NewSymbol("glojure.core/seq")
	symConcat        = value.NewSymbol("glojure.core/concat")
	symUnquote       = value.NewSymbol("glojure.core/unquote")
	symSpliceUnquote = value.NewSymbol("glojure.core/splice-unquote")

	specials = func() map[string]bool {
		specialStrs := []string{
			"def",
			"loop*",
			"recur",
			"if",
			"case",
			"let*",
			"letfn*",
			"do",
			"fn*",
			"quote",
			"var",
			"glojure.core/import*",
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
)

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
		getCurrentNS   func() *value.Namespace

		// map for function shorthand arguments.
		// non-nil only when reading a function shorthand.
		fnArgMap   map[int]*value.Symbol
		argCounter int

		posStack []pos
	}
)

type options struct {
	filename     string
	resolver     SymbolResolver
	getCurrentNS func() *value.Namespace
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
func WithGetCurrentNS(getCurrentNS func() *value.Namespace) Option {
	return func(o *options) {
		o.getCurrentNS = getCurrentNS
	}
}

func New(r io.RuneScanner, opts ...Option) *Reader {
	o := options{}

	for _, opt := range opts {
		opt(&o)
	}
	getCurrentNS := func() *value.Namespace {
		if value.GlobalEnv != nil { // TODO: should be unnecessary
			return value.GlobalEnv.CurrentNamespace()
		}
		return value.FindOrCreateNamespace(value.NewSymbol("user"))
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
func (r *Reader) popSection() value.IPersistentMap {
	top := r.posStack[len(r.posStack)-1]
	r.posStack = r.posStack[:len(r.posStack)-1]

	return value.NewMap(
		value.KWFile, r.rs.filename,
		value.KWLine, top.Line,
		value.KWColumn, top.Column,
		value.KWEndLine, r.rs.pos().Line,
		value.KWEndColumn, r.rs.pos().Column,
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

func (r *Reader) readExpr() (expr interface{}, err error) {
	rune, err := r.next()
	if err != nil {
		return nil, err
	}

	r.pushSection()
	defer func() {
		s := r.popSection()
		obj, ok := expr.(value.IObj)
		if !ok {
			return
		}
		meta := obj.Meta()
		for seq := value.Seq(s); seq != nil; seq = seq.Next() {
			entry := seq.First().(value.IMapEntry)
			meta = value.Assoc(meta, entry.Key(), entry.Val()).(value.IPersistentMap)
		}
		expr = obj.WithMeta(meta)
	}()

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
		meta, err := r.readMeta()
		if err != nil {
			return nil, err
		}
		val, err := r.readExpr()
		if err != nil {
			return nil, err
		}
		return value.WithMeta(val, meta)
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
	return value.NewList(nodes...), nil
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
	return value.NewVector(nodes...), nil
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

	return value.NewMap(keyVals...), nil
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
	return value.NewSet(vals...), nil
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
		if argIndex < 1 {
			return nil, r.error("arg literal must be %, %& or %integer > 0")
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

	const maxArgIndex = 20

	args := make([]interface{}, 0, len(r.fnArgMap))
	var restSym *value.Symbol
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
		args = append(args, value.NewSymbol("&"), restSym)
	}
	// fill in any missing args with generated args
	for i, arg := range args {
		if arg != nil {
			continue
		}
		args[i] = r.genArg(i + 1)
	}

	return value.NewList(
		value.NewSymbol("fn*"),
		value.NewVector(args...),
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

	rn, err := value.RuneFromCharLiteral("\\" + char)
	if err != nil {
		return nil, r.error("invalid character literal: %w", err)
	}
	return value.NewChar(rn), nil
}

func (r *Reader) readQuoteType(form string) (interface{}, error) {
	node, err := r.readExpr()
	if err != nil {
		return nil, err
	}

	return value.NewList(value.NewSymbol(form), node), nil
}

func (r *Reader) readQuote() (interface{}, error) {
	return r.readQuoteType("quote")
}

func (r *Reader) readSyntaxQuote() (interface{}, error) {
	node, err := r.readExpr()
	if err != nil {
		return nil, err
	}

	// symbolNameMap tracks the names of symbols that have been renamed.
	// symbols that end with a '#' have '#' replaced with a unique
	// suffix.
	symbolNameMap := make(map[string]*value.Symbol)
	return r.syntaxQuote(symbolNameMap, node), nil
}

func (r *Reader) syntaxQuote(symbolNameMap map[string]*value.Symbol, node interface{}) interface{} {
	switch node := node.(type) {
	case value.Keyword, value.Char, string:
		return node
	case *value.Symbol:
		sym := node
		if specials[sym.String()] {
			return value.NewList(symQuote, sym)
		}
		switch {
		case sym.Namespace() == "" && strings.HasSuffix(sym.Name(), "#"):
			gs, ok := symbolNameMap[sym.String()]
			if ok {
				sym = gs
				break
			}
			// TODO: use a global counter, not the length of this map
			newSym := value.NewSymbol(strings.TrimSuffix(sym.Name(), "#") + "__" + strconv.Itoa(len(symbolNameMap)) + "__auto__")
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
			var nsym *value.Symbol
			if sym.Namespace() != "" {
				alias := value.InternSymbol(nil, sym.Namespace())
				nsym = r.symbolResolver.ResolveStruct(alias)
				if nsym == nil {
					nsym = r.symbolResolver.ResolveAlias(alias)
				}
			}
			if nsym != nil {
				sym = value.InternSymbol(nsym.Name(), sym.Name())
			} else if sym.Namespace() == "" {
				rsym := r.symbolResolver.ResolveStruct(sym)
				if rsym == nil {
					rsym = r.symbolResolver.ResolveVar(sym)
				}
				if rsym != nil {
					sym = rsym
				} else {
					sym = value.InternSymbol(r.symbolResolver.CurrentNS().Name(), sym.Name())
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
		return value.NewList(symQuote, sym)
	case value.IPersistentMap:
		var keyvals []interface{}
		for seq := value.Seq(node); seq != nil; seq = seq.Next() {
			entry := seq.First().(value.IMapEntry)
			keyvals = append(keyvals, entry.Key(), entry.Val())
		}
		return value.NewList(
			value.NewSymbol("glojure.core/apply"),
			value.NewSymbol("glojure.core/hash-map"),
			value.NewList(
				value.NewSymbol("glojure.core/seq"),
				value.NewCons(
					value.NewSymbol("glojure.core/concat"),
					r.sqExpandList(symbolNameMap, keyvals),
				),
			),
		)
	case value.IPersistentList, value.IPersistentVector:
		_, isVector := node.(value.IPersistentVector)
		if value.Count(node) == 0 {
			if isVector {
				//(glojure.core/apply glojure.core/vector (glojure.core/seq (glojure.core/concat)))
				return value.NewList(
					value.NewSymbol("glojure.core/apply"),
					value.NewSymbol("glojure.core/vector"),
					value.NewList(
						value.NewSymbol("glojure.core/seq"),
						value.NewList(
							value.NewSymbol("glojure.core/concat"),
						),
					),
				)
			}
			return value.NewList(symList)
		}
		if r.isUnquote(node) {
			return value.First(value.Rest(node))
		}

		elements := []interface{}{symConcat}
		for seq := value.Seq(node); seq != nil; seq = seq.Next() {
			first := seq.First()
			if seq, ok := first.(value.ISeq); ok && value.Equal(value.First(seq), symSpliceUnquote) {
				elements = append(elements, value.First(value.Rest(first)))
			} else {
				elements = append(elements, value.NewList(symList, r.syntaxQuote(symbolNameMap, first)))
			}
		}

		ret := value.NewList(symSeq,
			value.NewList(elements...))
		if isVector {
			ret = value.NewList(
				value.NewSymbol("glojure.core/apply"),
				value.NewSymbol("glojure.core/vector"),
				ret)
		}
		return ret
	}
	return value.NewList(symQuote, node)
}

func (r *Reader) sqExpandList(symbolNameMap map[string]*value.Symbol, els []interface{}) value.ISeq {
	var ret value.IPersistentVector = value.NewVector()
	for _, v := range els {
		if r.isUnquote(v) {
			ret = ret.Cons(value.NewList(value.NewSymbol("glojure.core/list"), value.First(value.Rest(v))))
		} else if r.isUnquoteSplicing(v) {
			ret = ret.Cons(value.First(value.Rest(v)))
		} else {
			ret = ret.Cons(value.NewList(value.NewSymbol("glojure.core/list"), r.syntaxQuote(symbolNameMap, v)))
		}
	}
	return value.Seq(ret)
}

func (r *Reader) isUnquote(form interface{}) bool {
	seq, ok := form.(value.ISeq)
	return ok && value.Equal(seq.First(), symUnquote)
}

func (r *Reader) isUnquoteSplicing(form interface{}) bool {
	seq, ok := form.(value.ISeq)
	return ok && value.Equal(seq.First(), symSpliceUnquote)
}

func (r *Reader) readDeref() (interface{}, error) {
	// TODO: look up 'deref' with the symbol resolver
	// it should resolve to glojure.core/deref in the go case
	return r.readQuoteType("glojure.core/deref")
}

func (r *Reader) readUnquote() (interface{}, error) {
	rn, _, err := r.rs.ReadRune()
	if err != nil {
		return nil, r.error("error reading input: %w", err)
	}
	if rn == '@' {
		return r.readQuoteType("glojure.core/splice-unquote")
	}

	r.rs.UnreadRune()
	return r.readQuoteType("glojure.core/unquote")
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
		return value.NewList(value.NewSymbol("var"), expr), nil
	case '"':
		return r.readRegex()
	case '^':
		r.rs.UnreadRune()
		// just read normally
		return r.readExpr()
	default:
		return nil, r.error("invalid dispatch character: %c", rn)
	}
}

func (r *Reader) readNamespacedMap() (interface{}, error) {
	nsKWVal, err := r.readKeyword()
	if err != nil {
		return nil, err
	}

	nsKW := nsKWVal.(value.Keyword)
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
	for mp := value.Seq(mapVal); mp != nil; mp = mp.Next() {
		kv := mp.First()

		key := kv.(*value.MapEntry).Key()
		val := kv.(*value.MapEntry).Val()

		keyKW, ok := key.(value.Keyword)
		if !ok || keyKW.Namespace() != "" {
			newKeyVals = append(newKeyVals, key, val)
			continue
		}
		newKey := value.NewKeyword(nsKW.Name() + "/" + keyKW.Name())
		newKeyVals = append(newKeyVals, newKey, val)
	}

	m, err := value.WithMeta(value.NewMap(newKeyVals...), mapVal.(value.IMeta).Meta())
	if err != nil {
		// This should never happen. Maps can have metadata.
		panic(err)
	}
	return m, nil
}

var (
	numPrefixRegex = regexp.MustCompile(`^[-+]?[0-9]+`)
	intRegex       = regexp.MustCompile(`^[-+]?\d(\d|[a-fA-F])*N?$`)
	ratioRegex     = regexp.MustCompile(`^[-+]?\d+\/\d+$`)
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

			return bi, nil
		}

		intVal, err := strconv.ParseInt(numStr, 0, 64)
		if err != nil {
			return nil, r.error("invalid number: %s", numStr)
		}

		return int64(intVal), nil
	}

	if ratioRegex.MatchString(numStr) {
		parts := strings.Split(numStr, "/")

		numBig, err := value.NewBigInt(parts[0])
		if err != nil {
			return nil, r.error("invalid ratio: %s", numStr)
		}
		denomBig, err := value.NewBigInt(parts[1])
		if err != nil {
			return nil, r.error("invalid ratio: %s", numStr)
		}
		return value.NewRatioBigInt(numBig, denomBig), nil
	}

	// else, it's a float
	// if the last character is M, it's a big decimal
	if strings.HasSuffix(numStr, "M") {
		bd, err := value.NewBigDecimal(numStr[:len(numStr)-1])
		if err != nil {
			return nil, r.error("invalid big decimal: %w", err)
		}
		return bd, nil
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return nil, r.error("invalid number: %s", numStr)
	}

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

	// check if symbol is a keyword
	switch sym {
	case "nil":
		return nil, nil
	case "true":
		return true, nil
	case "false":
		return false, nil
	}

	symVal := value.NewSymbol(sym)
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
	if sym == "" || sym == ":" || strings.Contains(sym[1:], ":") {
		return nil, r.error("invalid keyword: :" + sym)
	}
	if sym[0] == ':' {
		// TODO: handle auto-resolving keywords with namespaces
		ns := r.getCurrentNS().Name().Name()
		sym = ns + "/" + sym[1:]
	}
	return value.NewKeyword(sym), nil
}

func (r *Reader) readMeta() (value.IPersistentMap, error) {
	res, err := r.readExpr()
	if err != nil {
		return nil, err
	}

	switch res := res.(type) {
	case *value.Map:
		return res, nil
	case *value.Symbol, string:
		return value.NewMap(value.KWTag, res), nil
	case value.Keyword:
		return value.NewMap(res, true), nil
	default:
		return nil, r.error("metadata must be a map, symbol, keyword, or string")
	}
}

// Translated from Clojure's Compiler.java
func (r *Reader) resolveSymbol(sym *value.Symbol) *value.Symbol {
	if strings.Contains(sym.Name(), ".") {
		return sym
	}
	if sym.Namespace() != "" {
		ns := value.NamespaceFor(r.getCurrentNS(), sym)
		if ns == nil || (ns.Name().Name() == "" && sym.Namespace() == "") ||
			(ns.Name().Name() != "" && ns.Name().Name() == sym.Namespace()) {
			return sym
		}
		return value.InternSymbol(ns.Name().Name(), sym.Name())
	}

	currentNS := r.getCurrentNS()
	o := currentNS.GetMapping(sym)
	switch o := o.(type) {
	case nil:
		return value.InternSymbol(currentNS.Name().Name(), sym.Name())
	case *value.Var:
		return value.InternSymbol(o.Namespace().Name().Name(), o.Symbol().Name())
	}
	return nil
}

func isSpace(r rune) bool {
	return r == ',' || unicode.IsSpace(r)
}
