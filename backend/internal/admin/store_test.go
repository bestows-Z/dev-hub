package admin

import "testing"

func TestOrderStatusTransitions(t *testing.T) {
	for _, tc := range []struct {
		from, to string
		allowed  bool
	}{
		{"pending_payment", "paid", true}, {"pending_payment", "cancelled", true},
		{"paid", "delivered", true}, {"delivered", "cancelled", false},
		{"cancelled", "paid", false}, {"paid", "cancelled", false},
	} {
		if allowedTransition(tc.from, tc.to) != tc.allowed {
			t.Fatalf("%s -> %s", tc.from, tc.to)
		}
	}
}
