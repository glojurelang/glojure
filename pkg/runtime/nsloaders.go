package runtime

var (
	// nsLoaders is a map of namespace resource names to their loader
	// functions. Used for pre-compiled namespaces.
	nsLoaders = map[string]func(){}
)

// RegisterNSLoader registers a loader function for a namespace given its resource name
// (i.e. root path with slashes, no extension).
func RegisterNSLoader(nsResource string, loader func()) {
	if _, exists := nsLoaders[nsResource]; exists {
		panic("namespace loader already registered for " + nsResource)
	}
	nsLoaders[nsResource] = loader
}

// GetNSLoader retrieves the loader function for a namespace given its resource name.
func GetNSLoader(nsResource string) func() {
	return nsLoaders[nsResource]
}
