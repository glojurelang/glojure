package runtime

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/glojurelang/glojure/pkg/lang"
)

var grenadineHTTPClient = &http.Client{Timeout: 45 * time.Second}

func installGrenadineHost() {
	ns := lang.FindOrCreateNamespace(lang.NewSymbol("glojure.deps.host"))
	lang.InternVar(ns, lang.NewSymbol("host"), lang.FnFunc0(func() any {
		return grenadineHost()
	}), true)
}

func grenadineHost() lang.IPersistentMap {
	return lang.NewMap(
		lang.NewKeyword("http-get"), lang.FnFunc1(func(rawURL any) any {
			return grenadineHTTPGet(rawURL.(string))
		}),
		lang.NewKeyword("read-bytes"), lang.FnFunc1(func(path any) any {
			data, err := os.ReadFile(path.(string))
			if err != nil {
				panic(err)
			}
			return data
		}),
		lang.NewKeyword("write-bytes!"), lang.FnFunc2(func(path, data any) any {
			if err := os.WriteFile(path.(string), data.([]byte), 0o644); err != nil {
				panic(err)
			}
			return nil
		}),
		lang.NewKeyword("bytes->utf8"), lang.FnFunc1(func(data any) any {
			return string(data.([]byte))
		}),
		lang.NewKeyword("utf8->bytes"), lang.FnFunc1(func(text any) any {
			return []byte(text.(string))
		}),
		lang.NewKeyword("digest"), lang.FnFunc2(func(algorithm, data any) any {
			return grenadineDigest(algorithm.(lang.Keyword), data.([]byte))
		}),
		lang.NewKeyword("byte-count"), lang.FnFunc1(func(data any) any {
			return int64(len(data.([]byte)))
		}),
		lang.NewKeyword("exists?"), lang.FnFunc1(func(path any) any {
			_, err := os.Stat(path.(string))
			return err == nil
		}),
		lang.NewKeyword("directory?"), lang.FnFunc1(func(path any) any {
			info, err := os.Stat(path.(string))
			return err == nil && info.IsDir()
		}),
		lang.NewKeyword("regular-file?"), lang.FnFunc1(func(path any) any {
			info, err := os.Stat(path.(string))
			return err == nil && info.Mode().IsRegular()
		}),
		lang.NewKeyword("canonical-path"), lang.FnFunc1(func(path any) any {
			return grenadineCanonicalPath(path.(string))
		}),
		lang.NewKeyword("absolute-path"), lang.FnFunc1(func(path any) any {
			return grenadineCanonicalPath(path.(string))
		}),
		lang.NewKeyword("find-files"), lang.FnFunc2(func(root, predicate any) any {
			return grenadineFindFiles(root.(string), predicate.(lang.IFn))
		}),
		lang.NewKeyword("run-process"), lang.FnFunc1(func(options any) any {
			return grenadineRunProcess(options.(lang.IPersistentMap))
		}),
		lang.NewKeyword("mkdirs!"), lang.FnFunc1(func(path any) any {
			if err := os.MkdirAll(path.(string), 0o755); err != nil {
				panic(err)
			}
			return nil
		}),
		lang.NewKeyword("atomic-move!"), lang.FnFunc2(func(source, destination any) any {
			if err := grenadineAtomicMove(source.(string), destination.(string)); err != nil {
				panic(err)
			}
			return nil
		}),
		lang.NewKeyword("delete!"), lang.FnFunc1(func(path any) any {
			if err := os.RemoveAll(path.(string)); err != nil {
				panic(err)
			}
			return nil
		}),
		lang.NewKeyword("delete-tree!"), lang.FnFunc1(func(path any) any {
			if err := os.RemoveAll(path.(string)); err != nil {
				panic(err)
			}
			return nil
		}),
		lang.NewKeyword("extract-jar!"), lang.FnFunc2(func(jar, destination any) any {
			if err := grenadineExtractJar(jar.(string), destination.(string)); err != nil {
				panic(err)
			}
			return destination
		}),
		lang.NewKeyword("home-dir"), lang.FnFunc0(func() any {
			home, err := os.UserHomeDir()
			if err != nil {
				panic(err)
			}
			return home
		}),
		lang.NewKeyword("getenv"), lang.FnFunc1(func(name any) any {
			return os.Getenv(name.(string))
		}),
	)
}

func grenadineCanonicalPath(path string) string {
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		real = path
	}
	absolute, err := filepath.Abs(real)
	if err != nil {
		panic(err)
	}
	return filepath.Clean(absolute)
}

func grenadineFindFiles(root string, predicate lang.IFn) any {
	paths := []any{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			canonical := grenadineCanonicalPath(path)
			if RT.BooleanCast(predicate.Invoke(canonical)) {
				paths = append(paths, canonical)
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return lang.NewVector(paths...)
}

func grenadineStrings(value any) []string {
	values := []string{}
	for seq := lang.Seq(value); seq != nil; seq = seq.Next() {
		values = append(values, seq.First().(string))
	}
	return values
}

func grenadineRunProcess(options lang.IPersistentMap) lang.IPersistentMap {
	args := grenadineStrings(lang.Get(options, lang.NewKeyword("args")))
	if len(args) == 0 {
		panic(fmt.Errorf("grenadine: run-process requires :args"))
	}
	command := exec.Command(args[0], args[1:]...)
	if dir := lang.Get(options, lang.NewKeyword("dir")); !lang.IsNil(dir) {
		command.Dir = dir.(string)
	}
	command.Env = os.Environ()
	if environment := lang.Get(options, lang.NewKeyword("env")); !lang.IsNil(environment) {
		for seq := lang.Seq(environment); seq != nil; seq = seq.Next() {
			entry := seq.First().(*lang.MapEntry)
			command.Env = append(command.Env,
				fmt.Sprintf("%s=%s", entry.Key(), entry.Val()))
		}
	}
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	exit := int64(0)
	if err != nil {
		exit = 1
		if exitError, ok := err.(*exec.ExitError); ok {
			exit = int64(exitError.ExitCode())
		} else if stderr.Len() == 0 {
			stderr.WriteString(err.Error())
		}
	}
	return lang.NewMap(
		lang.NewKeyword("exit"), exit,
		lang.NewKeyword("out"), stdout.String(),
		lang.NewKeyword("err"), stderr.String(),
	)
}

func grenadineHTTPGet(rawURL string) lang.IPersistentMap {
	request, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return grenadineHTTPResponse(0, nil)
	}
	request.Header.Set("User-Agent", "glojure-grenadine/"+Version)
	response, err := grenadineHTTPClient.Do(request)
	if err != nil {
		return grenadineHTTPResponse(0, nil)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return grenadineHTTPResponse(0, nil)
	}
	return grenadineHTTPResponse(response.StatusCode, body)
}

func grenadineHTTPResponse(status int, body []byte) lang.IPersistentMap {
	return lang.NewMap(
		lang.NewKeyword("status"), int64(status),
		lang.NewKeyword("headers"), lang.NewMap(),
		lang.NewKeyword("body"), body,
	)
}

func grenadineDigest(algorithm lang.Keyword, data []byte) string {
	switch algorithm.Name() {
	case "sha1":
		sum := sha1.Sum(data)
		return hex.EncodeToString(sum[:])
	case "sha256":
		sum := sha256.Sum256(data)
		return hex.EncodeToString(sum[:])
	default:
		panic(fmt.Errorf("grenadine: unsupported digest algorithm %s", algorithm))
	}
}

func grenadineAtomicMove(source, destination string) error {
	if err := os.Remove(destination); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(source, destination)
}

func grenadineExtractJar(jar, destination string) (err error) {
	marker := filepath.Join(destination, ".grenadine-complete")
	if _, err := os.Stat(marker); err == nil {
		return nil
	}

	temporary := destination + ".part"
	if err := os.RemoveAll(temporary); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(temporary)
		}
	}()
	if err := os.MkdirAll(temporary, 0o755); err != nil {
		return err
	}

	archive, err := zip.OpenReader(jar)
	if err != nil {
		return err
	}
	defer archive.Close()

	for _, entry := range archive.File {
		if err := grenadineExtractEntry(entry, temporary); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(temporary, ".grenadine-complete"), nil, 0o644); err != nil {
		return err
	}
	if err := os.RemoveAll(destination); err != nil {
		return err
	}
	return os.Rename(temporary, destination)
}

func grenadineExtractEntry(entry *zip.File, root string) error {
	if entry.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("grenadine: refusing symlink in jar: %s", entry.Name)
	}
	name := filepath.Clean(filepath.FromSlash(entry.Name))
	if name == "." {
		return nil
	}
	if filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
		return fmt.Errorf("grenadine: refusing path outside jar root: %s", entry.Name)
	}
	target := filepath.Join(root, name)
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("grenadine: refusing path outside jar root: %s", entry.Name)
	}
	if entry.FileInfo().IsDir() {
		return os.MkdirAll(target, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	source, err := entry.Open()
	if err != nil {
		return err
	}
	defer source.Close()
	destination, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(destination, source)
	closeErr := destination.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
