package database

import (
	"context"
	"errors"
	"testing"
)

func TestDirectTransactorRunsCallbackAndReturnsItsError(t *testing.T) {
	expected := errors.New("stop")
	called := false

	err := (DirectTransactor{}).WithinTransaction(t.Context(), func(context.Context) error {
		called = true
		return expected
	})

	if !called {
		t.Fatal("transaction callback was not called")
	}
	if !errors.Is(err, expected) {
		t.Fatalf("got error %v, want %v", err, expected)
	}
}
