//go:build go1.19 && !go1.20
// +build go1.19,!go1.20

// MIT license, copy and modify from https://github.com/tlog-dev/loc

//nolint:unused
package log

import (
	"unsafe"
)

// inlinedCall is the encoding of entries in the FUNCDATA_InlTree table.
type inlinedCall struct {
	parent   int16 // index of parent in the inltree, or < 0
	funcID   uint8 // type of the called function
	_        byte
	file     int32 // fileno index into filetab
	line     int32 // line number of the call site
	func_    int32 // offset into pclntab for name of called function
	parentPc int32 // position of an instruction whose source position is the call site (offset from entry)
}

func pcNameFileLine(pc uintptr) (name, file string, line int32) {
	funcInfo := findfunc(pc)
	if funcInfo._func == nil {
		return
	}

	entry := funcInfoEntry(funcInfo)

	if pc > entry {
		// We store the pc of the start of the instruction following
		// the instruction in question (the call or the inline mark).
		// This is done for historical reasons, and to make FuncForPC
		// work correctly for entries in the result of runtime.Callers.
		pc--
	}

	file, line = funcline1(funcInfo, pc, false)

	name = funcname(funcInfo)
	const _PCDATA_InlTreeIndex = 2
	const _FUNCDATA_InlTree = 3
	if inldata := funcdata(funcInfo, _FUNCDATA_InlTree); inldata != nil {
		ix := pcdatavalue1(funcInfo, _PCDATA_InlTreeIndex, pc, nil, false)
		if ix >= 0 {
			inltree := (*[1 << 20]inlinedCall)(inldata)
			// Note: entry is not modified. It always refers to a real frame, not an inlined one.
			name = funcnameFromNameoff(funcInfo, inltree[ix].func_)
			// File/line is already correct.
			// TODO: remove file/line from InlinedCall?
		}
	}

	return
}

//go:linkname funcname runtime.funcname
func funcname(f funcInfo) string

//go:linkname funcdata runtime.funcdata
func funcdata(f funcInfo, i uint8) unsafe.Pointer

//go:linkname pcdatavalue runtime.pcdatavalue
func pcdatavalue(f funcInfo, table int32, targetpc uintptr, cache unsafe.Pointer) int32

//go:linkname pcdatavalue1 runtime.pcdatavalue1
func pcdatavalue1(f funcInfo, table int32, targetpc uintptr, cache unsafe.Pointer, strict bool) int32

//go:linkname funcnameFromNameoff runtime.funcnameFromNameoff
func funcnameFromNameoff(f funcInfo, nameoff int32) string

// Fastrandn returns a pseudorandom uint32 in [0,n).
//
//go:noescape
//go:linkname Fastrandn runtime.fastrandn
func Fastrandn(x uint32) uint32
