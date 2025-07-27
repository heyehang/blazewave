package xsync

import (
	"errors"
	"testing"

	"github.com/heyehang/blazewave/internal/test/assert"
)

func TestGoRecover(t *testing.T) {
	t.Parallel()

	errs := Go(func() error {
		panic("anmol")
	})

	err := <-errs
	assert.Contains(t, err, "anmol")
}

func TestGoErrorReturn(t *testing.T) {
	errMsg := "test error"
	errs := Go(func() error {
		return errors.New(errMsg)
	})
	err := <-errs
	assert.Contains(t, err, errMsg)
}

func TestGoNoError(t *testing.T) {
	errs := Go(func() error {
		return nil
	})
	err := <-errs
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestGoPanicNonString(t *testing.T) {
	errs := Go(func() error {
		panic(12345)
	})
	err := <-errs
	assert.Contains(t, err, "12345")
}
