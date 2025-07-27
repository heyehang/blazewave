//go:build !js

package blazewave

func (c *Conn) flate() bool {
	return c.copts != nil
}
