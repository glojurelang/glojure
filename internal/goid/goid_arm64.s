// Copyright 2016 Huan Du. All rights reserved.
// Use of this source code is governed by a MIT license.

#include "textflag.h"

TEXT ·getg(SB), NOSPLIT, $0-8
    MOVD    g, R8
    MOVD    R8, ret+0(FP)
    RET
