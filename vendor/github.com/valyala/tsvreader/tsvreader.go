package tsvreader

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strconv"
	"time"
	"unsafe"
)

// New returns new Reader that reads TSV data from r.
func New(r io.Reader) *Reader {
	var tr Reader
	tr.Reset(r)
	return &tr
}

// Reader reads tab-separated data.
//
// Call New for creating new TSV reader.
// Call Next before reading the next row.
//
// It is expected that columns are separated by tabs while rows
// are separated by newlines.
type Reader struct {
	r    io.Reader
	rb   []byte
	rErr error
	rBuf [16*1024]byte

	col int
	row int

	rowBuf  []byte
	b       []byte
	scratch []byte

	err          error
	needUnescape bool
}

// Reset resets the reader for reading from r.
func (tr *Reader) Reset(r io.Reader) {
	tr.r = r
	tr.rb = nil
	tr.rErr = nil

	tr.col = 0
	tr.row = 0

	tr.rowBuf = nil
	tr.b = nil
	tr.scratch = tr.scratch[:0]

	tr.err = nil
	tr.needUnescape = false
}

// Error returns the last error.
func (tr *Reader) Error() error {
	if tr.err == io.EOF {
		return nil
	}
	return tr.err
}

// ResetError resets the current error, so the reader could proceed further.
func (tr *Reader) ResetError() {
	tr.err = nil
}

// HasCols returns true if the current row contains unread columns.
//
// An empty row doesn't contain columns.
//
// This function may be used if TSV stream contains rows with different
// number of colums.
func (tr *Reader) HasCols() bool {
	return len(tr.rowBuf) > 0 && tr.b != nil
}

// Next advances to the next row.
//
// Returns true if the next row does exist.
//
// Next must be called after reading all the columns on the previous row.
// Check Error after Next returns false.
//
// HasCols may be used for reading rows with variable number of columns.
func (tr *Reader) Next() bool {
	if tr.err != nil {
		return false
	}
	if tr.HasCols() {
		tr.err = fmt.Errorf("row #%d %q contains unread columns: %q", tr.row, tr.rowBuf, tr.b)
		return false
	}

	tr.row++
	tr.col = 0
	tr.rowBuf = nil

	for {
		if len(tr.rb) == 0 {
			// Read buffer is empty. Attempt to fill it.
			if tr.rErr != nil {
				tr.err = tr.rErr
				if tr.err != io.EOF {
					tr.err = fmt.Errorf("cannot read row #%d: %s", tr.row, tr.err)
				} else if len(tr.scratch) > 0 {
					tr.err = fmt.Errorf("cannot find newline at the end of row #%d; row: %q", tr.row, tr.scratch)
				}
				return false
			}
			n, err := tr.r.Read(tr.rBuf[:])
			tr.rb = tr.rBuf[:n]
			tr.needUnescape = (bytes.IndexByte(tr.rb, '\\') >= 0)
			tr.rErr = err
		}

		// Search for the end of the current row.
		n := bytes.IndexByte(tr.rb, '\n')
		if n >= 0 {
			// Fast path: the row has been found.
			b := tr.rb[:n]
			tr.rb = tr.rb[n+1:]
			if len(tr.scratch) > 0 {
				tr.scratch = append(tr.scratch, b...)
				b = tr.scratch
				tr.scratch = tr.scratch[:0]
			}
			tr.rowBuf = b
			tr.b = tr.rowBuf
			return true
		}

		// Slow path: cannot find the end of row.
		// Append tr.rb to tr.scratch and repeat.
		tr.scratch = append(tr.scratch, tr.rb...)
		tr.rb = nil
	}
}

// Int returns the next int column value from the current row.
func (tr *Reader) Int() int {
	if tr.err != nil {
		return 0
	}
	b, err := tr.nextCol()
	if err != nil {
		tr.setColError("cannot read `int`", err)
		return 0
	}

	n, err := strconv.Atoi(b2s(b))
	if err != nil {
		tr.setColError("cannot parse `int`", err)
		return 0
	}
	return n
}

// Uint returns the next uint column value from the current row.
func (tr *Reader) Uint() uint {
	if tr.err != nil {
		return 0
	}
	b, err := tr.nextCol()
	if err != nil {
		tr.setColError("cannot read `uint`", err)
		return 0
	}
	s := b2s(b)

	// Fast path - attempt to use Atoi
	n, err := strconv.Atoi(s)
	if err == nil && n >= 0 {
		return uint(n)
	}

	// Slow path - use ParseUint
	nu, err := strconv.ParseUint(s, 10, strconv.IntSize)
	if err != nil {
		tr.setColError("cannot parse `uint`", err)
		return 0
	}
	return uint(nu)
}

// Int32 returns the next int32 column value from the current row.
func (tr *Reader) Int32() int32 {
	if tr.err != nil {
		return 0
	}
	b, err := tr.nextCol()
	if err != nil {
		tr.setColError("cannot read `int32`", err)
		return 0
	}
	s := b2s(b)

	// Fast path - attempt to use Atoi
	n, err := strconv.Atoi(s)
	if err == nil && n >= math.MinInt32 && n <= math.MaxInt32 {
		return int32(n)
	}

	// Slow path - use ParseInt
	n32, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		tr.setColError("cannot parse `int32`", err)
		return 0
	}
	return int32(n32)
}

// Uint32 returns the next uint32 column value from the current row.
func (tr *Reader) Uint32() uint32 {
	if tr.err != nil {
		return 0
	}
	b, err := tr.nextCol()
	if err != nil {
		tr.setColError("cannot read `uint32`", err)
		return 0
	}
	s := b2s(b)

	// Fast path - attempt to use Atoi
	n, err := strconv.Atoi(s)
	if err == nil && n >= 0 && n <= math.MaxUint32 {
		return uint32(n)
	}

	// Slow path - use ParseUint
	n32, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		tr.setColError("cannot parse `uint32`", err)
		return 0
	}
	return uint32(n32)
}

// Int16 returns the next int16 column value from the current row.
func (tr *Reader) Int16() int16 {
	if tr.err != nil {
		return 0
	}
	b, err := tr.nextCol()
	if err != nil {
		tr.setColError("cannot read `int16`", err)
		return 0
	}
	n, err := strconv.Atoi(b2s(b))
	if err != nil {
		tr.setColError("cannot parse `int16`", err)
		return 0
	}
	if n < math.MinInt16 || n > math.MaxInt16 {
		tr.setColError("cannot parse `int16`", fmt.Errorf("out of range"))
		return 0
	}
	return int16(n)
}

// Uint16 returns the next uint16 column value from the current row.
func (tr *Reader) Uint16() uint16 {
	if tr.err != nil {
		return 0
	}
	b, err := tr.nextCol()
	if err != nil {
		tr.setColError("cannot read `uint16`", err)
		return 0
	}
	n, err := strconv.Atoi(b2s(b))
	if err != nil {
		tr.setColError("cannot parse `uint16`", err)
		return 0
	}
	if n < 0 {
		tr.setColError("cannot parse `uint16`", fmt.Errorf("invalid syntax"))
		return 0
	}
	if n > math.MaxUint16 {
		tr.setColError("cannot parse `uint16`", fmt.Errorf("out of range"))
		return 0
	}
	return uint16(n)
}

// Int8 returns the next int8 column value from the current row.
func (tr *Reader) Int8() int8 {
	if tr.err != nil {
		return 0
	}
	b, err := tr.nextCol()
	if err != nil {
		tr.setColError("cannot read `int8`", err)
		return 0
	}
	n, err := strconv.Atoi(b2s(b))
	if err != nil {
		tr.setColError("cannot parse `int8`", err)
		return 0
	}
	if n < math.MinInt8 || n > math.MaxInt8 {
		tr.setColError("cannot parse `int8`", fmt.Errorf("out of range"))
		return 0
	}
	return int8(n)
}

// Uint8 returns the next uint8 column value from the current row.
func (tr *Reader) Uint8() uint8 {
	if tr.err != nil {
		return 0
	}
	b, err := tr.nextCol()
	if err != nil {
		tr.setColError("cannot read `uint8`", err)
		return 0
	}
	n, err := strconv.Atoi(b2s(b))
	if err != nil {
		tr.setColError("cannot parse `uint8`", err)
		return 0
	}
	if n < 0 {
		tr.setColError("cannot parse `uint8`", fmt.Errorf("invalid syntax"))
		return 0
	}
	if n > math.MaxUint8 {
		tr.setColError("cannot parse `uint8`", fmt.Errorf("out of range"))
		return 0
	}
	return uint8(n)
}

// Int64 returns the next int64 column value from the current row.
func (tr *Reader) Int64() int64 {
	if tr.err != nil {
		return 0
	}
	b, err := tr.nextCol()
	if err != nil {
		tr.setColError("cannot read `int64`", err)
		return 0
	}
	s := b2s(b)

	// Fast path - attempt to use Atoi
	n, err := strconv.Atoi(s)
	if err == nil && int64(n) >= math.MinInt64 && int64(n) <= math.MaxInt64 {
		return int64(n)
	}

	// Slow path - use ParseInt
	n64, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		tr.setColError("cannot parse `int64`", err)
		return 0
	}
	return n64
}

// Uint64 returns the next uint64 column value from the current row.
func (tr *Reader) Uint64() uint64 {
	if tr.err != nil {
		return 0
	}
	b, err := tr.nextCol()
	if err != nil {
		tr.setColError("cannot read `uint64`", err)
		return 0
	}
	s := b2s(b)

	// Fast path - attempt to use Atoi
	n, err := strconv.Atoi(s)
	if err == nil && n >= 0 && uint64(n) <= math.MaxUint64 {
		return uint64(n)
	}

	// Slow path - use ParseUint
	n64, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		tr.setColError("cannot parse `uint64`", err)
		return 0
	}
	return n64
}

// Float32 returns the next float32 column value from the current row.
func (tr *Reader) Float32() float32 {
	if tr.err != nil {
		return 0
	}
	b, err := tr.nextCol()
	if err != nil {
		tr.setColError("cannot read `float32`", err)
		return 0
	}
	s := b2s(b)

	f32, err := strconv.ParseFloat(s, 32)
	if err != nil {
		tr.setColError("cannot parse `float32`", err)
		return 0
	}
	return float32(f32)
}

// Float64 returns the next float64 column value from the current row.
func (tr *Reader) Float64() float64 {
	if tr.err != nil {
		return 0
	}
	b, err := tr.nextCol()
	if err != nil {
		tr.setColError("cannot read `float64`", err)
		return 0
	}
	s := b2s(b)

	f64, err := strconv.ParseFloat(s, 64)
	if err != nil {
		tr.setColError("cannot parse `float64`", err)
		return 0
	}
	return f64
}

// SkipCol skips the next column from the current row.
func (tr *Reader) SkipCol() {
	if tr.err != nil {
		return
	}
	_, err := tr.nextCol()
	if err != nil {
		tr.setColError("cannot skip column", err)
	}
}

// Bytes returns the next bytes column value from the current row.
//
// The returned value is valid until the next call to Reader.
func (tr *Reader) Bytes() []byte {
	if tr.err != nil {
		return nil
	}
	b, err := tr.nextCol()
	if err != nil {
		tr.setColError("cannot read `bytes`", err)
		return nil
	}

	if !tr.needUnescape {
		// Fast path - nothing to unescape.
		return b
	}

	// Unescape b
	n := bytes.IndexByte(b, '\\')
	if n < 0 {
		// Nothing to unescape in the current column.
		return b
	}

	// Slow path - in-place unescaping compatible with ClickHouse.
	n++
	d := b[:n]
	b = b[n:]
	for len(b) > 0 {
		switch b[0] {
		case 'b':
			d[len(d)-1] = '\b'
		case 'f':
			d[len(d)-1] = '\f'
		case 'r':
			d[len(d)-1] = '\r'
		case 'n':
			d[len(d)-1] = '\n'
		case 't':
			d[len(d)-1] = '\t'
		case '0':
			d[len(d)-1] = 0
		case '\'':
			d[len(d)-1] = '\''
		case '\\':
			d[len(d)-1] = '\\'
		default:
			d[len(d)-1] = b[0]
		}

		b = b[1:]
		n = bytes.IndexByte(b, '\\')
		if n < 0 {
			d = append(d, b...)
			break
		}
		n++
		d = append(d, b[:n]...)
		b = b[n:]
	}
	return d
}

// String returns the next string column value from the current row.
//
// String allocates memory. Use Bytes to avoid memory allocations.
func (tr *Reader) String() string {
	return string(tr.Bytes())
}

// Date returns the next date column value from the current row.
//
// date must be in the format YYYY-MM-DD
func (tr *Reader) Date() time.Time {
	if tr.err != nil {
		return zeroTime
	}
	b, err := tr.nextCol()
	if err != nil {
		tr.setColError("cannot read `date`", err)
		return zeroTime
	}
	s := b2s(b)

	y, m, d, err := parseDate(s)
	if err != nil {
		tr.setColError("cannot parse `date`", err)
		return zeroTime
	}
	if y == 0 && m == 0 && d == 0 {
		// special case for ClickHouse
		return zeroTime
	}
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

// DateTime returns the next datetime column value from the current row.
//
// datetime must be in the format YYYY-MM-DD hh:mm:ss.
func (tr *Reader) DateTime() time.Time {
	if tr.err != nil {
		return zeroTime
	}
	b, err := tr.nextCol()
	if err != nil {
		tr.setColError("cannot read `datetime`", err)
		return zeroTime
	}
	s := b2s(b)

	dt, err := parseDateTime(s)
	if err != nil {
		tr.setColError("cannot parse `datetime`", err)
		return zeroTime
	}
	return dt
}

func parseDateTime(s string) (time.Time, error) {
	if len(s) != len("YYYY-MM-DD hh:mm:ss") {
		return zeroTime, fmt.Errorf("too short datetime")
	}
	y, m, d, err := parseDate(s[:len("YYYY-MM-DD")])
	if err != nil {
		return zeroTime, err
	}
	s = s[len("YYYY-MM-DD"):]
	if s[0] != ' ' || s[3] != ':' || s[6] != ':' {
		return zeroTime, fmt.Errorf("invalid time format. Must be hh:mm:ss")
	}
	hS := s[1:3]
	minS := s[4:6]
	secS := s[7:]
	h, err := strconv.Atoi(hS)
	if err != nil {
		return zeroTime, fmt.Errorf("invalid hour: %s", err)
	}
	min, err := strconv.Atoi(minS)
	if err != nil {
		return zeroTime, fmt.Errorf("invalid minute: %s", err)
	}
	sec, err := strconv.Atoi(secS)
	if err != nil {
		return zeroTime, fmt.Errorf("invalid second: %s", err)
	}
	if y == 0 && m == 0 && d == 0 {
		// Special case for ClickHouse
		return zeroTime, nil
	}
	return time.Date(y, time.Month(m), d, h, min, sec, 0, time.UTC), nil
}

func parseDate(s string) (y, m, d int, err error) {
	if len(s) != len("YYYY-MM-DD") {
		err = fmt.Errorf("too short date")
		return
	}
	s = s[:len("YYYY-MM-DD")]
	if s[4] != '-' && s[7] != '-' {
		err = fmt.Errorf("invalid date format. Must be YYYY-MM-DD")
		return
	}
	yS := s[:4]
	mS := s[5:7]
	dS := s[8:]
	y, err = strconv.Atoi(yS)
	if err != nil {
		err = fmt.Errorf("invalid year: %s", err)
		return
	}
	m, err = strconv.Atoi(mS)
	if err != nil {
		err = fmt.Errorf("invalid month: %s", err)
		return
	}
	d, err = strconv.Atoi(dS)
	if err != nil {
		err = fmt.Errorf("invalid day: %s", err)
		return
	}
	return y, m, d, nil
}

var zeroTime time.Time

func (tr *Reader) nextCol() ([]byte, error) {
	if tr.row == 0 {
		return nil, fmt.Errorf("missing Next call")
	}

	tr.col++
	if tr.b == nil {
		return nil, fmt.Errorf("no more columns")
	}

	n := bytes.IndexByte(tr.b, '\t')
	if n < 0 {
		// last column
		b := tr.b
		tr.b = nil
		return b, nil
	}

	b := tr.b[:n]
	tr.b = tr.b[n+1:]
	return b, nil
}

func (tr *Reader) setColError(msg string, err error) {
	tr.err = fmt.Errorf("%s at row #%d, col #%d %q: %s", msg, tr.row, tr.col, tr.rowBuf, err)
}

func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
