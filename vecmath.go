package main

import "math"

// --- Vector Math (Simplified) ---
type Vector3 struct {
	X, Y, Z float64
}

func (v Vector3) Add(other Vector3) Vector3 {
	return Vector3{X: v.X + other.X, Y: v.Y + other.Y, Z: v.Z + other.Z}
}

func (v Vector3) Sub(other Vector3) Vector3 {
	return Vector3{X: v.X - other.X, Y: v.Y - other.Y, Z: v.Z - other.Z}
}

func (v Vector3) Scale(s float64) Vector3 {
	return Vector3{X: v.X * s, Y: v.Y * s, Z: v.Z * s}
}

func (v Vector3) Dot(other Vector3) float64 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

func (v Vector3) LengthSquared() float64 { // Often useful to avoid sqrt
	return v.X*v.X + v.Y*v.Y + v.Z*v.Z
}

func (v Vector3) Length() float64 {
	return math.Sqrt(v.LengthSquared())
}

func (v Vector3) Normalize() Vector3 {
	l := v.Length()
	if l == 0 {
		return Vector3{} // Or handle error, return original, etc.
	}
	return Vector3{X: v.X / l, Y: v.Y / l, Z: v.Z / l}
}

func (v Vector3) Reflect(normal Vector3) Vector3 {
	// Assumes normal is a unit vector
	// R = V - 2 * dot(V, N) * N
	dotProduct := v.Dot(normal)
	return v.Sub(normal.Scale(2 * dotProduct))
}

// SetFromSphericalCoords sets the vector from spherical coordinates.
// Phi is the polar angle (from y-axis, 0 to Pi). Theta is the azimuthal angle (around y-axis, 0 to 2Pi).
func SetFromSphericalCoords(radius, phi, theta float64) Vector3 {
	sinPhiRadius := math.Sin(phi) * radius
	return Vector3{
		X: sinPhiRadius * math.Sin(theta),
		Y: math.Cos(phi) * radius,
		Z: sinPhiRadius * math.Cos(theta),
	}
}

// DistanceTo calculates the distance between two Vector3 points.
func (v Vector3) DistanceTo(other Vector3) float64 {
	return v.Sub(other).Length()
}

// DistanceToSquared calculates the squared distance between two Vector3 points.
func (v Vector3) DistanceToSquared(other Vector3) float64 {
	return v.Sub(other).LengthSquared()
}
