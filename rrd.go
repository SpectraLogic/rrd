// Simple wrapper for rrdtool C library
package rrd

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

type Error string

func (e Error) Error() string {
	return string(e)
}

/*
type cstring []byte

func newCstring(s string) cstring {
	cs := make(cstring, len(s)+1)
	copy(cs, s)
	return cs
}

func (cs cstring) p() unsafe.Pointer {
	if len(cs) == 0 {
		return nil
	}
	return unsafe.Pointer(&cs[0])
}

func (cs cstring) String() string {
	return string(cs[:len(cs)-1])
}
*/

func join(args []interface{}) string {
	sa := make([]string, len(args))
	for i, a := range args {
		var s string
		switch v := a.(type) {
		case time.Time:
			s = i64toa(v.Unix())
		default:
			s = fmt.Sprint(v)
		}
		sa[i] = s
	}
	return strings.Join(sa, ":")
}

type Creator struct {
	filename string
	start    time.Time
	step     uint
	args     []string
}

// NewCreator returns new Creator object. You need to call Create to really
// create database file.
//
//	filename - name of database file
//	start    - don't accept any data timed before or at time specified
//	step     - base interval in seconds with which data will be fed into RRD
func NewCreator(filename string, start time.Time, step uint) *Creator {
	return &Creator{
		filename: filename,
		start:    start,
		step:     step,
	}
}

// DS formats a DS argument and appends it to the list of arguments to be
// passed to rrdcreate(). Each element of args is formatted with fmt.Sprint().
// Please see the rrdcreate(1) manual page for in-depth documentation.
func (c *Creator) DS(name, compute string, args ...interface{}) {
	c.args = append(c.args, "DS:"+name+":"+compute+":"+join(args))
}

// RRA formats an RRA argument and appends it to the list of arguments to be
// passed to rrdcreate(). Each element of args is formatted with fmt.Sprint().
// Please see the rrdcreate(1) manual page for in-depth documentation.
func (c *Creator) RRA(cf string, args ...interface{}) {
	c.args = append(c.args, "RRA:"+cf+":"+join(args))
}

// Create creates new database file. If overwrite is true it overwrites
// database file if exists. If overwrite is false it returns error if file
// exists (you can use os.IsExist function to check this case).
func (c *Creator) Create(overwrite bool) error {
	if !overwrite {
		f, err := os.OpenFile(
			c.filename,
			os.O_WRONLY|os.O_CREATE|os.O_EXCL,
			0666,
		)
		if err != nil {
			return err
		}
		f.Close()
	}
	return c.create()
}

// Use cstring and unsafe.Pointer to avoid allocations for C calls

type Updater struct {
	filename *cstring
	template *cstring
	daemon   *cstring

	args []*cstring
}

func NewUpdater(filename string) *Updater {
	return NewDaemonUpdater(filename, nil)
}

func NewDaemonUpdater(filename string, daemon *string) *Updater {
	u := &Updater{filename: newCstring(filename)}
	if daemon != nil {
		u.daemon = newCstring(*daemon)
	}
	runtime.SetFinalizer(u, cfree)
	return u
}

func cfree(u *Updater) {
	u.filename.Free()
	u.template.Free()
	for _, a := range u.args {
		a.Free()
	}
}

func (u *Updater) SetTemplate(dsName ...string) {
	u.template.Free()
	u.template = newCstring(strings.Join(dsName, ":"))
}

// Cache chaches data for later save using Update(). Use it to avoid
// open/read/write/close for every update.
func (u *Updater) Cache(args ...interface{}) {
	u.args = append(u.args, newCstring(join(args)))
}

// Update saves data in RRDB.
// Without args Update saves all subsequent updates buffered by Cache method.
// If you specify args it saves them immediately.
func (u *Updater) Update(args ...interface{}) error {
	if len(args) != 0 {
		cs := newCstring(join(args))
		err := u.update([]*cstring{cs})
		cs.Free()
		return err
	} else if len(u.args) != 0 {
		err := u.update(u.args)
		for _, a := range u.args {
			a.Free()
		}
		u.args = nil
		return err
	}
	return nil
}

// Update saves data in RRDB.
// Without args Update saves all subsequent updates buffered by Cache method.
// If you specify args it saves them immediately.
func (u *Updater) UpdateNoDaemon(args ...interface{}) error {
	if len(args) != 0 {
		cs := newCstring(join(args))
		err := u.updatenodaemon([]*cstring{cs})
		cs.Free()
		return err
	} else if len(u.args) != 0 {
		err := u.updatenodaemon(u.args)
		for _, a := range u.args {
			a.Free()
		}
		u.args = nil
		return err
	}
	return nil
}

type GraphInfo struct {
	Print         []string
	Width, Height uint
	Ymin, Ymax    float64
}

const (
	maxUint  = ^uint(0)
	maxInt   = int(maxUint >> 1)
	minInt   = -maxInt - 1
	defWidth = 2
)

type FetchResult struct {
	Filename string
	Cf       string
	Start    time.Time
	End      time.Time
	Step     time.Duration
	DsNames  []string
	RowCnt   int
	values   []float64
}

func (r *FetchResult) ValueAt(dsIndex, rowIndex int) float64 {
	return r.values[len(r.DsNames)*rowIndex+dsIndex]
}
