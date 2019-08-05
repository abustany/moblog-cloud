package netscapecookies

import (
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

func boolStr(b bool) string {
	if b {
		return "TRUE"
	} else {
		return "FALSE"
	}
}

var errNoDomain = errors.New("Cookie has no domain")
var errNoExpires = errors.New("Cookie has no expiration time")
var errNoName = errors.New("Cookie has no name")
var errNoValue = errors.New("Cookie has no value")

func WriteCookie(w io.Writer, cookie *http.Cookie) error {
	if cookie.Domain == "" {
		return errNoDomain
	}

	if cookie.Expires.IsZero() {
		return errNoExpires
	}

	if cookie.Name == "" {
		return errNoName
	}

	if cookie.Value == "" {
		return errNoValue
	}

	path := cookie.Path

	if path == "" {
		path = "/"
	}

	_, err := fmt.Fprintf(w,
		"%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
		cookie.Domain,
		boolStr(false),
		path,
		boolStr(cookie.Secure),
		cookie.Expires.Unix(),
		cookie.Name,
		cookie.Value)

	return err
}
