package powerrouter

import (
	"strings"
	"testing"
	"time"
)

var testKey = []byte("test-session-key")

func TestTokenRoundTrip(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	tok := makeToken(testKey, "admin", now)
	if !validToken(testKey, "admin", tok, now.Add(time.Minute)) {
		t.Error("a fresh token for the same user was rejected")
	}
	if !validToken(testKey, "admin", tok, now.Add(tokenTTL)) {
		t.Error("a token exactly at its lifetime was rejected")
	}
}

// Everything that is not "this user was shown this page a moment ago" must be
// refused: restarting the machine is the one thing here that cannot be undone.
func TestTokenRejectsEverythingElse(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	tok := makeToken(testKey, "admin", now)
	ts, sig, _ := strings.Cut(tok, ".")

	cases := map[string]struct {
		user, tok string
		at        time.Time
	}{
		"empty token":          {"admin", "", now},
		"no separator":         {"admin", "garbage", now},
		"another user":         {"operator", tok, now},
		"no user (signed out)": {"", makeToken(testKey, "", now), now},
		"expired":              {"admin", tok, now.Add(tokenTTL + time.Second)},
		"from the future":      {"admin", tok, now.Add(-time.Minute)},
		"signature changed":    {"admin", ts + "." + strings.Repeat("0", len(sig)), now},
		"time changed":         {"admin", "1800000500." + sig, now.Add(time.Minute)}, // a later time with the old signature
		"time not a number":    {"admin", "abc." + sig, now},
		"truncated signature":  {"admin", ts + "." + sig[:10], now},
	}
	for name, c := range cases {
		if validToken(testKey, c.user, c.tok, c.at) {
			t.Errorf("%s: token accepted, want it refused", name)
		}
	}
	// A token made with some other key (another install, or an attacker's
	// guess) is not valid here.
	if validToken([]byte("a different key"), "admin", tok, now) {
		t.Error("token signed with a different key was accepted")
	}
}

func TestTokensDifferPerUserAndPerTime(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	if makeToken(testKey, "a", now) == makeToken(testKey, "b", now) {
		t.Error("two users share a token")
	}
	if makeToken(testKey, "a", now) == makeToken(testKey, "a", now.Add(time.Second)) {
		t.Error("tokens do not change over time")
	}
}
