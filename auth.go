package main

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/thecsw/katya/storage"
	"github.com/thecsw/katya/utils"
)

var (
	// attemptCooldown is how many times you have between bad logins
	attemptCooldown = 14 * time.Minute
	// badLoginAttempts caches users' bad login attempts
	badLoginAttempts = cache.New(attemptCooldown, attemptCooldown)

	// usernameRegexp is a regex that every username should follow
	usernameRegexp = regexp.MustCompile(`^[-a-zA-Z0-9]{3,16}$`)
	// passwordRegexp is a regex that every password should follow
	passwordRegexp = regexp.MustCompile(`^[^ ]{2,32}$`)
)

// ContextKey is a type alias to string
type ContextKey string

// loggingMiddleware does a full validation AND authentication for a Basic Auth attempt
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ipAddr, err := utils.ExtractIP(r)
		if err != nil {
			httpJSON(w, nil, http.StatusBadRequest, errors.New("unknown origin"))
			return
		}
		if number, found := badLoginAttempts.Get(ipAddr); found && number.(uint) >= 4 {
			httpJSON(w, nil, http.StatusForbidden, errors.New("origin blocked"))
			return
		}
		tokens := strings.Split(r.Header.Get("Authorization"), "Basic ")
		if len(tokens) != 2 {
			httpJSON(w, nil, http.StatusBadRequest, errors.New("no basic auth provided"))
			return
		}
		decoded, err := base64.StdEncoding.DecodeString(tokens[1])
		if err != nil {
			httpJSON(w, nil, http.StatusBadRequest, errors.New("base64 decoding failed"))
			return
		}
		credentials := strings.Split(string(decoded), ":")
		if len(credentials) != 2 {
			httpJSON(w, nil, http.StatusBadRequest, errors.New("basic auth is malformed"))
			return
		}
		user, pass := credentials[0], credentials[1]
		if user == "" {
			httpJSON(w, nil, http.StatusBadRequest, errors.New("bad user credentials"))
			return
		}
		// Quickly sanitize the username and the password
		if !usernameRegexp.MatchString(user) || !passwordRegexp.MatchString(pass) {
			httpJSON(w, nil, http.StatusBadRequest, errors.New("bad user credentials"))
			return
		}
		foundUser, err := storage.GetUser(user, true)
		if err != nil || foundUser.Name == "" {
			httpJSON(w, nil, http.StatusForbidden, errors.New("bad user credentials"))
			return
		}
		if foundUser.Password != utils.ShaEncode(pass) {
			httpJSON(w, nil, http.StatusForbidden, errors.New("bad user credentials"))
			// Someone is maybe trying to guess the password
			_ = badLoginAttempts.Add(ipAddr, uint(0), cache.DefaultExpiration)
			_, _ = badLoginAttempts.IncrementUint(ipAddr, 1)
			return
		}
		userCookie := http.Cookie{
			Name:     "user",
			Value:    "Basic " + tokens[1],
			Domain:   "katya-api.sandyuraz.com",
			Expires:  time.Now().Add(72 * time.Hour),
			MaxAge:   2592000,
			Secure:   true,
			HttpOnly: false,
		}
		http.SetCookie(w, &userCookie)
		newContext := context.WithValue(context.TODO(), ContextKey("user"), *foundUser)
		next.ServeHTTP(w, r.WithContext(newContext))
	})
}

// verifyAuth verifies that the credentials are OK
func verifyAuth(w http.ResponseWriter, r *http.Request) {
	httpJSON(w, httpMessageReturn{Message: "OK"}, http.StatusOK, nil)
}
