package escapement

import "github.com/lestrrat-3d/decad"

// springAnchorSocket builds the cutter for the blind socket in the front plate's inner face that takes the spring anchor
// pin: a circle of radius 1.55 (0.05 over the pin's 1.5, a press fit) about the pin's centre, from 1 mm before the face
// to z 31, so the socket is 3 mm deep and leaves 3 mm of plate above it.
func (b *builder) springAnchorSocket() (*decad.Body, error) {
	q := b.g.SpringAnchor
	return b.discAt("spring anchor socket", q.X, q.Y, springSocketLow, springSocketR, springSocketHigh-springSocketLow)
}
