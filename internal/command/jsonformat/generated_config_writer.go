package jsonformat

import (
	"fmt"
	"io"
)

type generatedConfigWriter struct {
	createGeneratedConfigWriter CreateGeneratedConfigWriterFn

	writer    io.Writer
	closer    func() error
	writerErr error

	failed map[string]error
}

func newGeneratedConfigWriter(renderer Renderer) *generatedConfigWriter {
	return &generatedConfigWriter{
		createGeneratedConfigWriter: renderer.CreateGeneratedConfigWriter,
		failed:                      make(map[string]error),
	}
}

// MaybeWriteGeneratedConfig checks the provided diff for any generated config
// and attempts to write into the writer provided by the renderer.
func (writer *generatedConfigWriter) MaybeWriteGeneratedConfig(diff diff) bool {
	if len(diff.change.Change.GeneratedConfig) == 0 {
		// Then we have nothing to write out, so for simplicity we'll just
		// report this as a success.
		return true
	}

	if writer.writerErr != nil {
		// We couldn't create the writer previously, no point trying again just
		// mark this as failed and move on.
		writer.failed[diff.change.Address] = nil
		return false
	}

	w, wErr := writer.GetWriter()
	if wErr != nil {
		// This means we tried to create the writer and couldn't. The error has
		// been saved in the generatedConfigWriter so we'll just mark this
		// change as failed without a reason and move on.
		writer.failed[diff.change.Address] = nil
		return false
	}

	var header string
	if diff.change.Change.Importing != nil && len(diff.change.Change.Importing.ID) > 0 {
		header = fmt.Sprintf("\n# __generated__ by Terraform from %q\n", diff.change.Change.Importing.ID)
	} else {
		header = "\n# __generated__ by Terraform\n"
	}

	_, err := w.Write([]byte(fmt.Sprintf("%s%s\n", header, diff.change.Change.GeneratedConfig)))
	if err != nil {
		writer.failed[diff.change.Address] = err
		return false
	}

	return true
}

func (writer *generatedConfigWriter) GetWriter() (io.Writer, error) {
	if writer.writerErr != nil {
		return nil, writer.writerErr
	}
	if writer.writer != nil {
		return writer.writer, nil
	}
	writer.writer, writer.closer, writer.writerErr = writer.createGeneratedConfigWriter()
	return writer.writer, writer.writerErr
}

func (writer *generatedConfigWriter) Close() error {
	if writer.closer != nil {
		return writer.closer()
	}
	return nil
}

func (writer *generatedConfigWriter) Failed() (map[string]error, error) {
	return writer.failed, writer.writerErr
}
