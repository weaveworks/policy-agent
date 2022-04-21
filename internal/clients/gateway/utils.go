package gateway

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/golang/snappy"
)

func withBackOff(retries int, timeout time.Duration, fn func() error) error {
	for i := 1; i <= retries; i++ {
		err := fn()
		if err == nil {
			return nil
		}

		if err, ok := err.(ConnectionError); ok {
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
	defer lock.Unlock()
	fn()
}

func encodeSnappy(in interface{}) (out []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			err = fmt.Errorf("%s panic: %v", stack, r)
		}
	}()

	js, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("unable to encode to snappy, error: %w", err)
	}
	out = snappy.Encode(nil, js)
	return out, err
}

func decodeSnappy(in []byte, out interface{}) error {
	js, err := snappy.Decode(nil, in)
	if err != nil {
		return fmt.Errorf("unable to decode to snappy, error: %w", err)
	}
	return json.Unmarshal(js, out)
}
