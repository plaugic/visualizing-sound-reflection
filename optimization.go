package main

import (
	"log"
	"math"
	"math/rand"
	"syscall/js"
	"time"
)

// Basic Sphere-AABB intersection check (can be used as a utility or fallback)
func sphereIntersectsBox(spherePos Vector3, sphereRadius float64, box *SceneObject) bool {
	if box.ShapeType != "box" {
		return false
	}
	boxMin := box.Position.Sub(box.Scale.Scale(0.5))
	boxMax := box.Position.Add(box.Scale.Scale(0.5))
	closestX := math.Max(boxMin.X, math.Min(spherePos.X, boxMax.X))
	closestY := math.Max(boxMin.Y, math.Min(spherePos.Y, boxMax.Y))
	closestZ := math.Max(boxMin.Z, math.Min(spherePos.Z, boxMax.Z))
	distanceSq := (closestX-spherePos.X)*(closestX-spherePos.X) +
		(closestY-spherePos.Y)*(closestY-spherePos.Y) +
		(closestZ-spherePos.Z)*(closestZ-spherePos.Z)
	return distanceSq < (sphereRadius * sphereRadius)
}

// Basic Sphere-Sphere intersection check (can be used as a utility or fallback)
func spheresIntersect(pos1 Vector3, radius1 float64, pos2 Vector3, radius2 float64) bool {
	distanceSq := (pos1.X-pos2.X)*(pos1.X-pos2.X) +
		(pos1.Y-pos2.Y)*(pos1.Y-pos2.Y) +
		(pos1.Z-pos2.Z)*(pos1.Z-pos2.Z)
	sumRadiiSq := (radius1 + radius2) * (radius1 + radius2)
	return distanceSq < (sumRadiiSq + EPSILON)
}

func findAndApplyBestMoveForLearning(movingObject *SceneObject, fixedObject *SceneObject, goal string /* "maximize" */) {
	originalPos := movingObject.Position // Position of the object at the start of this optimization step
	var currentScore int
	var movingObjCloudState PointState
	var otherObjCurrentPos Vector3
	var otherObjScale Vector3

	if movingObject == soundSource {
		currentScore = calculateListenerScore(originalPos, fixedObject.Position)
		movingObjCloudState = StateSoundSource
	} else { // movingObject is listener
		currentScore = calculateListenerScore(fixedObject.Position, originalPos)
		movingObjCloudState = StateListener
	}
	otherObjCurrentPos = fixedObject.Position
	otherObjScale = fixedObject.Scale

	bestScore := currentScore
	bestPositions := []Vector3{originalPos}

	offsets := []float64{-OPTIMIZATION_STEP_SIZE, 0, OPTIMIZATION_STEP_SIZE}
	candidateTestPositions := []Vector3{}

	for _, dx := range offsets {
		for _, dy := range offsets {
			for _, dz := range offsets {
				if dx == 0 && dy == 0 && dz == 0 { // No change from originalPos (already evaluated as currentScore)
					continue
				}

				testPos := Vector3{
					X: math.Max(occupancyCloud.RoomMin.X+movingObject.Scale.X/2, math.Min(occupancyCloud.RoomMax.X-movingObject.Scale.X/2, originalPos.X+dx)),
					Y: math.Max(occupancyCloud.RoomMin.Y+movingObject.Scale.Y/2, math.Min(occupancyCloud.RoomMax.Y-movingObject.Scale.Y/2, originalPos.Y+dy)),
					Z: math.Max(occupancyCloud.RoomMin.Z+movingObject.Scale.Z/2, math.Min(occupancyCloud.RoomMax.Z-movingObject.Scale.Z/2, originalPos.Z+dz)),
				}

				// Ensure Y position is at least its own radius/scale from the effective ground (cloud min Y)
				minPossibleY := occupancyCloud.RoomMin.Y + movingObject.Scale.Y/2.0
				if testPos.Y < minPossibleY {
					testPos.Y = minPossibleY
				}
				maxPossibleY := occupancyCloud.RoomMax.Y - movingObject.Scale.Y/2.0
				if testPos.Y > maxPossibleY {
					testPos.Y = maxPossibleY
				}

				// Use OccupancyCloud for collision checks
				if occupancyCloud != nil {
					isValidCloudPos := occupancyCloud.IsPositionAttemptValid(testPos, movingObject.Scale, movingObjCloudState, otherObjCurrentPos, otherObjScale)
					if !isValidCloudPos {
						if occupancyCloud.DebugLogging {
							// log.Printf("Cloud: Candidate pos %v for %s rejected.", testPos, movingObject.Name)
						}
						continue // Skip this candidate position
					}
				} else {
					// Fallback to old direct collision logic if cloud is not initialized
					if spheresIntersect(testPos, movingObject.Scale.X/2.0, otherObjCurrentPos, otherObjScale.X/2.0) {
						continue
					}
					collidesWithStatic := false
					for _, staticObj := range staticSceneObjects { // Assuming staticSceneObjects is accessible
						if staticObj.ShapeType == "box" && sphereIntersectsBox(testPos, movingObject.Scale.X/2.0, staticObj) {
							collidesWithStatic = true
							break
						}
						// Add sphere-sphere for static spheres if any
					}
					if collidesWithStatic {
						continue
					}
				}

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

	// Also consider the original position if no other candidates were found (though it's already in bestPositions)
	if len(candidateTestPositions) == 0 && len(bestPositions) == 1 && bestPositions[0] == originalPos {
		// No valid moves found, will stick to original or try random jump
	}

	for _, testPos := range candidateTestPositions {
		var score int
		if movingObject == soundSource {
			score = calculateListenerScore(testPos, fixedObject.Position)
		} else {
			score = calculateListenerScore(fixedObject.Position, testPos)
		}

		if goal == "maximize" {
			if score > bestScore {
				bestScore = score
				bestPositions = []Vector3{testPos}
			} else if score == bestScore {
				isNewBestPos := true
				for _, bp := range bestPositions { // Avoid adding duplicates to bestPositions
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
	}

	chosenPos := originalPos
	if len(bestPositions) > 0 {
		if bestScore > currentScore {
			chosenPos = bestPositions[rand.Intn(len(bestPositions))]
		} else { // No improvement or score is the same
			if rand.Float64() < randomJumpProbability*explorationFactor {
				jumpMagnitude := (rand.Float64()*2.0 + 2.0) * explorationFactor
				dx := (rand.Float64()*2 - 1) * OPTIMIZATION_STEP_SIZE * jumpMagnitude
				dy := (rand.Float64()*0.5 - 0.25) * OPTIMIZATION_STEP_SIZE * jumpMagnitude // Smaller vertical jumps
				dz := (rand.Float64()*2 - 1) * OPTIMIZATION_STEP_SIZE * jumpMagnitude

				jumpPos := Vector3{
					X: math.Max(occupancyCloud.RoomMin.X+movingObject.Scale.X/2, math.Min(occupancyCloud.RoomMax.X-movingObject.Scale.X/2, originalPos.X+dx)),
					Y: math.Max(occupancyCloud.RoomMin.Y+movingObject.Scale.Y/2, math.Min(occupancyCloud.RoomMax.Y-movingObject.Scale.Y/2, originalPos.Y+dy)),
					Z: math.Max(occupancyCloud.RoomMin.Z+movingObject.Scale.Z/2, math.Min(occupancyCloud.RoomMax.Z-movingObject.Scale.Z/2, originalPos.Z+dz)),
				}
				minPossibleY := occupancyCloud.RoomMin.Y + movingObject.Scale.Y/2.0
				if jumpPos.Y < minPossibleY {
					jumpPos.Y = minPossibleY
				}
				maxPossibleY := occupancyCloud.RoomMax.Y - movingObject.Scale.Y/2.0
				if jumpPos.Y > maxPossibleY {
					jumpPos.Y = maxPossibleY
				}

				isValidJump := false
				if occupancyCloud != nil {
					isValidJump = occupancyCloud.IsPositionAttemptValid(jumpPos, movingObject.Scale, movingObjCloudState, otherObjCurrentPos, otherObjScale)
				} else {
					// Fallback jump collision check
					if !spheresIntersect(jumpPos, movingObject.Scale.X/2.0, otherObjCurrentPos, otherObjScale.X/2.0) {
						collidesWithStaticJump := false
						for _, staticObj := range staticSceneObjects {
							if staticObj.ShapeType == "box" && sphereIntersectsBox(jumpPos, movingObject.Scale.X/2.0, staticObj) {
								collidesWithStaticJump = true
								break
							}
						}
						if !collidesWithStaticJump {
							isValidJump = true
						}
					}
				}

				if isValidJump {
					chosenPos = jumpPos
					if occupancyCloud.DebugLogging {
						log.Printf("Cloud: %s made a random jump to %v", movingObject.Name, chosenPos)
					}
				} else if len(bestPositions) > 0 { // Fallback if jump is invalid
					chosenPos = bestPositions[rand.Intn(len(bestPositions))]
				}
			} else if len(bestPositions) > 0 { // No jump, but pick from existing (equally good or original) positions
				chosenPos = bestPositions[rand.Intn(len(bestPositions))]
				// Try to pick a non-original position if current is original and others exist
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

	// Commit the move
	movingObject.Position = chosenPos

	// Update the occupancy cloud with the new position of the object that moved
	if occupancyCloud != nil {
		occupancyCloud.UpdateObjectInCloud(movingObject.Name, originalPos, movingObject.Position, movingObject.Scale, movingObjCloudState)
	}
}

func runLearningCycle() {
	defer recoverFromPanic("runLearningCycle")
	log.Println("Learning cycle goroutine started.")

	// Initial cloud update for sound source and listener based on their starting positions in the scene
	if occupancyCloud != nil {
		if soundSource != nil {
			occupancyCloud.UpdateObjectInCloud("SoundSource", soundSource.Position, soundSource.Position, soundSource.Scale, StateSoundSource)
		}
		if listener != nil {
			occupancyCloud.UpdateObjectInCloud("Listener", listener.Position, listener.Position, listener.Scale, StateListener)
		}
	}

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
			learningModeActive = false
			break
		}

		findAndApplyBestMoveForLearning(movingObject, fixedObject, "maximize")
		// Note: OccupancyCloud is updated *inside* findAndApplyBestMoveForLearning after the move.

		visualizeSoundPropagation() // This updates global listenerRayScore and sends data to JS

		js.Global().Call("updateLearningProgress", currentLearningIteration, maxLearningIterations, globalBestScore)
		js.Global().Call("updateSliderValuesForObject", "SoundSource", soundSource.Position.X, soundSource.Position.Y, soundSource.Position.Z)
		js.Global().Call("updateSliderValuesForObject", "Listener", listener.Position.X, listener.Position.Y, listener.Position.Z)

		isSoundSourceTurn = !isSoundSourceTurn

		if autoTurnDelay > 0 {
			time.Sleep(autoTurnDelay)
		}
		if !learningModeActive {
			log.Println("Learning mode stopped during iteration.")
			break
		}
	}

	if learningModeActive {
		log.Println("Max learning iterations reached.")
	}
	learningModeActive = false
	jsGlobal.Call("updateLearningButton", false, "Start Learning (Coop. Maximize)")

	if soundSource != nil && listener != nil && globalBestSettings.Score > -1 {
		log.Printf("Learning finished. Applying global best settings. Score: %d", globalBestSettings.Score)

		originalSoundSourcePos := soundSource.Position
		originalListenerPos := listener.Position

		soundSource.Position = globalBestSettings.SoundSourcePos
		listener.Position = globalBestSettings.ListenerPos

		// Update cloud for final positions
		if occupancyCloud != nil {
			occupancyCloud.UpdateObjectInCloud("SoundSource", originalSoundSourcePos, soundSource.Position, soundSource.Scale, StateSoundSource)
			occupancyCloud.UpdateObjectInCloud("Listener", originalListenerPos, listener.Position, listener.Scale, StateListener)
		}

		numRays = globalBestSettings.NumRays
		initialRayOpacity = globalBestSettings.InitialRayOpacity
		maxReflections = globalBestSettings.MaxReflections
		volumeAttenuationFactor = globalBestSettings.VolumeAttenuationFactor
		explorationFactor = globalBestSettings.ExplorationFactor
		showOnlyListenerRays = globalBestSettings.ShowOnlyListenerRays

		jsGlobal.Call("updateAllUISliders",
			numRays, initialRayOpacity, maxReflections, volumeAttenuationFactor, explorationFactor,
			soundSource.Position.X, soundSource.Position.Y, soundSource.Position.Z,
			listener.Position.X, listener.Position.Y, listener.Position.Z,
			showOnlyListenerRays,
		)
		jsGlobal.Call("updateLearningProgress", currentLearningIteration, maxLearningIterations, globalBestScore)
		visualizeSoundPropagation()
		log.Printf("Best settings applied: %+v", globalBestSettings)
	} else {
		log.Println("Learning finished. No global best settings to apply or objects are nil.")
	}
	log.Printf("Learning cycle finished. Final best score: %d. Iterations: %d", globalBestScore, currentLearningIteration)
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
	globalBestScore = -1

	if soundSource != nil {
		globalBestSettings.SoundSourcePos = soundSource.Position
	}
	if listener != nil {
		globalBestSettings.ListenerPos = listener.Position
	}
	globalBestSettings.Score = -1
	globalBestSettings.Iteration = 0
	globalBestSettings.NumRays = numRays
	globalBestSettings.InitialRayOpacity = initialRayOpacity
	globalBestSettings.MaxReflections = maxReflections
	globalBestSettings.VolumeAttenuationFactor = volumeAttenuationFactor
	globalBestSettings.ExplorationFactor = explorationFactor
	globalBestSettings.ShowOnlyListenerRays = showOnlyListenerRays

	isSoundSourceTurn = true

	// Ensure cloud is up-to-date with initial positions before starting learning cycle
	if occupancyCloud != nil {
		if soundSource != nil {
			occupancyCloud.UpdateObjectInCloud("SoundSource", soundSource.Position, soundSource.Position, soundSource.Scale, StateSoundSource)
		}
		if listener != nil {
			occupancyCloud.UpdateObjectInCloud("Listener", listener.Position, listener.Position, listener.Scale, StateListener)
		}
		if occupancyCloud.DebugLogging {
			log.Println("Occupancy cloud states confirmed for SoundSource and Listener before starting learning.")
		}
	}

	jsGlobal.Call("updateLearningButton", true, "Stop Learning (Coop. Maximize)")
	jsGlobal.Call("updateLearningProgress", 0, maxLearningIterations, globalBestScore)

	go runLearningCycle()
	return nil
}

func goStopLearningMode(this js.Value, args []js.Value) interface{} {
	defer recoverFromPanic("goStopLearningMode")
	if !learningModeActive {
		log.Println("Learning mode is not running.")
		return nil
	}
	log.Println("Stopping Learning Mode requested...")
	learningModeActive = false
	return nil
}
