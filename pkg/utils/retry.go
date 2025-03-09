package utils

import (
	"fmt"
	"math/rand"
	"time"
)

func Retry(fn func() error, maxAttempts int, delay time.Duration) error {

	for i := 0; i < maxAttempts; i++ {
		if err := fn(); err != nil {
			time.Sleep(delay)
			continue
		}
		return nil
	}
	return fmt.Errorf("failed to execute function after %d attempts", maxAttempts)
}

func RetryWithBackoff(fn func() error, maxAttempts int, initialDelay time.Duration, maxDelay time.Duration) error {
	delay := initialDelay
	for i := 0; i < maxAttempts; i++ {
		if err := fn(); err != nil {
			time.Sleep(delay)
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}
	return nil
}

func RetryWithJitter(fn func() error, maxAttempts int, initialDelay time.Duration, maxDelay time.Duration) error {
	delay := initialDelay
	for i := 0; i < maxAttempts; i++ {
		if err := fn(); err != nil {
			time.Sleep(delay)
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
			delay += time.Duration(rand.Intn(1000)) * time.Millisecond
		}
	}
	return nil
}
