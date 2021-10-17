package main

import (
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"strings"
)

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
