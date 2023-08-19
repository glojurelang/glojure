package deps

import (
	"bufio"
	"fmt"
	"os"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
	"github.com/glojurelang/glojure/pkg/reader"
)

func Load() (*Deps, error) {
	// look for a deps.edn file in the current directory if it exists,
	// read it and parse it into a Deps struct.

	// if it doesn't exist, return nil, nil

	const filename = "deps.edn"

	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
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
		Deps: make(map[string]map[string]string),
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
