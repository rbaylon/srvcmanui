package loginguard

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rbaylon/srvcman/database"
)

// Challenge is a math question for the login form: a three-number
// expression using multiplication, division, addition and subtraction, to
// be answered with the usual order of operations (MDAS).
type Challenge struct {
	Question string
	Token    string
}

const captchaTTL = 5 * time.Minute

var opSymbol = map[byte]string{'*': "×", '/': "÷", '+': "+", '-': "−"}

func randN(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		// crypto/rand failing means the OS entropy source is broken; there
		// is no safe fallback for a security check.
		panic("loginguard: crypto/rand: " + err.Error())
	}
	return int(v.Int64())
}

// evaluate applies MDAS to nums[0] ops[0] nums[1] ops[1] nums[2]:
// multiplication and division first, left to right, then addition and
// subtraction. It reports false for a division that is not exact.
func evaluate(nums [3]int, ops [2]byte) (int, bool) {
	vals := []int{nums[0]}
	var addOps []byte
	for i, op := range ops {
		n := nums[i+1]
		switch op {
		case '*':
			vals[len(vals)-1] *= n
		case '/':
			last := vals[len(vals)-1]
			if n == 0 || last%n != 0 {
				return 0, false
			}
			vals[len(vals)-1] = last / n
		default:
			addOps = append(addOps, op)
			vals = append(vals, n)
		}
	}
	res := vals[0]
	for i, op := range addOps {
		if op == '+' {
			res += vals[i+1]
		} else {
			res -= vals[i+1]
		}
	}
	return res, true
}

// NewChallenge builds a question and the token that proves, later, that
// it was this server that asked it. The answer is not in the token: the
// token carries a MAC over (nonce, expiry, answer), so the browser cannot
// read the answer out of it and only the right answer reproduces the MAC.
func NewChallenge() (Challenge, error) {
	ops := [4]byte{'*', '/', '+', '-'}
	for tries := 0; tries < 1000; tries++ {
		var nums [3]int
		for i := range nums {
			nums[i] = 2 + randN(11) // 2..12
		}
		opPair := [2]byte{ops[randN(4)], ops[randN(4)]}
		answer, ok := evaluate(nums, opPair)
		// Keep the answer a small non-negative whole number: easy to type,
		// and no unicode minus for a user to get wrong.
		if !ok || answer < 0 || answer > 150 {
			continue
		}
		q := strconv.Itoa(nums[0]) + " " + opSymbol[opPair[0]] + " " +
			strconv.Itoa(nums[1]) + " " + opSymbol[opPair[1]] + " " +
			strconv.Itoa(nums[2])

		raw := make([]byte, 16, 16+sha256.Size)
		if _, err := rand.Read(raw[:8]); err != nil {
			return Challenge{}, err
		}
		binary.BigEndian.PutUint64(raw[8:], uint64(time.Now().Add(captchaTTL).Unix()))
		raw = append(raw, sign(raw[:8], raw[8:16], answer)...)
		return Challenge{Question: q, Token: base64.RawURLEncoding.EncodeToString(raw)}, nil
	}
	return Challenge{}, errors.New("loginguard: could not generate a challenge")
}

func captchaKey() []byte {
	sum := sha256.Sum256([]byte("billingportal/login-captcha/v1\x00" + database.GetEnvVariable("APP_SESSION_KEY")))
	return sum[:]
}

func sign(nonce, exp []byte, answer int) []byte {
	m := hmac.New(sha256.New, captchaKey())
	m.Write(nonce)
	m.Write(exp)
	m.Write([]byte(strconv.Itoa(answer)))
	return m.Sum(nil)
}

var (
	usedMu sync.Mutex
	used   = map[string]time.Time{} // nonce -> expiry, so a token works once
)

// claim marks a nonce as spent, reporting false if it already was.
func claim(nonce string, exp time.Time) bool {
	usedMu.Lock()
	defer usedMu.Unlock()
	now := time.Now()
	if len(used) > 1000 {
		for k, e := range used {
			if now.After(e) {
				delete(used, k)
			}
		}
	}
	if _, seen := used[nonce]; seen {
		return false
	}
	used[nonce] = exp
	return true
}

// VerifyChallenge checks an answer against a token from NewChallenge. The
// token is spent by the first check that reaches it, right or wrong, so one
// question cannot be answered by trying every number: a failed attempt
// needs a fresh question.
func VerifyChallenge(token, answer string) bool {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil || len(raw) != 16+sha256.Size {
		return false
	}
	exp := time.Unix(int64(binary.BigEndian.Uint64(raw[8:16])), 0)
	if time.Now().After(exp) {
		return false
	}
	if !claim(string(raw[:8]), exp) {
		return false
	}
	n, err := strconv.Atoi(strings.TrimSpace(answer))
	if err != nil {
		return false
	}
	return hmac.Equal(raw[16:], sign(raw[:8], raw[8:16], n))
}
