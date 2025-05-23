package main

import (
	"log"
	"math"
	"math/rand"
	"runtime/debug" // For more detailed panic stack
	"strconv"
	"syscall/js"
	"time"
)

// --- Global Constants ---
const (
	MAX_RAY_DISTANCE          float64 = 50.0
	EPSILON                   float64 = 0.00001
	OPTIMIZATION_STEP_SIZE    float64 = 0.5 // Step size for object movement in optimization
	FIBONACCI_SCORE_CAP_INDEX int     = 20  // Cap Fibonacci index for scoring
	BASE_DIRECT_HIT_SCORE     int     = 10  // Score for a direct hit
)

// --- Global State ---
var (
	jsGlobal      js.Value    // Global JavaScript object
	debounceTimer *time.Timer // Timer for debouncing UI updates

	// Scene objects
	allSceneObjects    []*SceneObject // All objects in the scene
	staticSceneObjects []*SceneObject // Objects that don't move during optimization
	soundSource        *SceneObject   // The source of the sound rays
	listener           *SceneObject   // The target for the sound rays
	wallCeilingMeshes  []*SceneObject // Specific meshes for walls/ceiling for opacity updates

	// Ray visualization & scoring
	rayVisuals       []*RayLine // Holds data for rays to be visualized
	listenerRayScore int        // Current score based on rays reaching the listener

	// Camera (from JS perspective)
	mainCamera struct {
		Position Vector3
		Target   Vector3
	}

	// Room dimensions
	roomWidth     float64 = 40
	roomDepth     float64 = 40
	roomHeight    float64 = 10
	wallThickness float64 = 0.2

	// Simulation parameters (can be changed by UI)
	numRays                 int           = 1000
	initialRayOpacity       float64       = 0.6
	maxReflections          int           = 3
	currentWallOpacity      float64       = 1.0  // Opacity for walls/ceiling
	showOnlyListenerRays    bool          = true // Filter for ray visualization
	currentDebounceTime     time.Duration = 500 * time.Millisecond
	debouncedVisualizeFunc  func()               // Debounced version of visualizeSoundPropagation
	volumeAttenuationFactor float64       = 0.85 // How much opacity reduces per bounce
	explorationFactor       float64       = 1.0  // Multiplier for randomness in learning

	// Learning Mode State
	learningModeActive       bool = false
	currentLearningIteration int
	maxLearningIterations    int               = 50000
	globalBestScore          int               = -1                   // Stores the highest score found during learning
	globalBestSettings       BestScoreSettings                        // Stores all settings related to globalBestScore
	isSoundSourceTurn        bool              = true                 // For alternating moves in learning mode
	randomJumpProbability    float64           = 0.1                  // Base probability of a random jump if no improvement
	autoTurnDelay            time.Duration     = 5 * time.Microsecond // Delay between learning turns

	// Ray colors
	bounceColors = []uint32{
		0xffff00, // 0 bounces (direct - though listenerRayColor often overrides)
		0xffa500, // 1 bounce
		0xff00ff, // 2 bounces
		0x00ffff, // 3 bounces
		0x00fa9a, // 4
		0xdda0dd, // 5
		0xfa8072, // 6
		0xadd8e6, // 7
		0xf0e68c, // 8
		0x90ee90, // 9
		0xffc0cb, // 10
		// Add more if maxReflections can be higher and distinct colors are desired
	}
	listenerRayColor uint32 = 0x00ff00 // Green for rays hitting the listener

	// Precomputed data
	fibonacciSequence []int         // Stores Fibonacci numbers for scoring
	recordsManager    RecordManager // Manages best score records
)

func precomputeFibonacci(n int) {
	fibonacciSequence = make([]int, n+1)
	if n >= 0 {
		fibonacciSequence[0] = 0 // Or 1, depending on how you want to score 0 bounces (direct)
	}
	if n >= 1 {
		fibonacciSequence[1] = 1
	}
	for i := 2; i <= n; i++ {
		fibonacciSequence[i] = fibonacciSequence[i-1] + fibonacciSequence[i-2]
		if fibonacciSequence[i] < 0 { // Overflow protection for int
			fibonacciSequence[i] = fibonacciSequence[i-1] // Cap at previous value
		}
	}
}

func recoverFromPanic(funcName string) {
	if r := recover(); r != nil {
		log.Printf("PANIC RECOVERED in %s: %v\n%s", funcName, r, string(debug.Stack()))
		// If panic occurs during learning, try to stop learning mode gracefully
		if funcName == "runLearningCycle" || funcName == "findAndApplyBestMoveForLearning" {
			if learningModeActive {
				learningModeActive = false
				jsGlobal.Call("updateLearningButton", false, "Start Learning (Coop. Maximize)")
			}
		}
	}
}

func debounce(f func(), d time.Duration) func() {
	return func() {
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(d, f)
	}
}

func main() {
	defer recoverFromPanic("main") // Catch panics in the main setup

	jsGlobal = js.Global()
	log.Println("Go WASM Initializing...")
	rand.Seed(time.Now().UnixNano()) // Seed random number generator

	precomputeFibonacci(FIBONACCI_SCORE_CAP_INDEX)
	recordsManager = *NewRecordManager(10) // Store top 10 records

	createSceneContent() // Initialize 3D objects

	// --- Register Go functions to be callable from JavaScript ---
	jsGlobal.Set("goUpdateSliderValue", js.FuncOf(goUpdateSliderValue))
	jsGlobal.Set("goUpdateToggleValue", js.FuncOf(goUpdateToggleValue))
	jsGlobal.Set("goTriggerVisualizeSound", js.FuncOf(goTriggerVisualizeSound))
	jsGlobal.Set("goTriggerClearRays", js.FuncOf(goTriggerClearRays))
	jsGlobal.Set("goUpdateCameraState", js.FuncOf(goUpdateCameraState)) // For JS to inform Go about camera changes
	jsGlobal.Set("goUpdateSoundSourcePositionAndVisualize", js.FuncOf(goUpdateSoundSourcePositionAndVisualize))
	jsGlobal.Set("goUpdateListenerPositionAndVisualize", js.FuncOf(goUpdateListenerPositionAndVisualize))

	// Learning mode JS functions
	jsGlobal.Set("goStartLearningMode", js.FuncOf(goStartLearningMode))
	jsGlobal.Set("goStopLearningMode", js.FuncOf(goStopLearningMode))
	jsGlobal.Set("goApplyRecordedSettingsByIndex", js.FuncOf(goApplyRecordedSettingsByIndex))
	// jsGlobal.Set("goToggleAutoOptimization", js.FuncOf(goToggleAutoOptimization)) // If you add another optimization mode

	debouncedVisualizeFunc = debounce(visualizeSoundPropagation, currentDebounceTime)

	jsGlobal.Call("goWasmReady") // Signal to JS that WASM is ready

	// Perform initial visualization
	go func() { // Run in a goroutine to avoid blocking main, though JS interop needs care
		defer recoverFromPanic("initialVisualizeAndLegend")
		debouncedVisualizeFunc()
		updateRayLegendJS()
	}()

	log.Println("Go WASM setup complete. Entering blocking select to keep alive.")
	select {} // Keep the Go program running (WASM requirement)
}

// --- JS Interop Functions (Callbacks from JavaScript) ---

func goUpdateSliderValue(this js.Value, args []js.Value) interface{} {
	defer recoverFromPanic("goUpdateSliderValue")
	if len(args) != 2 {
		log.Println("Error: goUpdateSliderValue expects 2 arguments (sliderName, value)")
		return nil
	}
	sliderName := args[0].String()
	value := args[1].Float()

	needsVisualUpdate := true
	switch sliderName {
	// Sound Source Position
	case "soundSourceX":
		if soundSource != nil {
			soundSource.Position.X = value
		}
	case "soundSourceY":
		if soundSource != nil {
			soundSource.Position.Y = value
		}
	case "soundSourceZ":
		if soundSource != nil {
			soundSource.Position.Z = value
		}
	// Listener Position
	case "listenerX":
		if listener != nil {
			listener.Position.X = value
		}
	case "listenerY":
		if listener != nil {
			listener.Position.Y = value
		}
	case "listenerZ":
		if listener != nil {
			listener.Position.Z = value
		}
	// Ray & Simulation Parameters
	case "numRays":
		numRays = int(value)
	case "rayOpacity":
		initialRayOpacity = value
	case "maxBounces":
		maxReflections = int(value)
		updateRayLegendJS() // Legend depends on max bounces
	case "volume": // This is volumeAttenuationFactor
		volumeAttenuationFactor = value
	case "explorationFactor":
		explorationFactor = value
	// Environment & Performance
	case "wallOpacity":
		currentWallOpacity = value
		for _, wallObj := range wallCeilingMeshes { // Update material properties directly
			wallObj.Material.Color[3] = float32(currentWallOpacity)
			wallObj.Material.IsTransparent = currentWallOpacity < 1.0
		}
		needsVisualUpdate = false      // Does not require re-casting rays, just re-render
		jsGlobal.Call("requestRender") // Tell JS to re-render the scene graph
	case "debounceTime":
		newDebounceTime := time.Duration(int(value)) * time.Millisecond
		if newDebounceTime != currentDebounceTime {
			currentDebounceTime = newDebounceTime
			debouncedVisualizeFunc = debounce(visualizeSoundPropagation, currentDebounceTime)
		}
		needsVisualUpdate = false // No immediate visual update from this change
	default:
		log.Printf("Unknown slider: %s", sliderName)
		needsVisualUpdate = false
	}

	if needsVisualUpdate {
		if !learningModeActive {
			debouncedVisualizeFunc()
		} else {
			visualizeSoundPropagation() // In learning mode, update immediately
		}
	}
	return nil
}

func goUpdateSoundSourcePositionAndVisualize(this js.Value, args []js.Value) interface{} {
	defer recoverFromPanic("goUpdateSoundSourcePositionAndVisualize")
	if len(args) != 3 || soundSource == nil {
		return nil
	}
	soundSource.Position.X = args[0].Float()
	soundSource.Position.Y = args[1].Float()
	soundSource.Position.Z = args[2].Float()
	if !learningModeActive { // Only visualize if not in learning mode (learning mode has its own viz calls)
		visualizeSoundPropagation()
	}
	return nil
}

func goUpdateListenerPositionAndVisualize(this js.Value, args []js.Value) interface{} {
	defer recoverFromPanic("goUpdateListenerPositionAndVisualize")
	if len(args) != 3 || listener == nil {
		return nil
	}
	listener.Position.X = args[0].Float()
	listener.Position.Y = args[1].Float()
	listener.Position.Z = args[2].Float()
	if !learningModeActive {
		visualizeSoundPropagation()
	}
	return nil
}

func goUpdateToggleValue(this js.Value, args []js.Value) interface{} {
	defer recoverFromPanic("goUpdateToggleValue")
	if len(args) != 2 {
		return nil
	}
	toggleName := args[0].String()
	checked := args[1].Bool()
	switch toggleName {
	case "showOnlyListenerRays":
		showOnlyListenerRays = checked
		if !learningModeActive {
			debouncedVisualizeFunc()
		} else {
			visualizeSoundPropagation()
		}
	default:
		log.Printf("Unknown toggle: %s", toggleName)
	}
	return nil
}

func goTriggerVisualizeSound(this js.Value, args []js.Value) interface{} {
	defer recoverFromPanic("goTriggerVisualizeSound")
	if !learningModeActive {
		debouncedVisualizeFunc()
	} else {
		visualizeSoundPropagation() // If learning, visualize immediately
	}
	return nil
}

func goTriggerClearRays(this js.Value, args []js.Value) interface{} {
	defer recoverFromPanic("goTriggerClearRays")
	clearRayVisualsAndNotifyJS()
	return nil
}

func goUpdateCameraState(this js.Value, args []js.Value) interface{} {
	defer recoverFromPanic("goUpdateCameraState")
	if len(args) != 6 {
		return nil
	}
	// These are for Go to be aware of camera state if needed, not typically used by simulation
	mainCamera.Position.X = args[0].Float()
	mainCamera.Position.Y = args[1].Float()
	mainCamera.Position.Z = args[2].Float()
	mainCamera.Target.X = args[3].Float()
	mainCamera.Target.Y = args[4].Float()
	mainCamera.Target.Z = args[5].Float()
	return nil
}

func clearRayVisualsAndNotifyJS() {
	defer recoverFromPanic("clearRayVisualsAndNotifyJS")
	rayVisuals = []*RayLine{}      // Clear the Go-side ray data
	jsGlobal.Call("clearRaysJS")   // Tell JS to clear Three.js ray objects
	jsGlobal.Call("requestRender") // Tell JS to re-render the (now empty of rays) scene
}

// --- Core Simulation & Visualization Logic ---
func visualizeSoundPropagation() {
	defer recoverFromPanic("visualizeSoundPropagation")

	if soundSource == nil || listener == nil {
		log.Println("Sound source or listener is nil, cannot visualize.")
		return
	}

	rayVisuals = []*RayLine{} // Clear previous rays before new calculation
	currentWeightedScore := 0

	sourcePos := soundSource.Position
	listenerPos := listener.Position
	listenerRadius := listener.Scale.X // Assuming uniform scale for listener sphere

	// Prepare collidable objects (all except the source itself for the first ray segment)
	var collidables []*SceneObject
	for _, obj := range allSceneObjects {
		if obj != soundSource { // Direct rays from source don't collide with source itself
			collidables = append(collidables, obj)
		}
	}

	for i := 0; i < numRays; i++ {
		// Fibonacci sphere algorithm for even ray distribution
		phi := math.Acos(-1 + (2*float64(i))/float64(numRays))
		theta := math.Sqrt(float64(numRays)*math.Pi) * phi
		direction := SetFromSphericalCoords(1, phi, theta).Normalize()

		hitData := castRayAndAddVisuals(sourcePos, direction, 0, collidables, listenerPos, listenerRadius)
		if hitData.hitListener {
			if hitData.bounces == 0 {
				currentWeightedScore += BASE_DIRECT_HIT_SCORE
			} else {
				fibIndex := hitData.bounces
				if fibIndex > FIBONACCI_SCORE_CAP_INDEX {
					fibIndex = FIBONACCI_SCORE_CAP_INDEX
				}
				if fibIndex >= 0 && fibIndex < len(fibonacciSequence) {
					currentWeightedScore += fibonacciSequence[fibIndex]
				}
			}
		}
	}

	listenerRayScore = currentWeightedScore

	// If in learning mode, check if this is a new best score
	if learningModeActive && listenerRayScore > globalBestScore {
		globalBestScore = listenerRayScore

		// Capture all settings that led to this new best score
		currentSettingsSnapshot := BestScoreSettings{
			Score:                   globalBestScore,
			Iteration:               currentLearningIteration,
			NumRays:                 numRays,
			InitialRayOpacity:       initialRayOpacity,
			MaxReflections:          maxReflections,
			VolumeAttenuationFactor: volumeAttenuationFactor,
			ExplorationFactor:       explorationFactor,
			SoundSourcePos:          soundSource.Position, // Current position that yielded this score
			ListenerPos:             listener.Position,    // Current position
			ShowOnlyListenerRays:    showOnlyListenerRays,
			// AllObjectSnapshots:   takeSnapshots(), // If you want to save the state of ALL objects
		}
		recordsManager.AddRecord(currentSettingsSnapshot) // Add to historical records list
		globalBestSettings = currentSettingsSnapshot      // This is the current best for this learning session

		log.Printf("New global best score in learning: %d (S: %.1f,%.1f,%.1f L: %.1f,%.1f,%.1f)",
			globalBestScore,
			soundSource.Position.X, soundSource.Position.Y, soundSource.Position.Z,
			listener.Position.X, listener.Position.Y, listener.Position.Z)
		jsGlobal.Call("updateLearningProgress", currentLearningIteration, maxLearningIterations, globalBestScore)
		// No need to call updateRecordsDisplay here, AddRecord does it.
	}

	// Update JS display with current score and render the scene
	jsGlobal.Call("updateListenerRayCountJS", listenerRayScore)
	jsGlobal.Call("renderSceneJS", prepareSceneDataJS(), prepareRayDataJS())
}

// --- Data Preparation for JavaScript ---

func prepareSceneDataJS() js.Value {
	defer recoverFromPanic("prepareSceneDataJS")
	jsObjects := make([]interface{}, len(allSceneObjects))
	for i, obj := range allSceneObjects {
		jsObjects[i] = map[string]interface{}{
			"name": obj.Name, "type": obj.ShapeType,
			"position": map[string]interface{}{"x": obj.Position.X, "y": obj.Position.Y, "z": obj.Position.Z},
			"scale":    map[string]interface{}{"x": obj.Scale.X, "y": obj.Scale.Y, "z": obj.Scale.Z},
			"rotation": map[string]interface{}{"x": obj.Rotation.X, "y": obj.Rotation.Y, "z": obj.Rotation.Z}, // Degrees
			"color":    map[string]interface{}{"r": obj.Material.Color[0], "g": obj.Material.Color[1], "b": obj.Material.Color[2], "a": obj.Material.Color[3]},
		}
	}
	return js.ValueOf(jsObjects)
}

func prepareRayDataJS() js.Value {
	defer recoverFromPanic("prepareRayDataJS")
	jsRays := make([]interface{}, len(rayVisuals))
	for i, ray := range rayVisuals {
		jsRays[i] = map[string]interface{}{
			"start":   map[string]interface{}{"x": ray.Start.X, "y": ray.Start.Y, "z": ray.Start.Z},
			"end":     map[string]interface{}{"x": ray.End.X, "y": ray.End.Y, "z": ray.End.Z},
			"color":   float64(ray.Color), // Pass color as a number (hex)
			"opacity": ray.Opacity,
		}
	}
	return js.ValueOf(jsRays)
}

// --- Misc Utility Functions ---

// goToggleAutoOptimization is a placeholder if you add other optimization modes.
// For now, learning mode is the primary "auto optimization".
func goToggleAutoOptimization(this js.Value, args []js.Value) interface{} {
	defer recoverFromPanic("goToggleAutoOptimization")
	log.Println("General auto-optimization toggle called, but learning mode is primary focus. Use 'Start/Stop Learning' button.")
	// If you had a simpler, non-iterative auto-optimization, you might toggle it here.
	return nil
}

// IfThenElse is a generic helper, though not strictly necessary with Go's syntax.
func IfThenElse(condition bool, a interface{}, b interface{}) interface{} {
	if condition {
		return a
	}
	return b
}

func updateRayLegendJS() {
	defer recoverFromPanic("updateRayLegendJS")
	legendData := make([]map[string]interface{}, 0)

	// Add listener ray color first
	legendData = append(legendData, map[string]interface{}{
		"color": float64(listenerRayColor), // Ensure color is float64 for JS
		"label": "Reaches Listener",
	})

	// Determine how many bounce colors to show in legend
	displayBouncesInLegend := maxReflections
	if displayBouncesInLegend > len(bounceColors)-1 { // Don't try to show more colors than defined
		displayBouncesInLegend = len(bounceColors) - 1
	}
	if displayBouncesInLegend > 10 { // Cap legend items to a reasonable number
		displayBouncesInLegend = 10
	}

	for i := 0; i <= displayBouncesInLegend; i++ {
		colorIdx := i
		if colorIdx >= len(bounceColors) { // Should be prevented by above cap
			colorIdx = len(bounceColors) - 1
		}

		label := ""
		if i == 0 { // For the "direct path" or 0th bounce color if not hitting listener
			label = "Direct Path (Non-Listener)"
		} else {
			suffix := "th"
			switch i % 10 {
			case 1:
				if i%100 != 11 {
					suffix = "st"
				}
			case 2:
				if i%100 != 12 {
					suffix = "nd"
				}
			case 3:
				if i%100 != 13 {
					suffix = "rd"
				}
			}
			label = strconv.Itoa(i) + suffix + " Bounce"
		}
		// Avoid duplicating "Direct Path" if bounceColors[0] is used for non-listener direct hits
		if i == 0 && bounceColors[colorIdx] == listenerRayColor {
			continue
		}

		legendData = append(legendData, map[string]interface{}{
			"color": float64(bounceColors[colorIdx]),
			"label": label,
		})
	}

	// If there are more possible reflections than shown colors, add a generic "further bounces"
	if maxReflections > displayBouncesInLegend && displayBouncesInLegend < len(bounceColors)-1 {
		legendData = append(legendData, map[string]interface{}{
			"color": float64(bounceColors[len(bounceColors)-1]), // Use the last defined color
			"label": "Further Bounces",
		})
	}

	jsGlobal.Call("updateLegendOnPage", js.ValueOf(legendData))
}
