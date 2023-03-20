package value

var (
	KWTag       = NewKeyword("tag")
	KWFile      = NewKeyword("file")
	KWLine      = NewKeyword("line")
	KWColumn    = NewKeyword("column")
	KWEndLine   = NewKeyword("end-line")
	KWEndColumn = NewKeyword("end-column")

	KWMethods       = NewKeyword("methods")
	KWIsVariadic    = NewKeyword("variadic?")
	KWMaxFixedArity = NewKeyword("max-fixed-arity")
	KWLocal         = NewKeyword("local")
	KWName          = NewKeyword("name")
	KWFixedArity    = NewKeyword("fixed-arity")
	KWBody          = NewKeyword("body")
	KWParams        = NewKeyword("params")

	KWMacro   = NewKeyword("macro")
	KWPrivate = NewKeyword("private")
	KWDynamic = NewKeyword("dynamic")
	KWNS      = NewKeyword("ns")
)
