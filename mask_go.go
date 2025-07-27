//go:build !amd64 && !arm64 && !js

package blazewave

func mask(b []byte, key uint32) uint32 {
	return maskGo(b, key)
}
