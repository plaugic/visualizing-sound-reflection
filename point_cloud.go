package main

import (
	"log"
	"math"
	"syscall/js"
)

// PointState defines the occupancy state of a cell in the point cloud.
type PointState uint8 // Using uint8 for memory efficiency

const (
	StateEmpty          PointState = 0 // Cell is traversable and not occupied by a persistent element
	StateStaticObstacle PointState = 1 // Cell is occupied by a static, immovable obstacle
	StateSoundSource    PointState = 2 // Cell is currently occupied by the sound source
	StateListener       PointState = 3 // Cell is currently occupied by the listener
	StateOutOfBounds    PointState = 4 // Query was for a point outside the defined cloud boundaries
	// Future states could include: StateExploredLowPotential, StateExploredHighPotential, etc.
)

// OccupancyCloud represents the discretized 3D space.
// For simplicity, this initial version uses a 3D grid.
// An Octree could be a future optimization for sparse environments.
type OccupancyCloud struct {
	Grid         [][][]PointState // The 3D grid storing the state of each cell
	RoomMin      Vector3          // Min corner of the room in world coordinates (e.g., floor, back-left)
	RoomMax      Vector3          // Max corner of the room in world coordinates (e.g., ceiling, front-right)
	CellSize     Vector3          // Size of each cell in world units (x, y, z)
	CellsX       int              // Number of cells along X-axis
	CellsY       int              // Number of cells along Y-axis
	CellsZ       int              // Number of cells along Z-axis
	DebugLogging bool
}

// NewOccupancyCloud creates and initializes a new occupancy cloud.
// roomMin: The minimum corner of the bounding box for the cloud (e.g., {-roomWidth/2, 0, -roomDepth/2}).
// roomMax: The maximum corner of the bounding box for the cloud (e.g., {roomWidth/2, roomHeight, roomDepth/2}).
// cellSize: The desired size for each cell. Smaller cells = higher resolution but more memory/computation.
func NewOccupancyCloud(roomMin, roomMax Vector3, cellSize Vector3, debugLogging bool) *OccupancyCloud {
	if cellSize.X <= 0 || cellSize.Y <= 0 || cellSize.Z <= 0 {
		log.Fatalf("OccupancyCloud cell dimensions must be positive. Got: %.2f, %.2f, %.2f", cellSize.X, cellSize.Y, cellSize.Z)
	}

	cellsX := int(math.Ceil((roomMax.X - roomMin.X) / cellSize.X))
	cellsY := int(math.Ceil((roomMax.Y - roomMin.Y) / cellSize.Y))
	cellsZ := int(math.Ceil((roomMax.Z - roomMin.Z) / cellSize.Z))

	if cellsX == 0 {
		cellsX = 1
	}
	if cellsY == 0 {
		cellsY = 1
	}
	if cellsZ == 0 {
		cellsZ = 1
	}

	grid := make([][][]PointState, cellsX)
	for i := range grid {
		grid[i] = make([][]PointState, cellsY)
		for j := range grid[i] {
			grid[i][j] = make([]PointState, cellsZ)
			// All cells initially empty
			for k := range grid[i][j] {
				grid[i][j][k] = StateEmpty
			}
		}
	}
	if debugLogging {
		log.Printf("OccupancyCloud initialized: Dimensions [%.1f, %.1f, %.1f] to [%.1f, %.1f, %.1f]", roomMin.X, roomMin.Y, roomMin.Z, roomMax.X, roomMax.Y, roomMax.Z)
		log.Printf("OccupancyCloud initialized: Cells %d x %d x %d, CellSize: %.2f x %.2f x %.2f", cellsX, cellsY, cellsZ, cellSize.X, cellSize.Y, cellSize.Z)
	}

	return &OccupancyCloud{
		Grid:         grid,
		RoomMin:      roomMin,
		RoomMax:      roomMax, // Store actual max based on cells and cellsize for precision later
		CellSize:     cellSize,
		CellsX:       cellsX,
		CellsY:       cellsY,
		CellsZ:       cellsZ,
		DebugLogging: debugLogging,
	}
}

// worldToGridCoords converts world coordinates to grid cell indices.
// Returns indices and a bool indicating if the coordinates are within bounds.
func (oc *OccupancyCloud) worldToGridCoords(worldPos Vector3) (ix, iy, iz int, inBounds bool) {
	if worldPos.X < oc.RoomMin.X || worldPos.X >= oc.RoomMin.X+float64(oc.CellsX)*oc.CellSize.X ||
		worldPos.Y < oc.RoomMin.Y || worldPos.Y >= oc.RoomMin.Y+float64(oc.CellsY)*oc.CellSize.Y ||
		worldPos.Z < oc.RoomMin.Z || worldPos.Z >= oc.RoomMin.Z+float64(oc.CellsZ)*oc.CellSize.Z {
		return -1, -1, -1, false // Out of defined cloud bounds
	}

	ix = int(math.Floor((worldPos.X - oc.RoomMin.X) / oc.CellSize.X))
	iy = int(math.Floor((worldPos.Y - oc.RoomMin.Y) / oc.CellSize.Y))
	iz = int(math.Floor((worldPos.Z - oc.RoomMin.Z) / oc.CellSize.Z))

	// Clamp to valid grid indices just in case of floating point issues at the boundary
	ix = clampInt(ix, 0, oc.CellsX-1)
	iy = clampInt(iy, 0, oc.CellsY-1)
	iz = clampInt(iz, 0, oc.CellsZ-1)

	return ix, iy, iz, true
}

func clampInt(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// getCellState retrieves the state of a cell by its grid indices.
func (oc *OccupancyCloud) getCellState(ix, iy, iz int) PointState {
	if ix < 0 || ix >= oc.CellsX || iy < 0 || iy >= oc.CellsY || iz < 0 || iz >= oc.CellsZ {
		return StateOutOfBounds
	}
	return oc.Grid[ix][iy][iz]
}

// setCellState sets the state of a cell by its grid indices.
func (oc *OccupancyCloud) setCellState(ix, iy, iz int, state PointState) {
	if ix >= 0 && ix < oc.CellsX && iy >= 0 && iy < oc.CellsY && iz >= 0 && iz < oc.CellsZ {
		oc.Grid[ix][iy][iz] = state
	} else {
		if oc.DebugLogging {
			log.Printf("Attempted to set state for out-of-bounds cell: (%d, %d, %d)", ix, iy, iz)
		}
	}
}

// MarkStaticObstacles populates the cloud with static obstacles from the scene.
// This should be called once after scene creation.
func (oc *OccupancyCloud) MarkStaticObstacles(staticObjects []*SceneObject) {
	if oc.DebugLogging {
		log.Printf("Marking %d static obstacles in occupancy cloud...", len(staticObjects))
	}
	for _, obj := range staticObjects {
		if !obj.IsStatic { // Should only be static objects
			continue
		}
		// For each object, determine the AABB of cells it occupies.
		// This is a simplification; more accurate rasterization might be needed for non-box shapes or rotated boxes.
		objMin := obj.Position.Sub(obj.Scale.Scale(0.5)) // Assumes scale is full dimensions
		objMax := obj.Position.Add(obj.Scale.Scale(0.5))

		minIX, minIY, minIZ, inBoundsMin := oc.worldToGridCoords(objMin)
		maxIX, maxIY, maxIZ, inBoundsMax := oc.worldToGridCoords(objMax)

		if !(inBoundsMin && inBoundsMax) {
			// If even parts of the object are out of bounds, it might be an issue with room/cloud setup.
			// For now, we'll only mark the parts that are in bounds.
			if oc.DebugLogging {
				log.Printf("Static object %s partially or fully out of cloud bounds during marking.", obj.Name)
			}
		}

		for ix := minIX; ix <= maxIX; ix++ {
			for iy := minIY; iy <= maxIY; iy++ {
				for iz := minIZ; iz <= maxIZ; iz++ {
					// Further check if cell center is within object for non-box shapes (approx)
					// For boxes aligned with grid, this AABB approach is okay.
					// For spheres, one would check if cell_center to obj_center distance < radius
					// This basic version marks the AABB of the object's AABB in the grid.
					oc.setCellState(ix, iy, iz, StateStaticObstacle)
				}
			}
		}
	}
	if oc.DebugLogging {
		log.Println("Static obstacles marked.")
	}
}

// updateObjectInCloud updates the cloud for a movable object (Source or Listener).
// It clears its old position and marks its new position.
// oldPosition must be the object's center *before* the move.
// newPosition is the object's center *after* the move.
func (oc *OccupancyCloud) UpdateObjectInCloud(objName string, oldPosition, newPosition Vector3, objScale Vector3, newState PointState) {
	// 1. Clear the old position
	// Iterate over cells occupied by the object at its oldPosition
	// This needs to know the object's extent (e.g., radius for sphere, AABB for box)
	// For simplicity, assume spherical objects for dynamic ones initially.
	// Effective radius for cell marking (can be larger than actual radius to be conservative)
	markRadius := math.Max(objScale.X, math.Max(objScale.Y, objScale.Z))/2.0 + oc.CellSize.X // Add cellsize for safety margin

	// Clear old cells
	// Iterate over a bounding box of cells around the old position
	oldMin := oldPosition.Sub(Vector3{markRadius, markRadius, markRadius})
	oldMax := oldPosition.Add(Vector3{markRadius, markRadius, markRadius})
	oldMinIX, oldMinIY, oldMinIZ, _ := oc.worldToGridCoords(oldMin)
	oldMaxIX, oldMaxIY, oldMaxIZ, _ := oc.worldToGridCoords(oldMax)

	for ix := oldMinIX; ix <= oldMaxIX; ix++ {
		for iy := oldMinIY; iy <= oldMaxIY; iy++ {
			for iz := oldMinIZ; iz <= oldMaxIZ; iz++ {
				if oc.getCellState(ix, iy, iz) == newState { // Only clear if it was marked by this object type
					oc.setCellState(ix, iy, iz, StateEmpty)
				}
			}
		}
	}

	// 2. Mark the new position
	newMin := newPosition.Sub(Vector3{markRadius, markRadius, markRadius})
	newMax := newPosition.Add(Vector3{markRadius, markRadius, markRadius})
	newMinIX, newMinIY, newMinIZ, _ := oc.worldToGridCoords(newMin)
	newMaxIX, newMaxIY, newMaxIZ, _ := oc.worldToGridCoords(newMax)

	for ix := newMinIX; ix <= newMaxIX; ix++ {
		for iy := newMinIY; iy <= newMaxIY; iy++ {
			for iz := newMinIZ; iz <= newMaxIZ; iz++ {
				// Check if cell is within actual object sphere at new position
				cellCenterX := oc.RoomMin.X + (float64(ix)+0.5)*oc.CellSize.X
				cellCenterY := oc.RoomMin.Y + (float64(iy)+0.5)*oc.CellSize.Y
				cellCenterZ := oc.RoomMin.Z + (float64(iz)+0.5)*oc.CellSize.Z
				cellCenter := Vector3{cellCenterX, cellCenterY, cellCenterZ}

				if cellCenter.Sub(newPosition).Length() < markRadius { // Using markRadius, effectively rasterizing a sphere
					currentState := oc.getCellState(ix, iy, iz)
					if currentState == StateEmpty { // Only mark if empty, don't overwrite obstacles
						oc.setCellState(ix, iy, iz, newState)
					} else if currentState != StateStaticObstacle && oc.DebugLogging {
						// log.Printf("Cloud conflict: %s wants to occupy cell (%d,%d,%d) with state %d, but it's %d", objName, ix,iy,iz, newState, currentState)
					}
				}
			}
		}
	}
	if oc.DebugLogging {
		// log.Printf("Object %s updated in cloud. Old: %.1f,%.1f,%.1f New: %.1f,%.1f,%.1f", objName, oldPosition.X, oldPosition.Y, oldPosition.Z, newPosition.X, newPosition.Y, newPosition.Z)
	}
}

// IsPositionAttemptValid checks if a proposed position for a dynamic object is valid according to the cloud.
// It checks against static obstacles and the *other* dynamic object.
// movingObjType should be StateSoundSource or StateListener.
// otherObjCurrentPos is the current center of the *other* dynamic object.
// otherObjScale is the scale of the *other* dynamic object.
func (oc *OccupancyCloud) IsPositionAttemptValid(proposedPos Vector3, movingObjScale Vector3, movingObjType PointState, otherObjCurrentPos Vector3, otherObjScale Vector3) bool {
	// Determine cells the moving object would occupy at proposedPos
	objRadius := math.Max(movingObjScale.X, math.Max(movingObjScale.Y, movingObjScale.Z)) / 2.0

	// Iterate over a bounding box of cells the object might touch
	objMin := proposedPos.Sub(Vector3{objRadius, objRadius, objRadius})
	objMax := proposedPos.Add(Vector3{objRadius, objRadius, objRadius})
	minIX, minIY, minIZ, _ := oc.worldToGridCoords(objMin)
	maxIX, maxIY, maxIZ, _ := oc.worldToGridCoords(objMax)

	for ix := minIX; ix <= maxIX; ix++ {
		for iy := minIY; iy <= maxIY; iy++ {
			for iz := minIZ; iz <= maxIZ; iz++ {
				cellCenterX := oc.RoomMin.X + (float64(ix)+0.5)*oc.CellSize.X
				cellCenterY := oc.RoomMin.Y + (float64(iy)+0.5)*oc.CellSize.Y
				cellCenterZ := oc.RoomMin.Z + (float64(iz)+0.5)*oc.CellSize.Z
				cellCenter := Vector3{cellCenterX, cellCenterY, cellCenterZ}

				// Check if this cell center is actually within the sphere of the moving object
				if cellCenter.Sub(proposedPos).Length() < objRadius {
					cellState := oc.getCellState(ix, iy, iz)

					if cellState == StateOutOfBounds {
						return false
					} // Trying to move out of defined cloud
					if cellState == StateStaticObstacle {
						return false
					} // Collision with static obstacle

					// Check collision with the *other* dynamic object directly (more accurate than relying on its cloud state for this check)
					// This avoids issues if the other object's cloud state hasn't updated yet or for precision.
					if spheresIntersect(proposedPos, objRadius, otherObjCurrentPos, math.Max(otherObjScale.X, otherObjScale.Z)/2.0) {
						return false
					}
				}
			}
		}
	}
	return true // Position is valid according to the cloud and direct other-object check
}

// PrepareCloudForJS converts the occupancy cloud data into a format suitable for JavaScript/Three.js visualization.
// This could be a list of occupied cells with their states and positions.
// For a "gold standard" this might involve sending only changes or a compressed format.
// Initial version: send all non-empty cells.
func (oc *OccupancyCloud) PrepareCloudForJS() js.Value {
	defer recoverFromPanic("PrepareCloudForJS_OccupancyCloud")

	var occupiedCells []interface{}
	for ix := 0; ix < oc.CellsX; ix++ {
		for iy := 0; iy < oc.CellsY; iy++ {
			for iz := 0; iz < oc.CellsZ; iz++ {
				state := oc.Grid[ix][iy][iz]
				if state != StateEmpty { // Only send non-empty cells
					// Calculate world position of the cell's center
					worldX := oc.RoomMin.X + (float64(ix)+0.5)*oc.CellSize.X
					worldY := oc.RoomMin.Y + (float64(iy)+0.5)*oc.CellSize.Y
					worldZ := oc.RoomMin.Z + (float64(iz)+0.5)*oc.CellSize.Z
					occupiedCells = append(occupiedCells, map[string]interface{}{
						"x":     worldX,
						"y":     worldY,
						"z":     worldZ,
						"state": uint8(state), // Send state as a number
						"sizeX": oc.CellSize.X,
						"sizeY": oc.CellSize.Y,
						"sizeZ": oc.CellSize.Z,
					})
				}
			}
		}
	}
	if oc.DebugLogging && len(occupiedCells) > 0 {
		log.Printf("Preparing %d occupied cloud cells for JS.", len(occupiedCells))
	}
	return js.ValueOf(occupiedCells)
}
