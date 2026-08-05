package httpjson

import (
	"compress/gzip"
	stderrors "errors"
	"io"
	"net/http"

	"github.com/FacileStudio/tronc/errors"
)

var errDecompressedTooLarge = stderrors.New("decompressed body too large")

// DecodeGzipJSON reads a gzip-compressed JSON object from the request into dst.
//
// Two independent caps apply: the compressed body is bounded by MaxBodyBytes,
// and the decompressed stream by maxDecompressedBytes. Both are needed — a
// small compressed body can expand without limit, so bounding only the request
// is a decompression bomb.
func DecodeGzipJSON(w http.ResponseWriter, request *http.Request, dst any, maxDecompressedBytes int64) error {
	return DecodeGzipJSONLimit(w, request, dst, MaxBodyBytes, maxDecompressedBytes)
}

// DecodeGzipJSONLimit is DecodeGzipJSON with both caps given explicitly.
func DecodeGzipJSONLimit(w http.ResponseWriter, request *http.Request, dst any, maxCompressedBytes, maxDecompressedBytes int64) error {
	defer func() { _ = request.Body.Close() }()
	request.Body = http.MaxBytesReader(w, request.Body, maxCompressedBytes)

	decompressed, err := gzip.NewReader(request.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if stderrors.As(err, &maxBytesErr) {
			return errors.TooLarge("request body too large")
		}
		return errors.Invalid("invalid gzip body")
	}
	defer func() { _ = decompressed.Close() }()

	return decodeStrict(&cappedReader{reader: decompressed, remaining: maxDecompressedBytes}, dst)
}

type cappedReader struct {
	reader    io.Reader
	remaining int64
}

func (c *cappedReader) Read(p []byte) (int, error) {
	n, err := c.reader.Read(p)
	c.remaining -= int64(n)
	if c.remaining < 0 {
		return n, errDecompressedTooLarge
	}
	return n, err
}
