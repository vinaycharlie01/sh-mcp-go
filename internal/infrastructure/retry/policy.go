package retry

import (
	"context"
	"log/slog"
	"time"

	"github.com/avast/retry-go/v4"
)

// Policy defines retry behaviour for an operation.
type Policy struct {
	Attempts    uint
	Delay       time.Duration
	MaxDelay    time.Duration
	DelayType   retry.DelayTypeFunc
	RetryIf     retry.RetryIfFunc
	OnRetry     retry.OnRetryFunc
	Logger      *slog.Logger
}

// DefaultHelmPolicy is a sensible retry policy for Helm operations.
func DefaultHelmPolicy(logger *slog.Logger) Policy {
	return Policy{
		Attempts:  3,
		Delay:     2 * time.Second,
		MaxDelay:  30 * time.Second,
		DelayType: retry.BackOffDelay,
		Logger:    logger,
	}
}

// DefaultK8sPolicy is a sensible retry policy for Kubernetes API calls.
func DefaultK8sPolicy(logger *slog.Logger) Policy {
	return Policy{
		Attempts:  5,
		Delay:     1 * time.Second,
		MaxDelay:  15 * time.Second,
		DelayType: retry.BackOffDelay,
		Logger:    logger,
	}
}

// Do executes fn according to the policy, propagating context cancellation.
func Do(ctx context.Context, policy Policy, fn func() error) error {
	opts := []retry.Option{
		retry.Attempts(policy.Attempts),
		retry.Delay(policy.Delay),
		retry.MaxDelay(policy.MaxDelay),
		retry.Context(ctx),
	}

	if policy.DelayType != nil {
		opts = append(opts, retry.DelayType(policy.DelayType))
	} else {
		opts = append(opts, retry.DelayType(retry.BackOffDelay))
	}

	if policy.RetryIf != nil {
		opts = append(opts, retry.RetryIf(policy.RetryIf))
	}

	if policy.OnRetry != nil {
		opts = append(opts, retry.OnRetry(policy.OnRetry))
	} else if policy.Logger != nil {
		logger := policy.Logger
		opts = append(opts, retry.OnRetry(func(n uint, err error) {
			logger.Warn("retrying operation",
				slog.Uint64("attempt", uint64(n)),
				slog.String("error", err.Error()),
			)
		}))
	}

	return retry.Do(fn, opts...)
}

// DoWithResult executes fn and returns both result and error, retrying on failure.
func DoWithResult[T any](ctx context.Context, policy Policy, fn func() (T, error)) (T, error) {
	var result T
	err := Do(ctx, policy, func() error {
		var e error
		result, e = fn()
		return e
	})
	return result, err
}
