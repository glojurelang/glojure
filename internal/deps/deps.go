package deps

// Deps is a struct that contains all the dependencies for the
// Glojure application.
type Deps struct {
	Pkgs []string
	Deps map[string]map[string]string
}
