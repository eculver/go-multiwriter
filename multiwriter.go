package multiwriter // import "go.enc.dev/multiwriter"

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/kataras/tablewriter"
)

const (
	defaultSize = 10000

	// CSVFormat sets output format to comma-separated values
	CSVFormat = "csv"
	// TableFormat sets output format to an ASCII table
	TableFormat = "table"
	// TextFormat sets the output format to a fmt-renderd text string
	TextFormat = "text"
)

// Formatter is a dumb way to inject custom formatting logic for column data.
// This can be useful for outputting prefix/suffixes or other basic translations
type Formatter interface {
	// Format formats as source string into a new string
	Format(string) string
}

// BasicFormatter uses fmt.Sprintf to format based on a static format string
type BasicFormatter struct {
	FmtString string
}

// Format formats value by applying format with fmt.Sprintf
func (bf BasicFormatter) Format(value string) string {
	return fmt.Sprintf(bf.FmtString, value)
}

// FuncFormatter wraps a user-defined function to apply formatting
type FuncFormatter func(string) string

// Format formats the value by calling the external function
func (ff FuncFormatter) Format(value string) string {
	return ff(value)
}

// AllFormats contains all the formats supported
var AllFormats = []string{CSVFormat, TableFormat, TextFormat}

// Writer writes structured data to an internal buffer and outputs it as a given format when flushed
type Writer struct {
	size       int
	basew      io.Writer
	csvw       *csv.Writer
	table      *tablewriter.Table
	str        strings.Builder
	strw       *bufio.Writer
	formatters map[string][]Formatter
	columns    []string
	format     string
	err        error
}

// Option modifies default options of the Writer
type Option func(*Writer)

// WithFormatter sets the text formatting for the column
func WithFormatter(column string, f Formatter) Option {
	return func(w *Writer) {
		formatters, ok := w.formatters[column]
		if !ok {
			w.formatters[column] = []Formatter{f}
			return
		}
		w.formatters[column] = append(formatters, f)
	}
}

// WithSize modifies the size of the internal buffer
func WithSize(size int) Option {
	return func(w *Writer) {
		w.size = size
	}
}

// New returns a new Writer for writing. Format should be one of AllFormats.
// The size value determines how big the internal buffer should be. When the
// buffer fills, the writer automatically flushes it.
func New(writer io.Writer, columns []string, format string, opts ...Option) *Writer {
	table := tablewriter.NewWriter(writer)
	table.SetHeader(columns)
	csvw := csv.NewWriter(writer)
	csvw.Write(columns)
	w := &Writer{
		basew:      writer,
		size:       defaultSize,
		csvw:       csvw,
		table:      table,
		formatters: map[string][]Formatter{},
		columns:    columns,
		format:     format,
	}
	for _, o := range opts {
		o(w)
	}
	w.strw = bufio.NewWriterSize(writer, w.size)
	return w
}

// Write writes the record to the internal buffer
func (w *Writer) Write(record []string) error {
	recordFormatted := w.formatRecord(record)
	switch w.format {
	case CSVFormat:
		if err := w.csvw.Write(recordFormatted); err != nil {
			w.err = multierror.Append(w.err, fmt.Errorf("error writing record to csv: %s", err))
			return err
		}
	case TableFormat:
		w.table.Append(recordFormatted)
	case TextFormat:
		w.str.WriteString("---\n")
		for i, v := range recordFormatted {
			w.str.WriteString(fmt.Sprintf("%s: %s\n", w.columns[i], v))
		}
		w.strw.WriteString(w.str.String())
		w.str.Reset()
	}
	return nil
}

// Flush flushes all records from the internal buffer to its output writer
func (w *Writer) Flush() {
	switch w.format {
	case CSVFormat:
		w.csvw.Flush()
		if err := w.csvw.Error(); err != nil {
			w.err = multierror.Append(w.err, fmt.Errorf("error flushing csv: %s", err))
		}
		break
	case TextFormat:
		if err := w.strw.Flush(); err != nil {
			w.err = multierror.Append(w.err, fmt.Errorf("error flushing text: %s", err))
		}
		w.strw.Reset(w.basew)
		w.str.Reset()
	case TableFormat:
		w.table.Render()
		w.table.ClearRows()
	}
}

// Error returns whether there was an error writing.
func (w *Writer) Error() error {
	return w.err
}

// formatRecord applies column formatters to column values
func (w *Writer) formatRecord(record []string) []string {
	final := make([]string, len(record))
	for i, val := range record {
		colName := w.columns[i]
		formatters := w.formatters[colName]
		for _, formatter := range formatters {
			val = formatter.Format(val)
		}
		final[i] = val
	}
	return final
}
