// GENERATED FILE. DO NOT EDIT.
package gljimports

import (
	bytes "bytes"
	context "context"
	fmt "fmt"
	io "io"
	io_fs "io/fs"
	io_ioutil "io/ioutil"
	net_http "net/http"
	regexp "regexp"
	strconv "strconv"
	strings "strings"
	time "time"
	math_big "math/big"
	math_rand "math/rand"
	math "math"
	"reflect"
	"github.com/glojurelang/glojure/value"
)

func RegisterImports(_register func(string, value.Value)) {
	// package bytes
	////////////////////////////////////////
	{
		var x bytes.Buffer
		_register("bytes.Buffer", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("bytes.Compare", value.NewGoVal(bytes.Compare))
	_register("bytes.Contains", value.NewGoVal(bytes.Contains))
	_register("bytes.ContainsAny", value.NewGoVal(bytes.ContainsAny))
	_register("bytes.ContainsRune", value.NewGoVal(bytes.ContainsRune))
	_register("bytes.Count", value.NewGoVal(bytes.Count))
	_register("bytes.Cut", value.NewGoVal(bytes.Cut))
	_register("bytes.Equal", value.NewGoVal(bytes.Equal))
	_register("bytes.EqualFold", value.NewGoVal(bytes.EqualFold))
	_register("bytes.ErrTooLarge", value.NewGoVal(bytes.ErrTooLarge))
	_register("bytes.Fields", value.NewGoVal(bytes.Fields))
	_register("bytes.FieldsFunc", value.NewGoVal(bytes.FieldsFunc))
	_register("bytes.HasPrefix", value.NewGoVal(bytes.HasPrefix))
	_register("bytes.HasSuffix", value.NewGoVal(bytes.HasSuffix))
	_register("bytes.Index", value.NewGoVal(bytes.Index))
	_register("bytes.IndexAny", value.NewGoVal(bytes.IndexAny))
	_register("bytes.IndexByte", value.NewGoVal(bytes.IndexByte))
	_register("bytes.IndexFunc", value.NewGoVal(bytes.IndexFunc))
	_register("bytes.IndexRune", value.NewGoVal(bytes.IndexRune))
	_register("bytes.Join", value.NewGoVal(bytes.Join))
	_register("bytes.LastIndex", value.NewGoVal(bytes.LastIndex))
	_register("bytes.LastIndexAny", value.NewGoVal(bytes.LastIndexAny))
	_register("bytes.LastIndexByte", value.NewGoVal(bytes.LastIndexByte))
	_register("bytes.LastIndexFunc", value.NewGoVal(bytes.LastIndexFunc))
	_register("bytes.Map", value.NewGoVal(bytes.Map))
	_register("bytes.MinRead", value.NewGoVal(bytes.MinRead))
	_register("bytes.NewBuffer", value.NewGoVal(bytes.NewBuffer))
	_register("bytes.NewBufferString", value.NewGoVal(bytes.NewBufferString))
	_register("bytes.NewReader", value.NewGoVal(bytes.NewReader))
	{
		var x bytes.Reader
		_register("bytes.Reader", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("bytes.Repeat", value.NewGoVal(bytes.Repeat))
	_register("bytes.Replace", value.NewGoVal(bytes.Replace))
	_register("bytes.ReplaceAll", value.NewGoVal(bytes.ReplaceAll))
	_register("bytes.Runes", value.NewGoVal(bytes.Runes))
	_register("bytes.Split", value.NewGoVal(bytes.Split))
	_register("bytes.SplitAfter", value.NewGoVal(bytes.SplitAfter))
	_register("bytes.SplitAfterN", value.NewGoVal(bytes.SplitAfterN))
	_register("bytes.SplitN", value.NewGoVal(bytes.SplitN))
	_register("bytes.Title", value.NewGoVal(bytes.Title))
	_register("bytes.ToLower", value.NewGoVal(bytes.ToLower))
	_register("bytes.ToLowerSpecial", value.NewGoVal(bytes.ToLowerSpecial))
	_register("bytes.ToTitle", value.NewGoVal(bytes.ToTitle))
	_register("bytes.ToTitleSpecial", value.NewGoVal(bytes.ToTitleSpecial))
	_register("bytes.ToUpper", value.NewGoVal(bytes.ToUpper))
	_register("bytes.ToUpperSpecial", value.NewGoVal(bytes.ToUpperSpecial))
	_register("bytes.ToValidUTF8", value.NewGoVal(bytes.ToValidUTF8))
	_register("bytes.Trim", value.NewGoVal(bytes.Trim))
	_register("bytes.TrimFunc", value.NewGoVal(bytes.TrimFunc))
	_register("bytes.TrimLeft", value.NewGoVal(bytes.TrimLeft))
	_register("bytes.TrimLeftFunc", value.NewGoVal(bytes.TrimLeftFunc))
	_register("bytes.TrimPrefix", value.NewGoVal(bytes.TrimPrefix))
	_register("bytes.TrimRight", value.NewGoVal(bytes.TrimRight))
	_register("bytes.TrimRightFunc", value.NewGoVal(bytes.TrimRightFunc))
	_register("bytes.TrimSpace", value.NewGoVal(bytes.TrimSpace))
	_register("bytes.TrimSuffix", value.NewGoVal(bytes.TrimSuffix))

	// package context
	////////////////////////////////////////
	_register("context.Background", value.NewGoVal(context.Background))
	{
		var x context.CancelFunc
		_register("context.CancelFunc", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("context.Canceled", value.NewGoVal(context.Canceled))
	{
		var x context.Context
		_register("context.Context", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("context.DeadlineExceeded", value.NewGoVal(context.DeadlineExceeded))
	_register("context.TODO", value.NewGoVal(context.TODO))
	_register("context.WithCancel", value.NewGoVal(context.WithCancel))
	_register("context.WithDeadline", value.NewGoVal(context.WithDeadline))
	_register("context.WithTimeout", value.NewGoVal(context.WithTimeout))
	_register("context.WithValue", value.NewGoVal(context.WithValue))

	// package fmt
	////////////////////////////////////////
	_register("fmt.Errorf", value.NewGoVal(fmt.Errorf))
	{
		var x fmt.Formatter
		_register("fmt.Formatter", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("fmt.Fprint", value.NewGoVal(fmt.Fprint))
	_register("fmt.Fprintf", value.NewGoVal(fmt.Fprintf))
	_register("fmt.Fprintln", value.NewGoVal(fmt.Fprintln))
	_register("fmt.Fscan", value.NewGoVal(fmt.Fscan))
	_register("fmt.Fscanf", value.NewGoVal(fmt.Fscanf))
	_register("fmt.Fscanln", value.NewGoVal(fmt.Fscanln))
	{
		var x fmt.GoStringer
		_register("fmt.GoStringer", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("fmt.Print", value.NewGoVal(fmt.Print))
	_register("fmt.Printf", value.NewGoVal(fmt.Printf))
	_register("fmt.Println", value.NewGoVal(fmt.Println))
	_register("fmt.Scan", value.NewGoVal(fmt.Scan))
	{
		var x fmt.ScanState
		_register("fmt.ScanState", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("fmt.Scanf", value.NewGoVal(fmt.Scanf))
	_register("fmt.Scanln", value.NewGoVal(fmt.Scanln))
	{
		var x fmt.Scanner
		_register("fmt.Scanner", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("fmt.Sprint", value.NewGoVal(fmt.Sprint))
	_register("fmt.Sprintf", value.NewGoVal(fmt.Sprintf))
	_register("fmt.Sprintln", value.NewGoVal(fmt.Sprintln))
	_register("fmt.Sscan", value.NewGoVal(fmt.Sscan))
	_register("fmt.Sscanf", value.NewGoVal(fmt.Sscanf))
	_register("fmt.Sscanln", value.NewGoVal(fmt.Sscanln))
	{
		var x fmt.State
		_register("fmt.State", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x fmt.Stringer
		_register("fmt.Stringer", value.NewGoTyp(reflect.TypeOf(x)))
	}

	// package io
	////////////////////////////////////////
	{
		var x io.ByteReader
		_register("io.ByteReader", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.ByteScanner
		_register("io.ByteScanner", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.ByteWriter
		_register("io.ByteWriter", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.Closer
		_register("io.Closer", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("io.Copy", value.NewGoVal(io.Copy))
	_register("io.CopyBuffer", value.NewGoVal(io.CopyBuffer))
	_register("io.CopyN", value.NewGoVal(io.CopyN))
	_register("io.Discard", value.NewGoVal(io.Discard))
	_register("io.EOF", value.NewGoVal(io.EOF))
	_register("io.ErrClosedPipe", value.NewGoVal(io.ErrClosedPipe))
	_register("io.ErrNoProgress", value.NewGoVal(io.ErrNoProgress))
	_register("io.ErrShortBuffer", value.NewGoVal(io.ErrShortBuffer))
	_register("io.ErrShortWrite", value.NewGoVal(io.ErrShortWrite))
	_register("io.ErrUnexpectedEOF", value.NewGoVal(io.ErrUnexpectedEOF))
	_register("io.LimitReader", value.NewGoVal(io.LimitReader))
	{
		var x io.LimitedReader
		_register("io.LimitedReader", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("io.MultiReader", value.NewGoVal(io.MultiReader))
	_register("io.MultiWriter", value.NewGoVal(io.MultiWriter))
	_register("io.NewSectionReader", value.NewGoVal(io.NewSectionReader))
	_register("io.NopCloser", value.NewGoVal(io.NopCloser))
	_register("io.Pipe", value.NewGoVal(io.Pipe))
	{
		var x io.PipeReader
		_register("io.PipeReader", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.PipeWriter
		_register("io.PipeWriter", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("io.ReadAll", value.NewGoVal(io.ReadAll))
	_register("io.ReadAtLeast", value.NewGoVal(io.ReadAtLeast))
	{
		var x io.ReadCloser
		_register("io.ReadCloser", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("io.ReadFull", value.NewGoVal(io.ReadFull))
	{
		var x io.ReadSeekCloser
		_register("io.ReadSeekCloser", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.ReadSeeker
		_register("io.ReadSeeker", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.ReadWriteCloser
		_register("io.ReadWriteCloser", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.ReadWriteSeeker
		_register("io.ReadWriteSeeker", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.ReadWriter
		_register("io.ReadWriter", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.Reader
		_register("io.Reader", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.ReaderAt
		_register("io.ReaderAt", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.ReaderFrom
		_register("io.ReaderFrom", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.RuneReader
		_register("io.RuneReader", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.RuneScanner
		_register("io.RuneScanner", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.SectionReader
		_register("io.SectionReader", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("io.SeekCurrent", value.NewGoVal(io.SeekCurrent))
	_register("io.SeekEnd", value.NewGoVal(io.SeekEnd))
	_register("io.SeekStart", value.NewGoVal(io.SeekStart))
	{
		var x io.Seeker
		_register("io.Seeker", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.StringWriter
		_register("io.StringWriter", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("io.TeeReader", value.NewGoVal(io.TeeReader))
	{
		var x io.WriteCloser
		_register("io.WriteCloser", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.WriteSeeker
		_register("io.WriteSeeker", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("io.WriteString", value.NewGoVal(io.WriteString))
	{
		var x io.Writer
		_register("io.Writer", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.WriterAt
		_register("io.WriterAt", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io.WriterTo
		_register("io.WriterTo", value.NewGoTyp(reflect.TypeOf(x)))
	}

	// package io/fs
	////////////////////////////////////////
	{
		var x io_fs.DirEntry
		_register("io/fs.DirEntry", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("io/fs.ErrClosed", value.NewGoVal(io_fs.ErrClosed))
	_register("io/fs.ErrExist", value.NewGoVal(io_fs.ErrExist))
	_register("io/fs.ErrInvalid", value.NewGoVal(io_fs.ErrInvalid))
	_register("io/fs.ErrNotExist", value.NewGoVal(io_fs.ErrNotExist))
	_register("io/fs.ErrPermission", value.NewGoVal(io_fs.ErrPermission))
	{
		var x io_fs.FS
		_register("io/fs.FS", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io_fs.File
		_register("io/fs.File", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io_fs.FileInfo
		_register("io/fs.FileInfo", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("io/fs.FileInfoToDirEntry", value.NewGoVal(io_fs.FileInfoToDirEntry))
	{
		var x io_fs.FileMode
		_register("io/fs.FileMode", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("io/fs.Glob", value.NewGoVal(io_fs.Glob))
	{
		var x io_fs.GlobFS
		_register("io/fs.GlobFS", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("io/fs.ModeAppend", value.NewGoVal(io_fs.ModeAppend))
	_register("io/fs.ModeCharDevice", value.NewGoVal(io_fs.ModeCharDevice))
	_register("io/fs.ModeDevice", value.NewGoVal(io_fs.ModeDevice))
	_register("io/fs.ModeDir", value.NewGoVal(io_fs.ModeDir))
	_register("io/fs.ModeExclusive", value.NewGoVal(io_fs.ModeExclusive))
	_register("io/fs.ModeIrregular", value.NewGoVal(io_fs.ModeIrregular))
	_register("io/fs.ModeNamedPipe", value.NewGoVal(io_fs.ModeNamedPipe))
	_register("io/fs.ModePerm", value.NewGoVal(io_fs.ModePerm))
	_register("io/fs.ModeSetgid", value.NewGoVal(io_fs.ModeSetgid))
	_register("io/fs.ModeSetuid", value.NewGoVal(io_fs.ModeSetuid))
	_register("io/fs.ModeSocket", value.NewGoVal(io_fs.ModeSocket))
	_register("io/fs.ModeSticky", value.NewGoVal(io_fs.ModeSticky))
	_register("io/fs.ModeSymlink", value.NewGoVal(io_fs.ModeSymlink))
	_register("io/fs.ModeTemporary", value.NewGoVal(io_fs.ModeTemporary))
	_register("io/fs.ModeType", value.NewGoVal(io_fs.ModeType))
	{
		var x io_fs.PathError
		_register("io/fs.PathError", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("io/fs.ReadDir", value.NewGoVal(io_fs.ReadDir))
	{
		var x io_fs.ReadDirFS
		_register("io/fs.ReadDirFS", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x io_fs.ReadDirFile
		_register("io/fs.ReadDirFile", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("io/fs.ReadFile", value.NewGoVal(io_fs.ReadFile))
	{
		var x io_fs.ReadFileFS
		_register("io/fs.ReadFileFS", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("io/fs.SkipDir", value.NewGoVal(io_fs.SkipDir))
	_register("io/fs.Stat", value.NewGoVal(io_fs.Stat))
	{
		var x io_fs.StatFS
		_register("io/fs.StatFS", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("io/fs.Sub", value.NewGoVal(io_fs.Sub))
	{
		var x io_fs.SubFS
		_register("io/fs.SubFS", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("io/fs.ValidPath", value.NewGoVal(io_fs.ValidPath))
	_register("io/fs.WalkDir", value.NewGoVal(io_fs.WalkDir))
	{
		var x io_fs.WalkDirFunc
		_register("io/fs.WalkDirFunc", value.NewGoTyp(reflect.TypeOf(x)))
	}

	// package io/ioutil
	////////////////////////////////////////
	_register("io/ioutil.Discard", value.NewGoVal(io_ioutil.Discard))
	_register("io/ioutil.NopCloser", value.NewGoVal(io_ioutil.NopCloser))
	_register("io/ioutil.ReadAll", value.NewGoVal(io_ioutil.ReadAll))
	_register("io/ioutil.ReadDir", value.NewGoVal(io_ioutil.ReadDir))
	_register("io/ioutil.ReadFile", value.NewGoVal(io_ioutil.ReadFile))
	_register("io/ioutil.TempDir", value.NewGoVal(io_ioutil.TempDir))
	_register("io/ioutil.TempFile", value.NewGoVal(io_ioutil.TempFile))
	_register("io/ioutil.WriteFile", value.NewGoVal(io_ioutil.WriteFile))

	// package net/http
	////////////////////////////////////////
	_register("net/http.AllowQuerySemicolons", value.NewGoVal(net_http.AllowQuerySemicolons))
	_register("net/http.CanonicalHeaderKey", value.NewGoVal(net_http.CanonicalHeaderKey))
	{
		var x net_http.Client
		_register("net/http.Client", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.CloseNotifier
		_register("net/http.CloseNotifier", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.ConnState
		_register("net/http.ConnState", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.Cookie
		_register("net/http.Cookie", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.CookieJar
		_register("net/http.CookieJar", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("net/http.DefaultClient", value.NewGoVal(net_http.DefaultClient))
	_register("net/http.DefaultMaxHeaderBytes", value.NewGoVal(net_http.DefaultMaxHeaderBytes))
	_register("net/http.DefaultMaxIdleConnsPerHost", value.NewGoVal(net_http.DefaultMaxIdleConnsPerHost))
	_register("net/http.DefaultServeMux", value.NewGoVal(net_http.DefaultServeMux))
	_register("net/http.DefaultTransport", value.NewGoVal(net_http.DefaultTransport))
	_register("net/http.DetectContentType", value.NewGoVal(net_http.DetectContentType))
	{
		var x net_http.Dir
		_register("net/http.Dir", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("net/http.ErrAbortHandler", value.NewGoVal(net_http.ErrAbortHandler))
	_register("net/http.ErrBodyNotAllowed", value.NewGoVal(net_http.ErrBodyNotAllowed))
	_register("net/http.ErrBodyReadAfterClose", value.NewGoVal(net_http.ErrBodyReadAfterClose))
	_register("net/http.ErrContentLength", value.NewGoVal(net_http.ErrContentLength))
	_register("net/http.ErrHandlerTimeout", value.NewGoVal(net_http.ErrHandlerTimeout))
	_register("net/http.ErrHeaderTooLong", value.NewGoVal(net_http.ErrHeaderTooLong))
	_register("net/http.ErrHijacked", value.NewGoVal(net_http.ErrHijacked))
	_register("net/http.ErrLineTooLong", value.NewGoVal(net_http.ErrLineTooLong))
	_register("net/http.ErrMissingBoundary", value.NewGoVal(net_http.ErrMissingBoundary))
	_register("net/http.ErrMissingContentLength", value.NewGoVal(net_http.ErrMissingContentLength))
	_register("net/http.ErrMissingFile", value.NewGoVal(net_http.ErrMissingFile))
	_register("net/http.ErrNoCookie", value.NewGoVal(net_http.ErrNoCookie))
	_register("net/http.ErrNoLocation", value.NewGoVal(net_http.ErrNoLocation))
	_register("net/http.ErrNotMultipart", value.NewGoVal(net_http.ErrNotMultipart))
	_register("net/http.ErrNotSupported", value.NewGoVal(net_http.ErrNotSupported))
	_register("net/http.ErrServerClosed", value.NewGoVal(net_http.ErrServerClosed))
	_register("net/http.ErrShortBody", value.NewGoVal(net_http.ErrShortBody))
	_register("net/http.ErrSkipAltProtocol", value.NewGoVal(net_http.ErrSkipAltProtocol))
	_register("net/http.ErrUnexpectedTrailer", value.NewGoVal(net_http.ErrUnexpectedTrailer))
	_register("net/http.ErrUseLastResponse", value.NewGoVal(net_http.ErrUseLastResponse))
	_register("net/http.ErrWriteAfterFlush", value.NewGoVal(net_http.ErrWriteAfterFlush))
	_register("net/http.Error", value.NewGoVal(net_http.Error))
	_register("net/http.FS", value.NewGoVal(net_http.FS))
	{
		var x net_http.File
		_register("net/http.File", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("net/http.FileServer", value.NewGoVal(net_http.FileServer))
	{
		var x net_http.FileSystem
		_register("net/http.FileSystem", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.Flusher
		_register("net/http.Flusher", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("net/http.Get", value.NewGoVal(net_http.Get))
	_register("net/http.Handle", value.NewGoVal(net_http.Handle))
	_register("net/http.HandleFunc", value.NewGoVal(net_http.HandleFunc))
	{
		var x net_http.Handler
		_register("net/http.Handler", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.HandlerFunc
		_register("net/http.HandlerFunc", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("net/http.Head", value.NewGoVal(net_http.Head))
	{
		var x net_http.Header
		_register("net/http.Header", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.Hijacker
		_register("net/http.Hijacker", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("net/http.ListenAndServe", value.NewGoVal(net_http.ListenAndServe))
	_register("net/http.ListenAndServeTLS", value.NewGoVal(net_http.ListenAndServeTLS))
	_register("net/http.LocalAddrContextKey", value.NewGoVal(net_http.LocalAddrContextKey))
	_register("net/http.MaxBytesHandler", value.NewGoVal(net_http.MaxBytesHandler))
	_register("net/http.MaxBytesReader", value.NewGoVal(net_http.MaxBytesReader))
	_register("net/http.MethodConnect", value.NewGoVal(net_http.MethodConnect))
	_register("net/http.MethodDelete", value.NewGoVal(net_http.MethodDelete))
	_register("net/http.MethodGet", value.NewGoVal(net_http.MethodGet))
	_register("net/http.MethodHead", value.NewGoVal(net_http.MethodHead))
	_register("net/http.MethodOptions", value.NewGoVal(net_http.MethodOptions))
	_register("net/http.MethodPatch", value.NewGoVal(net_http.MethodPatch))
	_register("net/http.MethodPost", value.NewGoVal(net_http.MethodPost))
	_register("net/http.MethodPut", value.NewGoVal(net_http.MethodPut))
	_register("net/http.MethodTrace", value.NewGoVal(net_http.MethodTrace))
	_register("net/http.NewFileTransport", value.NewGoVal(net_http.NewFileTransport))
	_register("net/http.NewRequest", value.NewGoVal(net_http.NewRequest))
	_register("net/http.NewRequestWithContext", value.NewGoVal(net_http.NewRequestWithContext))
	_register("net/http.NewServeMux", value.NewGoVal(net_http.NewServeMux))
	_register("net/http.NoBody", value.NewGoVal(net_http.NoBody))
	_register("net/http.NotFound", value.NewGoVal(net_http.NotFound))
	_register("net/http.NotFoundHandler", value.NewGoVal(net_http.NotFoundHandler))
	_register("net/http.ParseHTTPVersion", value.NewGoVal(net_http.ParseHTTPVersion))
	_register("net/http.ParseTime", value.NewGoVal(net_http.ParseTime))
	_register("net/http.Post", value.NewGoVal(net_http.Post))
	_register("net/http.PostForm", value.NewGoVal(net_http.PostForm))
	{
		var x net_http.ProtocolError
		_register("net/http.ProtocolError", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("net/http.ProxyFromEnvironment", value.NewGoVal(net_http.ProxyFromEnvironment))
	_register("net/http.ProxyURL", value.NewGoVal(net_http.ProxyURL))
	{
		var x net_http.PushOptions
		_register("net/http.PushOptions", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.Pusher
		_register("net/http.Pusher", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("net/http.ReadRequest", value.NewGoVal(net_http.ReadRequest))
	_register("net/http.ReadResponse", value.NewGoVal(net_http.ReadResponse))
	_register("net/http.Redirect", value.NewGoVal(net_http.Redirect))
	_register("net/http.RedirectHandler", value.NewGoVal(net_http.RedirectHandler))
	{
		var x net_http.Request
		_register("net/http.Request", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.Response
		_register("net/http.Response", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.ResponseWriter
		_register("net/http.ResponseWriter", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.RoundTripper
		_register("net/http.RoundTripper", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x net_http.SameSite
		_register("net/http.SameSite", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("net/http.SameSiteDefaultMode", value.NewGoVal(net_http.SameSiteDefaultMode))
	_register("net/http.SameSiteLaxMode", value.NewGoVal(net_http.SameSiteLaxMode))
	_register("net/http.SameSiteNoneMode", value.NewGoVal(net_http.SameSiteNoneMode))
	_register("net/http.SameSiteStrictMode", value.NewGoVal(net_http.SameSiteStrictMode))
	_register("net/http.Serve", value.NewGoVal(net_http.Serve))
	_register("net/http.ServeContent", value.NewGoVal(net_http.ServeContent))
	_register("net/http.ServeFile", value.NewGoVal(net_http.ServeFile))
	{
		var x net_http.ServeMux
		_register("net/http.ServeMux", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("net/http.ServeTLS", value.NewGoVal(net_http.ServeTLS))
	{
		var x net_http.Server
		_register("net/http.Server", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("net/http.ServerContextKey", value.NewGoVal(net_http.ServerContextKey))
	_register("net/http.SetCookie", value.NewGoVal(net_http.SetCookie))
	_register("net/http.StateActive", value.NewGoVal(net_http.StateActive))
	_register("net/http.StateClosed", value.NewGoVal(net_http.StateClosed))
	_register("net/http.StateHijacked", value.NewGoVal(net_http.StateHijacked))
	_register("net/http.StateIdle", value.NewGoVal(net_http.StateIdle))
	_register("net/http.StateNew", value.NewGoVal(net_http.StateNew))
	_register("net/http.StatusAccepted", value.NewGoVal(net_http.StatusAccepted))
	_register("net/http.StatusAlreadyReported", value.NewGoVal(net_http.StatusAlreadyReported))
	_register("net/http.StatusBadGateway", value.NewGoVal(net_http.StatusBadGateway))
	_register("net/http.StatusBadRequest", value.NewGoVal(net_http.StatusBadRequest))
	_register("net/http.StatusConflict", value.NewGoVal(net_http.StatusConflict))
	_register("net/http.StatusContinue", value.NewGoVal(net_http.StatusContinue))
	_register("net/http.StatusCreated", value.NewGoVal(net_http.StatusCreated))
	_register("net/http.StatusEarlyHints", value.NewGoVal(net_http.StatusEarlyHints))
	_register("net/http.StatusExpectationFailed", value.NewGoVal(net_http.StatusExpectationFailed))
	_register("net/http.StatusFailedDependency", value.NewGoVal(net_http.StatusFailedDependency))
	_register("net/http.StatusForbidden", value.NewGoVal(net_http.StatusForbidden))
	_register("net/http.StatusFound", value.NewGoVal(net_http.StatusFound))
	_register("net/http.StatusGatewayTimeout", value.NewGoVal(net_http.StatusGatewayTimeout))
	_register("net/http.StatusGone", value.NewGoVal(net_http.StatusGone))
	_register("net/http.StatusHTTPVersionNotSupported", value.NewGoVal(net_http.StatusHTTPVersionNotSupported))
	_register("net/http.StatusIMUsed", value.NewGoVal(net_http.StatusIMUsed))
	_register("net/http.StatusInsufficientStorage", value.NewGoVal(net_http.StatusInsufficientStorage))
	_register("net/http.StatusInternalServerError", value.NewGoVal(net_http.StatusInternalServerError))
	_register("net/http.StatusLengthRequired", value.NewGoVal(net_http.StatusLengthRequired))
	_register("net/http.StatusLocked", value.NewGoVal(net_http.StatusLocked))
	_register("net/http.StatusLoopDetected", value.NewGoVal(net_http.StatusLoopDetected))
	_register("net/http.StatusMethodNotAllowed", value.NewGoVal(net_http.StatusMethodNotAllowed))
	_register("net/http.StatusMisdirectedRequest", value.NewGoVal(net_http.StatusMisdirectedRequest))
	_register("net/http.StatusMovedPermanently", value.NewGoVal(net_http.StatusMovedPermanently))
	_register("net/http.StatusMultiStatus", value.NewGoVal(net_http.StatusMultiStatus))
	_register("net/http.StatusMultipleChoices", value.NewGoVal(net_http.StatusMultipleChoices))
	_register("net/http.StatusNetworkAuthenticationRequired", value.NewGoVal(net_http.StatusNetworkAuthenticationRequired))
	_register("net/http.StatusNoContent", value.NewGoVal(net_http.StatusNoContent))
	_register("net/http.StatusNonAuthoritativeInfo", value.NewGoVal(net_http.StatusNonAuthoritativeInfo))
	_register("net/http.StatusNotAcceptable", value.NewGoVal(net_http.StatusNotAcceptable))
	_register("net/http.StatusNotExtended", value.NewGoVal(net_http.StatusNotExtended))
	_register("net/http.StatusNotFound", value.NewGoVal(net_http.StatusNotFound))
	_register("net/http.StatusNotImplemented", value.NewGoVal(net_http.StatusNotImplemented))
	_register("net/http.StatusNotModified", value.NewGoVal(net_http.StatusNotModified))
	_register("net/http.StatusOK", value.NewGoVal(net_http.StatusOK))
	_register("net/http.StatusPartialContent", value.NewGoVal(net_http.StatusPartialContent))
	_register("net/http.StatusPaymentRequired", value.NewGoVal(net_http.StatusPaymentRequired))
	_register("net/http.StatusPermanentRedirect", value.NewGoVal(net_http.StatusPermanentRedirect))
	_register("net/http.StatusPreconditionFailed", value.NewGoVal(net_http.StatusPreconditionFailed))
	_register("net/http.StatusPreconditionRequired", value.NewGoVal(net_http.StatusPreconditionRequired))
	_register("net/http.StatusProcessing", value.NewGoVal(net_http.StatusProcessing))
	_register("net/http.StatusProxyAuthRequired", value.NewGoVal(net_http.StatusProxyAuthRequired))
	_register("net/http.StatusRequestEntityTooLarge", value.NewGoVal(net_http.StatusRequestEntityTooLarge))
	_register("net/http.StatusRequestHeaderFieldsTooLarge", value.NewGoVal(net_http.StatusRequestHeaderFieldsTooLarge))
	_register("net/http.StatusRequestTimeout", value.NewGoVal(net_http.StatusRequestTimeout))
	_register("net/http.StatusRequestURITooLong", value.NewGoVal(net_http.StatusRequestURITooLong))
	_register("net/http.StatusRequestedRangeNotSatisfiable", value.NewGoVal(net_http.StatusRequestedRangeNotSatisfiable))
	_register("net/http.StatusResetContent", value.NewGoVal(net_http.StatusResetContent))
	_register("net/http.StatusSeeOther", value.NewGoVal(net_http.StatusSeeOther))
	_register("net/http.StatusServiceUnavailable", value.NewGoVal(net_http.StatusServiceUnavailable))
	_register("net/http.StatusSwitchingProtocols", value.NewGoVal(net_http.StatusSwitchingProtocols))
	_register("net/http.StatusTeapot", value.NewGoVal(net_http.StatusTeapot))
	_register("net/http.StatusTemporaryRedirect", value.NewGoVal(net_http.StatusTemporaryRedirect))
	_register("net/http.StatusText", value.NewGoVal(net_http.StatusText))
	_register("net/http.StatusTooEarly", value.NewGoVal(net_http.StatusTooEarly))
	_register("net/http.StatusTooManyRequests", value.NewGoVal(net_http.StatusTooManyRequests))
	_register("net/http.StatusUnauthorized", value.NewGoVal(net_http.StatusUnauthorized))
	_register("net/http.StatusUnavailableForLegalReasons", value.NewGoVal(net_http.StatusUnavailableForLegalReasons))
	_register("net/http.StatusUnprocessableEntity", value.NewGoVal(net_http.StatusUnprocessableEntity))
	_register("net/http.StatusUnsupportedMediaType", value.NewGoVal(net_http.StatusUnsupportedMediaType))
	_register("net/http.StatusUpgradeRequired", value.NewGoVal(net_http.StatusUpgradeRequired))
	_register("net/http.StatusUseProxy", value.NewGoVal(net_http.StatusUseProxy))
	_register("net/http.StatusVariantAlsoNegotiates", value.NewGoVal(net_http.StatusVariantAlsoNegotiates))
	_register("net/http.StripPrefix", value.NewGoVal(net_http.StripPrefix))
	_register("net/http.TimeFormat", value.NewGoVal(net_http.TimeFormat))
	_register("net/http.TimeoutHandler", value.NewGoVal(net_http.TimeoutHandler))
	_register("net/http.TrailerPrefix", value.NewGoVal(net_http.TrailerPrefix))
	{
		var x net_http.Transport
		_register("net/http.Transport", value.NewGoTyp(reflect.TypeOf(x)))
	}

	// package regexp
	////////////////////////////////////////
	_register("regexp.Compile", value.NewGoVal(regexp.Compile))
	_register("regexp.CompilePOSIX", value.NewGoVal(regexp.CompilePOSIX))
	_register("regexp.Match", value.NewGoVal(regexp.Match))
	_register("regexp.MatchReader", value.NewGoVal(regexp.MatchReader))
	_register("regexp.MatchString", value.NewGoVal(regexp.MatchString))
	_register("regexp.MustCompile", value.NewGoVal(regexp.MustCompile))
	_register("regexp.MustCompilePOSIX", value.NewGoVal(regexp.MustCompilePOSIX))
	_register("regexp.QuoteMeta", value.NewGoVal(regexp.QuoteMeta))
	{
		var x regexp.Regexp
		_register("regexp.Regexp", value.NewGoTyp(reflect.TypeOf(x)))
	}

	// package strconv
	////////////////////////////////////////
	_register("strconv.AppendBool", value.NewGoVal(strconv.AppendBool))
	_register("strconv.AppendFloat", value.NewGoVal(strconv.AppendFloat))
	_register("strconv.AppendInt", value.NewGoVal(strconv.AppendInt))
	_register("strconv.AppendQuote", value.NewGoVal(strconv.AppendQuote))
	_register("strconv.AppendQuoteRune", value.NewGoVal(strconv.AppendQuoteRune))
	_register("strconv.AppendQuoteRuneToASCII", value.NewGoVal(strconv.AppendQuoteRuneToASCII))
	_register("strconv.AppendQuoteRuneToGraphic", value.NewGoVal(strconv.AppendQuoteRuneToGraphic))
	_register("strconv.AppendQuoteToASCII", value.NewGoVal(strconv.AppendQuoteToASCII))
	_register("strconv.AppendQuoteToGraphic", value.NewGoVal(strconv.AppendQuoteToGraphic))
	_register("strconv.AppendUint", value.NewGoVal(strconv.AppendUint))
	_register("strconv.Atoi", value.NewGoVal(strconv.Atoi))
	_register("strconv.CanBackquote", value.NewGoVal(strconv.CanBackquote))
	_register("strconv.ErrRange", value.NewGoVal(strconv.ErrRange))
	_register("strconv.ErrSyntax", value.NewGoVal(strconv.ErrSyntax))
	_register("strconv.FormatBool", value.NewGoVal(strconv.FormatBool))
	_register("strconv.FormatComplex", value.NewGoVal(strconv.FormatComplex))
	_register("strconv.FormatFloat", value.NewGoVal(strconv.FormatFloat))
	_register("strconv.FormatInt", value.NewGoVal(strconv.FormatInt))
	_register("strconv.FormatUint", value.NewGoVal(strconv.FormatUint))
	_register("strconv.IntSize", value.NewGoVal(strconv.IntSize))
	_register("strconv.IsGraphic", value.NewGoVal(strconv.IsGraphic))
	_register("strconv.IsPrint", value.NewGoVal(strconv.IsPrint))
	_register("strconv.Itoa", value.NewGoVal(strconv.Itoa))
	{
		var x strconv.NumError
		_register("strconv.NumError", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("strconv.ParseBool", value.NewGoVal(strconv.ParseBool))
	_register("strconv.ParseComplex", value.NewGoVal(strconv.ParseComplex))
	_register("strconv.ParseFloat", value.NewGoVal(strconv.ParseFloat))
	_register("strconv.ParseInt", value.NewGoVal(strconv.ParseInt))
	_register("strconv.ParseUint", value.NewGoVal(strconv.ParseUint))
	_register("strconv.Quote", value.NewGoVal(strconv.Quote))
	_register("strconv.QuoteRune", value.NewGoVal(strconv.QuoteRune))
	_register("strconv.QuoteRuneToASCII", value.NewGoVal(strconv.QuoteRuneToASCII))
	_register("strconv.QuoteRuneToGraphic", value.NewGoVal(strconv.QuoteRuneToGraphic))
	_register("strconv.QuoteToASCII", value.NewGoVal(strconv.QuoteToASCII))
	_register("strconv.QuoteToGraphic", value.NewGoVal(strconv.QuoteToGraphic))
	_register("strconv.QuotedPrefix", value.NewGoVal(strconv.QuotedPrefix))
	_register("strconv.Unquote", value.NewGoVal(strconv.Unquote))
	_register("strconv.UnquoteChar", value.NewGoVal(strconv.UnquoteChar))

	// package strings
	////////////////////////////////////////
	{
		var x strings.Builder
		_register("strings.Builder", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("strings.Clone", value.NewGoVal(strings.Clone))
	_register("strings.Compare", value.NewGoVal(strings.Compare))
	_register("strings.Contains", value.NewGoVal(strings.Contains))
	_register("strings.ContainsAny", value.NewGoVal(strings.ContainsAny))
	_register("strings.ContainsRune", value.NewGoVal(strings.ContainsRune))
	_register("strings.Count", value.NewGoVal(strings.Count))
	_register("strings.Cut", value.NewGoVal(strings.Cut))
	_register("strings.EqualFold", value.NewGoVal(strings.EqualFold))
	_register("strings.Fields", value.NewGoVal(strings.Fields))
	_register("strings.FieldsFunc", value.NewGoVal(strings.FieldsFunc))
	_register("strings.HasPrefix", value.NewGoVal(strings.HasPrefix))
	_register("strings.HasSuffix", value.NewGoVal(strings.HasSuffix))
	_register("strings.Index", value.NewGoVal(strings.Index))
	_register("strings.IndexAny", value.NewGoVal(strings.IndexAny))
	_register("strings.IndexByte", value.NewGoVal(strings.IndexByte))
	_register("strings.IndexFunc", value.NewGoVal(strings.IndexFunc))
	_register("strings.IndexRune", value.NewGoVal(strings.IndexRune))
	_register("strings.Join", value.NewGoVal(strings.Join))
	_register("strings.LastIndex", value.NewGoVal(strings.LastIndex))
	_register("strings.LastIndexAny", value.NewGoVal(strings.LastIndexAny))
	_register("strings.LastIndexByte", value.NewGoVal(strings.LastIndexByte))
	_register("strings.LastIndexFunc", value.NewGoVal(strings.LastIndexFunc))
	_register("strings.Map", value.NewGoVal(strings.Map))
	_register("strings.NewReader", value.NewGoVal(strings.NewReader))
	_register("strings.NewReplacer", value.NewGoVal(strings.NewReplacer))
	{
		var x strings.Reader
		_register("strings.Reader", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("strings.Repeat", value.NewGoVal(strings.Repeat))
	_register("strings.Replace", value.NewGoVal(strings.Replace))
	_register("strings.ReplaceAll", value.NewGoVal(strings.ReplaceAll))
	{
		var x strings.Replacer
		_register("strings.Replacer", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("strings.Split", value.NewGoVal(strings.Split))
	_register("strings.SplitAfter", value.NewGoVal(strings.SplitAfter))
	_register("strings.SplitAfterN", value.NewGoVal(strings.SplitAfterN))
	_register("strings.SplitN", value.NewGoVal(strings.SplitN))
	_register("strings.Title", value.NewGoVal(strings.Title))
	_register("strings.ToLower", value.NewGoVal(strings.ToLower))
	_register("strings.ToLowerSpecial", value.NewGoVal(strings.ToLowerSpecial))
	_register("strings.ToTitle", value.NewGoVal(strings.ToTitle))
	_register("strings.ToTitleSpecial", value.NewGoVal(strings.ToTitleSpecial))
	_register("strings.ToUpper", value.NewGoVal(strings.ToUpper))
	_register("strings.ToUpperSpecial", value.NewGoVal(strings.ToUpperSpecial))
	_register("strings.ToValidUTF8", value.NewGoVal(strings.ToValidUTF8))
	_register("strings.Trim", value.NewGoVal(strings.Trim))
	_register("strings.TrimFunc", value.NewGoVal(strings.TrimFunc))
	_register("strings.TrimLeft", value.NewGoVal(strings.TrimLeft))
	_register("strings.TrimLeftFunc", value.NewGoVal(strings.TrimLeftFunc))
	_register("strings.TrimPrefix", value.NewGoVal(strings.TrimPrefix))
	_register("strings.TrimRight", value.NewGoVal(strings.TrimRight))
	_register("strings.TrimRightFunc", value.NewGoVal(strings.TrimRightFunc))
	_register("strings.TrimSpace", value.NewGoVal(strings.TrimSpace))
	_register("strings.TrimSuffix", value.NewGoVal(strings.TrimSuffix))

	// package time
	////////////////////////////////////////
	_register("time.ANSIC", value.NewGoVal(time.ANSIC))
	_register("time.After", value.NewGoVal(time.After))
	_register("time.AfterFunc", value.NewGoVal(time.AfterFunc))
	_register("time.April", value.NewGoVal(time.April))
	_register("time.August", value.NewGoVal(time.August))
	_register("time.Date", value.NewGoVal(time.Date))
	_register("time.December", value.NewGoVal(time.December))
	{
		var x time.Duration
		_register("time.Duration", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("time.February", value.NewGoVal(time.February))
	_register("time.FixedZone", value.NewGoVal(time.FixedZone))
	_register("time.Friday", value.NewGoVal(time.Friday))
	_register("time.Hour", value.NewGoVal(time.Hour))
	_register("time.January", value.NewGoVal(time.January))
	_register("time.July", value.NewGoVal(time.July))
	_register("time.June", value.NewGoVal(time.June))
	_register("time.Kitchen", value.NewGoVal(time.Kitchen))
	_register("time.Layout", value.NewGoVal(time.Layout))
	_register("time.LoadLocation", value.NewGoVal(time.LoadLocation))
	_register("time.LoadLocationFromTZData", value.NewGoVal(time.LoadLocationFromTZData))
	_register("time.Local", value.NewGoVal(time.Local))
	{
		var x time.Location
		_register("time.Location", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("time.March", value.NewGoVal(time.March))
	_register("time.May", value.NewGoVal(time.May))
	_register("time.Microsecond", value.NewGoVal(time.Microsecond))
	_register("time.Millisecond", value.NewGoVal(time.Millisecond))
	_register("time.Minute", value.NewGoVal(time.Minute))
	_register("time.Monday", value.NewGoVal(time.Monday))
	{
		var x time.Month
		_register("time.Month", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("time.Nanosecond", value.NewGoVal(time.Nanosecond))
	_register("time.NewTicker", value.NewGoVal(time.NewTicker))
	_register("time.NewTimer", value.NewGoVal(time.NewTimer))
	_register("time.November", value.NewGoVal(time.November))
	_register("time.Now", value.NewGoVal(time.Now))
	_register("time.October", value.NewGoVal(time.October))
	_register("time.Parse", value.NewGoVal(time.Parse))
	_register("time.ParseDuration", value.NewGoVal(time.ParseDuration))
	{
		var x time.ParseError
		_register("time.ParseError", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("time.ParseInLocation", value.NewGoVal(time.ParseInLocation))
	_register("time.RFC1123", value.NewGoVal(time.RFC1123))
	_register("time.RFC1123Z", value.NewGoVal(time.RFC1123Z))
	_register("time.RFC3339", value.NewGoVal(time.RFC3339))
	_register("time.RFC3339Nano", value.NewGoVal(time.RFC3339Nano))
	_register("time.RFC822", value.NewGoVal(time.RFC822))
	_register("time.RFC822Z", value.NewGoVal(time.RFC822Z))
	_register("time.RFC850", value.NewGoVal(time.RFC850))
	_register("time.RubyDate", value.NewGoVal(time.RubyDate))
	_register("time.Saturday", value.NewGoVal(time.Saturday))
	_register("time.Second", value.NewGoVal(time.Second))
	_register("time.September", value.NewGoVal(time.September))
	_register("time.Since", value.NewGoVal(time.Since))
	_register("time.Sleep", value.NewGoVal(time.Sleep))
	_register("time.Stamp", value.NewGoVal(time.Stamp))
	_register("time.StampMicro", value.NewGoVal(time.StampMicro))
	_register("time.StampMilli", value.NewGoVal(time.StampMilli))
	_register("time.StampNano", value.NewGoVal(time.StampNano))
	_register("time.Sunday", value.NewGoVal(time.Sunday))
	_register("time.Thursday", value.NewGoVal(time.Thursday))
	_register("time.Tick", value.NewGoVal(time.Tick))
	{
		var x time.Ticker
		_register("time.Ticker", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x time.Time
		_register("time.Time", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x time.Timer
		_register("time.Timer", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("time.Tuesday", value.NewGoVal(time.Tuesday))
	_register("time.UTC", value.NewGoVal(time.UTC))
	_register("time.Unix", value.NewGoVal(time.Unix))
	_register("time.UnixDate", value.NewGoVal(time.UnixDate))
	_register("time.UnixMicro", value.NewGoVal(time.UnixMicro))
	_register("time.UnixMilli", value.NewGoVal(time.UnixMilli))
	_register("time.Until", value.NewGoVal(time.Until))
	_register("time.Wednesday", value.NewGoVal(time.Wednesday))
	{
		var x time.Weekday
		_register("time.Weekday", value.NewGoTyp(reflect.TypeOf(x)))
	}

	// package math/big
	////////////////////////////////////////
	_register("math/big.Above", value.NewGoVal(math_big.Above))
	{
		var x math_big.Accuracy
		_register("math/big.Accuracy", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("math/big.AwayFromZero", value.NewGoVal(math_big.AwayFromZero))
	_register("math/big.Below", value.NewGoVal(math_big.Below))
	{
		var x math_big.ErrNaN
		_register("math/big.ErrNaN", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("math/big.Exact", value.NewGoVal(math_big.Exact))
	{
		var x math_big.Float
		_register("math/big.Float", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x math_big.Int
		_register("math/big.Int", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("math/big.Jacobi", value.NewGoVal(math_big.Jacobi))
	_register("math/big.MaxBase", value.NewGoVal(math_big.MaxBase))
	_register("math/big.MaxExp", value.NewGoVal(math_big.MaxExp))
	_register("math/big.MaxPrec", value.NewGoVal(math_big.MaxPrec))
	_register("math/big.MinExp", value.NewGoVal(math_big.MinExp))
	_register("math/big.NewFloat", value.NewGoVal(math_big.NewFloat))
	_register("math/big.NewInt", value.NewGoVal(math_big.NewInt))
	_register("math/big.NewRat", value.NewGoVal(math_big.NewRat))
	_register("math/big.ParseFloat", value.NewGoVal(math_big.ParseFloat))
	{
		var x math_big.Rat
		_register("math/big.Rat", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x math_big.RoundingMode
		_register("math/big.RoundingMode", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("math/big.ToNearestAway", value.NewGoVal(math_big.ToNearestAway))
	_register("math/big.ToNearestEven", value.NewGoVal(math_big.ToNearestEven))
	_register("math/big.ToNegativeInf", value.NewGoVal(math_big.ToNegativeInf))
	_register("math/big.ToPositiveInf", value.NewGoVal(math_big.ToPositiveInf))
	_register("math/big.ToZero", value.NewGoVal(math_big.ToZero))
	{
		var x math_big.Word
		_register("math/big.Word", value.NewGoTyp(reflect.TypeOf(x)))
	}

	// package math/rand
	////////////////////////////////////////
	_register("math/rand.ExpFloat64", value.NewGoVal(math_rand.ExpFloat64))
	_register("math/rand.Float32", value.NewGoVal(math_rand.Float32))
	_register("math/rand.Float64", value.NewGoVal(math_rand.Float64))
	_register("math/rand.Int", value.NewGoVal(math_rand.Int))
	_register("math/rand.Int31", value.NewGoVal(math_rand.Int31))
	_register("math/rand.Int31n", value.NewGoVal(math_rand.Int31n))
	_register("math/rand.Int63", value.NewGoVal(math_rand.Int63))
	_register("math/rand.Int63n", value.NewGoVal(math_rand.Int63n))
	_register("math/rand.Intn", value.NewGoVal(math_rand.Intn))
	_register("math/rand.New", value.NewGoVal(math_rand.New))
	_register("math/rand.NewSource", value.NewGoVal(math_rand.NewSource))
	_register("math/rand.NewZipf", value.NewGoVal(math_rand.NewZipf))
	_register("math/rand.NormFloat64", value.NewGoVal(math_rand.NormFloat64))
	_register("math/rand.Perm", value.NewGoVal(math_rand.Perm))
	{
		var x math_rand.Rand
		_register("math/rand.Rand", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("math/rand.Read", value.NewGoVal(math_rand.Read))
	_register("math/rand.Seed", value.NewGoVal(math_rand.Seed))
	_register("math/rand.Shuffle", value.NewGoVal(math_rand.Shuffle))
	{
		var x math_rand.Source
		_register("math/rand.Source", value.NewGoTyp(reflect.TypeOf(x)))
	}
	{
		var x math_rand.Source64
		_register("math/rand.Source64", value.NewGoTyp(reflect.TypeOf(x)))
	}
	_register("math/rand.Uint32", value.NewGoVal(math_rand.Uint32))
	_register("math/rand.Uint64", value.NewGoVal(math_rand.Uint64))
	{
		var x math_rand.Zipf
		_register("math/rand.Zipf", value.NewGoTyp(reflect.TypeOf(x)))
	}

	// package math
	////////////////////////////////////////
	_register("math.Abs", value.NewGoVal(math.Abs))
	_register("math.Acos", value.NewGoVal(math.Acos))
	_register("math.Acosh", value.NewGoVal(math.Acosh))
	_register("math.Asin", value.NewGoVal(math.Asin))
	_register("math.Asinh", value.NewGoVal(math.Asinh))
	_register("math.Atan", value.NewGoVal(math.Atan))
	_register("math.Atan2", value.NewGoVal(math.Atan2))
	_register("math.Atanh", value.NewGoVal(math.Atanh))
	_register("math.Cbrt", value.NewGoVal(math.Cbrt))
	_register("math.Ceil", value.NewGoVal(math.Ceil))
	_register("math.Copysign", value.NewGoVal(math.Copysign))
	_register("math.Cos", value.NewGoVal(math.Cos))
	_register("math.Cosh", value.NewGoVal(math.Cosh))
	_register("math.Dim", value.NewGoVal(math.Dim))
	_register("math.E", value.NewGoVal(math.E))
	_register("math.Erf", value.NewGoVal(math.Erf))
	_register("math.Erfc", value.NewGoVal(math.Erfc))
	_register("math.Erfcinv", value.NewGoVal(math.Erfcinv))
	_register("math.Erfinv", value.NewGoVal(math.Erfinv))
	_register("math.Exp", value.NewGoVal(math.Exp))
	_register("math.Exp2", value.NewGoVal(math.Exp2))
	_register("math.Expm1", value.NewGoVal(math.Expm1))
	_register("math.FMA", value.NewGoVal(math.FMA))
	_register("math.Float32bits", value.NewGoVal(math.Float32bits))
	_register("math.Float32frombits", value.NewGoVal(math.Float32frombits))
	_register("math.Float64bits", value.NewGoVal(math.Float64bits))
	_register("math.Float64frombits", value.NewGoVal(math.Float64frombits))
	_register("math.Floor", value.NewGoVal(math.Floor))
	_register("math.Frexp", value.NewGoVal(math.Frexp))
	_register("math.Gamma", value.NewGoVal(math.Gamma))
	_register("math.Hypot", value.NewGoVal(math.Hypot))
	_register("math.Ilogb", value.NewGoVal(math.Ilogb))
	_register("math.Inf", value.NewGoVal(math.Inf))
	_register("math.IsInf", value.NewGoVal(math.IsInf))
	_register("math.IsNaN", value.NewGoVal(math.IsNaN))
	_register("math.J0", value.NewGoVal(math.J0))
	_register("math.J1", value.NewGoVal(math.J1))
	_register("math.Jn", value.NewGoVal(math.Jn))
	_register("math.Ldexp", value.NewGoVal(math.Ldexp))
	_register("math.Lgamma", value.NewGoVal(math.Lgamma))
	_register("math.Ln10", value.NewGoVal(math.Ln10))
	_register("math.Ln2", value.NewGoVal(math.Ln2))
	_register("math.Log", value.NewGoVal(math.Log))
	_register("math.Log10", value.NewGoVal(math.Log10))
	_register("math.Log10E", value.NewGoVal(math.Log10E))
	_register("math.Log1p", value.NewGoVal(math.Log1p))
	_register("math.Log2", value.NewGoVal(math.Log2))
	_register("math.Log2E", value.NewGoVal(math.Log2E))
	_register("math.Logb", value.NewGoVal(math.Logb))
	_register("math.Max", value.NewGoVal(math.Max))
	_register("math.MaxFloat32", value.NewGoVal(math.MaxFloat32))
	_register("math.MaxFloat64", value.NewGoVal(math.MaxFloat64))
	_register("math.MaxInt", value.NewGoVal(math.MaxInt))
	_register("math.MaxInt16", value.NewGoVal(math.MaxInt16))
	_register("math.MaxInt32", value.NewGoVal(math.MaxInt32))
	_register("math.MaxInt64", value.NewGoVal(math.MaxInt64))
	_register("math.MaxInt8", value.NewGoVal(math.MaxInt8))
	_register("math.MaxUint", uint(math.MaxUint))
	_register("math.MaxUint16", value.NewGoVal(math.MaxUint16))
	_register("math.MaxUint32", value.NewGoVal(math.MaxUint32))
	_register("math.MaxUint64", uint64(math.MaxUint64))
	_register("math.MaxUint8", value.NewGoVal(math.MaxUint8))
	_register("math.Min", value.NewGoVal(math.Min))
	_register("math.MinInt", value.NewGoVal(math.MinInt))
	_register("math.MinInt16", value.NewGoVal(math.MinInt16))
	_register("math.MinInt32", value.NewGoVal(math.MinInt32))
	_register("math.MinInt64", value.NewGoVal(math.MinInt64))
	_register("math.MinInt8", value.NewGoVal(math.MinInt8))
	_register("math.Mod", value.NewGoVal(math.Mod))
	_register("math.Modf", value.NewGoVal(math.Modf))
	_register("math.NaN", value.NewGoVal(math.NaN))
	_register("math.Nextafter", value.NewGoVal(math.Nextafter))
	_register("math.Nextafter32", value.NewGoVal(math.Nextafter32))
	_register("math.Phi", value.NewGoVal(math.Phi))
	_register("math.Pi", value.NewGoVal(math.Pi))
	_register("math.Pow", value.NewGoVal(math.Pow))
	_register("math.Pow10", value.NewGoVal(math.Pow10))
	_register("math.Remainder", value.NewGoVal(math.Remainder))
	_register("math.Round", value.NewGoVal(math.Round))
	_register("math.RoundToEven", value.NewGoVal(math.RoundToEven))
	_register("math.Signbit", value.NewGoVal(math.Signbit))
	_register("math.Sin", value.NewGoVal(math.Sin))
	_register("math.Sincos", value.NewGoVal(math.Sincos))
	_register("math.Sinh", value.NewGoVal(math.Sinh))
	_register("math.SmallestNonzeroFloat32", value.NewGoVal(math.SmallestNonzeroFloat32))
	_register("math.SmallestNonzeroFloat64", value.NewGoVal(math.SmallestNonzeroFloat64))
	_register("math.Sqrt", value.NewGoVal(math.Sqrt))
	_register("math.Sqrt2", value.NewGoVal(math.Sqrt2))
	_register("math.SqrtE", value.NewGoVal(math.SqrtE))
	_register("math.SqrtPhi", value.NewGoVal(math.SqrtPhi))
	_register("math.SqrtPi", value.NewGoVal(math.SqrtPi))
	_register("math.Tan", value.NewGoVal(math.Tan))
	_register("math.Tanh", value.NewGoVal(math.Tanh))
	_register("math.Trunc", value.NewGoVal(math.Trunc))
	_register("math.Y0", value.NewGoVal(math.Y0))
	_register("math.Y1", value.NewGoVal(math.Y1))
	_register("math.Yn", value.NewGoVal(math.Yn))
}
