//go:build !js

package blazewave_test

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/heyehang/blazewave"
	"github.com/heyehang/blazewave/internal/test/assert"
)

func Test_hijackerHTTPResponseControllerCompatibility(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	w := mockUnwrapper{
		ResponseWriter: rr,
		unwrap: func() http.ResponseWriter {
			return mockHijacker{
				ResponseWriter: rr,
				hijack: func() (conn net.Conn, writer *bufio.ReadWriter, err error) {
					return nil, nil, errors.New("haha")
				},
			}
		},
	}

	_, _, err := http.NewResponseController(w).Hijack()
	assert.Contains(t, err, "haha")
	hj, ok := blazewave.Hijacker(w)
	assert.Equal(t, "hijacker found", ok, true)
	_, _, err = hj.Hijack()
	assert.Contains(t, err, "haha")
}
