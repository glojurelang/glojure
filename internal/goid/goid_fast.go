//go:build amd64 || arm64

package goid

import (
	"reflect"
	"unsafe"
)

type emptyInterface struct {
	typ  unsafe.Pointer
	data unsafe.Pointer
}

var goidOffset = findGoidOffset()

func findGoidOffset() uintptr {
	sections, offsets := reflectTypeLinks()
	var holder any = reflect.TypeOf(0)
	for sectionIndex, sectionOffsets := range offsets {
		section := sections[sectionIndex]
		for _, offset := range sectionOffsets {
			(*emptyInterface)(unsafe.Pointer(&holder)).data = reflectResolveTypeOff(section, offset)
			candidate := holder.(reflect.Type)
			if candidate.Kind() != reflect.Pointer || candidate.Elem().Kind() != reflect.Struct {
				continue
			}
			candidate = candidate.Elem()
			if candidate.PkgPath() != "runtime" || candidate.Name() != "g" {
				continue
			}
			field, ok := candidate.FieldByName("goid")
			if !ok {
				panic("failed to find runtime.g.goid field")
			}
			return field.Offset
		}
	}
	panic("failed to find runtime.g type")
}

// Get returns the runtime's monotonic identity for the current goroutine.
//
//go:nocheckptr
func Get() int64 {
	g := getg()
	return *(*int64)(unsafe.Pointer(g + goidOffset))
}

func getg() uintptr

//go:linkname reflectTypeLinks reflect.typelinks
func reflectTypeLinks() (sections []unsafe.Pointer, offsets [][]int32)

//go:linkname reflectResolveTypeOff reflect.resolveTypeOff
func reflectResolveTypeOff(section unsafe.Pointer, offset int32) unsafe.Pointer
