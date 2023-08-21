package deps

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Generates an gljembed.go file that embeds the directories from
// deps.edn's pkgs and adds them to the Glojure load path when
// imported.
func (d *Deps) Embed() error {
	sort.Strings(d.Pkgs)

	modRootDir, err := goModRootDir()
	if err != nil {
		return err
	}

	modName, err := goModName()
	if err != nil {
		return err
	}

	modLastPart := filepath.Base(modName)

	f, err := os.Create(filepath.Join(modRootDir, "gljembed.go"))
	if err != nil {
		return err
	}
	defer f.Close()

	b := bytes.NewBuffer(nil)

	fmt.Fprintf(b, "// Code generated by glj. DO NOT EDIT.\n\n")
	fmt.Fprintf(b, "package %s\n\n", modLastPart)

	fmt.Fprintf(b, `import (
	"embed"
	"io/fs"

	"github.com/glojurelang/glojure/pkg/runtime"
)

`)

	fmt.Fprintf(b, "var (\n")
	for _, pkg := range d.Pkgs {
		fsName := mungePath(pkg) + "FS"
		fmt.Fprintf(b, "\t//go:embed %s/*\n", pkg)
		fmt.Fprintf(b, "\t%s embed.FS\n", fsName)
	}
	fmt.Fprintf(b, ")\n\n")

	fmt.Fprintf(b, `func subfs(efs embed.FS, dir string) fs.FS {
	d, err := fs.Sub(efs, dir)
	if err != nil {
		panic(err)
	}
	return d
}

`)

	fmt.Fprintf(b, "func init() {\n")
	for _, pkg := range d.Pkgs {
		fsName := mungePath(pkg) + "FS"
		fmt.Fprintf(b, "\truntime.AddLoadPath(subfs(%s, %q))\n", fsName, pkg)
	}
	fmt.Fprintf(b, "}\n")

	src, err := format.Source(b.Bytes())
	if err != nil {
		return err
	}
	f.Write(src)

	return nil
}

func mungePath(path string) string {
	return strings.Replace(path, "/", "__", -1)
}

func goModRootDir() (string, error) {
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return "", err
	}
	return filepath.Dir(string(out)), nil
}

func goModName() (string, error) {
	out, err := exec.Command("go", "list", "-m").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
