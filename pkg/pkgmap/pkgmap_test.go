package pkgmap

import "testing"

func resetImportRegistryForTest(t *testing.T) {
	t.Helper()

	loaderMtx.Lock()
	oldLoaders := importsLoaders
	oldLoaded := importsLoaded
	importsLoaders = nil
	importsLoaded = false
	loaderMtx.Unlock()

	mtx.Lock()
	oldPkgMap := pkgMap
	pkgMap = map[string]interface{}{}
	mtx.Unlock()

	t.Cleanup(func() {
		loaderMtx.Lock()
		importsLoaders = oldLoaders
		importsLoaded = oldLoaded
		loaderMtx.Unlock()

		mtx.Lock()
		pkgMap = oldPkgMap
		mtx.Unlock()
	})
}

func TestMultipleImportLoadersAreLoadedOnFirstLookup(t *testing.T) {
	resetImportRegistryForTest(t)

	SetLoader(func(register func(string, interface{})) {
		register("example/one.Value", 1)
	})
	SetLoader(func(register func(string, interface{})) {
		register("example/two.Value", 2)
	})

	if got, ok := Get("example/one.Value"); !ok || got != 1 {
		t.Fatalf("first loader value = %v, %v; want 1, true", got, ok)
	}
	if got, ok := Get("example/two.Value"); !ok || got != 2 {
		t.Fatalf("second loader value = %v, %v; want 2, true", got, ok)
	}
}

func TestImportLoaderInstalledAfterLookupLoadsImmediately(t *testing.T) {
	resetImportRegistryForTest(t)

	SetLoader(func(register func(string, interface{})) {
		register("example/initial.Value", 1)
	})
	Get("example/initial.Value")

	SetLoader(func(register func(string, interface{})) {
		register("example/late.Value", 3)
	})

	if got, ok := Get("example/late.Value"); !ok || got != 3 {
		t.Fatalf("late loader value = %v, %v; want 3, true", got, ok)
	}
}
