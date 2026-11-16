package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

func acceptsGzip(request *http.Request) bool {
	for _, part := range strings.Split(request.Header.Get("Accept-Encoding"), ",") {
		name, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if strings.EqualFold(strings.TrimSpace(name), "gzip") {
			params = strings.ReplaceAll(strings.ToLower(params), " ", "")
			quality, hasQuality := strings.CutPrefix(params, "q=")
			value, err := strconv.ParseFloat(quality, 64)
			return !hasQuality || err != nil || value > 0
		}
	}
	return false
}

func compressible(contentType string) bool {
	mediaType, _, _ := strings.Cut(strings.ToLower(contentType), ";")
	mediaType = strings.TrimSpace(mediaType)
	switch {
	case mediaType == "text/event-stream":
		return false
	case strings.HasPrefix(mediaType, "text/"):
		return true
	case strings.HasSuffix(mediaType, "+json"), strings.HasSuffix(mediaType, "+xml"):
		return true
	}
	switch mediaType {
	case "application/json", "application/xml", "application/javascript", "application/x-ndjson":
		return true
	}
	return false
}
