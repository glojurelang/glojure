// Package filesystem exposes the core java.io.File and java.nio.file
// pathname surface over Go's portable os/path/filepath APIs.
package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

type File struct{ Pathname string }
type Path struct{ Pathname string }
type Files struct{}
type Paths struct{}

func NewFile(args ...any) any {
	switch len(args) {
	case 1:
		return &File{Pathname: pathname(args[0])}
	case 2:
		return &File{Pathname: filepath.Join(pathname(args[0]), pathname(args[1]))}
	default:
		panic(fmt.Sprintf("File/new: wrong number of args (%d)", len(args)))
	}
}

func CreateTempFile(prefix, suffix any) *File {
	pattern := pathname(prefix) + "*"
	if suffix != nil {
		pattern += pathname(suffix)
	}
	file, err := os.CreateTemp("", pattern)
	if err != nil {
		panic(err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		panic(err)
	}
	return &File{Pathname: path}
}

func (f *File) GetPath() string         { return f.Pathname }
func (f *File) GetName() string         { return filepath.Base(f.Pathname) }
func (f *File) GetParent() string       { return filepath.Dir(f.Pathname) }
func (f *File) GetParentFile() *File    { return &File{Pathname: filepath.Dir(f.Pathname)} }
func (f *File) GetAbsolutePath() string { p, _ := filepath.Abs(f.Pathname); return p }
func (f *File) Path() string            { return f.GetPath() }
func (f *File) Name() string            { return f.GetName() }
func (f *File) Parent() string          { return f.GetParent() }
func (f *File) ParentFile() *File       { return f.GetParentFile() }
func (f *File) AbsolutePath() string    { return f.GetAbsolutePath() }
func (f *File) GetCanonicalPath() string {
	p, err := filepath.Abs(f.Pathname)
	if err != nil {
		panic(err)
	}
	return filepath.Clean(p)
}
func (f *File) CanonicalPath() string  { return f.GetCanonicalPath() }
func (f *File) GetAbsoluteFile() *File { return &File{Pathname: f.GetAbsolutePath()} }
func (f *File) GetCanonicalFile() *File {
	return &File{Pathname: f.GetCanonicalPath()}
}
func (f *File) AbsoluteFile() *File  { return f.GetAbsoluteFile() }
func (f *File) CanonicalFile() *File { return f.GetCanonicalFile() }
func (f *File) Exists() bool         { _, err := os.Stat(f.Pathname); return err == nil }
func (f *File) IsFile() bool {
	info, err := os.Stat(f.Pathname)
	return err == nil && info.Mode().IsRegular()
}
func (f *File) IsDirectory() bool {
	info, err := os.Stat(f.Pathname)
	return err == nil && info.IsDir()
}
func (f *File) CanRead() bool {
	file, err := os.Open(f.Pathname)
	if err != nil {
		return false
	}
	file.Close()
	return true
}
func (f *File) CanWrite() bool {
	file, err := os.OpenFile(f.Pathname, os.O_WRONLY, 0)
	if err != nil {
		return false
	}
	file.Close()
	return true
}
func (f *File) Length() int64 {
	info, err := os.Stat(f.Pathname)
	if err != nil {
		return 0
	}
	return info.Size()
}
func (f *File) LastModified() int64 {
	info, err := os.Stat(f.Pathname)
	if err != nil {
		return 0
	}
	return info.ModTime().UnixMilli()
}
func (f *File) Mkdir() bool  { return os.Mkdir(f.Pathname, 0o777) == nil }
func (f *File) Mkdirs() bool { return os.MkdirAll(f.Pathname, 0o777) == nil }
func (f *File) Delete() bool { return os.Remove(f.Pathname) == nil }
func (f *File) List() []string {
	entries, err := os.ReadDir(f.Pathname)
	if err != nil {
		return nil
	}
	out := make([]string, len(entries))
	for i, entry := range entries {
		out[i] = entry.Name()
	}
	return out
}
func (f *File) ListFiles() []*File {
	entries, err := os.ReadDir(f.Pathname)
	if err != nil {
		return nil
	}
	out := make([]*File, len(entries))
	for i, entry := range entries {
		out[i] = &File{Pathname: filepath.Join(f.Pathname, entry.Name())}
	}
	return out
}
func (f *File) ToPath() *Path    { return &Path{Pathname: f.Pathname} }
func (f *File) ToString() string { return f.Pathname }
func (f *File) String() string   { return f.Pathname }

func GetPath(args ...any) any {
	if len(args) == 0 {
		panic("Paths/get: missing path")
	}
	parts := []string{pathname(args[0])}
	for _, part := range args[1:] {
		switch value := part.(type) {
		case []string:
			parts = append(parts, value...)
		case lang.IPersistentVector:
			for i := 0; i < value.Count(); i++ {
				parts = append(parts, pathname(value.Nth(i)))
			}
		default:
			parts = append(parts, pathname(value))
		}
	}
	return &Path{Pathname: filepath.Join(parts...)}
}

func (p *Path) ToString() string   { return p.Pathname }
func (p *Path) String() string     { return p.Pathname }
func (p *Path) ToFile() *File      { return &File{Pathname: p.Pathname} }
func (p *Path) IsAbsolute() bool   { return filepath.IsAbs(p.Pathname) }
func (p *Path) GetFileName() *Path { return &Path{Pathname: filepath.Base(p.Pathname)} }
func (p *Path) GetParent() *Path   { return &Path{Pathname: filepath.Dir(p.Pathname)} }
func (p *Path) FileName() *Path    { return p.GetFileName() }
func (p *Path) Parent() *Path      { return p.GetParent() }
func (p *Path) Normalize() *Path   { return &Path{Pathname: filepath.Clean(p.Pathname)} }
func (p *Path) ToAbsolutePath() *Path {
	value, _ := filepath.Abs(p.Pathname)
	return &Path{Pathname: value}
}
func (p *Path) Resolve(other any) *Path {
	value := pathname(other)
	if filepath.IsAbs(value) {
		return &Path{Pathname: value}
	}
	return &Path{Pathname: filepath.Join(p.Pathname, value)}
}

func Exists(path any, _ ...any) bool { _, err := os.Stat(pathname(path)); return err == nil }
func IsRegularFile(path any, _ ...any) bool {
	info, err := os.Stat(pathname(path))
	return err == nil && info.Mode().IsRegular()
}
func IsDirectory(path any, _ ...any) bool {
	info, err := os.Stat(pathname(path))
	return err == nil && info.IsDir()
}
func CreateDirectories(path any, _ ...any) any {
	if err := os.MkdirAll(pathname(path), 0o777); err != nil {
		panic(err)
	}
	return path
}
func ReadString(path any, _ ...any) string {
	data, err := os.ReadFile(pathname(path))
	if err != nil {
		panic(err)
	}
	return string(data)
}
func WriteString(path, content any, _ ...any) any {
	if err := os.WriteFile(pathname(path), []byte(lang.ToString(content)), 0o666); err != nil {
		panic(err)
	}
	return path
}
func Delete(path any) {
	if err := os.Remove(pathname(path)); err != nil {
		panic(err)
	}
}
func Size(path any) int64 {
	info, err := os.Stat(pathname(path))
	if err != nil {
		panic(err)
	}
	return info.Size()
}
func GetLastModifiedTime(path any) time.Time {
	info, err := os.Stat(pathname(path))
	if err != nil {
		panic(err)
	}
	return info.ModTime()
}

func pathname(value any) string {
	switch value := value.(type) {
	case *File:
		return value.Pathname
	case *Path:
		return value.Pathname
	case string:
		return value
	default:
		return lang.ToString(value)
	}
}

func registerStatic(class, javaPackage, javaName, goName string, value any) {
	pkgmap.Set(class+"."+javaName, value)
	pkgmap.Set(javaPackage+"."+class+"."+javaName, value)
	pkgmap.SetHostClassPackage(class, javaPackage)
	_ = goName
}

func init() {
	pkgmap.SetHostClassPackage("File", "java.io")
	pkgmap.SetHostClass("File", lang.NewClass(reflect.TypeOf((*File)(nil)), "java.io.File"))
	lang.RegisterHostConstructor("java.io.File",
		lang.FnFunc(func(args ...any) any { return NewFile(args...) }))
	registerStatic("File", "java.io", "createTempFile", "CreateTempFile",
		lang.FnFunc(func(args ...any) any {
			if len(args) != 2 {
				panic(fmt.Sprintf("File/createTempFile: wrong number of args (%d)", len(args)))
			}
			return CreateTempFile(args[0], args[1])
		}))

	pkgmap.SetHostClassPackage("Path", "java.nio.file")
	pkgmap.SetHostClass("Path", lang.NewClass(reflect.TypeOf((*Path)(nil)), "java.nio.file.Path"))

	pkgmap.SetHostClassPackage("Paths", "java.nio.file")
	pkgmap.SetHostClass("Paths", lang.NewClass(reflect.TypeOf(Paths{}), "java.nio.file.Paths"))
	registerStatic("Paths", "java.nio.file", "get", "Get", lang.FnFunc(func(args ...any) any { return GetPath(args...) }))

	pkgmap.SetHostClassPackage("Files", "java.nio.file")
	pkgmap.SetHostClass("Files", lang.NewClass(reflect.TypeOf(Files{}), "java.nio.file.Files"))
	registerStatic("Files", "java.nio.file", "exists", "Exists", lang.FnFunc(func(args ...any) any { return Exists(args[0], args[1:]...) }))
	registerStatic("Files", "java.nio.file", "isRegularFile", "IsRegularFile", lang.FnFunc(func(args ...any) any { return IsRegularFile(args[0], args[1:]...) }))
	registerStatic("Files", "java.nio.file", "isDirectory", "IsDirectory", lang.FnFunc(func(args ...any) any { return IsDirectory(args[0], args[1:]...) }))
	registerStatic("Files", "java.nio.file", "createDirectories", "CreateDirectories", lang.FnFunc(func(args ...any) any { return CreateDirectories(args[0], args[1:]...) }))
	registerStatic("Files", "java.nio.file", "readString", "ReadString", lang.FnFunc(func(args ...any) any { return ReadString(args[0], args[1:]...) }))
	registerStatic("Files", "java.nio.file", "writeString", "WriteString", lang.FnFunc(func(args ...any) any { return WriteString(args[0], args[1], args[2:]...) }))
	registerStatic("Files", "java.nio.file", "delete", "Delete", lang.FnFunc(func(args ...any) any { Delete(args[0]); return nil }))
	registerStatic("Files", "java.nio.file", "size", "Size", lang.FnFunc(func(args ...any) any { return Size(args[0]) }))

	pkgmap.Set("File.separator", string(os.PathSeparator))
	pkgmap.Set("java.io.File.separator", string(os.PathSeparator))
	pkgmap.Set("File.separatorChar", lang.NewChar(rune(os.PathSeparator)))
	pkgmap.Set("java.io.File.separatorChar", lang.NewChar(rune(os.PathSeparator)))
	pkgmap.Set("File.pathSeparator", string(os.PathListSeparator))
	pkgmap.Set("java.io.File.pathSeparator", string(os.PathListSeparator))
	pkgmap.Set("File.pathSeparatorChar", lang.NewChar(rune(os.PathListSeparator)))
	pkgmap.Set("java.io.File.pathSeparatorChar", lang.NewChar(rune(os.PathListSeparator)))
}
