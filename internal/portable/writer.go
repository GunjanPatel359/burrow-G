package portable

import (
	"bufio"
	"encoding/json"
	"io"
	"time"
)

// Writer streams one session: Header, then Record for each record, then
// Close. Nothing is held back but the buffer, so any size session fits.
type Writer struct {
	buf     *bufio.Writer
	enc     *json.Encoder
	burrow  string
	records int
	dropped map[string]int
}

// NewWriter writes to out; burrow is the burrow-G version put in the header.
func NewWriter(out io.Writer, burrow string) *Writer {
	buf := bufio.NewWriterSize(out, 1<<20)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false) // keep code and tags readable: "<a>", not "<a>"
	return &Writer{buf: buf, enc: enc, burrow: burrow}
}

// Header fills in what every header carries, then writes it.
func (w *Writer) Header(h Header) error {
	h.Kind = "header"
	h.Format = Format
	h.Burrow = w.burrow
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
func (w *Writer) Drop(reason string) {
	if w.dropped == nil {
		w.dropped = map[string]int{}
	}
	w.dropped[reason]++
}

// Close writes the end line and flushes. It doesn't close the underlying
// writer, which belongs to the caller.
func (w *Writer) Close() (End, error) {
	end := End{Kind: "end", Records: w.records, Dropped: w.dropped}
	if err := w.enc.Encode(end); err != nil {
		return End{}, err
	}
	return end, w.buf.Flush()
}
