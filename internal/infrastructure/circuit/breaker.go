package circuit

import (
	"context"
	"fmt"
	"time"

	"github.com/sony/gobreaker"
)

// State mirrors gobreaker states for external consumers.
type State int

const (
	StateClosed   State = iota
	StateHalfOpen State = iota
	StateOpen     State = iota
)

// Breaker wraps gobreaker with a typed, context-aware interface.
type Breaker struct {
	cb *gobreaker.CircuitBreaker
}

// Settings configures the circuit breaker behaviour.
type Settings struct {
	Name        string
	MaxRequests uint32
	Interval    time.Duration
	Timeout     time.Duration
	ReadyToTrip func(counts gobreaker.Counts) bool
	OnStateChange func(name string, from, to State)
}

// NewBreaker creates a new circuit breaker with the given settings.
func NewBreaker(s Settings) *Breaker {
	if s.ReadyToTrip == nil {
		s.ReadyToTrip = defaultReadyToTrip
	}

	gs := gobreaker.Settings{
		Name:        s.Name,
		MaxRequests: s.MaxRequests,
		Interval:    s.Interval,
		Timeout:     s.Timeout,
		ReadyToTrip: s.ReadyToTrip,
	}

	if s.OnStateChange != nil {
		gs.OnStateChange = func(name string, from, to gobreaker.State) {
			s.OnStateChange(name, gbState(from), gbState(to))
		}
	}

	return &Breaker{cb: gobreaker.NewCircuitBreaker(gs)}
}

// Execute runs fn through the circuit breaker.
// Returns gobreaker.ErrOpenState if the circuit is open.
func (b *Breaker) Execute(ctx context.Context, fn func() (any, error)) (any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	result, err := b.cb.Execute(func() (any, error) {
		return fn()
	})
	if err != nil {
		return nil, fmt.Errorf("circuit breaker %q: %w", b.cb.Name(), err)
	}
	return result, nil
}

// State returns the current state of the circuit breaker.
func (b *Breaker) State() State {
	return gbState(b.cb.State())
}

// Name returns the circuit breaker name.
func (b *Breaker) Name() string { return b.cb.Name() }

func gbState(s gobreaker.State) State {
	switch s {
	case gobreaker.StateHalfOpen:
		return StateHalfOpen
	case gobreaker.StateOpen:
		return StateOpen
	default:
		return StateClosed
	}
}

const (
	helmBreakerInterval   = 60 * time.Second
	helmBreakerTimeout    = 30 * time.Second
	k8sBreakerMaxRequests = 2
	k8sBreakerInterval    = 30 * time.Second
	k8sBreakerTimeout     = 15 * time.Second

	tripMinRequests    uint32  = 3
	tripFailureRatio   float64 = 0.6
)

func defaultReadyToTrip(counts gobreaker.Counts) bool {
	failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
	return counts.Requests >= tripMinRequests && failureRatio >= tripFailureRatio
}

// DefaultHelmBreaker creates a circuit breaker tuned for Helm operations.
func DefaultHelmBreaker() *Breaker {
	return NewBreaker(Settings{
		Name:        "helm",
		MaxRequests: 1,
		Interval:    helmBreakerInterval,
		Timeout:     helmBreakerTimeout,
	})
}

// DefaultK8sBreaker creates a circuit breaker tuned for Kubernetes API calls.
func DefaultK8sBreaker() *Breaker {
	return NewBreaker(Settings{
		Name:        "kubernetes",
		MaxRequests: k8sBreakerMaxRequests,
		Interval:    k8sBreakerInterval,
		Timeout:     k8sBreakerTimeout,
	})
}
