// GENERATED FILE. DO NOT EDIT.
package gljimports

import (
	bytes "bytes"
	context "context"
	flag "flag"
	fmt "fmt"
	io "io"
	io_fs "io/fs"
	io_ioutil "io/ioutil"
	math "math"
	math_big "math/big"
	math_rand "math/rand"
	net_http "net/http"
	os "os"
	os_exec "os/exec"
	os_signal "os/signal"
	regexp "regexp"
	strconv "strconv"
	strings "strings"
	time "time"
	unicode "unicode"
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

	// package flag
	////////////////////////////////////////
	_register("flag.Arg", flag.Arg)
	_register("flag.Args", flag.Args)
	_register("flag.Bool", flag.Bool)
	_register("flag.BoolVar", flag.BoolVar)
	_register("flag.CommandLine", flag.CommandLine)
	_register("flag.ContinueOnError", flag.ContinueOnError)
	_register("flag.Duration", flag.Duration)
	_register("flag.DurationVar", flag.DurationVar)
	_register("flag.ErrHelp", flag.ErrHelp)
	{
		var x flag.ErrorHandling
		_register("flag.ErrorHandling", reflect.TypeOf(x))
	}
	_register("flag.ExitOnError", flag.ExitOnError)
	{
		var x flag.Flag
		_register("flag.Flag", reflect.TypeOf(x))
	}
	{
		var x flag.FlagSet
		_register("flag.FlagSet", reflect.TypeOf(x))
	}
	_register("flag.Float64", flag.Float64)
	_register("flag.Float64Var", flag.Float64Var)
	_register("flag.Func", flag.Func)
	{
		var x flag.Getter
		_register("flag.Getter", reflect.TypeOf(x))
	}
	_register("flag.Int", flag.Int)
	_register("flag.Int64", flag.Int64)
	_register("flag.Int64Var", flag.Int64Var)
	_register("flag.IntVar", flag.IntVar)
	_register("flag.Lookup", flag.Lookup)
	_register("flag.NArg", flag.NArg)
	_register("flag.NFlag", flag.NFlag)
	_register("flag.NewFlagSet", flag.NewFlagSet)
	_register("flag.PanicOnError", flag.PanicOnError)
	_register("flag.Parse", flag.Parse)
	_register("flag.Parsed", flag.Parsed)
	_register("flag.PrintDefaults", flag.PrintDefaults)
	_register("flag.Set", flag.Set)
	_register("flag.String", flag.String)
	_register("flag.StringVar", flag.StringVar)
	_register("flag.TextVar", flag.TextVar)
	_register("flag.Uint", flag.Uint)
	_register("flag.Uint64", flag.Uint64)
	_register("flag.Uint64Var", flag.Uint64Var)
	_register("flag.UintVar", flag.UintVar)
	_register("flag.UnquoteUsage", flag.UnquoteUsage)
	_register("flag.Usage", flag.Usage)
	{
		var x flag.Value
		_register("flag.Value", reflect.TypeOf(x))
	}
	_register("flag.Var", flag.Var)
	_register("flag.Visit", flag.Visit)
	_register("flag.VisitAll", flag.VisitAll)

	// package fmt
	////////////////////////////////////////
	_register("fmt.Append", fmt.Append)
	_register("fmt.Appendf", fmt.Appendf)
	_register("fmt.Appendln", fmt.Appendln)
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
	{
		var x net_http.MaxBytesError
		_register("net/http.MaxBytesError", reflect.TypeOf(x))
	}
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

	// package os
	////////////////////////////////////////
	_register("os.Args", os.Args)
	_register("os.Chdir", os.Chdir)
	_register("os.Chmod", os.Chmod)
	_register("os.Chown", os.Chown)
	_register("os.Chtimes", os.Chtimes)
	_register("os.Clearenv", os.Clearenv)
	_register("os.Create", os.Create)
	_register("os.CreateTemp", os.CreateTemp)
	_register("os.DevNull", os.DevNull)
	{
		var x os.DirEntry
		_register("os.DirEntry", reflect.TypeOf(x))
	}
	_register("os.DirFS", os.DirFS)
	_register("os.Environ", os.Environ)
	_register("os.ErrClosed", os.ErrClosed)
	_register("os.ErrDeadlineExceeded", os.ErrDeadlineExceeded)
	_register("os.ErrExist", os.ErrExist)
	_register("os.ErrInvalid", os.ErrInvalid)
	_register("os.ErrNoDeadline", os.ErrNoDeadline)
	_register("os.ErrNotExist", os.ErrNotExist)
	_register("os.ErrPermission", os.ErrPermission)
	_register("os.ErrProcessDone", os.ErrProcessDone)
	_register("os.Executable", os.Executable)
	_register("os.Exit", os.Exit)
	_register("os.Expand", os.Expand)
	_register("os.ExpandEnv", os.ExpandEnv)
	{
		var x os.File
		_register("os.File", reflect.TypeOf(x))
	}
	{
		var x os.FileInfo
		_register("os.FileInfo", reflect.TypeOf(x))
	}
	{
		var x os.FileMode
		_register("os.FileMode", reflect.TypeOf(x))
	}
	_register("os.FindProcess", os.FindProcess)
	_register("os.Getegid", os.Getegid)
	_register("os.Getenv", os.Getenv)
	_register("os.Geteuid", os.Geteuid)
	_register("os.Getgid", os.Getgid)
	_register("os.Getgroups", os.Getgroups)
	_register("os.Getpagesize", os.Getpagesize)
	_register("os.Getpid", os.Getpid)
	_register("os.Getppid", os.Getppid)
	_register("os.Getuid", os.Getuid)
	_register("os.Getwd", os.Getwd)
	_register("os.Hostname", os.Hostname)
	_register("os.Interrupt", os.Interrupt)
	_register("os.IsExist", os.IsExist)
	_register("os.IsNotExist", os.IsNotExist)
	_register("os.IsPathSeparator", os.IsPathSeparator)
	_register("os.IsPermission", os.IsPermission)
	_register("os.IsTimeout", os.IsTimeout)
	_register("os.Kill", os.Kill)
	_register("os.Lchown", os.Lchown)
	_register("os.Link", os.Link)
	{
		var x os.LinkError
		_register("os.LinkError", reflect.TypeOf(x))
	}
	_register("os.LookupEnv", os.LookupEnv)
	_register("os.Lstat", os.Lstat)
	_register("os.Mkdir", os.Mkdir)
	_register("os.MkdirAll", os.MkdirAll)
	_register("os.MkdirTemp", os.MkdirTemp)
	_register("os.ModeAppend", os.ModeAppend)
	_register("os.ModeCharDevice", os.ModeCharDevice)
	_register("os.ModeDevice", os.ModeDevice)
	_register("os.ModeDir", os.ModeDir)
	_register("os.ModeExclusive", os.ModeExclusive)
	_register("os.ModeIrregular", os.ModeIrregular)
	_register("os.ModeNamedPipe", os.ModeNamedPipe)
	_register("os.ModePerm", os.ModePerm)
	_register("os.ModeSetgid", os.ModeSetgid)
	_register("os.ModeSetuid", os.ModeSetuid)
	_register("os.ModeSocket", os.ModeSocket)
	_register("os.ModeSticky", os.ModeSticky)
	_register("os.ModeSymlink", os.ModeSymlink)
	_register("os.ModeTemporary", os.ModeTemporary)
	_register("os.ModeType", os.ModeType)
	_register("os.NewFile", os.NewFile)
	_register("os.NewSyscallError", os.NewSyscallError)
	_register("os.O_APPEND", os.O_APPEND)
	_register("os.O_CREATE", os.O_CREATE)
	_register("os.O_EXCL", os.O_EXCL)
	_register("os.O_RDONLY", os.O_RDONLY)
	_register("os.O_RDWR", os.O_RDWR)
	_register("os.O_SYNC", os.O_SYNC)
	_register("os.O_TRUNC", os.O_TRUNC)
	_register("os.O_WRONLY", os.O_WRONLY)
	_register("os.Open", os.Open)
	_register("os.OpenFile", os.OpenFile)
	{
		var x os.PathError
		_register("os.PathError", reflect.TypeOf(x))
	}
	_register("os.PathListSeparator", os.PathListSeparator)
	_register("os.PathSeparator", os.PathSeparator)
	_register("os.Pipe", os.Pipe)
	{
		var x os.ProcAttr
		_register("os.ProcAttr", reflect.TypeOf(x))
	}
	{
		var x os.Process
		_register("os.Process", reflect.TypeOf(x))
	}
	{
		var x os.ProcessState
		_register("os.ProcessState", reflect.TypeOf(x))
	}
	_register("os.ReadDir", os.ReadDir)
	_register("os.ReadFile", os.ReadFile)
	_register("os.Readlink", os.Readlink)
	_register("os.Remove", os.Remove)
	_register("os.RemoveAll", os.RemoveAll)
	_register("os.Rename", os.Rename)
	_register("os.SEEK_CUR", os.SEEK_CUR)
	_register("os.SEEK_END", os.SEEK_END)
	_register("os.SEEK_SET", os.SEEK_SET)
	_register("os.SameFile", os.SameFile)
	_register("os.Setenv", os.Setenv)
	{
		var x os.Signal
		_register("os.Signal", reflect.TypeOf(x))
	}
	_register("os.StartProcess", os.StartProcess)
	_register("os.Stat", os.Stat)
	_register("os.Stderr", os.Stderr)
	_register("os.Stdin", os.Stdin)
	_register("os.Stdout", os.Stdout)
	_register("os.Symlink", os.Symlink)
	{
		var x os.SyscallError
		_register("os.SyscallError", reflect.TypeOf(x))
	}
	_register("os.TempDir", os.TempDir)
	_register("os.Truncate", os.Truncate)
	_register("os.Unsetenv", os.Unsetenv)
	_register("os.UserCacheDir", os.UserCacheDir)
	_register("os.UserConfigDir", os.UserConfigDir)
	_register("os.UserHomeDir", os.UserHomeDir)
	_register("os.WriteFile", os.WriteFile)

	// package os/exec
	////////////////////////////////////////
	{
		var x os_exec.Cmd
		_register("os/exec.Cmd", reflect.TypeOf(x))
	}
	_register("os/exec.Command", os_exec.Command)
	_register("os/exec.CommandContext", os_exec.CommandContext)
	_register("os/exec.ErrDot", os_exec.ErrDot)
	_register("os/exec.ErrNotFound", os_exec.ErrNotFound)
	{
		var x os_exec.Error
		_register("os/exec.Error", reflect.TypeOf(x))
	}
	{
		var x os_exec.ExitError
		_register("os/exec.ExitError", reflect.TypeOf(x))
	}
	_register("os/exec.LookPath", os_exec.LookPath)

	// package os/signal
	////////////////////////////////////////
	_register("os/signal.Ignore", os_signal.Ignore)
	_register("os/signal.Ignored", os_signal.Ignored)
	_register("os/signal.Notify", os_signal.Notify)
	_register("os/signal.NotifyContext", os_signal.NotifyContext)
	_register("os/signal.Reset", os_signal.Reset)
	_register("os/signal.Stop", os_signal.Stop)

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

	// package unicode
	////////////////////////////////////////
	_register("unicode.ASCII_Hex_Digit", unicode.ASCII_Hex_Digit)
	_register("unicode.Adlam", unicode.Adlam)
	_register("unicode.Ahom", unicode.Ahom)
	_register("unicode.Anatolian_Hieroglyphs", unicode.Anatolian_Hieroglyphs)
	_register("unicode.Arabic", unicode.Arabic)
	_register("unicode.Armenian", unicode.Armenian)
	_register("unicode.Avestan", unicode.Avestan)
	_register("unicode.AzeriCase", unicode.AzeriCase)
	_register("unicode.Balinese", unicode.Balinese)
	_register("unicode.Bamum", unicode.Bamum)
	_register("unicode.Bassa_Vah", unicode.Bassa_Vah)
	_register("unicode.Batak", unicode.Batak)
	_register("unicode.Bengali", unicode.Bengali)
	_register("unicode.Bhaiksuki", unicode.Bhaiksuki)
	_register("unicode.Bidi_Control", unicode.Bidi_Control)
	_register("unicode.Bopomofo", unicode.Bopomofo)
	_register("unicode.Brahmi", unicode.Brahmi)
	_register("unicode.Braille", unicode.Braille)
	_register("unicode.Buginese", unicode.Buginese)
	_register("unicode.Buhid", unicode.Buhid)
	_register("unicode.C", unicode.C)
	_register("unicode.Canadian_Aboriginal", unicode.Canadian_Aboriginal)
	_register("unicode.Carian", unicode.Carian)
	{
		var x unicode.CaseRange
		_register("unicode.CaseRange", reflect.TypeOf(x))
	}
	_register("unicode.CaseRanges", unicode.CaseRanges)
	_register("unicode.Categories", unicode.Categories)
	_register("unicode.Caucasian_Albanian", unicode.Caucasian_Albanian)
	_register("unicode.Cc", unicode.Cc)
	_register("unicode.Cf", unicode.Cf)
	_register("unicode.Chakma", unicode.Chakma)
	_register("unicode.Cham", unicode.Cham)
	_register("unicode.Cherokee", unicode.Cherokee)
	_register("unicode.Chorasmian", unicode.Chorasmian)
	_register("unicode.Co", unicode.Co)
	_register("unicode.Common", unicode.Common)
	_register("unicode.Coptic", unicode.Coptic)
	_register("unicode.Cs", unicode.Cs)
	_register("unicode.Cuneiform", unicode.Cuneiform)
	_register("unicode.Cypriot", unicode.Cypriot)
	_register("unicode.Cyrillic", unicode.Cyrillic)
	_register("unicode.Dash", unicode.Dash)
	_register("unicode.Deprecated", unicode.Deprecated)
	_register("unicode.Deseret", unicode.Deseret)
	_register("unicode.Devanagari", unicode.Devanagari)
	_register("unicode.Diacritic", unicode.Diacritic)
	_register("unicode.Digit", unicode.Digit)
	_register("unicode.Dives_Akuru", unicode.Dives_Akuru)
	_register("unicode.Dogra", unicode.Dogra)
	_register("unicode.Duployan", unicode.Duployan)
	_register("unicode.Egyptian_Hieroglyphs", unicode.Egyptian_Hieroglyphs)
	_register("unicode.Elbasan", unicode.Elbasan)
	_register("unicode.Elymaic", unicode.Elymaic)
	_register("unicode.Ethiopic", unicode.Ethiopic)
	_register("unicode.Extender", unicode.Extender)
	_register("unicode.FoldCategory", unicode.FoldCategory)
	_register("unicode.FoldScript", unicode.FoldScript)
	_register("unicode.Georgian", unicode.Georgian)
	_register("unicode.Glagolitic", unicode.Glagolitic)
	_register("unicode.Gothic", unicode.Gothic)
	_register("unicode.Grantha", unicode.Grantha)
	_register("unicode.GraphicRanges", unicode.GraphicRanges)
	_register("unicode.Greek", unicode.Greek)
	_register("unicode.Gujarati", unicode.Gujarati)
	_register("unicode.Gunjala_Gondi", unicode.Gunjala_Gondi)
	_register("unicode.Gurmukhi", unicode.Gurmukhi)
	_register("unicode.Han", unicode.Han)
	_register("unicode.Hangul", unicode.Hangul)
	_register("unicode.Hanifi_Rohingya", unicode.Hanifi_Rohingya)
	_register("unicode.Hanunoo", unicode.Hanunoo)
	_register("unicode.Hatran", unicode.Hatran)
	_register("unicode.Hebrew", unicode.Hebrew)
	_register("unicode.Hex_Digit", unicode.Hex_Digit)
	_register("unicode.Hiragana", unicode.Hiragana)
	_register("unicode.Hyphen", unicode.Hyphen)
	_register("unicode.IDS_Binary_Operator", unicode.IDS_Binary_Operator)
	_register("unicode.IDS_Trinary_Operator", unicode.IDS_Trinary_Operator)
	_register("unicode.Ideographic", unicode.Ideographic)
	_register("unicode.Imperial_Aramaic", unicode.Imperial_Aramaic)
	_register("unicode.In", unicode.In)
	_register("unicode.Inherited", unicode.Inherited)
	_register("unicode.Inscriptional_Pahlavi", unicode.Inscriptional_Pahlavi)
	_register("unicode.Inscriptional_Parthian", unicode.Inscriptional_Parthian)
	_register("unicode.Is", unicode.Is)
	_register("unicode.IsControl", unicode.IsControl)
	_register("unicode.IsDigit", unicode.IsDigit)
	_register("unicode.IsGraphic", unicode.IsGraphic)
	_register("unicode.IsLetter", unicode.IsLetter)
	_register("unicode.IsLower", unicode.IsLower)
	_register("unicode.IsMark", unicode.IsMark)
	_register("unicode.IsNumber", unicode.IsNumber)
	_register("unicode.IsOneOf", unicode.IsOneOf)
	_register("unicode.IsPrint", unicode.IsPrint)
	_register("unicode.IsPunct", unicode.IsPunct)
	_register("unicode.IsSpace", unicode.IsSpace)
	_register("unicode.IsSymbol", unicode.IsSymbol)
	_register("unicode.IsTitle", unicode.IsTitle)
	_register("unicode.IsUpper", unicode.IsUpper)
	_register("unicode.Javanese", unicode.Javanese)
	_register("unicode.Join_Control", unicode.Join_Control)
	_register("unicode.Kaithi", unicode.Kaithi)
	_register("unicode.Kannada", unicode.Kannada)
	_register("unicode.Katakana", unicode.Katakana)
	_register("unicode.Kayah_Li", unicode.Kayah_Li)
	_register("unicode.Kharoshthi", unicode.Kharoshthi)
	_register("unicode.Khitan_Small_Script", unicode.Khitan_Small_Script)
	_register("unicode.Khmer", unicode.Khmer)
	_register("unicode.Khojki", unicode.Khojki)
	_register("unicode.Khudawadi", unicode.Khudawadi)
	_register("unicode.L", unicode.L)
	_register("unicode.Lao", unicode.Lao)
	_register("unicode.Latin", unicode.Latin)
	_register("unicode.Lepcha", unicode.Lepcha)
	_register("unicode.Letter", unicode.Letter)
	_register("unicode.Limbu", unicode.Limbu)
	_register("unicode.Linear_A", unicode.Linear_A)
	_register("unicode.Linear_B", unicode.Linear_B)
	_register("unicode.Lisu", unicode.Lisu)
	_register("unicode.Ll", unicode.Ll)
	_register("unicode.Lm", unicode.Lm)
	_register("unicode.Lo", unicode.Lo)
	_register("unicode.Logical_Order_Exception", unicode.Logical_Order_Exception)
	_register("unicode.Lower", unicode.Lower)
	_register("unicode.LowerCase", unicode.LowerCase)
	_register("unicode.Lt", unicode.Lt)
	_register("unicode.Lu", unicode.Lu)
	_register("unicode.Lycian", unicode.Lycian)
	_register("unicode.Lydian", unicode.Lydian)
	_register("unicode.M", unicode.M)
	_register("unicode.Mahajani", unicode.Mahajani)
	_register("unicode.Makasar", unicode.Makasar)
	_register("unicode.Malayalam", unicode.Malayalam)
	_register("unicode.Mandaic", unicode.Mandaic)
	_register("unicode.Manichaean", unicode.Manichaean)
	_register("unicode.Marchen", unicode.Marchen)
	_register("unicode.Mark", unicode.Mark)
	_register("unicode.Masaram_Gondi", unicode.Masaram_Gondi)
	_register("unicode.MaxASCII", unicode.MaxASCII)
	_register("unicode.MaxCase", unicode.MaxCase)
	_register("unicode.MaxLatin1", unicode.MaxLatin1)
	_register("unicode.MaxRune", unicode.MaxRune)
	_register("unicode.Mc", unicode.Mc)
	_register("unicode.Me", unicode.Me)
	_register("unicode.Medefaidrin", unicode.Medefaidrin)
	_register("unicode.Meetei_Mayek", unicode.Meetei_Mayek)
	_register("unicode.Mende_Kikakui", unicode.Mende_Kikakui)
	_register("unicode.Meroitic_Cursive", unicode.Meroitic_Cursive)
	_register("unicode.Meroitic_Hieroglyphs", unicode.Meroitic_Hieroglyphs)
	_register("unicode.Miao", unicode.Miao)
	_register("unicode.Mn", unicode.Mn)
	_register("unicode.Modi", unicode.Modi)
	_register("unicode.Mongolian", unicode.Mongolian)
	_register("unicode.Mro", unicode.Mro)
	_register("unicode.Multani", unicode.Multani)
	_register("unicode.Myanmar", unicode.Myanmar)
	_register("unicode.N", unicode.N)
	_register("unicode.Nabataean", unicode.Nabataean)
	_register("unicode.Nandinagari", unicode.Nandinagari)
	_register("unicode.Nd", unicode.Nd)
	_register("unicode.New_Tai_Lue", unicode.New_Tai_Lue)
	_register("unicode.Newa", unicode.Newa)
	_register("unicode.Nko", unicode.Nko)
	_register("unicode.Nl", unicode.Nl)
	_register("unicode.No", unicode.No)
	_register("unicode.Noncharacter_Code_Point", unicode.Noncharacter_Code_Point)
	_register("unicode.Number", unicode.Number)
	_register("unicode.Nushu", unicode.Nushu)
	_register("unicode.Nyiakeng_Puachue_Hmong", unicode.Nyiakeng_Puachue_Hmong)
	_register("unicode.Ogham", unicode.Ogham)
	_register("unicode.Ol_Chiki", unicode.Ol_Chiki)
	_register("unicode.Old_Hungarian", unicode.Old_Hungarian)
	_register("unicode.Old_Italic", unicode.Old_Italic)
	_register("unicode.Old_North_Arabian", unicode.Old_North_Arabian)
	_register("unicode.Old_Permic", unicode.Old_Permic)
	_register("unicode.Old_Persian", unicode.Old_Persian)
	_register("unicode.Old_Sogdian", unicode.Old_Sogdian)
	_register("unicode.Old_South_Arabian", unicode.Old_South_Arabian)
	_register("unicode.Old_Turkic", unicode.Old_Turkic)
	_register("unicode.Oriya", unicode.Oriya)
	_register("unicode.Osage", unicode.Osage)
	_register("unicode.Osmanya", unicode.Osmanya)
	_register("unicode.Other", unicode.Other)
	_register("unicode.Other_Alphabetic", unicode.Other_Alphabetic)
	_register("unicode.Other_Default_Ignorable_Code_Point", unicode.Other_Default_Ignorable_Code_Point)
	_register("unicode.Other_Grapheme_Extend", unicode.Other_Grapheme_Extend)
	_register("unicode.Other_ID_Continue", unicode.Other_ID_Continue)
	_register("unicode.Other_ID_Start", unicode.Other_ID_Start)
	_register("unicode.Other_Lowercase", unicode.Other_Lowercase)
	_register("unicode.Other_Math", unicode.Other_Math)
	_register("unicode.Other_Uppercase", unicode.Other_Uppercase)
	_register("unicode.P", unicode.P)
	_register("unicode.Pahawh_Hmong", unicode.Pahawh_Hmong)
	_register("unicode.Palmyrene", unicode.Palmyrene)
	_register("unicode.Pattern_Syntax", unicode.Pattern_Syntax)
	_register("unicode.Pattern_White_Space", unicode.Pattern_White_Space)
	_register("unicode.Pau_Cin_Hau", unicode.Pau_Cin_Hau)
	_register("unicode.Pc", unicode.Pc)
	_register("unicode.Pd", unicode.Pd)
	_register("unicode.Pe", unicode.Pe)
	_register("unicode.Pf", unicode.Pf)
	_register("unicode.Phags_Pa", unicode.Phags_Pa)
	_register("unicode.Phoenician", unicode.Phoenician)
	_register("unicode.Pi", unicode.Pi)
	_register("unicode.Po", unicode.Po)
	_register("unicode.Prepended_Concatenation_Mark", unicode.Prepended_Concatenation_Mark)
	_register("unicode.PrintRanges", unicode.PrintRanges)
	_register("unicode.Properties", unicode.Properties)
	_register("unicode.Ps", unicode.Ps)
	_register("unicode.Psalter_Pahlavi", unicode.Psalter_Pahlavi)
	_register("unicode.Punct", unicode.Punct)
	_register("unicode.Quotation_Mark", unicode.Quotation_Mark)
	_register("unicode.Radical", unicode.Radical)
	{
		var x unicode.Range16
		_register("unicode.Range16", reflect.TypeOf(x))
	}
	{
		var x unicode.Range32
		_register("unicode.Range32", reflect.TypeOf(x))
	}
	{
		var x unicode.RangeTable
		_register("unicode.RangeTable", reflect.TypeOf(x))
	}
	_register("unicode.Regional_Indicator", unicode.Regional_Indicator)
	_register("unicode.Rejang", unicode.Rejang)
	_register("unicode.ReplacementChar", unicode.ReplacementChar)
	_register("unicode.Runic", unicode.Runic)
	_register("unicode.S", unicode.S)
	_register("unicode.STerm", unicode.STerm)
	_register("unicode.Samaritan", unicode.Samaritan)
	_register("unicode.Saurashtra", unicode.Saurashtra)
	_register("unicode.Sc", unicode.Sc)
	_register("unicode.Scripts", unicode.Scripts)
	_register("unicode.Sentence_Terminal", unicode.Sentence_Terminal)
	_register("unicode.Sharada", unicode.Sharada)
	_register("unicode.Shavian", unicode.Shavian)
	_register("unicode.Siddham", unicode.Siddham)
	_register("unicode.SignWriting", unicode.SignWriting)
	_register("unicode.SimpleFold", unicode.SimpleFold)
	_register("unicode.Sinhala", unicode.Sinhala)
	_register("unicode.Sk", unicode.Sk)
	_register("unicode.Sm", unicode.Sm)
	_register("unicode.So", unicode.So)
	_register("unicode.Soft_Dotted", unicode.Soft_Dotted)
	_register("unicode.Sogdian", unicode.Sogdian)
	_register("unicode.Sora_Sompeng", unicode.Sora_Sompeng)
	_register("unicode.Soyombo", unicode.Soyombo)
	_register("unicode.Space", unicode.Space)
	{
		var x unicode.SpecialCase
		_register("unicode.SpecialCase", reflect.TypeOf(x))
	}
	_register("unicode.Sundanese", unicode.Sundanese)
	_register("unicode.Syloti_Nagri", unicode.Syloti_Nagri)
	_register("unicode.Symbol", unicode.Symbol)
	_register("unicode.Syriac", unicode.Syriac)
	_register("unicode.Tagalog", unicode.Tagalog)
	_register("unicode.Tagbanwa", unicode.Tagbanwa)
	_register("unicode.Tai_Le", unicode.Tai_Le)
	_register("unicode.Tai_Tham", unicode.Tai_Tham)
	_register("unicode.Tai_Viet", unicode.Tai_Viet)
	_register("unicode.Takri", unicode.Takri)
	_register("unicode.Tamil", unicode.Tamil)
	_register("unicode.Tangut", unicode.Tangut)
	_register("unicode.Telugu", unicode.Telugu)
	_register("unicode.Terminal_Punctuation", unicode.Terminal_Punctuation)
	_register("unicode.Thaana", unicode.Thaana)
	_register("unicode.Thai", unicode.Thai)
	_register("unicode.Tibetan", unicode.Tibetan)
	_register("unicode.Tifinagh", unicode.Tifinagh)
	_register("unicode.Tirhuta", unicode.Tirhuta)
	_register("unicode.Title", unicode.Title)
	_register("unicode.TitleCase", unicode.TitleCase)
	_register("unicode.To", unicode.To)
	_register("unicode.ToLower", unicode.ToLower)
	_register("unicode.ToTitle", unicode.ToTitle)
	_register("unicode.ToUpper", unicode.ToUpper)
	_register("unicode.TurkishCase", unicode.TurkishCase)
	_register("unicode.Ugaritic", unicode.Ugaritic)
	_register("unicode.Unified_Ideograph", unicode.Unified_Ideograph)
	_register("unicode.Upper", unicode.Upper)
	_register("unicode.UpperCase", unicode.UpperCase)
	_register("unicode.UpperLower", unicode.UpperLower)
	_register("unicode.Vai", unicode.Vai)
	_register("unicode.Variation_Selector", unicode.Variation_Selector)
	_register("unicode.Version", unicode.Version)
	_register("unicode.Wancho", unicode.Wancho)
	_register("unicode.Warang_Citi", unicode.Warang_Citi)
	_register("unicode.White_Space", unicode.White_Space)
	_register("unicode.Yezidi", unicode.Yezidi)
	_register("unicode.Yi", unicode.Yi)
	_register("unicode.Z", unicode.Z)
	_register("unicode.Zanabazar_Square", unicode.Zanabazar_Square)
	_register("unicode.Zl", unicode.Zl)
	_register("unicode.Zp", unicode.Zp)
	_register("unicode.Zs", unicode.Zs)
}
