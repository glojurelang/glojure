package lang

type (
	AFn interface {
		IFn
	}
)

func afnApplyTo(a AFn, args ISeq) any {
	slc := seqToSlice(args)
	return a.Invoke(slc...)
}
