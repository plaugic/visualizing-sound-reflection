package main

import "math"

type RayIntersectionResult struct {
	Hit           bool
	Point, Normal Vector3
	Distance      float64
	Object        *SceneObject
}

func performRaycast(origin Vector3, direction Vector3, maxDist float64, objects []*SceneObject, ignoreObject *SceneObject) RayIntersectionResult {
	closestHit := RayIntersectionResult{Hit: false, Distance: maxDist}
	for _, obj := range objects {
		if obj == ignoreObject || !obj.Visible {
			continue
		}
		var hitDistance float64 = -1
		if obj.ShapeType == "sphere" {
			oc := origin.Sub(obj.Position)
			a := direction.Dot(direction)
			b := 2.0 * oc.Dot(direction)
			c := oc.Dot(oc) - obj.Scale.X*obj.Scale.X // Assuming uniform scale for sphere radius
			discriminant := b*b - 4*a*c
			if discriminant >= 0 {
				t := (-b - math.Sqrt(discriminant)) / (2.0 * a)
				if t > EPSILON && t < closestHit.Distance {
					hitDistance = t
				}
			}
		} else if obj.ShapeType == "box" {
			// Simplified AABB intersection (assumes box is not rotated relative to world axes)
			// For rotated boxes, a more complex OBB intersection would be needed.
			minBound := obj.Position.Sub(obj.Scale.Scale(0.5))
			maxBound := obj.Position.Add(obj.Scale.Scale(0.5))
			tMin, tMax := 0.0, maxDist
			hitCurrentBox := true

			for i := 0; i < 3; i++ { // Iterate over X, Y, Z axes
				var invD, oComp, minB_i, maxB_i float64
				rayDirComp := 0.0

				switch i {
				case 0: // X
					rayDirComp = direction.X
					oComp = origin.X
					minB_i = minBound.X
					maxB_i = maxBound.X
				case 1: // Y
					rayDirComp = direction.Y
					oComp = origin.Y
					minB_i = minBound.Y
					maxB_i = maxBound.Y
				case 2: // Z
					rayDirComp = direction.Z
					oComp = origin.Z
					minB_i = minBound.Z
					maxB_i = maxBound.Z
				}

				if math.Abs(rayDirComp) < EPSILON { // Ray is parallel to this slab.
					if oComp < minB_i || oComp > maxB_i { // Origin is outside the slab.
						hitCurrentBox = false
						break
					}
					continue // Ray is parallel and inside slab, continue checking other slabs.
				}
				invD = 1.0 / rayDirComp

				t0 := (minB_i - oComp) * invD
				t1 := (maxB_i - oComp) * invD
				if invD < 0 {
					t0, t1 = t1, t0 // Swap if invD is negative
				}

				if t0 > tMin {
					tMin = t0
				}
				if t1 < tMax {
					tMax = t1
				}

				if tMin > tMax { // Ray misses the box
					hitCurrentBox = false
					break
				}
			} // End loop over axes

			if hitCurrentBox && tMin > EPSILON && tMin < closestHit.Distance {
				hitDistance = tMin
			}
		} // End box intersection

		if hitDistance > EPSILON && hitDistance < closestHit.Distance {
			closestHit.Hit = true
			closestHit.Distance = hitDistance
			closestHit.Point = origin.Add(direction.Scale(hitDistance))
			closestHit.Object = obj
			// Calculate normal (simplified for AABB and sphere)
			if obj.ShapeType == "sphere" {
				closestHit.Normal = closestHit.Point.Sub(obj.Position).Normalize()
			} else if obj.ShapeType == "box" {
				// Simplified normal calculation for AABB
				p := closestHit.Point
				c := obj.Position
				d := obj.Scale.Scale(0.5) // half dimensions
				if math.Abs(p.X-(c.X-d.X)) < EPSILON {
					closestHit.Normal = Vector3{-1, 0, 0}
				} else if math.Abs(p.X-(c.X+d.X)) < EPSILON {
					closestHit.Normal = Vector3{1, 0, 0}
				} else if math.Abs(p.Y-(c.Y-d.Y)) < EPSILON {
					closestHit.Normal = Vector3{0, -1, 0}
				} else if math.Abs(p.Y-(c.Y+d.Y)) < EPSILON {
					closestHit.Normal = Vector3{0, 1, 0}
				} else if math.Abs(p.Z-(c.Z-d.Z)) < EPSILON {
					closestHit.Normal = Vector3{0, 0, -1}
				} else if math.Abs(p.Z-(c.Z+d.Z)) < EPSILON {
					closestHit.Normal = Vector3{0, 0, 1}
				} else {
					// Fallback (should ideally not happen for precise AABB hits on faces)
					closestHit.Normal = p.Sub(c).Normalize()
				}
			}
		}
	}
	return closestHit
}

// castRayAndGetBounceCountForEvaluation: returns bounce count if listener hit, -1 otherwise. No visuals.
func castRayAndGetBounceCountForEvaluation(origin Vector3, direction Vector3, currentReflections int, collidables []*SceneObject, listenerPos Vector3, listenerRadius float64) int {
	if currentReflections > maxReflections {
		return -1
	}

	// Ensure soundSource is collidable for reflected rays
	effectiveCollidables := collidables
	if currentReflections > 0 { // For reflected rays, the source itself can be an occluder
		sourceInCollidables := false
		for _, obj := range collidables {
			if obj == soundSource {
				sourceInCollidables = true
				break
			}
		}
		if !sourceInCollidables && soundSource != nil { // Add soundSource if not already present
			tempCollidables := make([]*SceneObject, len(collidables)+1)
			copy(tempCollidables, collidables)
			tempCollidables[len(collidables)] = soundSource
			effectiveCollidables = tempCollidables
		}
	}

	intersection := performRaycast(origin, direction, MAX_RAY_DISTANCE, effectiveCollidables, nil)

	rayLength := MAX_RAY_DISTANCE
	if intersection.Hit {
		rayLength = intersection.Distance
	}
	endPoint := origin.Add(direction.Scale(rayLength))

	// Check if current ray segment hits the listener (sphere intersection)
	// Simplified: Check distance from listener center to the ray line segment
	dirToListener := listenerPos.Sub(origin)
	t := dirToListener.Dot(direction) // Project listener's origin onto the ray
	var closestPointOnLine Vector3

	if t <= 0 { // Closest point is the ray origin
		closestPointOnLine = origin
	} else if t >= rayLength { // Closest point is the ray endpoint (or hit point)
		closestPointOnLine = endPoint
	} else { // Closest point is on the segment
		closestPointOnLine = origin.Add(direction.Scale(t))
	}

	if closestPointOnLine.Sub(listenerPos).Length() < listenerRadius {
		// Check if this hit is occluded by anything *before* the listener along this segment
		distToClosestPointOnLine := origin.Sub(closestPointOnLine).Length()
		if !intersection.Hit || intersection.Distance > distToClosestPointOnLine {
			return currentReflections // Hit listener
		}
	}

	// If ray hit an object and we haven't exceeded max reflections
	if intersection.Hit && currentReflections < maxReflections {
		// Check for attenuation - if ray is too weak, stop.
		currentSegmentOpacity := initialRayOpacity * math.Pow(volumeAttenuationFactor, float64(currentReflections))
		if currentSegmentOpacity < 0.01 { // Threshold for ray being too weak
			return -1
		}

		reflectDirection := direction.Reflect(intersection.Normal)
		reflectionOrigin := intersection.Point.Add(reflectDirection.Scale(0.01)) // Move slightly off surface
		return castRayAndGetBounceCountForEvaluation(reflectionOrigin, reflectDirection, currentReflections+1, collidables, listenerPos, listenerRadius)
	}

	return -1 // No listener hit along this path
}

type HitData struct {
	hitListener bool
	bounces     int
}

// castRayAndAddVisuals: adds to rayVisuals and returns HitData.
func castRayAndAddVisuals(origin Vector3, direction Vector3, currentReflections int, collidables []*SceneObject, listenerPos Vector3, listenerRadius float64) HitData {
	if currentReflections > maxReflections {
		return HitData{hitListener: false, bounces: -1}
	}

	effectiveCollidables := collidables
	if currentReflections > 0 {
		sourceInCollidables := false
		for _, obj := range collidables {
			if obj == soundSource {
				sourceInCollidables = true
				break
			}
		}
		if !sourceInCollidables && soundSource != nil {
			tempCollidables := make([]*SceneObject, len(collidables)+1)
			copy(tempCollidables, collidables)
			tempCollidables[len(collidables)] = soundSource
			effectiveCollidables = tempCollidables
		}
	}

	intersection := performRaycast(origin, direction, MAX_RAY_DISTANCE, effectiveCollidables, nil)

	rayColorIdx := currentReflections
	if rayColorIdx >= len(bounceColors) {
		rayColorIdx = rayColorIdx % len(bounceColors) // Cycle through colors if not enough
	}
	rayColor := bounceColors[rayColorIdx]

	rayLength := MAX_RAY_DISTANCE
	if intersection.Hit {
		rayLength = intersection.Distance
	}
	endPoint := origin.Add(direction.Scale(rayLength))

	currentSegmentOpacity := initialRayOpacity * math.Pow(volumeAttenuationFactor, float64(currentReflections))

	result := HitData{hitListener: false, bounces: -1}

	// Check for listener intersection along this segment
	dirToListener := listenerPos.Sub(origin)
	t := dirToListener.Dot(direction)
	var closestPointOnLine Vector3
	if t <= 0 {
		closestPointOnLine = origin
	} else if t >= rayLength { // If projection is beyond current segment end
		closestPointOnLine = endPoint
	} else {
		closestPointOnLine = origin.Add(direction.Scale(t))
	}

	listenerHitThisSegment := false
	if closestPointOnLine.Sub(listenerPos).Length() < listenerRadius {
		// Ensure no object is hit *before* the listener on this segment
		distToClosestPointOnLine := origin.Sub(closestPointOnLine).Length()
		if !intersection.Hit || intersection.Distance > distToClosestPointOnLine {
			listenerHitThisSegment = true
		}
	}

	if listenerHitThisSegment {
		rayColor = listenerRayColor
		result.hitListener = true
		result.bounces = currentReflections
		currentSegmentOpacity = initialRayOpacity // Make listener rays fully opaque for clarity
	}

	// Store data for subsequent bounces even if this segment itself didn't hit the listener directly
	// The final hitListener status will be determined by the deepest reflection that hits.
	reflectionHitData := HitData{hitListener: false, bounces: -1}
	if intersection.Hit && currentReflections < maxReflections {
		if currentSegmentOpacity >= 0.01 || (showOnlyListenerRays && result.hitListener) { // Only reflect if ray is strong enough or it's a listener path
			reflectDirection := direction.Reflect(intersection.Normal)
			reflectionOrigin := intersection.Point.Add(reflectDirection.Scale(0.01)) // Offset to avoid self-intersection
			reflectionHitData = castRayAndAddVisuals(reflectionOrigin, reflectDirection, currentReflections+1, collidables, listenerPos, listenerRadius)

			if reflectionHitData.hitListener {
				result.hitListener = true // Propagate listener hit status upwards
				// If this path also hit listener, keep the lower bounce count. If not, take the reflection's.
				if result.bounces == -1 || reflectionHitData.bounces < result.bounces {
					result.bounces = reflectionHitData.bounces
				}
			}
		}
	}

	// Determine if this ray segment should be drawn
	shouldDraw := false
	if currentSegmentOpacity >= 0.01 { // Basic visibility
		if !showOnlyListenerRays {
			shouldDraw = true
		} else if result.hitListener || reflectionHitData.hitListener { // If showing only listener rays, and this path (current or future segment) hits.
			shouldDraw = true
			if listenerHitThisSegment { // if this segment is the one hitting, ensure its color is listenerRayColor
				rayColor = listenerRayColor
				currentSegmentOpacity = initialRayOpacity // And full opacity for the hitting segment
			} else if reflectionHitData.hitListener {
				// If a future segment hits, this segment's color remains its bounce color.
				// Opacity might be low, but it's part of a successful path.
			}
		}
	}

	if shouldDraw {
		rayVisuals = append(rayVisuals, &RayLine{
			Start:   Point3D{origin.X, origin.Y, origin.Z},
			End:     Point3D{endPoint.X, endPoint.Y, endPoint.Z},
			Color:   rayColor,
			Opacity: currentSegmentOpacity,
		})
	}

	return result
}

func calculateListenerScore(testSourcePos, testListenerPos Vector3) int {
	currentListenerScore := 0
	var tempCollidables []*SceneObject

	// Create a temporary list of collidables for this specific evaluation
	// Exclude the object being tested if it's the sound source,
	// but include it if it's a reflection point.
	for _, obj := range allSceneObjects {
		isCurrentTestedSource := (obj.Name == "SoundSource" && obj.Position.X == testSourcePos.X && obj.Position.Y == testSourcePos.Y && obj.Position.Z == testSourcePos.Z)
		// The listener itself should always be a target, not an occluder for its own rays.
		// The sound source is the origin, so it's not an occluder for direct rays.
		if !isCurrentTestedSource && obj.Name != "Listener" {
			tempCollidables = append(tempCollidables, obj)
		}
	}

	evalNumRays := numRays / 50 // Use fewer rays for faster evaluation during optimization
	if evalNumRays < 10 {
		evalNumRays = 10
	}
	if evalNumRays > 100 { // Cap eval rays
		evalNumRays = 100
	}

	var listenerObjForRadius *SceneObject
	if listener != nil && listener.Position.X == testListenerPos.X && listener.Position.Y == testListenerPos.Y && listener.Position.Z == testListenerPos.Z {
		listenerObjForRadius = listener
	} else { // Create a temporary listener representative if needed (e.g. if listener is nil globally)
		listenerObjForRadius = &SceneObject{Scale: Vector3{X: 0.25, Y: 0.25, Z: 0.25}} // Default listener radius
	}
	listenerRadius := listenerObjForRadius.Scale.X // Assuming uniform scale for radius

	for i := 0; i < evalNumRays; i++ {
		// Fibonacci spiral for even distribution
		phi := math.Acos(-1 + (2*float64(i))/float64(evalNumRays))
		theta := math.Sqrt(float64(evalNumRays)*math.Pi) * phi
		direction := SetFromSphericalCoords(1, phi, theta).Normalize()

		hitBounceCount := castRayAndGetBounceCountForEvaluation(testSourcePos, direction, 0, tempCollidables, testListenerPos, listenerRadius)
		if hitBounceCount == 0 { // Direct hit
			currentListenerScore += BASE_DIRECT_HIT_SCORE
		} else if hitBounceCount > 0 { // Indirect hit
			fibIndex := hitBounceCount
			if fibIndex > FIBONACCI_SCORE_CAP_INDEX { // Cap Fibonacci index
				fibIndex = FIBONACCI_SCORE_CAP_INDEX
			}
			if fibIndex < len(fibonacciSequence) { // Ensure index is within bounds
				currentListenerScore += fibonacciSequence[fibIndex]
			}
		}
	}
	return currentListenerScore
}
