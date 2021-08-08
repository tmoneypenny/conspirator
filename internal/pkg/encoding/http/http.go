package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// Protocol contains the major and minor version
// used in the request
type Protocol struct {
	ProtoMajor int
	ProtoMinor int
}

// protoAtLeast does a simple check to verify
// the HTTP protocol is at least 1.1... because
// you never know what you're gonna get.
func protoAtLeast(major, minor int) bool {
	return major > 1 || major == 1 && minor >= 1
}

// bodyAllowedForStatus reports whether a given response status code
// permits a body. See RFC 7230, section 3.3.
func bodyAllowedForStatus(status int) bool {
	switch {
	case status >= 100 && status <= 199:
		return false
	case status == 204:
		return false
	case status == 304:
		return false
	}
	return true
}

// Write writes a an HTTP request, which is the header and body,
// in wire format.
func WriteRequest(request *http.Request) []byte {
	var b bytes.Buffer
	request.Write(&b)

	// Client will always use HTTP/1.1 or HTTP/2
	if request.Proto == "HTTP/2.0" {
		writeHTTP2(&b)
	}

	return b.Bytes()
}

// Write writes an HTTP response, which is the header and body,
// in wire format.
func WriteResponse(response *echo.Response, proto Protocol, body []byte) []byte {
	var b bytes.Buffer
	var closeConnection bool = true
	var writeDate bool = true
	// Write status header in the format HTTP/Major.Minor <status> <status_txt>
	if _, err := fmt.Fprintf(&b, "HTTP/%d.%d %03d %s\r\n",
		proto.ProtoMajor,
		proto.ProtoMinor,
		response.Status,
		http.StatusText(response.Status)); err != nil {
		log.Error().Msg("failed to write status line")
		return nil
	}

	// create a clone of the headers to work with
	r1 := new(http.Header)
	*r1 = response.Header().Clone()

	// Write headers
	for h := range *r1 {
		if _, err := fmt.Fprintf(&b, "%s: %s\r\n", h, r1.Get(h)); err != nil {
			log.Error().Msg("failed to write header")
			return nil
		}
	}

	// Write content-length
	if response.Size != 0 && bodyAllowedForStatus(response.Status) {
		if _, err := fmt.Fprintf(&b, "Content-Length: %d\r\n", response.Size); err != nil {
			log.Error().Msg("failed to write content length")
			return nil
		}
	} else if response.Size == 0 && bodyAllowedForStatus(response.Status) {
		if _, err := io.WriteString(&b, "Content-Length: 0\r\n"); err != nil {
			log.Error().Msg("failed to write content length")
			return nil
		}
	}

	// Write Date -- optional
	if writeDate {
		if _, err := fmt.Fprintf(&b, "Date: %s\r\n", time.Now().UTC().Format(time.UnixDate)); err != nil {
			log.Error().Msg("failed to write date")
			return nil
		}
	}

	// Write Close
	if closeConnection {
		if _, err := fmt.Fprint(&b, "Connection: close\r\n"); err != nil {
			log.Error().Msg("failed to write close")
			return nil
		}
	}

	// Write end-of-header
	if _, err := io.WriteString(&b, "\r\n"); err != nil {
		log.Error().Msg("failed to write end-of-header")
		return nil
	}

	if _, err := io.WriteString(&b, string(body)); err != nil {
		log.Error().Msg("failed to write body")
		return nil
	}

	log.Debug().Msgf("Writing response: %s", b.String())

	return b.Bytes()
}

// writeHTTP2 overwrites the Proto in the default
// http.Request.Write() method
func writeHTTP2(b *bytes.Buffer) {
	*b = *bytes.NewBuffer(bytes.Replace(b.Bytes(), []byte("HTTP/1.1"), []byte("HTTP/2.0"), 1))
}
