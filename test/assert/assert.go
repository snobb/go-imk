package assert

import (
	"testing"
)

func Equal[T comparable](t *testing.T, a, b T) {
	if a != b {
		t.Logf("expected %v, got %v\n", a, b)
		t.FailNow()
	}
}

func Error(t *testing.T, err error) {
	if err == nil {
		t.Logf("expected error but got nil")
		t.FailNow()
	}
}

func NoError(t *testing.T, err error) {
	if err != nil {
		t.Logf("expected no error, got %q\n", err.Error())
		t.FailNow()
	}
}

func EqualError(t *testing.T, err error, expected string) {
	Error(t, err)

	if err.Error() != expected {
		t.Logf("expected error %q, got %q\n", expected, err.Error())
		t.FailNow()
	}
}
