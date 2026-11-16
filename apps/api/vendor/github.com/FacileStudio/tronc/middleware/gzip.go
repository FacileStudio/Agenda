package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"sync"
)

var gzipPool = sync.Pool{
	New: func() any {
		writer, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		return writer
	},
}

// Gzip compresses text responses (text/*, JSON, XML, JavaScript, SVG) for
// clients that send Accept-Encoding: gzip.
//
// It leaves a response alone when the handler already set Content-Encoding,
// when the body is HEAD, 1xx, 204 or 304, when the type is not compressible
// (images, archives, event streams), and when the request is a Range or an
// Upgrade, so downloads, SSE and WebSocket routes pass through untouched.
// It sets Vary: Accept-Encoding, drops the stale Content-Length, and keeps
// Flush working on compressed responses.
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if !acceptsGzip(request) {
			next.ServeHTTP(w, request)
			return
		}
		w.Header().Add("Vary", "Accept-Encoding")
		if request.Header.Get("Range") != "" || request.Header.Get("Upgrade") != "" {
			next.ServeHTTP(w, request)
			return
		}
		wrapped := &gzipResponseWriter{ResponseWriter: w, head: request.Method == http.MethodHead}
		defer wrapped.finish()
		next.ServeHTTP(wrapped, request)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gz      *gzip.Writer
	head    bool
	decided bool
}

func (g *gzipResponseWriter) Unwrap() http.ResponseWriter { return g.ResponseWriter }

func (g *gzipResponseWriter) WriteHeader(status int) {
	g.decide(status, nil)
	g.ResponseWriter.WriteHeader(status)
}

func (g *gzipResponseWriter) Write(data []byte) (int, error) {
	g.decide(http.StatusOK, data)
	if g.gz == nil {
		return g.ResponseWriter.Write(data)
	}
	return g.gz.Write(data)
}

func (g *gzipResponseWriter) Flush() {
	g.decide(http.StatusOK, nil)
	if g.gz != nil {
		_ = g.gz.Flush()
	}
	_ = http.NewResponseController(g.ResponseWriter).Flush()
}

func (g *gzipResponseWriter) decide(status int, first []byte) {
	if g.decided {
		return
	}
	g.decided = true
	header := g.Header()
	if g.head || status < 200 || status == http.StatusNoContent || status == http.StatusNotModified ||
		header.Get("Content-Encoding") != "" {
		return
	}
	contentType := header.Get("Content-Type")
	if contentType == "" && len(first) > 0 {
		contentType = http.DetectContentType(first)
	}
	if !compressible(contentType) {
		return
	}
	header.Set("Content-Encoding", "gzip")
	header.Del("Content-Length")
	g.gz, _ = gzipPool.Get().(*gzip.Writer)
	g.gz.Reset(g.ResponseWriter)
}

func (g *gzipResponseWriter) finish() {
	if g.gz == nil {
		return
	}
	_ = g.gz.Close()
	gzipPool.Put(g.gz)
}
