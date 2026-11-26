package portable

import (
	"bufio"
	"encoding/json"
	"io"
	"time"
)

// Writer streams one session: Header, then Record for each record, then
// Close. It holds one line at a time, so any size session fits.
type Writer struct {
	buf     *bufio.Writer // batches lines into large writes
	enc     *json.Encoder
	burrow  string // burrow-G version, stamped into the header
	records int
	dropped map[string]int
}

// NewWriter writes to out; burrow is the burrow-G version for the header.
func NewWriter(out io.Writer, burrow string) *Writer {
	buf := bufio.NewWriterSize(out, 1<<20)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false) // keep code readable: "a < b", not "a < b"
	return &Writer{buf: buf, enc: enc, burrow: burrow, dropped: map[string]int{}}
}

// Header fills in what every header carries, so agents can't get it wrong.
func (w *Writer) Header(h Header) error {
	h.Kind, h.Format, h.Burrow = "header", Format, w.burrow
	if h.Exported.IsZero() {
		h.Exported = time.Now().UTC()
	}
	return w.enc.Encode(h)
}

func (w *Writer) Record(r Record) error {
	if err := w.enc.Encode(r); err != nil {
		return err
	}
	w.records++
	return nil
}

// Drop counts one thing left out of the export, by reason.
func (w *Writer) Drop(reason string) { w.dropped[reason]++ }

// Close writes the end line and flushes. The caller owns out and closes it.
func (w *Writer) Close() (End, error) {
	end := End{Kind: "end", Records: w.records}
	if len(w.dropped) > 0 {
		end.Dropped = w.dropped
	}
	if err := w.enc.Encode(end); err != nil {
		return End{}, err
	}
	return end, w.buf.Flush()
}
