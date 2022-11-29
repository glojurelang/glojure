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
)

func RegisterImports(_register func(string, interface{})) {
	// package bytes
	////////////////////////////////////////
	{
		var x bytes.Buffer
		_register("bytes.Buffer", reflect.TypeOf(x))
	}
	_register("bytes.Compare", bytes.Compare)
	_register("bytes.Contains", bytes.Contains)
	_register("bytes.ContainsAny", bytes.ContainsAny)
	_register("bytes.ContainsRune", bytes.ContainsRune)
	_register("bytes.Count", bytes.Count)
	_register("bytes.Cut", bytes.Cut)
	_register("bytes.Equal", bytes.Equal)
	_register("bytes.EqualFold", bytes.EqualFold)
	_register("bytes.ErrTooLarge", bytes.ErrTooLarge)
	_register("bytes.Fields", bytes.Fields)
	_register("bytes.FieldsFunc", bytes.FieldsFunc)
	_register("bytes.HasPrefix", bytes.HasPrefix)
	_register("bytes.HasSuffix", bytes.HasSuffix)
	_register("bytes.Index", bytes.Index)
	_register("bytes.IndexAny", bytes.IndexAny)
	_register("bytes.IndexByte", bytes.IndexByte)
	_register("bytes.IndexFunc", bytes.IndexFunc)
	_register("bytes.IndexRune", bytes.IndexRune)
	_register("bytes.Join", bytes.Join)
	_register("bytes.LastIndex", bytes.LastIndex)
	_register("bytes.LastIndexAny", bytes.LastIndexAny)
	_register("bytes.LastIndexByte", bytes.LastIndexByte)
	_register("bytes.LastIndexFunc", bytes.LastIndexFunc)
	_register("bytes.Map", bytes.Map)
	_register("bytes.MinRead", bytes.MinRead)
	_register("bytes.NewBuffer", bytes.NewBuffer)
	_register("bytes.NewBufferString", bytes.NewBufferString)
	_register("bytes.NewReader", bytes.NewReader)
	{
		var x bytes.Reader
		_register("bytes.Reader", reflect.TypeOf(x))
	}
	_register("bytes.Repeat", bytes.Repeat)
	_register("bytes.Replace", bytes.Replace)
	_register("bytes.ReplaceAll", bytes.ReplaceAll)
	_register("bytes.Runes", bytes.Runes)
	_register("bytes.Split", bytes.Split)
	_register("bytes.SplitAfter", bytes.SplitAfter)
	_register("bytes.SplitAfterN", bytes.SplitAfterN)
	_register("bytes.SplitN", bytes.SplitN)
	_register("bytes.Title", bytes.Title)
	_register("bytes.ToLower", bytes.ToLower)
	_register("bytes.ToLowerSpecial", bytes.ToLowerSpecial)
	_register("bytes.ToTitle", bytes.ToTitle)
	_register("bytes.ToTitleSpecial", bytes.ToTitleSpecial)
	_register("bytes.ToUpper", bytes.ToUpper)
	_register("bytes.ToUpperSpecial", bytes.ToUpperSpecial)
	_register("bytes.ToValidUTF8", bytes.ToValidUTF8)
	_register("bytes.Trim", bytes.Trim)
	_register("bytes.TrimFunc", bytes.TrimFunc)
	_register("bytes.TrimLeft", bytes.TrimLeft)
	_register("bytes.TrimLeftFunc", bytes.TrimLeftFunc)
	_register("bytes.TrimPrefix", bytes.TrimPrefix)
	_register("bytes.TrimRight", bytes.TrimRight)
	_register("bytes.TrimRightFunc", bytes.TrimRightFunc)
	_register("bytes.TrimSpace", bytes.TrimSpace)
	_register("bytes.TrimSuffix", bytes.TrimSuffix)

	// package context
	////////////////////////////////////////
	_register("context.Background", context.Background)
	{
		var x context.CancelFunc
		_register("context.CancelFunc", reflect.TypeOf(x))
	}
	_register("context.Canceled", context.Canceled)
	{
		var x context.Context
		_register("context.Context", reflect.TypeOf(x))
	}
	_register("context.DeadlineExceeded", context.DeadlineExceeded)
	_register("context.TODO", context.TODO)
	_register("context.WithCancel", context.WithCancel)
	_register("context.WithDeadline", context.WithDeadline)
	_register("context.WithTimeout", context.WithTimeout)
	_register("context.WithValue", context.WithValue)

	// package fmt
	////////////////////////////////////////
	_register("fmt.Errorf", fmt.Errorf)
	{
		var x fmt.Formatter
		_register("fmt.Formatter", reflect.TypeOf(x))
	}
	_register("fmt.Fprint", fmt.Fprint)
	_register("fmt.Fprintf", fmt.Fprintf)
	_register("fmt.Fprintln", fmt.Fprintln)
	_register("fmt.Fscan", fmt.Fscan)
	_register("fmt.Fscanf", fmt.Fscanf)
	_register("fmt.Fscanln", fmt.Fscanln)
	{
		var x fmt.GoStringer
		_register("fmt.GoStringer", reflect.TypeOf(x))
	}
	_register("fmt.Print", fmt.Print)
	_register("fmt.Printf", fmt.Printf)
	_register("fmt.Println", fmt.Println)
	_register("fmt.Scan", fmt.Scan)
	{
		var x fmt.ScanState
		_register("fmt.ScanState", reflect.TypeOf(x))
	}
	_register("fmt.Scanf", fmt.Scanf)
	_register("fmt.Scanln", fmt.Scanln)
	{
		var x fmt.Scanner
		_register("fmt.Scanner", reflect.TypeOf(x))
	}
	_register("fmt.Sprint", fmt.Sprint)
	_register("fmt.Sprintf", fmt.Sprintf)
	_register("fmt.Sprintln", fmt.Sprintln)
	_register("fmt.Sscan", fmt.Sscan)
	_register("fmt.Sscanf", fmt.Sscanf)
	_register("fmt.Sscanln", fmt.Sscanln)
	{
		var x fmt.State
		_register("fmt.State", reflect.TypeOf(x))
	}
	{
		var x fmt.Stringer
		_register("fmt.Stringer", reflect.TypeOf(x))
	}

	// package io
	////////////////////////////////////////
	{
		var x io.ByteReader
		_register("io.ByteReader", reflect.TypeOf(x))
	}
	{
		var x io.ByteScanner
		_register("io.ByteScanner", reflect.TypeOf(x))
	}
	{
		var x io.ByteWriter
		_register("io.ByteWriter", reflect.TypeOf(x))
	}
	{
		var x io.Closer
		_register("io.Closer", reflect.TypeOf(x))
	}
	_register("io.Copy", io.Copy)
	_register("io.CopyBuffer", io.CopyBuffer)
	_register("io.CopyN", io.CopyN)
	_register("io.Discard", io.Discard)
	_register("io.EOF", io.EOF)
	_register("io.ErrClosedPipe", io.ErrClosedPipe)
	_register("io.ErrNoProgress", io.ErrNoProgress)
	_register("io.ErrShortBuffer", io.ErrShortBuffer)
	_register("io.ErrShortWrite", io.ErrShortWrite)
	_register("io.ErrUnexpectedEOF", io.ErrUnexpectedEOF)
	_register("io.LimitReader", io.LimitReader)
	{
		var x io.LimitedReader
		_register("io.LimitedReader", reflect.TypeOf(x))
	}
	_register("io.MultiReader", io.MultiReader)
	_register("io.MultiWriter", io.MultiWriter)
	_register("io.NewSectionReader", io.NewSectionReader)
	_register("io.NopCloser", io.NopCloser)
	_register("io.Pipe", io.Pipe)
	{
		var x io.PipeReader
		_register("io.PipeReader", reflect.TypeOf(x))
	}
	{
		var x io.PipeWriter
		_register("io.PipeWriter", reflect.TypeOf(x))
	}
	_register("io.ReadAll", io.ReadAll)
	_register("io.ReadAtLeast", io.ReadAtLeast)
	{
		var x io.ReadCloser
		_register("io.ReadCloser", reflect.TypeOf(x))
	}
	_register("io.ReadFull", io.ReadFull)
	{
		var x io.ReadSeekCloser
		_register("io.ReadSeekCloser", reflect.TypeOf(x))
	}
	{
		var x io.ReadSeeker
		_register("io.ReadSeeker", reflect.TypeOf(x))
	}
	{
		var x io.ReadWriteCloser
		_register("io.ReadWriteCloser", reflect.TypeOf(x))
	}
	{
		var x io.ReadWriteSeeker
		_register("io.ReadWriteSeeker", reflect.TypeOf(x))
	}
	{
		var x io.ReadWriter
		_register("io.ReadWriter", reflect.TypeOf(x))
	}
	{
		var x io.Reader
		_register("io.Reader", reflect.TypeOf(x))
	}
	{
		var x io.ReaderAt
		_register("io.ReaderAt", reflect.TypeOf(x))
	}
	{
		var x io.ReaderFrom
		_register("io.ReaderFrom", reflect.TypeOf(x))
	}
	{
		var x io.RuneReader
		_register("io.RuneReader", reflect.TypeOf(x))
	}
	{
		var x io.RuneScanner
		_register("io.RuneScanner", reflect.TypeOf(x))
	}
	{
		var x io.SectionReader
		_register("io.SectionReader", reflect.TypeOf(x))
	}
	_register("io.SeekCurrent", io.SeekCurrent)
	_register("io.SeekEnd", io.SeekEnd)
	_register("io.SeekStart", io.SeekStart)
	{
		var x io.Seeker
		_register("io.Seeker", reflect.TypeOf(x))
	}
	{
		var x io.StringWriter
		_register("io.StringWriter", reflect.TypeOf(x))
	}
	_register("io.TeeReader", io.TeeReader)
	{
		var x io.WriteCloser
		_register("io.WriteCloser", reflect.TypeOf(x))
	}
	{
		var x io.WriteSeeker
		_register("io.WriteSeeker", reflect.TypeOf(x))
	}
	_register("io.WriteString", io.WriteString)
	{
		var x io.Writer
		_register("io.Writer", reflect.TypeOf(x))
	}
	{
		var x io.WriterAt
		_register("io.WriterAt", reflect.TypeOf(x))
	}
	{
		var x io.WriterTo
		_register("io.WriterTo", reflect.TypeOf(x))
	}

	// package io/fs
	////////////////////////////////////////
	{
		var x io_fs.DirEntry
		_register("io/fs.DirEntry", reflect.TypeOf(x))
	}
	_register("io/fs.ErrClosed", io_fs.ErrClosed)
	_register("io/fs.ErrExist", io_fs.ErrExist)
	_register("io/fs.ErrInvalid", io_fs.ErrInvalid)
	_register("io/fs.ErrNotExist", io_fs.ErrNotExist)
	_register("io/fs.ErrPermission", io_fs.ErrPermission)
	{
		var x io_fs.FS
		_register("io/fs.FS", reflect.TypeOf(x))
	}
	{
		var x io_fs.File
		_register("io/fs.File", reflect.TypeOf(x))
	}
	{
		var x io_fs.FileInfo
		_register("io/fs.FileInfo", reflect.TypeOf(x))
	}
	_register("io/fs.FileInfoToDirEntry", io_fs.FileInfoToDirEntry)
	{
		var x io_fs.FileMode
		_register("io/fs.FileMode", reflect.TypeOf(x))
	}
	_register("io/fs.Glob", io_fs.Glob)
	{
		var x io_fs.GlobFS
		_register("io/fs.GlobFS", reflect.TypeOf(x))
	}
	_register("io/fs.ModeAppend", io_fs.ModeAppend)
	_register("io/fs.ModeCharDevice", io_fs.ModeCharDevice)
	_register("io/fs.ModeDevice", io_fs.ModeDevice)
	_register("io/fs.ModeDir", io_fs.ModeDir)
	_register("io/fs.ModeExclusive", io_fs.ModeExclusive)
	_register("io/fs.ModeIrregular", io_fs.ModeIrregular)
	_register("io/fs.ModeNamedPipe", io_fs.ModeNamedPipe)
	_register("io/fs.ModePerm", io_fs.ModePerm)
	_register("io/fs.ModeSetgid", io_fs.ModeSetgid)
	_register("io/fs.ModeSetuid", io_fs.ModeSetuid)
	_register("io/fs.ModeSocket", io_fs.ModeSocket)
	_register("io/fs.ModeSticky", io_fs.ModeSticky)
	_register("io/fs.ModeSymlink", io_fs.ModeSymlink)
	_register("io/fs.ModeTemporary", io_fs.ModeTemporary)
	_register("io/fs.ModeType", io_fs.ModeType)
	{
		var x io_fs.PathError
		_register("io/fs.PathError", reflect.TypeOf(x))
	}
	_register("io/fs.ReadDir", io_fs.ReadDir)
	{
		var x io_fs.ReadDirFS
		_register("io/fs.ReadDirFS", reflect.TypeOf(x))
	}
	{
		var x io_fs.ReadDirFile
		_register("io/fs.ReadDirFile", reflect.TypeOf(x))
	}
	_register("io/fs.ReadFile", io_fs.ReadFile)
	{
		var x io_fs.ReadFileFS
		_register("io/fs.ReadFileFS", reflect.TypeOf(x))
	}
	_register("io/fs.SkipDir", io_fs.SkipDir)
	_register("io/fs.Stat", io_fs.Stat)
	{
		var x io_fs.StatFS
		_register("io/fs.StatFS", reflect.TypeOf(x))
	}
	_register("io/fs.Sub", io_fs.Sub)
	{
		var x io_fs.SubFS
		_register("io/fs.SubFS", reflect.TypeOf(x))
	}
	_register("io/fs.ValidPath", io_fs.ValidPath)
	_register("io/fs.WalkDir", io_fs.WalkDir)
	{
		var x io_fs.WalkDirFunc
		_register("io/fs.WalkDirFunc", reflect.TypeOf(x))
	}

	// package io/ioutil
	////////////////////////////////////////
	_register("io/ioutil.Discard", io_ioutil.Discard)
	_register("io/ioutil.NopCloser", io_ioutil.NopCloser)
	_register("io/ioutil.ReadAll", io_ioutil.ReadAll)
	_register("io/ioutil.ReadDir", io_ioutil.ReadDir)
	_register("io/ioutil.ReadFile", io_ioutil.ReadFile)
	_register("io/ioutil.TempDir", io_ioutil.TempDir)
	_register("io/ioutil.TempFile", io_ioutil.TempFile)
	_register("io/ioutil.WriteFile", io_ioutil.WriteFile)

	// package net/http
	////////////////////////////////////////
	_register("net/http.AllowQuerySemicolons", net_http.AllowQuerySemicolons)
	_register("net/http.CanonicalHeaderKey", net_http.CanonicalHeaderKey)
	{
		var x net_http.Client
		_register("net/http.Client", reflect.TypeOf(x))
	}
	{
		var x net_http.CloseNotifier
		_register("net/http.CloseNotifier", reflect.TypeOf(x))
	}
	{
		var x net_http.ConnState
		_register("net/http.ConnState", reflect.TypeOf(x))
	}
	{
		var x net_http.Cookie
		_register("net/http.Cookie", reflect.TypeOf(x))
	}
	{
		var x net_http.CookieJar
		_register("net/http.CookieJar", reflect.TypeOf(x))
	}
	_register("net/http.DefaultClient", net_http.DefaultClient)
	_register("net/http.DefaultMaxHeaderBytes", net_http.DefaultMaxHeaderBytes)
	_register("net/http.DefaultMaxIdleConnsPerHost", net_http.DefaultMaxIdleConnsPerHost)
	_register("net/http.DefaultServeMux", net_http.DefaultServeMux)
	_register("net/http.DefaultTransport", net_http.DefaultTransport)
	_register("net/http.DetectContentType", net_http.DetectContentType)
	{
		var x net_http.Dir
		_register("net/http.Dir", reflect.TypeOf(x))
	}
	_register("net/http.ErrAbortHandler", net_http.ErrAbortHandler)
	_register("net/http.ErrBodyNotAllowed", net_http.ErrBodyNotAllowed)
	_register("net/http.ErrBodyReadAfterClose", net_http.ErrBodyReadAfterClose)
	_register("net/http.ErrContentLength", net_http.ErrContentLength)
	_register("net/http.ErrHandlerTimeout", net_http.ErrHandlerTimeout)
	_register("net/http.ErrHeaderTooLong", net_http.ErrHeaderTooLong)
	_register("net/http.ErrHijacked", net_http.ErrHijacked)
	_register("net/http.ErrLineTooLong", net_http.ErrLineTooLong)
	_register("net/http.ErrMissingBoundary", net_http.ErrMissingBoundary)
	_register("net/http.ErrMissingContentLength", net_http.ErrMissingContentLength)
	_register("net/http.ErrMissingFile", net_http.ErrMissingFile)
	_register("net/http.ErrNoCookie", net_http.ErrNoCookie)
	_register("net/http.ErrNoLocation", net_http.ErrNoLocation)
	_register("net/http.ErrNotMultipart", net_http.ErrNotMultipart)
	_register("net/http.ErrNotSupported", net_http.ErrNotSupported)
	_register("net/http.ErrServerClosed", net_http.ErrServerClosed)
	_register("net/http.ErrShortBody", net_http.ErrShortBody)
	_register("net/http.ErrSkipAltProtocol", net_http.ErrSkipAltProtocol)
	_register("net/http.ErrUnexpectedTrailer", net_http.ErrUnexpectedTrailer)
	_register("net/http.ErrUseLastResponse", net_http.ErrUseLastResponse)
	_register("net/http.ErrWriteAfterFlush", net_http.ErrWriteAfterFlush)
	_register("net/http.Error", net_http.Error)
	_register("net/http.FS", net_http.FS)
	{
		var x net_http.File
		_register("net/http.File", reflect.TypeOf(x))
	}
	_register("net/http.FileServer", net_http.FileServer)
	{
		var x net_http.FileSystem
		_register("net/http.FileSystem", reflect.TypeOf(x))
	}
	{
		var x net_http.Flusher
		_register("net/http.Flusher", reflect.TypeOf(x))
	}
	_register("net/http.Get", net_http.Get)
	_register("net/http.Handle", net_http.Handle)
	_register("net/http.HandleFunc", net_http.HandleFunc)
	{
		var x net_http.Handler
		_register("net/http.Handler", reflect.TypeOf(x))
	}
	{
		var x net_http.HandlerFunc
		_register("net/http.HandlerFunc", reflect.TypeOf(x))
	}
	_register("net/http.Head", net_http.Head)
	{
		var x net_http.Header
		_register("net/http.Header", reflect.TypeOf(x))
	}
	{
		var x net_http.Hijacker
		_register("net/http.Hijacker", reflect.TypeOf(x))
	}
	_register("net/http.ListenAndServe", net_http.ListenAndServe)
	_register("net/http.ListenAndServeTLS", net_http.ListenAndServeTLS)
	_register("net/http.LocalAddrContextKey", net_http.LocalAddrContextKey)
	_register("net/http.MaxBytesHandler", net_http.MaxBytesHandler)
	_register("net/http.MaxBytesReader", net_http.MaxBytesReader)
	_register("net/http.MethodConnect", net_http.MethodConnect)
	_register("net/http.MethodDelete", net_http.MethodDelete)
	_register("net/http.MethodGet", net_http.MethodGet)
	_register("net/http.MethodHead", net_http.MethodHead)
	_register("net/http.MethodOptions", net_http.MethodOptions)
	_register("net/http.MethodPatch", net_http.MethodPatch)
	_register("net/http.MethodPost", net_http.MethodPost)
	_register("net/http.MethodPut", net_http.MethodPut)
	_register("net/http.MethodTrace", net_http.MethodTrace)
	_register("net/http.NewFileTransport", net_http.NewFileTransport)
	_register("net/http.NewRequest", net_http.NewRequest)
	_register("net/http.NewRequestWithContext", net_http.NewRequestWithContext)
	_register("net/http.NewServeMux", net_http.NewServeMux)
	_register("net/http.NoBody", net_http.NoBody)
	_register("net/http.NotFound", net_http.NotFound)
	_register("net/http.NotFoundHandler", net_http.NotFoundHandler)
	_register("net/http.ParseHTTPVersion", net_http.ParseHTTPVersion)
	_register("net/http.ParseTime", net_http.ParseTime)
	_register("net/http.Post", net_http.Post)
	_register("net/http.PostForm", net_http.PostForm)
	{
		var x net_http.ProtocolError
		_register("net/http.ProtocolError", reflect.TypeOf(x))
	}
	_register("net/http.ProxyFromEnvironment", net_http.ProxyFromEnvironment)
	_register("net/http.ProxyURL", net_http.ProxyURL)
	{
		var x net_http.PushOptions
		_register("net/http.PushOptions", reflect.TypeOf(x))
	}
	{
		var x net_http.Pusher
		_register("net/http.Pusher", reflect.TypeOf(x))
	}
	_register("net/http.ReadRequest", net_http.ReadRequest)
	_register("net/http.ReadResponse", net_http.ReadResponse)
	_register("net/http.Redirect", net_http.Redirect)
	_register("net/http.RedirectHandler", net_http.RedirectHandler)
	{
		var x net_http.Request
		_register("net/http.Request", reflect.TypeOf(x))
	}
	{
		var x net_http.Response
		_register("net/http.Response", reflect.TypeOf(x))
	}
	{
		var x net_http.ResponseWriter
		_register("net/http.ResponseWriter", reflect.TypeOf(x))
	}
	{
		var x net_http.RoundTripper
		_register("net/http.RoundTripper", reflect.TypeOf(x))
	}
	{
		var x net_http.SameSite
		_register("net/http.SameSite", reflect.TypeOf(x))
	}
	_register("net/http.SameSiteDefaultMode", net_http.SameSiteDefaultMode)
	_register("net/http.SameSiteLaxMode", net_http.SameSiteLaxMode)
	_register("net/http.SameSiteNoneMode", net_http.SameSiteNoneMode)
	_register("net/http.SameSiteStrictMode", net_http.SameSiteStrictMode)
	_register("net/http.Serve", net_http.Serve)
	_register("net/http.ServeContent", net_http.ServeContent)
	_register("net/http.ServeFile", net_http.ServeFile)
	{
		var x net_http.ServeMux
		_register("net/http.ServeMux", reflect.TypeOf(x))
	}
	_register("net/http.ServeTLS", net_http.ServeTLS)
	{
		var x net_http.Server
		_register("net/http.Server", reflect.TypeOf(x))
	}
	_register("net/http.ServerContextKey", net_http.ServerContextKey)
	_register("net/http.SetCookie", net_http.SetCookie)
	_register("net/http.StateActive", net_http.StateActive)
	_register("net/http.StateClosed", net_http.StateClosed)
	_register("net/http.StateHijacked", net_http.StateHijacked)
	_register("net/http.StateIdle", net_http.StateIdle)
	_register("net/http.StateNew", net_http.StateNew)
	_register("net/http.StatusAccepted", net_http.StatusAccepted)
	_register("net/http.StatusAlreadyReported", net_http.StatusAlreadyReported)
	_register("net/http.StatusBadGateway", net_http.StatusBadGateway)
	_register("net/http.StatusBadRequest", net_http.StatusBadRequest)
	_register("net/http.StatusConflict", net_http.StatusConflict)
	_register("net/http.StatusContinue", net_http.StatusContinue)
	_register("net/http.StatusCreated", net_http.StatusCreated)
	_register("net/http.StatusEarlyHints", net_http.StatusEarlyHints)
	_register("net/http.StatusExpectationFailed", net_http.StatusExpectationFailed)
	_register("net/http.StatusFailedDependency", net_http.StatusFailedDependency)
	_register("net/http.StatusForbidden", net_http.StatusForbidden)
	_register("net/http.StatusFound", net_http.StatusFound)
	_register("net/http.StatusGatewayTimeout", net_http.StatusGatewayTimeout)
	_register("net/http.StatusGone", net_http.StatusGone)
	_register("net/http.StatusHTTPVersionNotSupported", net_http.StatusHTTPVersionNotSupported)
	_register("net/http.StatusIMUsed", net_http.StatusIMUsed)
	_register("net/http.StatusInsufficientStorage", net_http.StatusInsufficientStorage)
	_register("net/http.StatusInternalServerError", net_http.StatusInternalServerError)
	_register("net/http.StatusLengthRequired", net_http.StatusLengthRequired)
	_register("net/http.StatusLocked", net_http.StatusLocked)
	_register("net/http.StatusLoopDetected", net_http.StatusLoopDetected)
	_register("net/http.StatusMethodNotAllowed", net_http.StatusMethodNotAllowed)
	_register("net/http.StatusMisdirectedRequest", net_http.StatusMisdirectedRequest)
	_register("net/http.StatusMovedPermanently", net_http.StatusMovedPermanently)
	_register("net/http.StatusMultiStatus", net_http.StatusMultiStatus)
	_register("net/http.StatusMultipleChoices", net_http.StatusMultipleChoices)
	_register("net/http.StatusNetworkAuthenticationRequired", net_http.StatusNetworkAuthenticationRequired)
	_register("net/http.StatusNoContent", net_http.StatusNoContent)
	_register("net/http.StatusNonAuthoritativeInfo", net_http.StatusNonAuthoritativeInfo)
	_register("net/http.StatusNotAcceptable", net_http.StatusNotAcceptable)
	_register("net/http.StatusNotExtended", net_http.StatusNotExtended)
	_register("net/http.StatusNotFound", net_http.StatusNotFound)
	_register("net/http.StatusNotImplemented", net_http.StatusNotImplemented)
	_register("net/http.StatusNotModified", net_http.StatusNotModified)
	_register("net/http.StatusOK", net_http.StatusOK)
	_register("net/http.StatusPartialContent", net_http.StatusPartialContent)
	_register("net/http.StatusPaymentRequired", net_http.StatusPaymentRequired)
	_register("net/http.StatusPermanentRedirect", net_http.StatusPermanentRedirect)
	_register("net/http.StatusPreconditionFailed", net_http.StatusPreconditionFailed)
	_register("net/http.StatusPreconditionRequired", net_http.StatusPreconditionRequired)
	_register("net/http.StatusProcessing", net_http.StatusProcessing)
	_register("net/http.StatusProxyAuthRequired", net_http.StatusProxyAuthRequired)
	_register("net/http.StatusRequestEntityTooLarge", net_http.StatusRequestEntityTooLarge)
	_register("net/http.StatusRequestHeaderFieldsTooLarge", net_http.StatusRequestHeaderFieldsTooLarge)
	_register("net/http.StatusRequestTimeout", net_http.StatusRequestTimeout)
	_register("net/http.StatusRequestURITooLong", net_http.StatusRequestURITooLong)
	_register("net/http.StatusRequestedRangeNotSatisfiable", net_http.StatusRequestedRangeNotSatisfiable)
	_register("net/http.StatusResetContent", net_http.StatusResetContent)
	_register("net/http.StatusSeeOther", net_http.StatusSeeOther)
	_register("net/http.StatusServiceUnavailable", net_http.StatusServiceUnavailable)
	_register("net/http.StatusSwitchingProtocols", net_http.StatusSwitchingProtocols)
	_register("net/http.StatusTeapot", net_http.StatusTeapot)
	_register("net/http.StatusTemporaryRedirect", net_http.StatusTemporaryRedirect)
	_register("net/http.StatusText", net_http.StatusText)
	_register("net/http.StatusTooEarly", net_http.StatusTooEarly)
	_register("net/http.StatusTooManyRequests", net_http.StatusTooManyRequests)
	_register("net/http.StatusUnauthorized", net_http.StatusUnauthorized)
	_register("net/http.StatusUnavailableForLegalReasons", net_http.StatusUnavailableForLegalReasons)
	_register("net/http.StatusUnprocessableEntity", net_http.StatusUnprocessableEntity)
	_register("net/http.StatusUnsupportedMediaType", net_http.StatusUnsupportedMediaType)
	_register("net/http.StatusUpgradeRequired", net_http.StatusUpgradeRequired)
	_register("net/http.StatusUseProxy", net_http.StatusUseProxy)
	_register("net/http.StatusVariantAlsoNegotiates", net_http.StatusVariantAlsoNegotiates)
	_register("net/http.StripPrefix", net_http.StripPrefix)
	_register("net/http.TimeFormat", net_http.TimeFormat)
	_register("net/http.TimeoutHandler", net_http.TimeoutHandler)
	_register("net/http.TrailerPrefix", net_http.TrailerPrefix)
	{
		var x net_http.Transport
		_register("net/http.Transport", reflect.TypeOf(x))
	}

	// package regexp
	////////////////////////////////////////
	_register("regexp.Compile", regexp.Compile)
	_register("regexp.CompilePOSIX", regexp.CompilePOSIX)
	_register("regexp.Match", regexp.Match)
	_register("regexp.MatchReader", regexp.MatchReader)
	_register("regexp.MatchString", regexp.MatchString)
	_register("regexp.MustCompile", regexp.MustCompile)
	_register("regexp.MustCompilePOSIX", regexp.MustCompilePOSIX)
	_register("regexp.QuoteMeta", regexp.QuoteMeta)
	{
		var x regexp.Regexp
		_register("regexp.Regexp", reflect.TypeOf(x))
	}

	// package strconv
	////////////////////////////////////////
	_register("strconv.AppendBool", strconv.AppendBool)
	_register("strconv.AppendFloat", strconv.AppendFloat)
	_register("strconv.AppendInt", strconv.AppendInt)
	_register("strconv.AppendQuote", strconv.AppendQuote)
	_register("strconv.AppendQuoteRune", strconv.AppendQuoteRune)
	_register("strconv.AppendQuoteRuneToASCII", strconv.AppendQuoteRuneToASCII)
	_register("strconv.AppendQuoteRuneToGraphic", strconv.AppendQuoteRuneToGraphic)
	_register("strconv.AppendQuoteToASCII", strconv.AppendQuoteToASCII)
	_register("strconv.AppendQuoteToGraphic", strconv.AppendQuoteToGraphic)
	_register("strconv.AppendUint", strconv.AppendUint)
	_register("strconv.Atoi", strconv.Atoi)
	_register("strconv.CanBackquote", strconv.CanBackquote)
	_register("strconv.ErrRange", strconv.ErrRange)
	_register("strconv.ErrSyntax", strconv.ErrSyntax)
	_register("strconv.FormatBool", strconv.FormatBool)
	_register("strconv.FormatComplex", strconv.FormatComplex)
	_register("strconv.FormatFloat", strconv.FormatFloat)
	_register("strconv.FormatInt", strconv.FormatInt)
	_register("strconv.FormatUint", strconv.FormatUint)
	_register("strconv.IntSize", strconv.IntSize)
	_register("strconv.IsGraphic", strconv.IsGraphic)
	_register("strconv.IsPrint", strconv.IsPrint)
	_register("strconv.Itoa", strconv.Itoa)
	{
		var x strconv.NumError
		_register("strconv.NumError", reflect.TypeOf(x))
	}
	_register("strconv.ParseBool", strconv.ParseBool)
	_register("strconv.ParseComplex", strconv.ParseComplex)
	_register("strconv.ParseFloat", strconv.ParseFloat)
	_register("strconv.ParseInt", strconv.ParseInt)
	_register("strconv.ParseUint", strconv.ParseUint)
	_register("strconv.Quote", strconv.Quote)
	_register("strconv.QuoteRune", strconv.QuoteRune)
	_register("strconv.QuoteRuneToASCII", strconv.QuoteRuneToASCII)
	_register("strconv.QuoteRuneToGraphic", strconv.QuoteRuneToGraphic)
	_register("strconv.QuoteToASCII", strconv.QuoteToASCII)
	_register("strconv.QuoteToGraphic", strconv.QuoteToGraphic)
	_register("strconv.QuotedPrefix", strconv.QuotedPrefix)
	_register("strconv.Unquote", strconv.Unquote)
	_register("strconv.UnquoteChar", strconv.UnquoteChar)

	// package strings
	////////////////////////////////////////
	{
		var x strings.Builder
		_register("strings.Builder", reflect.TypeOf(x))
	}
	_register("strings.Clone", strings.Clone)
	_register("strings.Compare", strings.Compare)
	_register("strings.Contains", strings.Contains)
	_register("strings.ContainsAny", strings.ContainsAny)
	_register("strings.ContainsRune", strings.ContainsRune)
	_register("strings.Count", strings.Count)
	_register("strings.Cut", strings.Cut)
	_register("strings.EqualFold", strings.EqualFold)
	_register("strings.Fields", strings.Fields)
	_register("strings.FieldsFunc", strings.FieldsFunc)
	_register("strings.HasPrefix", strings.HasPrefix)
	_register("strings.HasSuffix", strings.HasSuffix)
	_register("strings.Index", strings.Index)
	_register("strings.IndexAny", strings.IndexAny)
	_register("strings.IndexByte", strings.IndexByte)
	_register("strings.IndexFunc", strings.IndexFunc)
	_register("strings.IndexRune", strings.IndexRune)
	_register("strings.Join", strings.Join)
	_register("strings.LastIndex", strings.LastIndex)
	_register("strings.LastIndexAny", strings.LastIndexAny)
	_register("strings.LastIndexByte", strings.LastIndexByte)
	_register("strings.LastIndexFunc", strings.LastIndexFunc)
	_register("strings.Map", strings.Map)
	_register("strings.NewReader", strings.NewReader)
	_register("strings.NewReplacer", strings.NewReplacer)
	{
		var x strings.Reader
		_register("strings.Reader", reflect.TypeOf(x))
	}
	_register("strings.Repeat", strings.Repeat)
	_register("strings.Replace", strings.Replace)
	_register("strings.ReplaceAll", strings.ReplaceAll)
	{
		var x strings.Replacer
		_register("strings.Replacer", reflect.TypeOf(x))
	}
	_register("strings.Split", strings.Split)
	_register("strings.SplitAfter", strings.SplitAfter)
	_register("strings.SplitAfterN", strings.SplitAfterN)
	_register("strings.SplitN", strings.SplitN)
	_register("strings.Title", strings.Title)
	_register("strings.ToLower", strings.ToLower)
	_register("strings.ToLowerSpecial", strings.ToLowerSpecial)
	_register("strings.ToTitle", strings.ToTitle)
	_register("strings.ToTitleSpecial", strings.ToTitleSpecial)
	_register("strings.ToUpper", strings.ToUpper)
	_register("strings.ToUpperSpecial", strings.ToUpperSpecial)
	_register("strings.ToValidUTF8", strings.ToValidUTF8)
	_register("strings.Trim", strings.Trim)
	_register("strings.TrimFunc", strings.TrimFunc)
	_register("strings.TrimLeft", strings.TrimLeft)
	_register("strings.TrimLeftFunc", strings.TrimLeftFunc)
	_register("strings.TrimPrefix", strings.TrimPrefix)
	_register("strings.TrimRight", strings.TrimRight)
	_register("strings.TrimRightFunc", strings.TrimRightFunc)
	_register("strings.TrimSpace", strings.TrimSpace)
	_register("strings.TrimSuffix", strings.TrimSuffix)

	// package time
	////////////////////////////////////////
	_register("time.ANSIC", time.ANSIC)
	_register("time.After", time.After)
	_register("time.AfterFunc", time.AfterFunc)
	_register("time.April", time.April)
	_register("time.August", time.August)
	_register("time.Date", time.Date)
	_register("time.December", time.December)
	{
		var x time.Duration
		_register("time.Duration", reflect.TypeOf(x))
	}
	_register("time.February", time.February)
	_register("time.FixedZone", time.FixedZone)
	_register("time.Friday", time.Friday)
	_register("time.Hour", time.Hour)
	_register("time.January", time.January)
	_register("time.July", time.July)
	_register("time.June", time.June)
	_register("time.Kitchen", time.Kitchen)
	_register("time.Layout", time.Layout)
	_register("time.LoadLocation", time.LoadLocation)
	_register("time.LoadLocationFromTZData", time.LoadLocationFromTZData)
	_register("time.Local", time.Local)
	{
		var x time.Location
		_register("time.Location", reflect.TypeOf(x))
	}
	_register("time.March", time.March)
	_register("time.May", time.May)
	_register("time.Microsecond", time.Microsecond)
	_register("time.Millisecond", time.Millisecond)
	_register("time.Minute", time.Minute)
	_register("time.Monday", time.Monday)
	{
		var x time.Month
		_register("time.Month", reflect.TypeOf(x))
	}
	_register("time.Nanosecond", time.Nanosecond)
	_register("time.NewTicker", time.NewTicker)
	_register("time.NewTimer", time.NewTimer)
	_register("time.November", time.November)
	_register("time.Now", time.Now)
	_register("time.October", time.October)
	_register("time.Parse", time.Parse)
	_register("time.ParseDuration", time.ParseDuration)
	{
		var x time.ParseError
		_register("time.ParseError", reflect.TypeOf(x))
	}
	_register("time.ParseInLocation", time.ParseInLocation)
	_register("time.RFC1123", time.RFC1123)
	_register("time.RFC1123Z", time.RFC1123Z)
	_register("time.RFC3339", time.RFC3339)
	_register("time.RFC3339Nano", time.RFC3339Nano)
	_register("time.RFC822", time.RFC822)
	_register("time.RFC822Z", time.RFC822Z)
	_register("time.RFC850", time.RFC850)
	_register("time.RubyDate", time.RubyDate)
	_register("time.Saturday", time.Saturday)
	_register("time.Second", time.Second)
	_register("time.September", time.September)
	_register("time.Since", time.Since)
	_register("time.Sleep", time.Sleep)
	_register("time.Stamp", time.Stamp)
	_register("time.StampMicro", time.StampMicro)
	_register("time.StampMilli", time.StampMilli)
	_register("time.StampNano", time.StampNano)
	_register("time.Sunday", time.Sunday)
	_register("time.Thursday", time.Thursday)
	_register("time.Tick", time.Tick)
	{
		var x time.Ticker
		_register("time.Ticker", reflect.TypeOf(x))
	}
	{
		var x time.Time
		_register("time.Time", reflect.TypeOf(x))
	}
	{
		var x time.Timer
		_register("time.Timer", reflect.TypeOf(x))
	}
	_register("time.Tuesday", time.Tuesday)
	_register("time.UTC", time.UTC)
	_register("time.Unix", time.Unix)
	_register("time.UnixDate", time.UnixDate)
	_register("time.UnixMicro", time.UnixMicro)
	_register("time.UnixMilli", time.UnixMilli)
	_register("time.Until", time.Until)
	_register("time.Wednesday", time.Wednesday)
	{
		var x time.Weekday
		_register("time.Weekday", reflect.TypeOf(x))
	}

	// package math/big
	////////////////////////////////////////
	_register("math/big.Above", math_big.Above)
	{
		var x math_big.Accuracy
		_register("math/big.Accuracy", reflect.TypeOf(x))
	}
	_register("math/big.AwayFromZero", math_big.AwayFromZero)
	_register("math/big.Below", math_big.Below)
	{
		var x math_big.ErrNaN
		_register("math/big.ErrNaN", reflect.TypeOf(x))
	}
	_register("math/big.Exact", math_big.Exact)
	{
		var x math_big.Float
		_register("math/big.Float", reflect.TypeOf(x))
	}
	{
		var x math_big.Int
		_register("math/big.Int", reflect.TypeOf(x))
	}
	_register("math/big.Jacobi", math_big.Jacobi)
	_register("math/big.MaxBase", math_big.MaxBase)
	_register("math/big.MaxExp", math_big.MaxExp)
	_register("math/big.MaxPrec", math_big.MaxPrec)
	_register("math/big.MinExp", math_big.MinExp)
	_register("math/big.NewFloat", math_big.NewFloat)
	_register("math/big.NewInt", math_big.NewInt)
	_register("math/big.NewRat", math_big.NewRat)
	_register("math/big.ParseFloat", math_big.ParseFloat)
	{
		var x math_big.Rat
		_register("math/big.Rat", reflect.TypeOf(x))
	}
	{
		var x math_big.RoundingMode
		_register("math/big.RoundingMode", reflect.TypeOf(x))
	}
	_register("math/big.ToNearestAway", math_big.ToNearestAway)
	_register("math/big.ToNearestEven", math_big.ToNearestEven)
	_register("math/big.ToNegativeInf", math_big.ToNegativeInf)
	_register("math/big.ToPositiveInf", math_big.ToPositiveInf)
	_register("math/big.ToZero", math_big.ToZero)
	{
		var x math_big.Word
		_register("math/big.Word", reflect.TypeOf(x))
	}

	// package math/rand
	////////////////////////////////////////
	_register("math/rand.ExpFloat64", math_rand.ExpFloat64)
	_register("math/rand.Float32", math_rand.Float32)
	_register("math/rand.Float64", math_rand.Float64)
	_register("math/rand.Int", math_rand.Int)
	_register("math/rand.Int31", math_rand.Int31)
	_register("math/rand.Int31n", math_rand.Int31n)
	_register("math/rand.Int63", math_rand.Int63)
	_register("math/rand.Int63n", math_rand.Int63n)
	_register("math/rand.Intn", math_rand.Intn)
	_register("math/rand.New", math_rand.New)
	_register("math/rand.NewSource", math_rand.NewSource)
	_register("math/rand.NewZipf", math_rand.NewZipf)
	_register("math/rand.NormFloat64", math_rand.NormFloat64)
	_register("math/rand.Perm", math_rand.Perm)
	{
		var x math_rand.Rand
		_register("math/rand.Rand", reflect.TypeOf(x))
	}
	_register("math/rand.Read", math_rand.Read)
	_register("math/rand.Seed", math_rand.Seed)
	_register("math/rand.Shuffle", math_rand.Shuffle)
	{
		var x math_rand.Source
		_register("math/rand.Source", reflect.TypeOf(x))
	}
	{
		var x math_rand.Source64
		_register("math/rand.Source64", reflect.TypeOf(x))
	}
	_register("math/rand.Uint32", math_rand.Uint32)
	_register("math/rand.Uint64", math_rand.Uint64)
	{
		var x math_rand.Zipf
		_register("math/rand.Zipf", reflect.TypeOf(x))
	}

	// package math
	////////////////////////////////////////
	_register("math.Abs", math.Abs)
	_register("math.Acos", math.Acos)
	_register("math.Acosh", math.Acosh)
	_register("math.Asin", math.Asin)
	_register("math.Asinh", math.Asinh)
	_register("math.Atan", math.Atan)
	_register("math.Atan2", math.Atan2)
	_register("math.Atanh", math.Atanh)
	_register("math.Cbrt", math.Cbrt)
	_register("math.Ceil", math.Ceil)
	_register("math.Copysign", math.Copysign)
	_register("math.Cos", math.Cos)
	_register("math.Cosh", math.Cosh)
	_register("math.Dim", math.Dim)
	_register("math.E", math.E)
	_register("math.Erf", math.Erf)
	_register("math.Erfc", math.Erfc)
	_register("math.Erfcinv", math.Erfcinv)
	_register("math.Erfinv", math.Erfinv)
	_register("math.Exp", math.Exp)
	_register("math.Exp2", math.Exp2)
	_register("math.Expm1", math.Expm1)
	_register("math.FMA", math.FMA)
	_register("math.Float32bits", math.Float32bits)
	_register("math.Float32frombits", math.Float32frombits)
	_register("math.Float64bits", math.Float64bits)
	_register("math.Float64frombits", math.Float64frombits)
	_register("math.Floor", math.Floor)
	_register("math.Frexp", math.Frexp)
	_register("math.Gamma", math.Gamma)
	_register("math.Hypot", math.Hypot)
	_register("math.Ilogb", math.Ilogb)
	_register("math.Inf", math.Inf)
	_register("math.IsInf", math.IsInf)
	_register("math.IsNaN", math.IsNaN)
	_register("math.J0", math.J0)
	_register("math.J1", math.J1)
	_register("math.Jn", math.Jn)
	_register("math.Ldexp", math.Ldexp)
	_register("math.Lgamma", math.Lgamma)
	_register("math.Ln10", math.Ln10)
	_register("math.Ln2", math.Ln2)
	_register("math.Log", math.Log)
	_register("math.Log10", math.Log10)
	_register("math.Log10E", math.Log10E)
	_register("math.Log1p", math.Log1p)
	_register("math.Log2", math.Log2)
	_register("math.Log2E", math.Log2E)
	_register("math.Logb", math.Logb)
	_register("math.Max", math.Max)
	_register("math.MaxFloat32", math.MaxFloat32)
	_register("math.MaxFloat64", math.MaxFloat64)
	_register("math.MaxInt", math.MaxInt)
	_register("math.MaxInt16", math.MaxInt16)
	_register("math.MaxInt32", math.MaxInt32)
	_register("math.MaxInt64", math.MaxInt64)
	_register("math.MaxInt8", math.MaxInt8)
	_register("math.MaxUint", uint(math.MaxUint))
	_register("math.MaxUint16", math.MaxUint16)
	_register("math.MaxUint32", math.MaxUint32)
	_register("math.MaxUint64", uint64(math.MaxUint64))
	_register("math.MaxUint8", math.MaxUint8)
	_register("math.Min", math.Min)
	_register("math.MinInt", math.MinInt)
	_register("math.MinInt16", math.MinInt16)
	_register("math.MinInt32", math.MinInt32)
	_register("math.MinInt64", math.MinInt64)
	_register("math.MinInt8", math.MinInt8)
	_register("math.Mod", math.Mod)
	_register("math.Modf", math.Modf)
	_register("math.NaN", math.NaN)
	_register("math.Nextafter", math.Nextafter)
	_register("math.Nextafter32", math.Nextafter32)
	_register("math.Phi", math.Phi)
	_register("math.Pi", math.Pi)
	_register("math.Pow", math.Pow)
	_register("math.Pow10", math.Pow10)
	_register("math.Remainder", math.Remainder)
	_register("math.Round", math.Round)
	_register("math.RoundToEven", math.RoundToEven)
	_register("math.Signbit", math.Signbit)
	_register("math.Sin", math.Sin)
	_register("math.Sincos", math.Sincos)
	_register("math.Sinh", math.Sinh)
	_register("math.SmallestNonzeroFloat32", math.SmallestNonzeroFloat32)
	_register("math.SmallestNonzeroFloat64", math.SmallestNonzeroFloat64)
	_register("math.Sqrt", math.Sqrt)
	_register("math.Sqrt2", math.Sqrt2)
	_register("math.SqrtE", math.SqrtE)
	_register("math.SqrtPhi", math.SqrtPhi)
	_register("math.SqrtPi", math.SqrtPi)
	_register("math.Tan", math.Tan)
	_register("math.Tanh", math.Tanh)
	_register("math.Trunc", math.Trunc)
	_register("math.Y0", math.Y0)
	_register("math.Y1", math.Y1)
	_register("math.Yn", math.Yn)
}
