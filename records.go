package main

import (
	"log"
	"sort"
	"syscall/js"
)

// Struct to hold all settings for the best score
type BestScoreSettings struct {
	Score                   int
	Iteration               int
	NumRays                 int
	InitialRayOpacity       float64
	MaxReflections          int
	VolumeAttenuationFactor float64
	ExplorationFactor       float64
	SoundSourcePos          Vector3
	ListenerPos             Vector3
	ShowOnlyListenerRays    bool
	AllObjectSnapshots      []SceneObjectSnapshot // Optional: for restoring entire scene states
}

// RecordManager handles storing and retrieving best scores
type RecordManager struct {
	BestRecords []BestScoreSettings
	MaxRecords  int
}

func NewRecordManager(maxRecords int) *RecordManager {
	return &RecordManager{
		BestRecords: make([]BestScoreSettings, 0, maxRecords),
		MaxRecords:  maxRecords,
	}
}

func (rm *RecordManager) AddRecord(settings BestScoreSettings) {
	log.Printf("New record candidate: Score %d at iter %d", settings.Score, settings.Iteration)

	// Add the new record
	rm.BestRecords = append(rm.BestRecords, settings)

	// Sort records by score in descending order
	sort.Slice(rm.BestRecords, func(i, j int) bool {
		return rm.BestRecords[i].Score > rm.BestRecords[j].Score
	})

	// If the number of records exceeds MaxRecords, truncate the list
	if len(rm.BestRecords) > rm.MaxRecords {
		rm.BestRecords = rm.BestRecords[:rm.MaxRecords]
	}

	log.Printf("RecordManager updated. Current top %d scores: ", len(rm.BestRecords))
	for i, rec := range rm.BestRecords {
		log.Printf("  %d. Score: %d, Iter: %d", i+1, rec.Score, rec.Iteration)
	}

	// Notify JavaScript to update the records display
	jsGlobal.Call("updateRecordsDisplay", rm.prepareRecordsForJS())
}

func (rm *RecordManager) prepareRecordsForJS() js.Value {
	jsRecords := make([]interface{}, len(rm.BestRecords))
	for i, rec := range rm.BestRecords {
		jsRecords[i] = map[string]interface{}{
			"score":     rec.Score,
			"iteration": rec.Iteration,
			"numRays":   rec.NumRays, // Example of including more data
			// Add other relevant fields if you want them in the JS display object
		}
	}
	return js.ValueOf(jsRecords)
}

func goApplyRecordedSettingsByIndex(this js.Value, args []js.Value) interface{} {
	defer recoverFromPanic("goApplyRecordedSettingsByIndex")
	if len(args) != 1 {
		log.Println("Error: goApplyRecordedSettingsByIndex expects 1 argument (index)")
		return nil
	}
	index := args[0].Int()

	if index < 0 || index >= len(recordsManager.BestRecords) {
		log.Printf("Error: Invalid record index %d. Max index %d", index, len(recordsManager.BestRecords)-1)
		return nil
	}

	settings := recordsManager.BestRecords[index]
	log.Printf("Applying recorded settings from record %d (Score: %d)", index, settings.Score)

	// Apply settings
	numRays = settings.NumRays
	initialRayOpacity = settings.InitialRayOpacity
	maxReflections = settings.MaxReflections
	volumeAttenuationFactor = settings.VolumeAttenuationFactor
	explorationFactor = settings.ExplorationFactor // Apply exploration factor as well
	showOnlyListenerRays = settings.ShowOnlyListenerRays

	if soundSource != nil {
		soundSource.Position = settings.SoundSourcePos
	}
	if listener != nil {
		listener.Position = settings.ListenerPos
	}

	// TODO: If AllObjectSnapshots were populated and you want to restore them, do it here.
	// This would involve iterating settings.AllObjectSnapshots and updating allSceneObjects.
	// Be careful with this, as it could be complex if objects can be added/removed.
	// For now, we only restore sound source and listener positions.

	// Update UI sliders to reflect the applied settings
	jsGlobal.Call("updateAllUISliders",
		numRays, initialRayOpacity, maxReflections, volumeAttenuationFactor, explorationFactor,
		soundSource.Position.X, soundSource.Position.Y, soundSource.Position.Z,
		listener.Position.X, listener.Position.Y, listener.Position.Z,
		showOnlyListenerRays,
	)

	visualizeSoundPropagation() // Re-visualize with the new settings
	updateRayLegendJS()         // Update legend if maxReflections changed
	return nil
}
