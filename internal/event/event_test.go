package event

import "testing"

func TestError(t *testing.T) {
	t.Run("no-ops without panicking", func(t *testing.T) {
		Error(nil)
		Error("some error")
		Error("test error", "key", "value")
	})
}
