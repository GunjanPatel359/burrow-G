package portable

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Reader reads a session written by Writer, one record at a time.
type Reader struct {
	sc     *bufio.Scanner
	header Header
	end    End
	done   bool // the end line was read
}

// NewReader reads the header line and refuses anything that isn't a
// burrow-session file.
func NewReader(in io.Reader) (*Reader, error) {
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 64<<10), maxLine)
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return nil, err
		}
		return nil, errors.New("not a burrow-G export: empty file")
	}
	var h Header
	if json.Unmarshal(sc.Bytes(), &h) != nil || h.Kind != "header" || !strings.HasPrefix(h.Format, "burrow-session/") {
		return nil, errors.New("not a burrow-G export: no burrow-session header")
	}
	return &Reader{sc: sc, header: h}, nil
}

func (r *Reader) Header() Header { return r.header }

// Next returns the next record, io.EOF after the end line, or ErrTruncated
// when the input stops without one.
func (r *Reader) Next() (Record, error) {
	if r.done {
		return Record{}, io.EOF
	}
	if !r.sc.Scan() {
		if err := r.sc.Err(); err != nil {
			return Record{}, err
		}
		return Record{}, ErrTruncated
	}
	var rec Record
	if err := json.Unmarshal(r.sc.Bytes(), &rec); err != nil {
		return Record{}, fmt.Errorf("bad line: %w", err)
	}
	if rec.Kind == "end" {
		if err := json.Unmarshal(r.sc.Bytes(), &r.end); err != nil {
			return Record{}, fmt.Errorf("bad end line: %w", err)
		}
		r.done = true
		return Record{}, io.EOF
	}
	return rec, nil
}

// End is the end line, once Next has returned io.EOF.
func (r *Reader) End() End { return r.end }
