package main

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
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

	csvHeader = []string{
		"left", "center", "right", "start_url", "url",
	}
)

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
		next.ServeHTTP(w, r)
	})
}

// extractIP makes sure the request has a proper request IP.
func extractIP(r *http.Request) (string, error) {
	// if not a proper remote addr, return empty
	if !strings.ContainsRune(r.RemoteAddr, ':') {
		return "", errors.New("lol")
	}
	ipAddr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || ipAddr == "" {
		return "", errors.New("request has failed origin validation, retry")
	}
	return ipAddr, nil
}

// shaEncode return SHA512 sum of a string.
func shaEncode(input string) string {
	sha := sha512.Sum512([]byte(input))
	return hex.EncodeToString(sha[:])
}

// verifyAuth verifies that the credentials are OK
func verifyAuth(w http.ResponseWriter, r *http.Request) {
	httpJSON(w, httpMessageReturn{Message: "OK"}, http.StatusOK, nil)
}

// httpJSON is a generic http object passer.
func httpJSON(w http.ResponseWriter, data interface{}, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if err != nil && status >= 400 && status < 600 {
		json.NewEncoder(w).Encode(httpErrorReturn{Error: err.Error()})
		return
	}
	json.NewEncoder(w).Encode(data)
}

func httpCSV(w http.ResponseWriter, results []SearchResult, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	toWrite := make([][]string, 0, len(results)+1)
	toWrite = append(toWrite, csvHeader)
	for _, v := range results {
		toWrite = append(toWrite, []string{
			v.Left, v.Center, v.Right, v.Source, v.URL,
		})
	}
	csv.NewWriter(w).WriteAll(toWrite)
}

// httpHTML sends a good HTML response.
func httpHTML(w http.ResponseWriter, data interface{}) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, data)
}

// httpMessageReturn defines a generic HTTP return message.
type httpMessageReturn struct {
	Message interface{} `json:"message"`
}

// httpErrorReturn defines a generic HTTP error message.
type httpErrorReturn struct {
	Error string `json:"error"`
}
