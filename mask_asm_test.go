//go:build !js

package blazewave_test

import (
	"testing"

	"github.com/heyehang/blazewave"
)

func TestMaskASM(t *testing.T) {
	t.Parallel()

	testMask(t, "maskASM", blazewave.Mask)
}
