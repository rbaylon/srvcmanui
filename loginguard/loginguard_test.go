package loginguard

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// srvcman's database.GetEnvVariable loads ./.env and exits if it is
// missing, so the tests supply one (it is git-ignored) and clean it up.
func TestMain(m *testing.M) {
	_, statErr := os.Stat(".env")
	if os.IsNotExist(statErr) {
		os.WriteFile(".env", []byte("APP_SESSION_KEY=test-session-key-0123456789\n"), 0o600)
		code := m.Run()
		os.Remove(".env")
		os.Exit(code)
	}
	os.Exit(m.Run())
}

func TestEvaluateFollowsMDAS(t *testing.T) {
	cases := []struct {
		nums [3]int
		ops  [2]byte
		want int
		ok   bool
	}{
		{[3]int{2, 3, 4}, [2]byte{'+', '*'}, 14, true},  // not (2+3)*4
		{[3]int{10, 2, 3}, [2]byte{'-', '*'}, 4, true},  // not (10-2)*3
		{[3]int{12, 3, 2}, [2]byte{'/', '*'}, 8, true},  // left to right: (12/3)*2
		{[3]int{12, 4, 2}, [2]byte{'-', '/'}, 10, true}, // 12-(4/2)
		{[3]int{8, 2, 3}, [2]byte{'*', '-'}, 13, true},
		{[3]int{7, 2, 3}, [2]byte{'/', '+'}, 0, false}, // 7/2 is not whole
		{[3]int{9, 3, 2}, [2]byte{'+', '/'}, 0, false}, // 3/2 is not whole
	}
	for _, c := range cases {
		got, ok := evaluate(c.nums, c.ops)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("evaluate(%v,%q) = %d,%v; want %d,%v", c.nums, c.ops, got, ok, c.want, c.ok)
		}
	}
}

// solve re-derives the answer from the displayed question text, using only
// the MDAS rules, independently of how the challenge computed it.
func solve(q string) int {
	f := strings.Fields(q)
	nums := [3]int{}
	var ops [2]byte
	for i := 0; i < 3; i++ {
		nums[i], _ = strconv.Atoi(f[i*2])
	}
	for i := 0; i < 2; i++ {
		switch f[i*2+1] {
		case "×":
			ops[i] = '*'
		case "÷":
			ops[i] = '/'
		case "+":
			ops[i] = '+'
		case "−":
			ops[i] = '-'
		}
	}
	v, _ := evaluate(nums, ops)
	return v
}

func TestChallengeRoundTripAndSingleUse(t *testing.T) {
	seenOps := map[string]bool{}
	for i := 0; i < 300; i++ {
		ch, err := NewChallenge()
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range strings.Fields(ch.Question) {
			seenOps[s] = true
		}
		ans := solve(ch.Question)
		if ans < 0 || ans > 150 {
			t.Fatalf("answer %d out of range for %q", ans, ch.Question)
		}
		if !VerifyChallenge(ch.Token, strconv.Itoa(ans)) {
			t.Fatalf("correct answer %d rejected for %q", ans, ch.Question)
		}
		if VerifyChallenge(ch.Token, strconv.Itoa(ans)) {
			t.Fatal("token accepted twice")
		}
	}
	for _, op := range []string{"×", "÷", "+", "−"} {
		if !seenOps[op] {
			t.Errorf("operator %q never generated in 300 challenges", op)
		}
	}
}

func TestWrongAnswerBurnsTokenAndGarbageRejected(t *testing.T) {
	ch, _ := NewChallenge()
	ans := solve(ch.Question)
	if VerifyChallenge(ch.Token, strconv.Itoa(ans+1)) {
		t.Fatal("wrong answer accepted")
	}
	if VerifyChallenge(ch.Token, strconv.Itoa(ans)) {
		t.Fatal("token still usable after a wrong attempt")
	}
	for _, bad := range []string{"", "x", ch.Token[:10]} {
		if VerifyChallenge(bad, "1") {
			t.Fatalf("garbage token %q accepted", bad)
		}
	}
	ch2, _ := NewChallenge()
	if VerifyChallenge(ch2.Token, "abc") || VerifyChallenge(ch2.Token, "") {
		t.Fatal("non-numeric answer accepted")
	}
}

func TestTokenCannotBeForgedForAnotherAnswer(t *testing.T) {
	a, _ := NewChallenge()
	b, _ := NewChallenge()
	// b's token with a's answer must not verify unless the answers collide.
	ansA, ansB := solve(a.Question), solve(b.Question)
	if ansA != ansB && VerifyChallenge(b.Token, strconv.Itoa(ansA)) {
		t.Fatal("answer for one question accepted on another's token")
	}
}

func TestLimiter(t *testing.T) {
	l := NewLimiter(3, 100*time.Millisecond)
	for i := 0; i < 3; i++ {
		if b, _ := l.Blocked("k"); b {
			t.Fatalf("blocked after %d hits", i)
		}
		l.Record("k")
	}
	b, wait := l.Blocked("k")
	if !b || wait <= 0 || wait > 100*time.Millisecond {
		t.Fatalf("expected block with sane wait, got %v %v", b, wait)
	}
	if b, _ := l.Blocked("other"); b {
		t.Fatal("limit leaked across keys")
	}
	time.Sleep(120 * time.Millisecond)
	if b, _ := l.Blocked("k"); b {
		t.Fatal("still blocked after window")
	}
	l.Record("k")
	l.Reset("k")
	if b, _ := l.Blocked("k"); b {
		t.Fatal("blocked after reset")
	}
}

func TestBeginLocksOutRepeatedFailures(t *testing.T) {
	ip, user := "203.0.113.9", "victim-x"
	for i := 0; i < 8; i++ {
		if ok, _ := Begin(ip, user); !ok {
			t.Fatalf("locked out early at attempt %d", i)
		}
		Fail(ip, user)
	}
	if ok, wait := Begin(ip, user); ok || wait <= 0 {
		t.Fatal("expected lockout after 8 failures from one address")
	}
	// A different address is not locked by the per-address counter...
	if ok, _ := Begin("198.51.100.7", "someone-else"); !ok {
		t.Fatal("unrelated address blocked")
	}
	// ...but the per-username counter follows the username across addresses.
	for i := 0; i < 10; i++ {
		Fail("192.0.2."+strconv.Itoa(i+1), "target-y")
	}
	if ok, _ := Begin("192.0.2.200", "target-y"); ok {
		t.Fatal("username not locked after 10 failures across addresses")
	}
	Succeed("target-y")
	if ok, _ := Begin("192.0.2.201", "target-y"); !ok {
		t.Fatal("username still locked after a successful sign-in")
	}
}

func TestClientIP(t *testing.T) {
	req := func(remote, realip string) *http.Request {
		r, _ := http.NewRequest("POST", "/", nil)
		r.RemoteAddr = remote
		if realip != "" {
			r.Header.Set("X-Real-IP", realip)
		}
		return r
	}
	if got := ClientIP(req("172.18.0.5:4000", "203.0.113.4")); got != "203.0.113.4" {
		t.Errorf("proxy peer: got %s", got)
	}
	// From a public peer the header is attacker-controlled and must be ignored.
	if got := ClientIP(req("198.51.100.20:4000", "203.0.113.4")); got != "198.51.100.20" {
		t.Errorf("public peer spoofing X-Real-IP: got %s", got)
	}
	if got := ClientIP(req("198.51.100.20:4000", "")); got != "198.51.100.20" {
		t.Errorf("no header: got %s", got)
	}
	a := ClientIP(req("[2001:db8:1:2:aaaa::1]:1", ""))
	b := ClientIP(req("[2001:db8:1:2:bbbb::9]:1", ""))
	if a != b || a != "2001:db8:1:2::/64" {
		t.Errorf("ipv6 /64 keying: %s vs %s", a, b)
	}
}
