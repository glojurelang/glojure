// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.

#ifdef GOARCH_amd64
#define	get_tls(r)	MOVQ TLS, r
#define	g(r)	0(r)(TLS*1)
#endif
