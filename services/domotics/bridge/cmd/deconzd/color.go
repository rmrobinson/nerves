package main

import (
	"math"

	"github.com/lucasb-eyer/go-colorful"
)

const (
	colorPointRed   = 0
	colorPointGreen = 1
	colorPointBlue  = 2
)

// XY is a colour represented using the CIE colour space.
type XY struct {
	X float64
	Y float64
}

// colourPointsForModel returns the XY bounds for the specified lightbulb model.
// The returned array always has the red, then green, then blue points in that order.
func colorPointsForModel(model string) (points []XY) {
	points = make([]XY, 3)
	switch model {
	case "LCT001", "LCT002", "LCT003":
		points[colorPointRed].X = 0.674
		points[colorPointRed].Y = 0.322
		points[colorPointGreen].X = 0.408
		points[colorPointGreen].Y = 0.517
		points[colorPointBlue].X = 0.168
		points[colorPointBlue].Y = 0.041
		return

	case "LLC001", "LLC005", "LLC006", "LLC007", "LLC011", "LLC012", "LLC013", "LST001":
		points[colorPointRed].X = 0.703
		points[colorPointRed].Y = 0.296
		points[colorPointGreen].X = 0.214
		points[colorPointGreen].Y = 0.709
		points[colorPointBlue].X = 0.139
		points[colorPointBlue].Y = 0.081
		return
	}

	points[colorPointRed].X = 1.0
	points[colorPointRed].Y = 0.0
	points[colorPointGreen].X = 0.0
	points[colorPointGreen].Y = 1.0
	points[colorPointBlue].X = 0.0
	points[colorPointBlue].Y = 0.0
	return
}

func crossProduct(p1, p2 XY) float64 {
	return p1.X*p2.Y - p1.Y*p2.X
}

func getClosestPointToPoints(a, b, p XY) XY {
	ap := XY{X: p.X - a.X, Y: p.Y - a.Y}
	ab := XY{X: b.X - a.X, Y: b.Y - a.Y}

	ab2 := ab.X*ab.X + ab.Y*ab.Y
	ap_ab := ap.X*ab.X + ap.Y*ab.Y

	t := ap_ab / ab2

	if t < 0.0 {
		t = 0.0
	} else if t > 1.0 {
		t = 1.0
	}

	return XY{X: a.X + ab.X*t, Y: a.Y + ab.Y*t}
}

func getDistanceBetweenTwoPoints(p1, p2 XY) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y

	return math.Sqrt(dx*dx + dy*dy)
}

func checkPointInColorPointsReach(p XY, colorPoints []XY) bool {
	if len(colorPoints) != 3 {
		return false
	}

	red := colorPoints[colorPointRed]
	green := colorPoints[colorPointGreen]
	blue := colorPoints[colorPointBlue]

	v1 := XY{X: green.X - red.X, Y: green.Y - red.Y}
	v2 := XY{X: blue.X - red.X, Y: blue.Y - red.Y}
	q := XY{X: p.X - red.X, Y: p.Y - red.Y}

	s := crossProduct(q, v2) / crossProduct(v1, v2)
	t := crossProduct(v1, q) / crossProduct(v1, v2)

	if s >= 0.0 && t >= 0.0 && s+t <= 1.0 {
		return true
	}

	return false
}

func getHueXYBrightnessFromColor(c colorful.Color, model string) (float64, float64, float64) {
	X, Y, Z := c.Xyz()
	cx := X / (X + Y + Z)
	cy := Y / (X + Y + Z)

	xy := XY{X: cx, Y: cy}
	colorPoints := colorPointsForModel(model)

	if !checkPointInColorPointsReach(xy, colorPoints) {
		// Find the closest color we can reach and send this instead
		pAB := getClosestPointToPoints(colorPoints[colorPointRed], colorPoints[colorPointGreen], xy)
		pAC := getClosestPointToPoints(colorPoints[colorPointBlue], colorPoints[colorPointRed], xy)
		pBC := getClosestPointToPoints(colorPoints[colorPointGreen], colorPoints[colorPointBlue], xy)

		dAB := getDistanceBetweenTwoPoints(xy, pAB)
		dAC := getDistanceBetweenTwoPoints(xy, pAC)
		dBC := getDistanceBetweenTwoPoints(xy, pBC)

		lowest := dAB
		closestPoint := pAB

		if dAC < lowest {
			lowest = dAC
			closestPoint = pAC
		}
		if dBC < lowest {
			lowest = dBC
			closestPoint = pBC
		}

		cx = closestPoint.X
		cy = closestPoint.Y
	}

	return cx, cy, Y
}
