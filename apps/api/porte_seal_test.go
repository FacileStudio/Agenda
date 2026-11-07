package main

import (
	"strings"
	"testing"
)

func TestNewPorteStoreKey(t *testing.T) {
	cases := []struct {
		name, key string
		wantErr   bool
	}{
		{"unset", "", false},
		{"hex", strings.Repeat("ab", 32), false},
		{"rotation", strings.Repeat("ab", 32) + "," + strings.Repeat("cd", 32), false},
		{"short", "abcd", true},
		{"garbage", "not-a-key", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(porteTokenKeyEnv, tc.key)
			store, err := newPorteStore(nil)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				if strings.Contains(err.Error(), tc.key) {
					t.Fatal("error leaks the key")
				}
				return
			}
			if err != nil || store == nil {
				t.Fatalf("store=%v err=%v", store, err)
			}
		})
	}
}
