package apiv1

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"
)

// parseHeaders takes a string from formValue and converts it
// into a map to be used with Header().Add(k, v)
func parseHeaders(headers string) map[string]string {
	hMap := make(map[string]string)

	if headers != "" {
		for _, h := range strings.Split(headers, "\r\n") {
			hS := strings.SplitN(h, ":", 2)
			if len(hS) == 2 {
				hMap[hS[0]] = hS[1]
			} else {
				hMap[hS[0]] = ""
			}
		}
	}

	return hMap
}

// parseUrl validates the URL
func parseUrl(urlPath string) string {
	u, err := url.Parse(urlPath)
	if err != nil {
		log.Error().Msg("could not parse urlPath")
	}

	if !strings.HasPrefix(u.Path, "/") {
		return fmt.Sprintf("/%s", u.Path)
	}

	return u.Path
}

// parseMethods parses the selected method options. It is
// necessary to add methods individually, since Router().Add()
// does not support the "ANY" type
func parseMethods(m []byte) []string {
	return strings.Split(string(m), ",")
}
