package gljimports

import (
	bytes "bytes"
	fmt "fmt"
	io "io"
	io_fs "io/fs"
	io_ioutil "io/ioutil"
	net_http "net/http"
	"reflect"
	regexp "regexp"
	strings "strings"
	time "time"

	"github.com/glojurelang/glojure/value"
)

func RegisterImports(_register func(string, value.Value)) {
	// package fmt
	////////////////////////////////////////
	_register("go/fmt.Errorf", value.NewGoVal(fmt.Errorf))
	{
		var x fmt.Formatter
		_register("go/fmt.Formatter", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/fmt.Fprint", value.NewGoVal(fmt.Fprint))
	_register("go/fmt.Fprintf", value.NewGoVal(fmt.Fprintf))
	_register("go/fmt.Fprintln", value.NewGoVal(fmt.Fprintln))
	_register("go/fmt.Fscan", value.NewGoVal(fmt.Fscan))
	_register("go/fmt.Fscanf", value.NewGoVal(fmt.Fscanf))
	_register("go/fmt.Fscanln", value.NewGoVal(fmt.Fscanln))
	{
		var x fmt.GoStringer
		_register("go/fmt.GoStringer", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/fmt.Print", value.NewGoVal(fmt.Print))
	_register("go/fmt.Printf", value.NewGoVal(fmt.Printf))
	_register("go/fmt.Println", value.NewGoVal(fmt.Println))
	_register("go/fmt.Scan", value.NewGoVal(fmt.Scan))
	{
		var x fmt.ScanState
		_register("go/fmt.ScanState", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/fmt.Scanf", value.NewGoVal(fmt.Scanf))
	_register("go/fmt.Scanln", value.NewGoVal(fmt.Scanln))
	{
		var x fmt.Scanner
		_register("go/fmt.Scanner", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/fmt.Sprint", value.NewGoVal(fmt.Sprint))
	_register("go/fmt.Sprintf", value.NewGoVal(fmt.Sprintf))
	_register("go/fmt.Sprintln", value.NewGoVal(fmt.Sprintln))
	_register("go/fmt.Sscan", value.NewGoVal(fmt.Sscan))
	_register("go/fmt.Sscanf", value.NewGoVal(fmt.Sscanf))
	_register("go/fmt.Sscanln", value.NewGoVal(fmt.Sscanln))
	{
		var x fmt.State
		_register("go/fmt.State", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x fmt.Stringer
		_register("go/fmt.Stringer", value.NewGoTyp(reflect.TypeOf(x)))
	}

	// package time
	////////////////////////////////////////
	_register("go/time.ANSIC", value.NewGoVal(time.ANSIC))
	_register("go/time.After", value.NewGoVal(time.After))
	_register("go/time.AfterFunc", value.NewGoVal(time.AfterFunc))
	_register("go/time.April", value.NewGoVal(time.April))
	_register("go/time.August", value.NewGoVal(time.August))
	_register("go/time.Date", value.NewGoVal(time.Date))
	_register("go/time.December", value.NewGoVal(time.December))
	{
		var x time.Duration
		_register("go/time.Duration", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/time.February", value.NewGoVal(time.February))
	_register("go/time.FixedZone", value.NewGoVal(time.FixedZone))
	_register("go/time.Friday", value.NewGoVal(time.Friday))
	_register("go/time.Hour", value.NewGoVal(time.Hour))
	_register("go/time.January", value.NewGoVal(time.January))
	_register("go/time.July", value.NewGoVal(time.July))
	_register("go/time.June", value.NewGoVal(time.June))
	_register("go/time.Kitchen", value.NewGoVal(time.Kitchen))
	_register("go/time.Layout", value.NewGoVal(time.Layout))
	_register("go/time.LoadLocation", value.NewGoVal(time.LoadLocation))
	_register("go/time.LoadLocationFromTZData", value.NewGoVal(time.LoadLocationFromTZData))
	_register("go/time.Local", value.NewGoVal(time.Local))
	{
		var x time.Location
		_register("go/time.Location", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/time.March", value.NewGoVal(time.March))
	_register("go/time.May", value.NewGoVal(time.May))
	_register("go/time.Microsecond", value.NewGoVal(time.Microsecond))
	_register("go/time.Millisecond", value.NewGoVal(time.Millisecond))
	_register("go/time.Minute", value.NewGoVal(time.Minute))
	_register("go/time.Monday", value.NewGoVal(time.Monday))
	{
		var x time.Month
		_register("go/time.Month", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/time.Nanosecond", value.NewGoVal(time.Nanosecond))
	_register("go/time.NewTicker", value.NewGoVal(time.NewTicker))
	_register("go/time.NewTimer", value.NewGoVal(time.NewTimer))
	_register("go/time.November", value.NewGoVal(time.November))
	_register("go/time.Now", value.NewGoVal(time.Now))
	_register("go/time.October", value.NewGoVal(time.October))
	_register("go/time.Parse", value.NewGoVal(time.Parse))
	_register("go/time.ParseDuration", value.NewGoVal(time.ParseDuration))
	{
		var x time.ParseError
		_register("go/time.ParseError", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/time.ParseInLocation", value.NewGoVal(time.ParseInLocation))
	_register("go/time.RFC1123", value.NewGoVal(time.RFC1123))
	_register("go/time.RFC1123Z", value.NewGoVal(time.RFC1123Z))
	_register("go/time.RFC3339", value.NewGoVal(time.RFC3339))
	_register("go/time.RFC3339Nano", value.NewGoVal(time.RFC3339Nano))
	_register("go/time.RFC822", value.NewGoVal(time.RFC822))
	_register("go/time.RFC822Z", value.NewGoVal(time.RFC822Z))
	_register("go/time.RFC850", value.NewGoVal(time.RFC850))
	_register("go/time.RubyDate", value.NewGoVal(time.RubyDate))
	_register("go/time.Saturday", value.NewGoVal(time.Saturday))
	_register("go/time.Second", value.NewGoVal(time.Second))
	_register("go/time.September", value.NewGoVal(time.September))
	_register("go/time.Since", value.NewGoVal(time.Since))
	_register("go/time.Sleep", value.NewGoVal(time.Sleep))
	_register("go/time.Stamp", value.NewGoVal(time.Stamp))
	_register("go/time.StampMicro", value.NewGoVal(time.StampMicro))
	_register("go/time.StampMilli", value.NewGoVal(time.StampMilli))
	_register("go/time.StampNano", value.NewGoVal(time.StampNano))
	_register("go/time.Sunday", value.NewGoVal(time.Sunday))
	_register("go/time.Thursday", value.NewGoVal(time.Thursday))
	_register("go/time.Tick", value.NewGoVal(time.Tick))
	{
		var x time.Ticker
		_register("go/time.Ticker", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x time.Time
		_register("go/time.Time", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x time.Timer
		_register("go/time.Timer", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/time.Tuesday", value.NewGoVal(time.Tuesday))
	_register("go/time.UTC", value.NewGoVal(time.UTC))
	_register("go/time.Unix", value.NewGoVal(time.Unix))
	_register("go/time.UnixDate", value.NewGoVal(time.UnixDate))
	_register("go/time.UnixMicro", value.NewGoVal(time.UnixMicro))
	_register("go/time.UnixMilli", value.NewGoVal(time.UnixMilli))
	_register("go/time.Until", value.NewGoVal(time.Until))
	_register("go/time.Wednesday", value.NewGoVal(time.Wednesday))
	{
		var x time.Weekday
		_register("go/time.Weekday", value.NewGoTyp(reflect.TypeOf(x)))
	}

	// package regexp
	////////////////////////////////////////
	_register("go/regexp.Compile", value.NewGoVal(regexp.Compile))
	_register("go/regexp.CompilePOSIX", value.NewGoVal(regexp.CompilePOSIX))
	_register("go/regexp.Match", value.NewGoVal(regexp.Match))
	_register("go/regexp.MatchReader", value.NewGoVal(regexp.MatchReader))
	_register("go/regexp.MatchString", value.NewGoVal(regexp.MatchString))
	_register("go/regexp.MustCompile", value.NewGoVal(regexp.MustCompile))
	_register("go/regexp.MustCompilePOSIX", value.NewGoVal(regexp.MustCompilePOSIX))
	_register("go/regexp.QuoteMeta", value.NewGoVal(regexp.QuoteMeta))
	{
		var x regexp.Regexp
		_register("go/regexp.Regexp", value.NewGoTyp(reflect.TypeOf(x)))
	}

	// package strings
	////////////////////////////////////////
	{
		var x strings.Builder
		_register("go/strings.Builder", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/strings.Clone", value.NewGoVal(strings.Clone))
	_register("go/strings.Compare", value.NewGoVal(strings.Compare))
	_register("go/strings.Contains", value.NewGoVal(strings.Contains))
	_register("go/strings.ContainsAny", value.NewGoVal(strings.ContainsAny))
	_register("go/strings.ContainsRune", value.NewGoVal(strings.ContainsRune))
	_register("go/strings.Count", value.NewGoVal(strings.Count))
	_register("go/strings.Cut", value.NewGoVal(strings.Cut))
	_register("go/strings.EqualFold", value.NewGoVal(strings.EqualFold))
	_register("go/strings.Fields", value.NewGoVal(strings.Fields))
	_register("go/strings.FieldsFunc", value.NewGoVal(strings.FieldsFunc))
	_register("go/strings.HasPrefix", value.NewGoVal(strings.HasPrefix))
	_register("go/strings.HasSuffix", value.NewGoVal(strings.HasSuffix))
	_register("go/strings.Index", value.NewGoVal(strings.Index))
	_register("go/strings.IndexAny", value.NewGoVal(strings.IndexAny))
	_register("go/strings.IndexByte", value.NewGoVal(strings.IndexByte))
	_register("go/strings.IndexFunc", value.NewGoVal(strings.IndexFunc))
	_register("go/strings.IndexRune", value.NewGoVal(strings.IndexRune))
	_register("go/strings.Join", value.NewGoVal(strings.Join))
	_register("go/strings.LastIndex", value.NewGoVal(strings.LastIndex))
	_register("go/strings.LastIndexAny", value.NewGoVal(strings.LastIndexAny))
	_register("go/strings.LastIndexByte", value.NewGoVal(strings.LastIndexByte))
	_register("go/strings.LastIndexFunc", value.NewGoVal(strings.LastIndexFunc))
	_register("go/strings.Map", value.NewGoVal(strings.Map))
	_register("go/strings.NewReader", value.NewGoVal(strings.NewReader))
	_register("go/strings.NewReplacer", value.NewGoVal(strings.NewReplacer))
	{
		var x strings.Reader
		_register("go/strings.Reader", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/strings.Repeat", value.NewGoVal(strings.Repeat))
	_register("go/strings.Replace", value.NewGoVal(strings.Replace))
	_register("go/strings.ReplaceAll", value.NewGoVal(strings.ReplaceAll))
	{
		var x strings.Replacer
		_register("go/strings.Replacer", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/strings.Split", value.NewGoVal(strings.Split))
	_register("go/strings.SplitAfter", value.NewGoVal(strings.SplitAfter))
	_register("go/strings.SplitAfterN", value.NewGoVal(strings.SplitAfterN))
	_register("go/strings.SplitN", value.NewGoVal(strings.SplitN))
	_register("go/strings.Title", value.NewGoVal(strings.Title))
	_register("go/strings.ToLower", value.NewGoVal(strings.ToLower))
	_register("go/strings.ToLowerSpecial", value.NewGoVal(strings.ToLowerSpecial))
	_register("go/strings.ToTitle", value.NewGoVal(strings.ToTitle))
	_register("go/strings.ToTitleSpecial", value.NewGoVal(strings.ToTitleSpecial))
	_register("go/strings.ToUpper", value.NewGoVal(strings.ToUpper))
	_register("go/strings.ToUpperSpecial", value.NewGoVal(strings.ToUpperSpecial))
	_register("go/strings.ToValidUTF8", value.NewGoVal(strings.ToValidUTF8))
	_register("go/strings.Trim", value.NewGoVal(strings.Trim))
	_register("go/strings.TrimFunc", value.NewGoVal(strings.TrimFunc))
	_register("go/strings.TrimLeft", value.NewGoVal(strings.TrimLeft))
	_register("go/strings.TrimLeftFunc", value.NewGoVal(strings.TrimLeftFunc))
	_register("go/strings.TrimPrefix", value.NewGoVal(strings.TrimPrefix))
	_register("go/strings.TrimRight", value.NewGoVal(strings.TrimRight))
	_register("go/strings.TrimRightFunc", value.NewGoVal(strings.TrimRightFunc))
	_register("go/strings.TrimSpace", value.NewGoVal(strings.TrimSpace))
	_register("go/strings.TrimSuffix", value.NewGoVal(strings.TrimSuffix))

	// package bytes
	////////////////////////////////////////
	{
		var x bytes.Buffer
		_register("go/bytes.Buffer", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/bytes.Compare", value.NewGoVal(bytes.Compare))
	_register("go/bytes.Contains", value.NewGoVal(bytes.Contains))
	_register("go/bytes.ContainsAny", value.NewGoVal(bytes.ContainsAny))
	_register("go/bytes.ContainsRune", value.NewGoVal(bytes.ContainsRune))
	_register("go/bytes.Count", value.NewGoVal(bytes.Count))
	_register("go/bytes.Cut", value.NewGoVal(bytes.Cut))
	_register("go/bytes.Equal", value.NewGoVal(bytes.Equal))
	_register("go/bytes.EqualFold", value.NewGoVal(bytes.EqualFold))
	_register("go/bytes.ErrTooLarge", value.NewGoVal(bytes.ErrTooLarge))
	_register("go/bytes.Fields", value.NewGoVal(bytes.Fields))
	_register("go/bytes.FieldsFunc", value.NewGoVal(bytes.FieldsFunc))
	_register("go/bytes.HasPrefix", value.NewGoVal(bytes.HasPrefix))
	_register("go/bytes.HasSuffix", value.NewGoVal(bytes.HasSuffix))
	_register("go/bytes.Index", value.NewGoVal(bytes.Index))
	_register("go/bytes.IndexAny", value.NewGoVal(bytes.IndexAny))
	_register("go/bytes.IndexByte", value.NewGoVal(bytes.IndexByte))
	_register("go/bytes.IndexFunc", value.NewGoVal(bytes.IndexFunc))
	_register("go/bytes.IndexRune", value.NewGoVal(bytes.IndexRune))
	_register("go/bytes.Join", value.NewGoVal(bytes.Join))
	_register("go/bytes.LastIndex", value.NewGoVal(bytes.LastIndex))
	_register("go/bytes.LastIndexAny", value.NewGoVal(bytes.LastIndexAny))
	_register("go/bytes.LastIndexByte", value.NewGoVal(bytes.LastIndexByte))
	_register("go/bytes.LastIndexFunc", value.NewGoVal(bytes.LastIndexFunc))
	_register("go/bytes.Map", value.NewGoVal(bytes.Map))
	_register("go/bytes.MinRead", value.NewGoVal(bytes.MinRead))
	_register("go/bytes.NewBuffer", value.NewGoVal(bytes.NewBuffer))
	_register("go/bytes.NewBufferString", value.NewGoVal(bytes.NewBufferString))
	_register("go/bytes.NewReader", value.NewGoVal(bytes.NewReader))
	{
		var x bytes.Reader
		_register("go/bytes.Reader", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/bytes.Repeat", value.NewGoVal(bytes.Repeat))
	_register("go/bytes.Replace", value.NewGoVal(bytes.Replace))
	_register("go/bytes.ReplaceAll", value.NewGoVal(bytes.ReplaceAll))
	_register("go/bytes.Runes", value.NewGoVal(bytes.Runes))
	_register("go/bytes.Split", value.NewGoVal(bytes.Split))
	_register("go/bytes.SplitAfter", value.NewGoVal(bytes.SplitAfter))
	_register("go/bytes.SplitAfterN", value.NewGoVal(bytes.SplitAfterN))
	_register("go/bytes.SplitN", value.NewGoVal(bytes.SplitN))
	_register("go/bytes.Title", value.NewGoVal(bytes.Title))
	_register("go/bytes.ToLower", value.NewGoVal(bytes.ToLower))
	_register("go/bytes.ToLowerSpecial", value.NewGoVal(bytes.ToLowerSpecial))
	_register("go/bytes.ToTitle", value.NewGoVal(bytes.ToTitle))
	_register("go/bytes.ToTitleSpecial", value.NewGoVal(bytes.ToTitleSpecial))
	_register("go/bytes.ToUpper", value.NewGoVal(bytes.ToUpper))
	_register("go/bytes.ToUpperSpecial", value.NewGoVal(bytes.ToUpperSpecial))
	_register("go/bytes.ToValidUTF8", value.NewGoVal(bytes.ToValidUTF8))
	_register("go/bytes.Trim", value.NewGoVal(bytes.Trim))
	_register("go/bytes.TrimFunc", value.NewGoVal(bytes.TrimFunc))
	_register("go/bytes.TrimLeft", value.NewGoVal(bytes.TrimLeft))
	_register("go/bytes.TrimLeftFunc", value.NewGoVal(bytes.TrimLeftFunc))
	_register("go/bytes.TrimPrefix", value.NewGoVal(bytes.TrimPrefix))
	_register("go/bytes.TrimRight", value.NewGoVal(bytes.TrimRight))
	_register("go/bytes.TrimRightFunc", value.NewGoVal(bytes.TrimRightFunc))
	_register("go/bytes.TrimSpace", value.NewGoVal(bytes.TrimSpace))
	_register("go/bytes.TrimSuffix", value.NewGoVal(bytes.TrimSuffix))

	// package net/http
	////////////////////////////////////////
	_register("go/net/http.AllowQuerySemicolons", value.NewGoVal(net_http.AllowQuerySemicolons))
	_register("go/net/http.CanonicalHeaderKey", value.NewGoVal(net_http.CanonicalHeaderKey))
	{
		var x net_http.Client
		_register("go/net/http.Client", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.CloseNotifier
		_register("go/net/http.CloseNotifier", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.ConnState
		_register("go/net/http.ConnState", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.Cookie
		_register("go/net/http.Cookie", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.CookieJar
		_register("go/net/http.CookieJar", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/net/http.DefaultClient", value.NewGoVal(net_http.DefaultClient))
	_register("go/net/http.DefaultMaxHeaderBytes", value.NewGoVal(net_http.DefaultMaxHeaderBytes))
	_register("go/net/http.DefaultMaxIdleConnsPerHost", value.NewGoVal(net_http.DefaultMaxIdleConnsPerHost))
	_register("go/net/http.DefaultServeMux", value.NewGoVal(net_http.DefaultServeMux))
	_register("go/net/http.DefaultTransport", value.NewGoVal(net_http.DefaultTransport))
	_register("go/net/http.DetectContentType", value.NewGoVal(net_http.DetectContentType))
	{
		var x net_http.Dir
		_register("go/net/http.Dir", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/net/http.ErrAbortHandler", value.NewGoVal(net_http.ErrAbortHandler))
	_register("go/net/http.ErrBodyNotAllowed", value.NewGoVal(net_http.ErrBodyNotAllowed))
	_register("go/net/http.ErrBodyReadAfterClose", value.NewGoVal(net_http.ErrBodyReadAfterClose))
	_register("go/net/http.ErrContentLength", value.NewGoVal(net_http.ErrContentLength))
	_register("go/net/http.ErrHandlerTimeout", value.NewGoVal(net_http.ErrHandlerTimeout))
	_register("go/net/http.ErrHeaderTooLong", value.NewGoVal(net_http.ErrHeaderTooLong))
	_register("go/net/http.ErrHijacked", value.NewGoVal(net_http.ErrHijacked))
	_register("go/net/http.ErrLineTooLong", value.NewGoVal(net_http.ErrLineTooLong))
	_register("go/net/http.ErrMissingBoundary", value.NewGoVal(net_http.ErrMissingBoundary))
	_register("go/net/http.ErrMissingContentLength", value.NewGoVal(net_http.ErrMissingContentLength))
	_register("go/net/http.ErrMissingFile", value.NewGoVal(net_http.ErrMissingFile))
	_register("go/net/http.ErrNoCookie", value.NewGoVal(net_http.ErrNoCookie))
	_register("go/net/http.ErrNoLocation", value.NewGoVal(net_http.ErrNoLocation))
	_register("go/net/http.ErrNotMultipart", value.NewGoVal(net_http.ErrNotMultipart))
	_register("go/net/http.ErrNotSupported", value.NewGoVal(net_http.ErrNotSupported))
	_register("go/net/http.ErrServerClosed", value.NewGoVal(net_http.ErrServerClosed))
	_register("go/net/http.ErrShortBody", value.NewGoVal(net_http.ErrShortBody))
	_register("go/net/http.ErrSkipAltProtocol", value.NewGoVal(net_http.ErrSkipAltProtocol))
	_register("go/net/http.ErrUnexpectedTrailer", value.NewGoVal(net_http.ErrUnexpectedTrailer))
	_register("go/net/http.ErrUseLastResponse", value.NewGoVal(net_http.ErrUseLastResponse))
	_register("go/net/http.ErrWriteAfterFlush", value.NewGoVal(net_http.ErrWriteAfterFlush))
	_register("go/net/http.Error", value.NewGoVal(net_http.Error))
	_register("go/net/http.FS", value.NewGoVal(net_http.FS))
	{
		var x net_http.File
		_register("go/net/http.File", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/net/http.FileServer", value.NewGoVal(net_http.FileServer))
	{
		var x net_http.FileSystem
		_register("go/net/http.FileSystem", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.Flusher
		_register("go/net/http.Flusher", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/net/http.Get", value.NewGoVal(net_http.Get))
	_register("go/net/http.Handle", value.NewGoVal(net_http.Handle))
	_register("go/net/http.HandleFunc", value.NewGoVal(net_http.HandleFunc))
	{
		var x net_http.Handler
		_register("go/net/http.Handler", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.HandlerFunc
		_register("go/net/http.HandlerFunc", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/net/http.Head", value.NewGoVal(net_http.Head))
	{
		var x net_http.Header
		_register("go/net/http.Header", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.Hijacker
		_register("go/net/http.Hijacker", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/net/http.ListenAndServe", value.NewGoVal(net_http.ListenAndServe))
	_register("go/net/http.ListenAndServeTLS", value.NewGoVal(net_http.ListenAndServeTLS))
	_register("go/net/http.LocalAddrContextKey", value.NewGoVal(net_http.LocalAddrContextKey))
	_register("go/net/http.MaxBytesHandler", value.NewGoVal(net_http.MaxBytesHandler))
	_register("go/net/http.MaxBytesReader", value.NewGoVal(net_http.MaxBytesReader))
	_register("go/net/http.MethodConnect", value.NewGoVal(net_http.MethodConnect))
	_register("go/net/http.MethodDelete", value.NewGoVal(net_http.MethodDelete))
	_register("go/net/http.MethodGet", value.NewGoVal(net_http.MethodGet))
	_register("go/net/http.MethodHead", value.NewGoVal(net_http.MethodHead))
	_register("go/net/http.MethodOptions", value.NewGoVal(net_http.MethodOptions))
	_register("go/net/http.MethodPatch", value.NewGoVal(net_http.MethodPatch))
	_register("go/net/http.MethodPost", value.NewGoVal(net_http.MethodPost))
	_register("go/net/http.MethodPut", value.NewGoVal(net_http.MethodPut))
	_register("go/net/http.MethodTrace", value.NewGoVal(net_http.MethodTrace))
	_register("go/net/http.NewFileTransport", value.NewGoVal(net_http.NewFileTransport))
	_register("go/net/http.NewRequest", value.NewGoVal(net_http.NewRequest))
	_register("go/net/http.NewRequestWithContext", value.NewGoVal(net_http.NewRequestWithContext))
	_register("go/net/http.NewServeMux", value.NewGoVal(net_http.NewServeMux))
	_register("go/net/http.NoBody", value.NewGoVal(net_http.NoBody))
	_register("go/net/http.NotFound", value.NewGoVal(net_http.NotFound))
	_register("go/net/http.NotFoundHandler", value.NewGoVal(net_http.NotFoundHandler))
	_register("go/net/http.ParseHTTPVersion", value.NewGoVal(net_http.ParseHTTPVersion))
	_register("go/net/http.ParseTime", value.NewGoVal(net_http.ParseTime))
	_register("go/net/http.Post", value.NewGoVal(net_http.Post))
	_register("go/net/http.PostForm", value.NewGoVal(net_http.PostForm))
	{
		var x net_http.ProtocolError
		_register("go/net/http.ProtocolError", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/net/http.ProxyFromEnvironment", value.NewGoVal(net_http.ProxyFromEnvironment))
	_register("go/net/http.ProxyURL", value.NewGoVal(net_http.ProxyURL))
	{
		var x net_http.PushOptions
		_register("go/net/http.PushOptions", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.Pusher
		_register("go/net/http.Pusher", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/net/http.ReadRequest", value.NewGoVal(net_http.ReadRequest))
	_register("go/net/http.ReadResponse", value.NewGoVal(net_http.ReadResponse))
	_register("go/net/http.Redirect", value.NewGoVal(net_http.Redirect))
	_register("go/net/http.RedirectHandler", value.NewGoVal(net_http.RedirectHandler))
	{
		var x net_http.Request
		_register("go/net/http.Request", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.Response
		_register("go/net/http.Response", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.ResponseWriter
		_register("go/net/http.ResponseWriter", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.RoundTripper
		_register("go/net/http.RoundTripper", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.SameSite
		_register("go/net/http.SameSite", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/net/http.SameSiteDefaultMode", value.NewGoVal(net_http.SameSiteDefaultMode))
	_register("go/net/http.SameSiteLaxMode", value.NewGoVal(net_http.SameSiteLaxMode))
	_register("go/net/http.SameSiteNoneMode", value.NewGoVal(net_http.SameSiteNoneMode))
	_register("go/net/http.SameSiteStrictMode", value.NewGoVal(net_http.SameSiteStrictMode))
	_register("go/net/http.Serve", value.NewGoVal(net_http.Serve))
	_register("go/net/http.ServeContent", value.NewGoVal(net_http.ServeContent))
	_register("go/net/http.ServeFile", value.NewGoVal(net_http.ServeFile))
	{
		var x net_http.ServeMux
		_register("go/net/http.ServeMux", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/net/http.ServeTLS", value.NewGoVal(net_http.ServeTLS))
	{
		var x net_http.Server
		_register("go/net/http.Server", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/net/http.ServerContextKey", value.NewGoVal(net_http.ServerContextKey))
	_register("go/net/http.SetCookie", value.NewGoVal(net_http.SetCookie))
	_register("go/net/http.StateActive", value.NewGoVal(net_http.StateActive))
	_register("go/net/http.StateClosed", value.NewGoVal(net_http.StateClosed))
	_register("go/net/http.StateHijacked", value.NewGoVal(net_http.StateHijacked))
	_register("go/net/http.StateIdle", value.NewGoVal(net_http.StateIdle))
	_register("go/net/http.StateNew", value.NewGoVal(net_http.StateNew))
	_register("go/net/http.StatusAccepted", value.NewGoVal(net_http.StatusAccepted))
	_register("go/net/http.StatusAlreadyReported", value.NewGoVal(net_http.StatusAlreadyReported))
	_register("go/net/http.StatusBadGateway", value.NewGoVal(net_http.StatusBadGateway))
	_register("go/net/http.StatusBadRequest", value.NewGoVal(net_http.StatusBadRequest))
	_register("go/net/http.StatusConflict", value.NewGoVal(net_http.StatusConflict))
	_register("go/net/http.StatusContinue", value.NewGoVal(net_http.StatusContinue))
	_register("go/net/http.StatusCreated", value.NewGoVal(net_http.StatusCreated))
	_register("go/net/http.StatusEarlyHints", value.NewGoVal(net_http.StatusEarlyHints))
	_register("go/net/http.StatusExpectationFailed", value.NewGoVal(net_http.StatusExpectationFailed))
	_register("go/net/http.StatusFailedDependency", value.NewGoVal(net_http.StatusFailedDependency))
	_register("go/net/http.StatusForbidden", value.NewGoVal(net_http.StatusForbidden))
	_register("go/net/http.StatusFound", value.NewGoVal(net_http.StatusFound))
	_register("go/net/http.StatusGatewayTimeout", value.NewGoVal(net_http.StatusGatewayTimeout))
	_register("go/net/http.StatusGone", value.NewGoVal(net_http.StatusGone))
	_register("go/net/http.StatusHTTPVersionNotSupported", value.NewGoVal(net_http.StatusHTTPVersionNotSupported))
	_register("go/net/http.StatusIMUsed", value.NewGoVal(net_http.StatusIMUsed))
	_register("go/net/http.StatusInsufficientStorage", value.NewGoVal(net_http.StatusInsufficientStorage))
	_register("go/net/http.StatusInternalServerError", value.NewGoVal(net_http.StatusInternalServerError))
	_register("go/net/http.StatusLengthRequired", value.NewGoVal(net_http.StatusLengthRequired))
	_register("go/net/http.StatusLocked", value.NewGoVal(net_http.StatusLocked))
	_register("go/net/http.StatusLoopDetected", value.NewGoVal(net_http.StatusLoopDetected))
	_register("go/net/http.StatusMethodNotAllowed", value.NewGoVal(net_http.StatusMethodNotAllowed))
	_register("go/net/http.StatusMisdirectedRequest", value.NewGoVal(net_http.StatusMisdirectedRequest))
	_register("go/net/http.StatusMovedPermanently", value.NewGoVal(net_http.StatusMovedPermanently))
	_register("go/net/http.StatusMultiStatus", value.NewGoVal(net_http.StatusMultiStatus))
	_register("go/net/http.StatusMultipleChoices", value.NewGoVal(net_http.StatusMultipleChoices))
	_register("go/net/http.StatusNetworkAuthenticationRequired", value.NewGoVal(net_http.StatusNetworkAuthenticationRequired))
	_register("go/net/http.StatusNoContent", value.NewGoVal(net_http.StatusNoContent))
	_register("go/net/http.StatusNonAuthoritativeInfo", value.NewGoVal(net_http.StatusNonAuthoritativeInfo))
	_register("go/net/http.StatusNotAcceptable", value.NewGoVal(net_http.StatusNotAcceptable))
	_register("go/net/http.StatusNotExtended", value.NewGoVal(net_http.StatusNotExtended))
	_register("go/net/http.StatusNotFound", value.NewGoVal(net_http.StatusNotFound))
	_register("go/net/http.StatusNotImplemented", value.NewGoVal(net_http.StatusNotImplemented))
	_register("go/net/http.StatusNotModified", value.NewGoVal(net_http.StatusNotModified))
	_register("go/net/http.StatusOK", value.NewGoVal(net_http.StatusOK))
	_register("go/net/http.StatusPartialContent", value.NewGoVal(net_http.StatusPartialContent))
	_register("go/net/http.StatusPaymentRequired", value.NewGoVal(net_http.StatusPaymentRequired))
	_register("go/net/http.StatusPermanentRedirect", value.NewGoVal(net_http.StatusPermanentRedirect))
	_register("go/net/http.StatusPreconditionFailed", value.NewGoVal(net_http.StatusPreconditionFailed))
	_register("go/net/http.StatusPreconditionRequired", value.NewGoVal(net_http.StatusPreconditionRequired))
	_register("go/net/http.StatusProcessing", value.NewGoVal(net_http.StatusProcessing))
	_register("go/net/http.StatusProxyAuthRequired", value.NewGoVal(net_http.StatusProxyAuthRequired))
	_register("go/net/http.StatusRequestEntityTooLarge", value.NewGoVal(net_http.StatusRequestEntityTooLarge))
	_register("go/net/http.StatusRequestHeaderFieldsTooLarge", value.NewGoVal(net_http.StatusRequestHeaderFieldsTooLarge))
	_register("go/net/http.StatusRequestTimeout", value.NewGoVal(net_http.StatusRequestTimeout))
	_register("go/net/http.StatusRequestURITooLong", value.NewGoVal(net_http.StatusRequestURITooLong))
	_register("go/net/http.StatusRequestedRangeNotSatisfiable", value.NewGoVal(net_http.StatusRequestedRangeNotSatisfiable))
	_register("go/net/http.StatusResetContent", value.NewGoVal(net_http.StatusResetContent))
	_register("go/net/http.StatusSeeOther", value.NewGoVal(net_http.StatusSeeOther))
	_register("go/net/http.StatusServiceUnavailable", value.NewGoVal(net_http.StatusServiceUnavailable))
	_register("go/net/http.StatusSwitchingProtocols", value.NewGoVal(net_http.StatusSwitchingProtocols))
	_register("go/net/http.StatusTeapot", value.NewGoVal(net_http.StatusTeapot))
	_register("go/net/http.StatusTemporaryRedirect", value.NewGoVal(net_http.StatusTemporaryRedirect))
	_register("go/net/http.StatusText", value.NewGoVal(net_http.StatusText))
	_register("go/net/http.StatusTooEarly", value.NewGoVal(net_http.StatusTooEarly))
	_register("go/net/http.StatusTooManyRequests", value.NewGoVal(net_http.StatusTooManyRequests))
	_register("go/net/http.StatusUnauthorized", value.NewGoVal(net_http.StatusUnauthorized))
	_register("go/net/http.StatusUnavailableForLegalReasons", value.NewGoVal(net_http.StatusUnavailableForLegalReasons))
	_register("go/net/http.StatusUnprocessableEntity", value.NewGoVal(net_http.StatusUnprocessableEntity))
	_register("go/net/http.StatusUnsupportedMediaType", value.NewGoVal(net_http.StatusUnsupportedMediaType))
	_register("go/net/http.StatusUpgradeRequired", value.NewGoVal(net_http.StatusUpgradeRequired))
	_register("go/net/http.StatusUseProxy", value.NewGoVal(net_http.StatusUseProxy))
	_register("go/net/http.StatusVariantAlsoNegotiates", value.NewGoVal(net_http.StatusVariantAlsoNegotiates))
	_register("go/net/http.StripPrefix", value.NewGoVal(net_http.StripPrefix))
	_register("go/net/http.TimeFormat", value.NewGoVal(net_http.TimeFormat))
	_register("go/net/http.TimeoutHandler", value.NewGoVal(net_http.TimeoutHandler))
	_register("go/net/http.TrailerPrefix", value.NewGoVal(net_http.TrailerPrefix))
	{
		var x net_http.Transport
		_register("go/net/http.Transport", value.NewGoTyp(reflect.TypeOf(x)))
	}

	// package io
	////////////////////////////////////////
	{
		var x io.ByteReader
		_register("go/io.ByteReader", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.ByteScanner
		_register("go/io.ByteScanner", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.ByteWriter
		_register("go/io.ByteWriter", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.Closer
		_register("go/io.Closer", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/io.Copy", value.NewGoVal(io.Copy))
	_register("go/io.CopyBuffer", value.NewGoVal(io.CopyBuffer))
	_register("go/io.CopyN", value.NewGoVal(io.CopyN))
	_register("go/io.Discard", value.NewGoVal(io.Discard))
	_register("go/io.EOF", value.NewGoVal(io.EOF))
	_register("go/io.ErrClosedPipe", value.NewGoVal(io.ErrClosedPipe))
	_register("go/io.ErrNoProgress", value.NewGoVal(io.ErrNoProgress))
	_register("go/io.ErrShortBuffer", value.NewGoVal(io.ErrShortBuffer))
	_register("go/io.ErrShortWrite", value.NewGoVal(io.ErrShortWrite))
	_register("go/io.ErrUnexpectedEOF", value.NewGoVal(io.ErrUnexpectedEOF))
	_register("go/io.LimitReader", value.NewGoVal(io.LimitReader))
	{
		var x io.LimitedReader
		_register("go/io.LimitedReader", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/io.MultiReader", value.NewGoVal(io.MultiReader))
	_register("go/io.MultiWriter", value.NewGoVal(io.MultiWriter))
	_register("go/io.NewSectionReader", value.NewGoVal(io.NewSectionReader))
	_register("go/io.NopCloser", value.NewGoVal(io.NopCloser))
	_register("go/io.Pipe", value.NewGoVal(io.Pipe))
	{
		var x io.PipeReader
		_register("go/io.PipeReader", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.PipeWriter
		_register("go/io.PipeWriter", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/io.ReadAll", value.NewGoVal(io.ReadAll))
	_register("go/io.ReadAtLeast", value.NewGoVal(io.ReadAtLeast))
	{
		var x io.ReadCloser
		_register("go/io.ReadCloser", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/io.ReadFull", value.NewGoVal(io.ReadFull))
	{
		var x io.ReadSeekCloser
		_register("go/io.ReadSeekCloser", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.ReadSeeker
		_register("go/io.ReadSeeker", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.ReadWriteCloser
		_register("go/io.ReadWriteCloser", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.ReadWriteSeeker
		_register("go/io.ReadWriteSeeker", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.ReadWriter
		_register("go/io.ReadWriter", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.Reader
		_register("go/io.Reader", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.ReaderAt
		_register("go/io.ReaderAt", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.ReaderFrom
		_register("go/io.ReaderFrom", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.RuneReader
		_register("go/io.RuneReader", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.RuneScanner
		_register("go/io.RuneScanner", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.SectionReader
		_register("go/io.SectionReader", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/io.SeekCurrent", value.NewGoVal(io.SeekCurrent))
	_register("go/io.SeekEnd", value.NewGoVal(io.SeekEnd))
	_register("go/io.SeekStart", value.NewGoVal(io.SeekStart))
	{
		var x io.Seeker
		_register("go/io.Seeker", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.StringWriter
		_register("go/io.StringWriter", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/io.TeeReader", value.NewGoVal(io.TeeReader))
	{
		var x io.WriteCloser
		_register("go/io.WriteCloser", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.WriteSeeker
		_register("go/io.WriteSeeker", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/io.WriteString", value.NewGoVal(io.WriteString))
	{
		var x io.Writer
		_register("go/io.Writer", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.WriterAt
		_register("go/io.WriterAt", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.WriterTo
		_register("go/io.WriterTo", value.NewGoTyp(reflect.TypeOf(x)))
	}

	// package io/ioutil
	////////////////////////////////////////
	_register("go/io/ioutil.Discard", value.NewGoVal(io_ioutil.Discard))
	_register("go/io/ioutil.NopCloser", value.NewGoVal(io_ioutil.NopCloser))
	_register("go/io/ioutil.ReadAll", value.NewGoVal(io_ioutil.ReadAll))
	_register("go/io/ioutil.ReadDir", value.NewGoVal(io_ioutil.ReadDir))
	_register("go/io/ioutil.ReadFile", value.NewGoVal(io_ioutil.ReadFile))
	_register("go/io/ioutil.TempDir", value.NewGoVal(io_ioutil.TempDir))
	_register("go/io/ioutil.TempFile", value.NewGoVal(io_ioutil.TempFile))
	_register("go/io/ioutil.WriteFile", value.NewGoVal(io_ioutil.WriteFile))

	// package io/fs
	////////////////////////////////////////
	{
		var x io_fs.DirEntry
		_register("go/io/fs.DirEntry", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/io/fs.ErrClosed", value.NewGoVal(io_fs.ErrClosed))
	_register("go/io/fs.ErrExist", value.NewGoVal(io_fs.ErrExist))
	_register("go/io/fs.ErrInvalid", value.NewGoVal(io_fs.ErrInvalid))
	_register("go/io/fs.ErrNotExist", value.NewGoVal(io_fs.ErrNotExist))
	_register("go/io/fs.ErrPermission", value.NewGoVal(io_fs.ErrPermission))
	{
		var x io_fs.FS
		_register("go/io/fs.FS", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io_fs.File
		_register("go/io/fs.File", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io_fs.FileInfo
		_register("go/io/fs.FileInfo", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/io/fs.FileInfoToDirEntry", value.NewGoVal(io_fs.FileInfoToDirEntry))
	{
		var x io_fs.FileMode
		_register("go/io/fs.FileMode", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/io/fs.Glob", value.NewGoVal(io_fs.Glob))
	{
		var x io_fs.GlobFS
		_register("go/io/fs.GlobFS", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/io/fs.ModeAppend", value.NewGoVal(io_fs.ModeAppend))
	_register("go/io/fs.ModeCharDevice", value.NewGoVal(io_fs.ModeCharDevice))
	_register("go/io/fs.ModeDevice", value.NewGoVal(io_fs.ModeDevice))
	_register("go/io/fs.ModeDir", value.NewGoVal(io_fs.ModeDir))
	_register("go/io/fs.ModeExclusive", value.NewGoVal(io_fs.ModeExclusive))
	_register("go/io/fs.ModeIrregular", value.NewGoVal(io_fs.ModeIrregular))
	_register("go/io/fs.ModeNamedPipe", value.NewGoVal(io_fs.ModeNamedPipe))
	_register("go/io/fs.ModePerm", value.NewGoVal(io_fs.ModePerm))
	_register("go/io/fs.ModeSetgid", value.NewGoVal(io_fs.ModeSetgid))
	_register("go/io/fs.ModeSetuid", value.NewGoVal(io_fs.ModeSetuid))
	_register("go/io/fs.ModeSocket", value.NewGoVal(io_fs.ModeSocket))
	_register("go/io/fs.ModeSticky", value.NewGoVal(io_fs.ModeSticky))
	_register("go/io/fs.ModeSymlink", value.NewGoVal(io_fs.ModeSymlink))
	_register("go/io/fs.ModeTemporary", value.NewGoVal(io_fs.ModeTemporary))
	_register("go/io/fs.ModeType", value.NewGoVal(io_fs.ModeType))
	{
		var x io_fs.PathError
		_register("go/io/fs.PathError", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/io/fs.ReadDir", value.NewGoVal(io_fs.ReadDir))
	{
		var x io_fs.ReadDirFS
		_register("go/io/fs.ReadDirFS", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io_fs.ReadDirFile
		_register("go/io/fs.ReadDirFile", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/io/fs.ReadFile", value.NewGoVal(io_fs.ReadFile))
	{
		var x io_fs.ReadFileFS
		_register("go/io/fs.ReadFileFS", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/io/fs.SkipDir", value.NewGoVal(io_fs.SkipDir))
	_register("go/io/fs.Stat", value.NewGoVal(io_fs.Stat))
	{
		var x io_fs.StatFS
		_register("go/io/fs.StatFS", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/io/fs.Sub", value.NewGoVal(io_fs.Sub))
	{
		var x io_fs.SubFS
		_register("go/io/fs.SubFS", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("go/io/fs.ValidPath", value.NewGoVal(io_fs.ValidPath))
	_register("go/io/fs.WalkDir", value.NewGoVal(io_fs.WalkDir))
	{
		var x io_fs.WalkDirFunc
		_register("go/io/fs.WalkDirFunc", value.NewGoTyp(reflect.TypeOf(x)))
	}
}
