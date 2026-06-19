package userauth

import (
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

type perKeyRateLimiter struct {
	limiters    sync.Map
	rps         rate.Limit
	burst       int
	callCounter atomic.Int64
}

type perKeyEntry struct {
	limiter  *rate.Limiter
	lastSeen atomic.Int64
}

const sweepInterval = 1024
const staleAfter = 10 * time.Minute

func newPerKeyRateLimiter(rpm, burst int) *perKeyRateLimiter {
	if burst <= 0 {
		burst = 5
	}
	r := rate.Limit(0)
	if rpm > 0 {
		r = rate.Limit(float64(rpm) / 60.0)
	}
	return &perKeyRateLimiter{rps: r, burst: burst}
}

func (rl *perKeyRateLimiter) allow(key string) bool {
	if rl.rps == 0 {
		return true
	}
	nowNs := time.Now().UnixNano()

	fresh := &perKeyEntry{limiter: rate.NewLimiter(rl.rps, rl.burst)}
	fresh.lastSeen.Store(nowNs)

	v, _ := rl.limiters.LoadOrStore(key, fresh)
	entry := v.(*perKeyEntry)
	if !entry.limiter.Allow() {
		return false
	}
	entry.lastSeen.Store(nowNs)

	if rl.callCounter.Add(1)%sweepInterval == 0 {
		rl.sweepStale()
	}
	return true
}

func (rl *perKeyRateLimiter) sweepStale() {
	cutoffNs := time.Now().Add(-staleAfter).UnixNano()
	rl.limiters.Range(func(k, v any) bool {
		if v.(*perKeyEntry).lastSeen.Load() < cutoffNs {
			rl.limiters.Delete(k)
		}
		return true
	})
}

type AuthRateLimiter struct {
	login   *perKeyRateLimiter
	verify  *perKeyRateLimiter
	register *perKeyRateLimiter
}

func NewAuthRateLimiter() *AuthRateLimiter {
	return &AuthRateLimiter{
		login:    newPerKeyRateLimiter(5, 3),
		verify:   newPerKeyRateLimiter(10, 5),
		register: newPerKeyRateLimiter(3, 2),
	}
}

func (a *AuthRateLimiter) AllowLogin(email string) bool {
	return a.login.allow("auth:" + email)
}

func (a *AuthRateLimiter) AllowVerify(userID string) bool {
	return a.verify.allow("verify:" + userID)
}

func (a *AuthRateLimiter) AllowRegister(ip string) bool {
	return a.register.allow("register:" + ip)
}
