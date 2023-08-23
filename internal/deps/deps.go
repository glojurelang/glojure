package deps

import "fmt"

// Deps is a struct that contains all the dependencies for the
// Glojure application.
type Deps struct {
	goModName string
	goModDir  string

	Pkgs []string
	Deps map[string]map[string]string
}

func (d *Deps) Gen() error {
	if err := d.Get(); err != nil {
		return fmt.Errorf("failed to fetch dependencies: %w", err)
	}
	if err := d.Embed(); err != nil {
		return fmt.Errorf("failed to generate embed for packages: %w", err)
	}
	if err := d.GLJ(); err != nil {
		return fmt.Errorf("failed to generate glj main: %w", err)
	}

	return nil
}
