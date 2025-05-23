package main

import (
	"log"
	"math"
	"math/rand"
	"syscall/js"
	"time"
)

// Basic Sphere-AABB intersection check
func sphereIntersectsBox(spherePos Vector3, sphereRadius float64, box *SceneObject) bool {
	if box.ShapeType != "box" {
		return false // Should not happen if called correctly
	}

	// Closest point on AABB to sphere center
	boxMin := box.Position.Sub(box.Scale.Scale(0.5))
	boxMax := box.Position.Add(box.Scale.Scale(0.5))

	closestX := math.Max(boxMin.X, math.Min(spherePos.X, boxMax.X))
	closestY := math.Max(boxMin.Y, math.Min(spherePos.Y, boxMax.Y))
	closestZ := math.Max(boxMin.Z, math.Min(spherePos.Z, boxMax.Z))

	// Distance squared from sphere center to closest point on AABB
	distanceSq := (closestX-spherePos.X)*(closestX-spherePos.X) +
		(closestY-spherePos.Y)*(closestY-spherePos.Y) +
		(closestZ-spherePos.Z)*(closestZ-spherePos.Z)

	// If distance squared is less than radius squared, they intersect
	return distanceSq < (sphereRadius * sphereRadius)
}

func spheresIntersect(pos1 Vector3, radius1 float64, pos2 Vector3, radius2 float64) bool {
	distanceSq := (pos1.X-pos2.X)*(pos1.X-pos2.X) +
		(pos1.Y-pos2.Y)*(pos1.Y-pos2.Y) +
		(pos1.Z-pos2.Z)*(pos1.Z-pos2.Z)
	sumRadiiSq := (radius1 + radius2) * (radius1 + radius2)
	return distanceSq < (sumRadiiSq + EPSILON) // Add EPSILON for floating point robustness
}

func findAndApplyBestMoveForLearning(movingObject *SceneObject, fixedObject *SceneObject, goal string /* "maximize" */) {
	originalPos := movingObject.Position
	var currentScore int
	if movingObject == soundSource {
		currentScore = calculateListenerScore(originalPos, fixedObject.Position)
	} else { // movingObject is listener
		currentScore = calculateListenerScore(fixedObject.Position, originalPos)
	}

	bestScore := currentScore
	bestPositions := []Vector3{originalPos} // Store all positions that yield the best score

	// Define offsets for neighboring positions
	offsets := []float64{-OPTIMIZATION_STEP_SIZE, 0, OPTIMIZATION_STEP_SIZE}
	candidateTestPositions := []Vector3{}

	// Generate candidate positions
	for _, dx := range offsets {
		for _, dy := range offsets {
			for _, dz := range offsets {
				if dx == 0 && dy == 0 && dz == 0 {
					continue
				} // Skip original position for now

				testPos := Vector3{
					X: math.Max(-roomWidth/2+movingObject.Scale.X, math.Min(roomWidth/2-movingObject.Scale.X, originalPos.X+dx)),
					Y: math.Max(movingObject.Scale.Y, math.Min(roomHeight-wallThickness-movingObject.Scale.Y, originalPos.Y+dy)), // Ensure Y is above ground and below ceiling
					Z: math.Max(-roomDepth/2+movingObject.Scale.Z, math.Min(roomDepth/2-movingObject.Scale.Z, originalPos.Z+dz)),
				}
				// Ensure Y position is at least its own radius/scale from the effective ground
				if testPos.Y < movingObject.Scale.Y { // Assuming Scale.Y is relevant for height from ground
					testPos.Y = movingObject.Scale.Y
				}

				// Collision check: movingObject vs fixedObject
				if movingObject == soundSource && spheresIntersect(testPos, movingObject.Scale.X, fixedObject.Position, fixedObject.Scale.X) {
					continue
				}
				if movingObject == listener && spheresIntersect(testPos, movingObject.Scale.X, fixedObject.Position, fixedObject.Scale.X) {
					continue
				}

				// Collision check: movingObject vs staticSceneObjects
				collidesWithStatic := false
				for _, staticObj := range staticSceneObjects {
					if staticObj.ShapeType == "box" && sphereIntersectsBox(testPos, movingObject.Scale.X, staticObj) {
						collidesWithStatic = true
						break
					}
					// TODO: Add sphere-sphere check if static spheres are introduced
				}
				if collidesWithStatic {
					continue
				}

				// Avoid duplicate test positions
				isDuplicate := false
				for _, p := range candidateTestPositions {
					if math.Abs(p.X-testPos.X) < EPSILON && math.Abs(p.Y-testPos.Y) < EPSILON && math.Abs(p.Z-testPos.Z) < EPSILON {
						isDuplicate = true
						break
					}
				}
				if !isDuplicate {
					candidateTestPositions = append(candidateTestPositions, testPos)
				}
			}
		}
	}
	if len(candidateTestPositions) == 0 { // If all neighbors were invalid, consider original
		candidateTestPositions = append(candidateTestPositions, originalPos)
	}

	// Evaluate candidate positions
	for _, testPos := range candidateTestPositions {
		var score int
		if movingObject == soundSource {
			score = calculateListenerScore(testPos, fixedObject.Position)
		} else { // movingObject is listener
			score = calculateListenerScore(fixedObject.Position, testPos)
		}

		if goal == "maximize" {
			if score > bestScore {
				bestScore = score
				bestPositions = []Vector3{testPos} // New best, reset ties
			} else if score == bestScore {
				// Add to ties, ensuring it's a new position not already in bestPositions
				isNewBestPos := true
				for _, bp := range bestPositions {
					if math.Abs(bp.X-testPos.X) < EPSILON && math.Abs(bp.Y-testPos.Y) < EPSILON && math.Abs(bp.Z-testPos.Z) < EPSILON {
						isNewBestPos = false
						break
					}
				}
				if isNewBestPos {
					bestPositions = append(bestPositions, testPos)
				}
			}
		}
		// Add "minimize" goal logic if needed later
	}

	// Apply the best move found
	chosenPos := originalPos // Default to original if no improvement or no valid moves
	if len(bestPositions) > 0 {
		if bestScore > currentScore { // Improvement found
			chosenPos = bestPositions[rand.Intn(len(bestPositions))] // Pick randomly among the best
		} else { // No improvement or score is the same, consider random jump or sticking to one of the current bests
			if rand.Float64() < randomJumpProbability*explorationFactor {
				// Attempt a larger random jump
				jumpMagnitude := (rand.Float64()*2.0 + 2.0) * explorationFactor // More aggressive jump
				dx := (rand.Float64()*2 - 1) * OPTIMIZATION_STEP_SIZE * jumpMagnitude
				dy := (rand.Float64()*0.5 - 0.25) * OPTIMIZATION_STEP_SIZE * jumpMagnitude // Less vertical jump
				dz := (rand.Float64()*2 - 1) * OPTIMIZATION_STEP_SIZE * jumpMagnitude

				jumpPos := Vector3{
					X: math.Max(-roomWidth/2+movingObject.Scale.X, math.Min(roomWidth/2-movingObject.Scale.X, originalPos.X+dx)),
					Y: math.Max(movingObject.Scale.Y, math.Min(roomHeight-wallThickness-movingObject.Scale.Y, originalPos.Y+dy)),
					Z: math.Max(-roomDepth/2+movingObject.Scale.Z, math.Min(roomDepth/2-movingObject.Scale.Z, originalPos.Z+dz)),
				}
				if jumpPos.Y < movingObject.Scale.Y {
					jumpPos.Y = movingObject.Scale.Y
				}

				collidesWithStaticJump := false
				for _, staticObj := range staticSceneObjects {
					if staticObj.ShapeType == "box" && sphereIntersectsBox(jumpPos, movingObject.Scale.X, staticObj) {
						collidesWithStaticJump = true
						break
					}
				}

				// Check if jump position is valid
				if !(movingObject == soundSource && spheresIntersect(jumpPos, movingObject.Scale.X, fixedObject.Position, fixedObject.Scale.X)) &&
					!(movingObject == listener && spheresIntersect(jumpPos, movingObject.Scale.X, fixedObject.Position, fixedObject.Scale.X)) &&
					!collidesWithStaticJump {
					chosenPos = jumpPos
				} else if len(bestPositions) > 0 { // Fallback to one of the (equally good or original) positions
					chosenPos = bestPositions[rand.Intn(len(bestPositions))]
				}

			} else if len(bestPositions) > 0 { // No jump, but pick from existing bests (could be original)
				chosenPos = bestPositions[rand.Intn(len(bestPositions))]
				// If chosenPos is still originalPos and there were other equally good options, try to pick one of those
				if chosenPos.X == originalPos.X && chosenPos.Y == originalPos.Y && chosenPos.Z == originalPos.Z && len(bestPositions) > 1 {
					tempBests := []Vector3{}
					for _, bp := range bestPositions {
						if math.Abs(bp.X-originalPos.X) > EPSILON || math.Abs(bp.Y-originalPos.Y) > EPSILON || math.Abs(bp.Z-originalPos.Z) > EPSILON {
							tempBests = append(tempBests, bp)
						}
					}
					if len(tempBests) > 0 {
						chosenPos = tempBests[rand.Intn(len(tempBests))]
					}
				}
			}
		}
	}
	movingObject.Position = chosenPos
}

func runLearningCycle() {
	defer recoverFromPanic("runLearningCycle")
	log.Println("Learning cycle goroutine started.")

	for currentLearningIteration < maxLearningIterations && learningModeActive {
		currentLearningIteration++

		var movingObject *SceneObject
		var fixedObject *SceneObject

		if isSoundSourceTurn {
			movingObject = soundSource
			fixedObject = listener
		} else {
			movingObject = listener
			fixedObject = soundSource
		}

		if movingObject == nil || fixedObject == nil {
			log.Println("Error: soundSource or listener is nil in learning cycle.")
			learningModeActive = false // Stop learning
			break
		}

		findAndApplyBestMoveForLearning(movingObject, fixedObject, "maximize")

		visualizeSoundPropagation() // This updates global listenerRayScore and sends data to JS
		// globalBestScore is updated inside visualizeSoundPropagation if listenerRayScore is higher

		// Update UI with progress
		js.Global().Call("updateLearningProgress", currentLearningIteration, maxLearningIterations, globalBestScore)
		// Update slider values for the moved object
		js.Global().Call("updateSliderValuesForObject", "SoundSource", soundSource.Position.X, soundSource.Position.Y, soundSource.Position.Z)
		js.Global().Call("updateSliderValuesForObject", "Listener", listener.Position.X, listener.Position.Y, listener.Position.Z)

		isSoundSourceTurn = !isSoundSourceTurn // Alternate turns

		if autoTurnDelay > 0 {
			time.Sleep(autoTurnDelay)
		}

		if !learningModeActive { // Check if stopped externally
			log.Println("Learning mode stopped during iteration.")
			break
		}
	}

	if learningModeActive { // If loop finished due to max iterations
		log.Println("Max learning iterations reached.")
	}
	learningModeActive = false
	jsGlobal.Call("updateLearningButton", false, "Start Learning (Coop. Maximize)") // Ensure button state is correct

	// After learning finishes (or is stopped), apply the globally best found settings
	if soundSource != nil && listener != nil && globalBestSettings.Score > -1 { // Check if any best score was actually found
		log.Printf("Learning finished. Applying global best settings. Score: %d", globalBestSettings.Score)
		soundSource.Position = globalBestSettings.SoundSourcePos
		listener.Position = globalBestSettings.ListenerPos

		// Apply other parameters from globalBestSettings
		numRays = globalBestSettings.NumRays
		initialRayOpacity = globalBestSettings.InitialRayOpacity
		maxReflections = globalBestSettings.MaxReflections
		volumeAttenuationFactor = globalBestSettings.VolumeAttenuationFactor
		explorationFactor = globalBestSettings.ExplorationFactor
		showOnlyListenerRays = globalBestSettings.ShowOnlyListenerRays

		// Update all UI sliders to reflect these best settings
		jsGlobal.Call("updateAllUISliders",
			numRays, initialRayOpacity, maxReflections, volumeAttenuationFactor, explorationFactor,
			soundSource.Position.X, soundSource.Position.Y, soundSource.Position.Z,
			listener.Position.X, listener.Position.Y, listener.Position.Z,
			showOnlyListenerRays,
		)
		jsGlobal.Call("updateLearningProgress", currentLearningIteration, maxLearningIterations, globalBestScore) // Final update
		visualizeSoundPropagation()                                                                               // Final visualization with best settings
		log.Printf("Best settings applied: %+v", globalBestSettings)
	} else {
		log.Println("Learning finished. No global best settings to apply or objects are nil.")
	}
	log.Printf("Learning finished. Final best score: %d. Iterations: %d", globalBestScore, currentLearningIteration)
}

func goStartLearningMode(this js.Value, args []js.Value) interface{} {
	defer recoverFromPanic("goStartLearningMode")
	if learningModeActive {
		log.Println("Learning mode already running.")
		return nil
	}
	log.Println("Starting Learning Mode (Cooperative Maximize)...")
	learningModeActive = true
	currentLearningIteration = 0
	globalBestScore = -1 // Reset global best score for this run

	// Initialize globalBestSettings with current settings as a baseline or reset
	if soundSource != nil {
		globalBestSettings.SoundSourcePos = soundSource.Position
	}
	if listener != nil {
		globalBestSettings.ListenerPos = listener.Position
	}
	globalBestSettings.Score = -1 // Indicates no score yet better than initial
	globalBestSettings.Iteration = 0
	globalBestSettings.NumRays = numRays
	globalBestSettings.InitialRayOpacity = initialRayOpacity
	globalBestSettings.MaxReflections = maxReflections
	globalBestSettings.VolumeAttenuationFactor = volumeAttenuationFactor
	globalBestSettings.ExplorationFactor = explorationFactor
	globalBestSettings.ShowOnlyListenerRays = showOnlyListenerRays
	// globalBestSettings.AllObjectSnapshots = takeSnapshots() // If you implement snapshotting for records

	isSoundSourceTurn = true // Sound source starts

	jsGlobal.Call("updateLearningButton", true, "Stop Learning (Coop. Maximize)")
	jsGlobal.Call("updateLearningProgress", 0, maxLearningIterations, 0)

	go runLearningCycle() // Start the learning process in a new goroutine
	return nil
}

func goStopLearningMode(this js.Value, args []js.Value) interface{} {
	defer recoverFromPanic("goStopLearningMode")
	if !learningModeActive {
		log.Println("Learning mode is not running.")
		return nil
	}
	log.Println("Stopping Learning Mode requested...")
	learningModeActive = false // Signal the learning goroutine to stop
	// The runLearningCycle will handle cleanup and applying best settings upon exit
	return nil
}
