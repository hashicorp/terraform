// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//
// System calls for amd64, Solaris are implemented in runtime/syscall_solaris.goc
//

TEXT ·sysvicall6(SB), 7, $0-64
	JMP	syscall·sysvicall6(SB)

TEXT ·rawSysvicall6(SB), 7, $0-64
	JMP	syscall·rawSysvicall6(SB)

TEXT ·dlopen(SB), 7, $0-16
	JMP	syscall·dlopen(SB)

TEXT ·dlclose(SB), 7, $0-8
	JMP	syscall·dlclose(SB)

TEXT ·dlsym(SB), 7, $0-16
	JMP	syscall·dlsym(SB)
