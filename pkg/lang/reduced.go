package lang

import "github.com/glojurelang/glojure/pkg/pkgmap"

type Reduced struct {
	val interface{}
}

var (
	_ IDeref = (*Reduced)(nil)
)

func IsReduced(v interface{}) bool {
	_, ok := v.(*Reduced)
	return ok
}

func NewReduced(v interface{}) *Reduced {
	return &Reduced{val: v}
}

func (r *Reduced) Deref() interface{} {
	return r.val
}

func init() {
	pkgmap.Set("github.com/glojurelang/glojure/pkg/lang.IsReduced", IsReduced)
	pkgmap.Set("github.com/glojurelang/glojure/pkg/lang.NewReduced", NewReduced)
}
