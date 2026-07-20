package report

import "io"

type errorWriter struct {
	writer io.Writer
	err    error
}

func newErrorWriter(writer io.Writer) *errorWriter {
	if checked, ok := writer.(*errorWriter); ok {
		return checked
	}
	return &errorWriter{writer: writer}
}

func (writer *errorWriter) Write(value []byte) (int, error) {
	if writer.err != nil {
		return 0, writer.err
	}
	written, err := writer.writer.Write(value)
	if err != nil {
		writer.err = err
	}
	return written, err
}

func (writer *errorWriter) Err() error { return writer.err }
