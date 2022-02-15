package gateway

import (
	"sync"
	"time"
)

func withBackOff(retries int, timeout time.Duration, fn func() error) error {
	for i := 1; i <= retries; i++ {
		err := fn()
		if err == nil {
			return nil
		}

		if err, ok := err.(FatalError); ok {
			return err
		}

		if i == retries {
			return err
		}

		time.Sleep(timeout)
	}
	return nil
}

func withLock(lock *sync.Mutex, fn func()) {
	lock.Lock()
	fn()
	lock.Unlock()
}
