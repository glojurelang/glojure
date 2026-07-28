package runtime

import "github.com/glojurelang/glojure/pkg/lang"

// OwnedLoopMap is a copy-on-write map value used for loop-local update
// regions. It keeps arbitrary initial values on the persistent path until the
// first effective map update, then uses a transient map for subsequent work.
type OwnedLoopMap struct {
	current   any
	transient *lang.TransientMap
	meta      lang.IPersistentMap
}

func NewOwnedLoopMap(value any) *OwnedLoopMap {
	return &OwnedLoopMap{current: value}
}

func (m *OwnedLoopMap) Assoc(key, value any) *OwnedLoopMap {
	if m.transient != nil {
		m.transient.Assoc(key, value)
		return m
	}
	previous := m.current
	next := lang.Assoc(m.current, key, value)
	m.current = next
	if lang.Identical(previous, next) {
		return m
	}
	m.startTransient(next)
	return m
}

func (m *OwnedLoopMap) ValAt(key any) any {
	return m.ValAtDefault(key, nil)
}

func (m *OwnedLoopMap) ValAtDefault(key, fallback any) any {
	if m.transient != nil {
		return m.transient.ValAtDefault(key, fallback)
	}
	return lang.GetDefault(m.current, key, fallback)
}

func (m *OwnedLoopMap) Persistent() any {
	if m.transient == nil {
		return m.current
	}
	result := m.transient.Persistent()
	m.transient = nil
	if m.meta != nil {
		result = result.(lang.IObj).WithMeta(m.meta).(lang.IPersistentCollection)
	}
	m.current = result
	return result
}

func (m *OwnedLoopMap) startTransient(value any) {
	if _, ok := value.(lang.IPersistentMap); !ok {
		return
	}
	editable, ok := value.(lang.IEditableCollection)
	if !ok {
		return
	}
	transient, ok := editable.AsTransient().(*lang.TransientMap)
	if !ok {
		return
	}
	if withMeta, ok := value.(lang.IMeta); ok {
		m.meta = withMeta.Meta()
	}
	m.transient = transient
}
