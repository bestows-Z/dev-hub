package auth

import "testing"

func TestTokenRoundTripAndTamper(t *testing.T) {
	tokens := NewTokens("this-is-a-test-secret-with-at-least-32-characters")
	raw, err := tokens.Issue(42)
	if err != nil {
		t.Fatal(err)
	}
	id, err := tokens.Verify(raw)
	if err != nil || id != 42 {
		t.Fatalf("id=%d err=%v", id, err)
	}
	if _, err := NewTokens("different-secret-with-at-least-32-characters").Verify(raw); err == nil {
		t.Fatal("accepted token signed by another key")
	}
}
