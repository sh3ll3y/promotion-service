package retry

import (
	"fmt"
	"time"
)

func Do(attempts int, sleep time.Duration, fn func() error) error {
	var err error
	for i := 0; ; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		if i >= (attempts - 1) {
			break
		}

		time.Sleep(sleep)
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}