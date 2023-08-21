package deps

import "fmt"

// Deps is a struct that contains all the dependencies for the
// Glojure application.
type Deps struct {
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

	return nil
}
