// Package loginguard defends the sign-in form against guessing: a rate
// limit and a math challenge (MDAS), both implemented here with the
// standard library only.
//
// Three limits apply to POST /ui/users/login:
//
//   - every submission counts against the client address (10 a minute), so
//     the form cannot be hammered at all, and neither can the challenge;
//   - failures count against the client address (8 per 15 minutes);
//   - failures count against the username (10 per 15 minutes), which is
//     what stops a guess spread across many addresses. The trade-off is
//     that someone can deliberately lock a known username out for a while;
//     the window is kept short for that reason.
//
// A failed challenge counts against the address but not the username: it
// says nothing about whoever owns that username.
package loginguard

import (
	"strings"
	"time"
)

var (
	ipAttempts   = NewLimiter(10, time.Minute)
	ipFailures   = NewLimiter(8, 15*time.Minute)
	userFailures = NewLimiter(10, 15*time.Minute)
)

func userKey(username string) string {
	u := strings.ToLower(strings.TrimSpace(username))
	if len(u) > 64 {
		u = u[:64]
	}
	return u
}

// Begin is called at the start of every login submission. It reports
// whether the attempt may proceed and, if not, how long to wait; an
// allowed attempt is counted.
func Begin(ip, username string) (bool, time.Duration) {
	if blocked, wait := ipAttempts.Blocked(ip); blocked {
		return false, wait
	}
	ipAttempts.Record(ip)
	if blocked, wait := ipFailures.Blocked(ip); blocked {
		return false, wait
	}
	if u := userKey(username); u != "" {
		if blocked, wait := userFailures.Blocked(u); blocked {
			return false, wait
		}
	}
	return true, 0
}

// Fail records a rejected attempt. Pass an empty username when the
// failure says nothing about the account (a wrong challenge answer).
func Fail(ip, username string) {
	ipFailures.Record(ip)
	if u := userKey(username); u != "" {
		userFailures.Record(u)
	}
}

// Succeed clears the username's failures after a good sign-in. The
// address's own failures are left to age out: a shared address (an office,
// a carrier NAT) should not be able to wipe an attacker's count by signing
// in once.
func Succeed(username string) {
	if u := userKey(username); u != "" {
		userFailures.Reset(u)
	}
}
