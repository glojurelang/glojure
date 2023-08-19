package deps

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/glojurelang/glojure/internal/genpkg"
)

// Get gets the dependencies for the given package and generates a
// package map for any go dependencies.
func (d *Deps) Get() error {
	packages := make([]string, 0, len(d.Deps))

	for dep, depMap := range d.Deps {
		version, ok := depMap["version"]
		if !ok {
			version = "latest"
		}

		if err := getDep(dep, version); err != nil {
			return err
		}

		packages = append(packages, dep)
	}

	if err := os.MkdirAll("./glj/gljimports", 0755); err != nil {
		return err
	}

	f, err := os.Create("./glj/gljimports/gljimports.go")
	if err != nil {
		return err
	}

	genpkg.GenPkgs(packages, genpkg.WithWriter(f))

	return nil
}

func getDep(dep, version string) error {
	out, err := exec.Command("go", "list", "-f", "{{.Module}}", dep).Output()
	if err == nil {
		// do we already have the right version?
		parts := strings.Split(string(out), " ")
		if len(parts) == 2 {
			curVersion := strings.TrimSpace(parts[1])
			if curVersion == version {
				// already installed with desired version
				return nil
			}
		}
	}

	// go get <dep>@<version>
	cmd := exec.Command("go", "get", dep+"@"+version)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to `go get %s@%s`: %w\n%s", dep, version, err, out)
	}
	return nil
}
