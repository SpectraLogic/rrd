package rrd

/*
#include <stdlib.h>
#include <rrd.h>
#include <rrd_client.h>
#include "rrdfunc.h"
#cgo pkg-config: librrd
*/
import "C"
import (
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

type cstring C.char

func newCstring(s string) *cstring {
	cs := C.malloc(C.size_t(len(s) + 1))
	buf := (*[1<<31 - 1]byte)(cs)[:len(s)+1]
	copy(buf, s)
	buf[len(s)] = 0
	return (*cstring)(cs)
}

func (cs *cstring) Free() {
	if cs != nil {
		C.free(unsafe.Pointer(cs))
	}
}

func (cs *cstring) String() string {
	buf := (*[1<<31 - 1]byte)(unsafe.Pointer(cs))
	for n, b := range buf {
		if b == 0 {
			return string(buf[:n])
		}
	}
	panic("rrd: bad C string")
}

var mutex sync.Mutex

func makeArgs(args []string) []*C.char {
	ret := make([]*C.char, len(args))
	for i, s := range args {
		ret[i] = C.CString(s)
	}
	return ret
}

func freeCString(s *C.char) {
	C.free(unsafe.Pointer(s))
}

func freeArgs(cArgs []*C.char) {
	for _, s := range cArgs {
		freeCString(s)
	}
}

func makeError(e *C.char) error {
	var null *C.char
	if e == null {
		return nil
	}
	defer freeCString(e)
	return Error(C.GoString(e))
}

func (c *Creator) create() error {
	filename := C.CString(c.filename)
	defer freeCString(filename)
	args := makeArgs(c.args)
	defer freeArgs(args)

	e := C.rrdCreate(
		filename,
		C.ulong(c.step),
		C.time_t(c.start.Unix()),
		C.int(len(args)),
		&args[0],
	)
	return makeError(e)
}

func (u *Updater) update(args []*cstring) error {
	var e *C.char

	if u.daemon != nil {
		e = C.rrdDaemonUpdate(
			(*C.char)(u.daemon),
			(*C.char)(u.filename),
			(*C.char)(u.template),
			C.int(len(args)),
			(**C.char)(unsafe.Pointer(&args[0])),
		)
	} else {
		e = C.rrdUpdate(
			(*C.char)(u.filename),
			(*C.char)(u.template),
			C.int(len(args)),
			(**C.char)(unsafe.Pointer(&args[0])),
		)
	}
	return makeError(e)
}

func (u *Updater) updatenodaemon(args []*cstring) error {
	var e *C.char

	e = C.rrdUpdatex(
		(*C.char)(u.filename),
		(*C.char)(u.template),
		C.int(len(args)),
		(**C.char)(unsafe.Pointer(&args[0])),
	)
	return makeError(e)
}

var (
	graphv = C.CString("graphv")
	xport  = C.CString("xport")

	oStart           = C.CString("-s")
	oEnd             = C.CString("-e")
	oTitle           = C.CString("-t")
	oVlabel          = C.CString("-v")
	oWidth           = C.CString("-w")
	oHeight          = C.CString("-h")
	oUpperLimit      = C.CString("-u")
	oLowerLimit      = C.CString("-l")
	oRigid           = C.CString("-r")
	oAltAutoscale    = C.CString("-A")
	oAltAutoscaleMin = C.CString("-J")
	oAltAutoscaleMax = C.CString("-M")
	oNoGridFit       = C.CString("-N")

	oLogarithmic   = C.CString("-o")
	oUnitsExponent = C.CString("-X")
	oUnitsLength   = C.CString("-L")

	oRightAxis      = C.CString("--right-axis")
	oRightAxisLabel = C.CString("--right-axis-label")

	oDaemon = C.CString("--daemon")

	oBorder = C.CString("--border")

	oNoLegend = C.CString("-g")

	oLazy = C.CString("-z")

	oColor = C.CString("-c")

	oSlopeMode   = C.CString("-E")
	oImageFormat = C.CString("-a")
	oInterlaced  = C.CString("-i")

	oBase      = C.CString("-b")
	oWatermark = C.CString("-W")

	oStep    = C.CString("--step")
	oMaxRows = C.CString("-m")
)

func ftoa(f float64) string {
	return strconv.FormatFloat(f, 'e', 10, 64)
}

func ftoc(f float64) *C.char {
	return C.CString(ftoa(f))
}

func i64toa(i int64) string {
	return strconv.FormatInt(i, 10)
}

func i64toc(i int64) *C.char {
	return C.CString(i64toa(i))
}

func u64toa(u uint64) string {
	return strconv.FormatUint(u, 10)
}

func u64toc(u uint64) *C.char {
	return C.CString(u64toa(u))
}
func itoa(i int) string {
	return i64toa(int64(i))
}

func itoc(i int) *C.char {
	return i64toc(int64(i))
}

func utoa(u uint) string {
	return u64toa(uint64(u))
}

func utoc(u uint) *C.char {
	return u64toc(uint64(u))
}

func parseInfoKey(ik string) (kname, kkey string, kid int) {
	kid = -1
	o := strings.IndexRune(ik, '[')
	if o == -1 {
		kname = ik
		return
	}
	c := strings.IndexRune(ik[o+1:], ']')
	if c == -1 {
		kname = ik
		return
	}
	c += o + 1
	kname = ik[:o] + ik[c+1:]
	kkey = ik[o+1 : c]
	if strings.HasPrefix(kname, "ds.") {
		return
	} else if id, err := strconv.Atoi(kkey); err == nil && id >= 0 {
		kid = id
	}
	return
}

func updateInfoValue(i *C.struct_rrd_info_t, v interface{}) interface{} {
	switch i._type {
	case C.RD_I_VAL:
		return float64(*(*C.rrd_value_t)(unsafe.Pointer(&i.value[0])))
	case C.RD_I_CNT:
		return uint(*(*C.ulong)(unsafe.Pointer(&i.value[0])))
	case C.RD_I_STR:
		return C.GoString(*(**C.char)(unsafe.Pointer(&i.value[0])))
	case C.RD_I_INT:
		return int(*(*C.int)(unsafe.Pointer(&i.value[0])))
	case C.RD_I_BLO:
		blob := *(*C.rrd_blob_t)(unsafe.Pointer(&i.value[0]))
		b := C.GoBytes(unsafe.Pointer(blob.ptr), C.int(blob.size))
		if v == nil {
			return b
		}
		return append(v.([]byte), b...)
	}

	return nil
}

func parseRRDInfo(i *C.rrd_info_t) map[string]interface{} {
	defer C.rrd_info_free(i)

	r := make(map[string]interface{})
	for w := (*C.struct_rrd_info_t)(i); w != nil; w = w.next {
		kname, kkey, kid := parseInfoKey(C.GoString(w.key))
		v, ok := r[kname]
		switch {
		case kid != -1:
			var a []interface{}
			if ok {
				a = v.([]interface{})
			}
			if len(a) < kid+1 {
				oldA := a
				a = make([]interface{}, kid+1)
				copy(a, oldA)
			}
			a[kid] = updateInfoValue(w, a[kid])
			v = a
		case kkey != "":
			var m map[string]interface{}
			if ok {
				m = v.(map[string]interface{})
			} else {
				m = make(map[string]interface{})
			}
			old, _ := m[kkey]
			m[kkey] = updateInfoValue(w, old)
			v = m
		default:
			v = updateInfoValue(w, v)
		}
		r[kname] = v
	}
	return r
}

// Info returns information about RRD file.
func Info(filename string) (map[string]interface{}, error) {
	return DaemonInfo(filename, nil)
}

// Info returns information about RRD file.
func DaemonInfo(filename string, daemon *string) (map[string]interface{}, error) {
	fn := C.CString(filename)
	defer freeCString(fn)
	var cDaemon *C.char
	var err error
	if daemon != nil {
		cDaemon = C.CString(*daemon)
		defer freeCString(cDaemon)
	}
	var i *C.rrd_info_t
	if daemon != nil {
		err = makeError(C.rrdDaemonInfo(&i, cDaemon, fn))
	} else {
		err = makeError(C.rrdInfo(&i, fn))
	}
	if err != nil {
		return nil, err
	}
	return parseRRDInfo(i), nil
}

// Fetch retrieves data from RRD file.
func Fetch(filename, cf string, start, end time.Time, step time.Duration) (FetchResult, error) {
	return DaemonFetch(filename, cf, start, end, step, nil)
}

// DaemonFetch retrieves data from RRD file or the RRD daemon.
func DaemonFetch(filename, cf string, start, end time.Time, step time.Duration, daemon *string) (FetchResult, error) {
	fn := C.CString(filename)
	defer freeCString(fn)
	cCf := C.CString(cf)
	defer freeCString(cCf)
	cStart := C.time_t(start.Unix())
	cEnd := C.time_t(end.Unix())
	cStep := C.ulong(step.Seconds())
	var cDaemon *C.char
	if daemon != nil {
		cDaemon = C.CString(*daemon)
		defer freeCString(cDaemon)
	}

	var (
		ret      C.int
		cDsCnt   C.ulong
		cDsNames **C.char
		cData    *C.double
	)
	var err error
	if daemon != nil {
		err = makeError(C.rrdDaemonFetch(&ret, cDaemon, fn, cCf, &cStart, &cEnd, &cStep, &cDsCnt, &cDsNames, &cData))
	} else {
		err = makeError(C.rrdFetch(&ret, fn, cCf, &cStart, &cEnd, &cStep, &cDsCnt, &cDsNames, &cData))
	}
	if err != nil {
		return FetchResult{filename, cf, start, end, step, nil, 0, nil}, err
	}

	start = time.Unix(int64(cStart), 0)
	end = time.Unix(int64(cEnd), 0)
	step = time.Duration(cStep) * time.Second
	dsCnt := int(cDsCnt)

	dsNames := make([]string, dsCnt)
	for i := 0; i < dsCnt; i++ {
		dsName := C.arrayGetCString(cDsNames, C.int(i))
		dsNames[i] = C.GoString(dsName)
		C.free(unsafe.Pointer(dsName))
	}
	C.free(unsafe.Pointer(cDsNames))

	rowCnt := (int(cEnd)-int(cStart))/int(cStep) + 1
	valuesLen := dsCnt * rowCnt
	var values []float64
	sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&values)))
	sliceHeader.Cap = valuesLen
	sliceHeader.Len = valuesLen
	sliceHeader.Data = uintptr(unsafe.Pointer(cData))
	return FetchResult{filename, cf, start, end, step, dsNames, rowCnt, values}, nil
}

// DaemonFlush instructs the daemon to flush the given RRD file.
func DaemonFlush(daemon, filename string) error {
	cDaemon := C.CString(daemon)
	defer freeCString(cDaemon)
	fn := C.CString(filename)
	defer freeCString(fn)

	var ret C.int
	return makeError(C.rrdDaemonFlush(&ret, cDaemon, fn))
}

// FreeValues free values memory allocated by C.
func (r *FetchResult) FreeValues() {
	sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&r.values)))
	C.free(unsafe.Pointer(sliceHeader.Data))
}

// Values returns copy of internal array of values.
func (r *FetchResult) Values() []float64 {
	return append([]float64{}, r.values...)
}
