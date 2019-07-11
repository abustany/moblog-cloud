package adminserver

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/pkg/errors"

	"github.com/abustany/moblog-cloud/pkg/sessionstore"
)

const AuthCookieName = "auth"

type AuthCookie struct {
	SessionID string
}

func EncodeAuthCookie(sc *securecookie.SecureCookie, cookie AuthCookie) (http.Cookie, error) {
	encoded, err := sc.Encode(AuthCookieName, &cookie)

	if err != nil {
		return http.Cookie{}, errors.Wrap(err, "Error while encoding auth cookie")
	}

	return http.Cookie{
		Name:     AuthCookieName,
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
	}, nil
}

func ResetAuthCookie() http.Cookie {
	return http.Cookie{
		Name:     AuthCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
	}
}

type contextKey int

const (
	sessionKey contextKey = iota
)

func sessionFromRequest(sc *securecookie.SecureCookie, sessionStore sessionstore.SessionStore, r *http.Request) (*sessionstore.Session, error) {
	authCookie, err := r.Cookie(AuthCookieName)

	if err == http.ErrNoCookie {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "Error while retrieving auth cookie")
	}

	var decoded AuthCookie

	if err := sc.Decode(AuthCookieName, authCookie.Value, &decoded); err != nil {
		// Probably wrong keys
		return nil, nil
	}

	session, err := sessionStore.Get(decoded.SessionID)

	if err != nil {
		return nil, errors.Wrapf(err, "Error while retrieving session %s", decoded.SessionID)
	}

	return session, nil
}

func WithSession(sc *securecookie.SecureCookie, sessionStore sessionstore.SessionStore, requireAuth bool, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := sessionFromRequest(sc, sessionStore, r)

		if err != nil {
			log.Printf("Error while decoding session: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if session == nil {
			if requireAuth {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		} else {
			r = r.WithContext(context.WithValue(r.Context(), sessionKey, session))
		}

		handler.ServeHTTP(w, r)
	})
}

func SessionFromContext(ctx context.Context) *sessionstore.Session {
	if value := ctx.Value(sessionKey); value != nil {
		return value.(*sessionstore.Session)
	}

	return nil
}
