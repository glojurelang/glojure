package deps

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
	"github.com/glojurelang/glojure/pkg/reader"
)

func Load() (*Deps, error) {
	const filename = "./gljdeps.edn"

	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	modRootDir, err := goModRootDir()
	if err != nil {
		return nil, fmt.Errorf("failed to find go mod dir: %w", err)
	}

	modName, err := goModName()
	if err != nil {
		return nil, fmt.Errorf("failed to find go mod name: %w", err)
	}

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rdr := reader.New(bufio.NewReader(f))
	d, err := rdr.ReadOne()
	if err != nil {
		return nil, err
	}

	dmap, ok := d.(lang.IPersistentMap)
	if !ok {
		return nil, fmt.Errorf("expected map, got %T", d)
	}

	deps := &Deps{
		Deps:      make(map[string]map[string]string),
		goModName: modName,
		goModDir:  modRootDir,
	}
	{ // paths
		for s := lang.Seq(dmap.ValAt(lang.NewKeyword("paths"))); s != nil; s = s.Next() {
			first := s.First()
			str, ok := first.(string)
			if !ok {
				return nil, fmt.Errorf("expected strings in :paths vector, got %T", first)
			}
			deps.Pkgs = append(deps.Pkgs, str)
		}
	}

	{ // deps
		var depsMap lang.IPersistentMap
		if dm := dmap.ValAt(lang.NewKeyword("deps")); dm != nil {
			if dm, ok := dm.(lang.IPersistentMap); !ok {
				return nil, fmt.Errorf("expected map for :deps, got %T", dm)
			} else {
				depsMap = dm
			}
		}
		for s := lang.Seq(depsMap); s != nil; s = s.Next() {
			entry := s.First().(lang.IMapEntry)
			k := entry.Key()
			sym, ok := k.(*lang.Symbol)
			if !ok {
				return nil, fmt.Errorf("expected symbol for :deps key, got %T", k)
			}

			v := entry.Val()
			valMap, ok := v.(lang.IPersistentMap)
			if !ok {
				return nil, fmt.Errorf("expected map for :deps value, got %T", v)
			}

			depMap := make(map[string]string)
			for s := lang.Seq(valMap); s != nil; s = s.Next() {
				entry := s.First().(lang.IMapEntry)
				k := entry.Key()
				kw, ok := k.(lang.Keyword)
				if !ok {
					return nil, fmt.Errorf("expected keyword for :deps key, got %T", k)
				}

				v := entry.Val()
				str, ok := v.(string)
				if !ok {
					return nil, fmt.Errorf("expected string for :deps value, got %T", v)
				}
				depMap[kw.Name()] = str
			}
			deps.Deps[pkgmap.UnmungePkg(sym.FullName())] = depMap
		}
	}

	return deps, nil
}

func goModRootDir() (string, error) {
	out, err := exec.Command("go", "env", "GOMOD").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w\n%s", err, string(out))
	}
	return filepath.Dir(string(out)), nil
}

func goModName() (string, error) {
	out, err := exec.Command("go", "list", "-m").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w\n%s", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}
