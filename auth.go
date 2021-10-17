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
)

var (
	attemptCooldown  = 14 * time.Minute
	badLoginAttempts = cache.New(attemptCooldown, attemptCooldown)

	usernameRegexp = regexp.MustCompile(`^[-a-zA-Z0-9]{3,16}$`)
	passwordRegexp = regexp.MustCompile(`^[^ ]{2,32}$`)
)

// ContextKey is a type alias to string
type ContextKey string

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ipAddr, err := extractIP(r)
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
		creds := strings.Split(string(decoded), ":")
		if len(creds) != 2 {
			httpJSON(w, nil, http.StatusBadRequest, errors.New("basic auth is malformed"))
			return
		}
		user, pass := creds[0], creds[1]
		if user == "" {
			httpJSON(w, nil, http.StatusBadRequest, errors.New("bad user credentials"))
			return
		}
		// Quickly sanitize the username and the password
		if !usernameRegexp.MatchString(user) || !passwordRegexp.MatchString(pass) {
			httpJSON(w, nil, http.StatusBadRequest, errors.New("bad user credentials"))
			return
		}
		foundUser, err := getUser(user, true)
		if err != nil || foundUser.Name == "" {
			httpJSON(w, nil, http.StatusForbidden, errors.New("bad user credentials"))
			return
		}
		if foundUser.Password != shaEncode(pass) {
			httpJSON(w, nil, http.StatusForbidden, errors.New("bad user credentials"))
			// Someone is maybe trying to guess the password
			badLoginAttempts.Add(ipAddr, uint(0), cache.DefaultExpiration)
			badLoginAttempts.IncrementUint(ipAddr, 1)
			return
		}
		newContext := context.WithValue(context.TODO(), ContextKey("user"), *foundUser)
		next.ServeHTTP(w, r.WithContext(newContext))
	})
}
