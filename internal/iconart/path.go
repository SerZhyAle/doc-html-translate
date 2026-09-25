// Package iconart draws the product's system-surface icons - the program ICO, the MSIX tile
// set, the Explorer verb and document-type icons, the extension's action icon - from one
// source: the product mark defined in mark.go and the vendored vocabulary glyphs under
// assets/glyphs (ICON-RENDER rule 9, ticket 32). Everything it writes is a render target;
// tests/icons_test.go fails when a committed file drifts from what this package draws.
package iconart

import (
	"fmt"
	"strconv"
)

// segment is one drawing command in absolute coordinates: 'M', 'L', 'C' or 'Z'.
// Quadratic curves are raised to cubics so the rasterizer sees only three kinds.
type segment struct {
	op  byte
	pts [3]point
}

type point struct{ x, y float64 }

// parsePath reads SVG path data - the subset the Material glyphs use (M L H V C S Q T Z,
// absolute and relative, implicit repeats, compact numbers such as "1.1.9" or "2-2").
// Arcs are refused rather than approximated: no vendored glyph needs one.
func parsePath(d string) ([]segment, error) {
	p := pathScanner{s: d}
	var out []segment
	var cur, start, lastCtrl point
	var prev byte
	cmd := byte(0)
	for {
		p.skipSep()
		if p.eof() {
			break
		}
		if c := p.s[p.i]; isCommand(c) {
			cmd = c
			p.i++
		} else if cmd == 0 {
			return nil, fmt.Errorf("path data starts without a command at %d", p.i)
		}
		rel := cmd >= 'a'
		abs := func(q point) point {
			if rel {
				return point{cur.x + q.x, cur.y + q.y}
			}
			return q
		}
		switch cmd | 0x20 {
		case 'z':
			out = append(out, segment{op: 'Z'})
			cur = start
			prev = 'z'
			// A closepath takes no arguments: a number here has no command to belong to,
			// and without this check the loop would re-read it forever.
			if p.skipSep(); !p.eof() && !isCommand(p.s[p.i]) {
				return nil, fmt.Errorf("number after closepath at %d in path data", p.i)
			}
			continue
		case 'm':
			q, err := p.point()
			if err != nil {
				return nil, err
			}
			cur = abs(q)
			start = cur
			out = append(out, segment{op: 'M', pts: [3]point{cur}})
			// Further pairs after a moveto are implicit linetos.
			if rel {
				cmd = 'l'
			} else {
				cmd = 'L'
			}
			prev = 'm'
			continue
		case 'l':
			q, err := p.point()
			if err != nil {
				return nil, err
			}
			cur = abs(q)
			out = append(out, segment{op: 'L', pts: [3]point{cur}})
		case 'h':
			v, err := p.number()
			if err != nil {
				return nil, err
			}
			if rel {
				cur.x += v
			} else {
				cur.x = v
			}
			out = append(out, segment{op: 'L', pts: [3]point{cur}})
		case 'v':
			v, err := p.number()
			if err != nil {
				return nil, err
			}
			if rel {
				cur.y += v
			} else {
				cur.y = v
			}
			out = append(out, segment{op: 'L', pts: [3]point{cur}})
		case 'c', 's':
			var c1 point
			if cmd|0x20 == 's' {
				c1 = cur
				if prev == 'c' || prev == 's' {
					c1 = point{2*cur.x - lastCtrl.x, 2*cur.y - lastCtrl.y}
				}
			} else {
				q, err := p.point()
				if err != nil {
					return nil, err
				}
				c1 = abs(q)
			}
			q2, err := p.point()
			if err != nil {
				return nil, err
			}
			q3, err := p.point()
			if err != nil {
				return nil, err
			}
			c2, end := abs(q2), abs(q3)
			out = append(out, segment{op: 'C', pts: [3]point{c1, c2, end}})
			lastCtrl, cur = c2, end
			prev = cmd | 0x20
			continue
		case 'q', 't':
			var c point
			if cmd|0x20 == 't' {
				c = cur
				if prev == 'q' || prev == 't' {
					c = point{2*cur.x - lastCtrl.x, 2*cur.y - lastCtrl.y}
				}
			} else {
				q, err := p.point()
				if err != nil {
					return nil, err
				}
				c = abs(q)
			}
			q2, err := p.point()
			if err != nil {
				return nil, err
			}
			end := abs(q2)
			out = append(out, segment{op: 'C', pts: [3]point{
				{cur.x + 2.0/3*(c.x-cur.x), cur.y + 2.0/3*(c.y-cur.y)},
				{end.x + 2.0/3*(c.x-end.x), end.y + 2.0/3*(c.y-end.y)},
				end,
			}})
			lastCtrl, cur = c, end
			prev = cmd | 0x20
			continue
		default:
			return nil, fmt.Errorf("path command %q is not supported", cmd)
		}
		prev = cmd | 0x20
	}
	return out, nil
}

func isCommand(c byte) bool {
	switch c | 0x20 {
	case 'm', 'l', 'h', 'v', 'c', 's', 'q', 't', 'a', 'z':
		return true
	}
	return false
}

type pathScanner struct {
	s string
	i int
}

func (p *pathScanner) eof() bool { return p.i >= len(p.s) }

func (p *pathScanner) skipSep() {
	for !p.eof() {
		switch p.s[p.i] {
		case ' ', ',', '\t', '\n', '\r':
			p.i++
		default:
			return
		}
	}
}

func (p *pathScanner) point() (point, error) {
	x, err := p.number()
	if err != nil {
		return point{}, err
	}
	y, err := p.number()
	return point{x, y}, err
}

// number reads one SVG number. A second '.' or a sign ends it, so "1.1.9" is 1.1 then .9.
func (p *pathScanner) number() (float64, error) {
	p.skipSep()
	start := p.i
	if !p.eof() && (p.s[p.i] == '-' || p.s[p.i] == '+') {
		p.i++
	}
	dot, digits := false, false
	for !p.eof() {
		c := p.s[p.i]
		switch {
		case c >= '0' && c <= '9':
			digits = true
		case c == '.' && !dot:
			dot = true
		case (c == 'e' || c == 'E') && digits:
			if p.i+1 < len(p.s) && (p.s[p.i+1] == '-' || p.s[p.i+1] == '+') {
				p.i++
			}
		default:
			goto done
		}
		p.i++
	}
done:
	if !digits {
		return 0, fmt.Errorf("expected a number at %d in path data", start)
	}
	return strconv.ParseFloat(p.s[start:p.i], 64)
}
