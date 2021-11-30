package utils

import (
	"net/http"
	"testing"
)

func TestExtractIP(t *testing.T) {
	type args struct {
		r *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractIP(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractIP() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShaEncode(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"sandy", args{"sandy"}, "8fe9d034afd54b2e9a72c481b846003e1367b630e6ce6899adc934ba1d2163d3e22d0305c343b0cee58eaa223bd90b995d3b16e14653afc96053f3ad52083891"},
		{"amadeus", args{"amadeus"}, "9b97c6af8a07ee847836656cfd269f190468c890e359426ff5fa8ef465391ebada0fd1c0e56b2e43589789482b00a749abcc19c348671d861a5f050f911663d5"},
		{"helltaker", args{"i love helltaker"}, "3fb3e2a84782c586a3700d20c34336acc3bfcd80382827dc1414d265af3e88838b835b49c38b031f46754dcb54addff086a1c6bb35a928db0a30163f6df78747"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShaEncode(tt.args.input); got != tt.want {
				t.Errorf("ShaEncode() = %v, want %v", got, tt.want)
			}
		})
	}
}
