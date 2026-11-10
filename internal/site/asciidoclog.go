package site

import (
	"context"
	"log/slog"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
)

// asciidocLog sends Asciidoctor's own messages (a section level skipped, an
// unknown attribute, ...) to the run's logger as records (CLI-6), instead of
// the raw lines Asciidoctor writes to stderr by default. Each record carries
// the page's element and where the line is in the model (MC-19's
// file:line): a representation is converted from a string, so Asciidoctor
// knows only the line in its body.
//
// It is asciidoctorg's SlogHandler with those two attributes in place of
// the body-relative cursor. Asciidoctor logs while a representation is
// converted, so the renderer's current page and representation are the ones
// the message is about.
type asciidocLog struct {
	r      *renderer
	logger *slog.Logger
}

var _ asciidoctor.ContextLogHandler = asciidocLog{}

func (h asciidocLog) Log(record asciidoctor.LogRecord) {
	h.LogContext(context.Background(), record)
}

func (h asciidocLog) LogContext(ctx context.Context, record asciidoctor.LogRecord) {
	level := record.Severity.SlogLevel()
	if !h.logger.Enabled(ctx, level) {
		return
	}
	var attrs []slog.Attr
	if n := h.r.currentPage; n != nil {
		attrs = append(attrs, slog.String("element", n.QualifiedName))
	}
	if c, rep := record.Cursor, h.r.current; c != nil && c.LineNumber > 0 && rep != nil {
		if loc := h.r.sources.lineOf(*rep, c.LineNumber); loc != "" {
			attrs = append(attrs, slog.String("location", loc))
		}
	}
	h.logger.LogAttrs(ctx, level, "asciidoc: "+record.Message, attrs...)
}
