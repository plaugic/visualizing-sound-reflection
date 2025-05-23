package main

import (
	"fmt"
	"math/rand"
)

// --- Scene & Object Representation ---
type MaterialProperties struct {
	Color         [4]float32 // R, G, B, A (0.0 to 1.0)
	IsTransparent bool
}

type SceneObject struct {
	Name            string
	ID              string
	Position        Vector3
	Rotation        Vector3 // Euler angles in degrees
	Scale           Vector3
	Visible         bool
	IsStatic        bool // True if the object cannot be moved by optimization/learning
	Material        MaterialProperties
	isWallOrCeiling bool
	ShapeType       string // "box", "sphere"
}

// Snapshot of an object's state for recording
type SceneObjectSnapshot struct {
	Name      string
	Position  Vector3
	Rotation  Vector3
	Scale     Vector3
	ShapeType string
}

func NewSceneObject(name, shapeType string) *SceneObject {
	return &SceneObject{
		Name:      name,
		ID:        fmt.Sprintf("%s-%d", name, rand.Intn(1000000)),
		Position:  Vector3{0, 0, 0},
		Rotation:  Vector3{0, 0, 0},
		Scale:     Vector3{1, 1, 1},
		Visible:   true,
		IsStatic:  true, // Default to static
		ShapeType: shapeType,
		Material: MaterialProperties{
			Color: [4]float32{0.7, 0.7, 0.7, 1.0},
		},
	}
}

type Point3D struct{ X, Y, Z float64 }
type RayLine struct {
	Start, End Point3D
	Color      uint32
	Opacity    float64
}

func createSceneContent() {
	allSceneObjects = make([]*SceneObject, 0)
	staticSceneObjects = make([]*SceneObject, 0)
	wallCeilingMeshes = make([]*SceneObject, 0)
	createEnvironment()
	createFurniture()
	createSoundSourceAndListener()
}

func createObject(name, shapeType string, pos, rotDegrees, scale Vector3, matProps MaterialProperties, isWall, isStatic bool) *SceneObject {
	obj := NewSceneObject(name, shapeType)
	obj.Position = pos
	obj.Rotation = rotDegrees
	obj.Scale = scale
	obj.Material = matProps
	obj.isWallOrCeiling = isWall
	obj.IsStatic = isStatic
	allSceneObjects = append(allSceneObjects, obj)
	if isWall {
		wallCeilingMeshes = append(wallCeilingMeshes, obj)
	}
	if isStatic && name != "SoundSource" && name != "Listener" {
		staticSceneObjects = append(staticSceneObjects, obj)
	}
	return obj
}

func createEnvironment() {
	groundMat := MaterialProperties{Color: [4]float32{0.6, 0.6, 0.6, 1.0}}
	createObject("Ground", "box", Vector3{0, 0, 0}, Vector3{}, Vector3{roomWidth, wallThickness, roomDepth}, groundMat, false, true)
	wallMat := MaterialProperties{Color: [4]float32{0.8, 0.8, 0.8, float32(currentWallOpacity)}, IsTransparent: currentWallOpacity < 1.0}
	createObject("BackWall", "box", Vector3{0, roomHeight / 2, -roomDepth / 2}, Vector3{}, Vector3{roomWidth, roomHeight, wallThickness}, wallMat, true, true)
	createObject("FrontWall", "box", Vector3{0, roomHeight / 2, roomDepth / 2}, Vector3{}, Vector3{roomWidth, roomHeight, wallThickness}, wallMat, true, true)
	createObject("LeftWall", "box", Vector3{-roomWidth / 2, roomHeight / 2, 0}, Vector3{}, Vector3{wallThickness, roomHeight, roomDepth}, wallMat, true, true)
	createObject("RightWall", "box", Vector3{roomWidth / 2, roomHeight / 2, 0}, Vector3{}, Vector3{wallThickness, roomHeight, roomDepth}, wallMat, true, true)
	createObject("Ceiling", "box", Vector3{0, roomHeight + wallThickness/2, 0}, Vector3{}, Vector3{roomWidth, wallThickness, roomDepth}, wallMat, true, true)
}

func createFurniture() {
	bookshelfMat := MaterialProperties{Color: [4]float32{0.54, 0.27, 0.07, 1.0}}
	tableMat := MaterialProperties{Color: [4]float32{0.63, 0.32, 0.18, 1.0}}
	pillarMat := MaterialProperties{Color: [4]float32{0.5, 0.5, 0.5, 1.0}}
	plantPotMat := MaterialProperties{Color: [4]float32{0.4, 0.2, 0.1, 1.0}}
	plantLeavesMat := MaterialProperties{Color: [4]float32{0.1, 0.5, 0.1, 1.0}}
	couchMat := MaterialProperties{Color: [4]float32{0.3, 0.3, 0.4, 1.0}}
	lampMat := MaterialProperties{Color: [4]float32{0.9, 0.9, 0.7, 1.0}}

	createObject("Bookshelf-Main-Left", "box", Vector3{-roomWidth/2 + 5, 1.5, 0}, Vector3{}, Vector3{2, 3, 6}, bookshelfMat, false, true)
	createObject("Bookshelf-Main-Right", "box", Vector3{roomWidth/2 - 5, 1.5, 0}, Vector3{}, Vector3{2, 3, 6}, bookshelfMat, false, true)
	createObject("Bookshelf-Back", "box", Vector3{0, 1.5, -roomDepth/2 + 3}, Vector3{0, 90, 0}, Vector3{6, 3, 1.5}, bookshelfMat, false, true)

	createObject("Table-Center-Large", "box", Vector3{0, 0.75, 0}, Vector3{}, Vector3{5, 0.2, 2.5}, tableMat, false, true)
	createObject("Table-Side-Left", "box", Vector3{-roomWidth / 4, 0.70, roomDepth / 3}, Vector3{}, Vector3{2, 0.2, 1.2}, tableMat, false, true)
	createObject("Table-Side-Right", "box", Vector3{roomWidth / 4, 0.70, -roomDepth / 3}, Vector3{0, 30, 0}, Vector3{2.5, 0.2, 1.5}, tableMat, false, true)

	createObject("Bookshelf-Corner-BL", "box", Vector3{-roomWidth/2 + 3, 2.0, -roomDepth/2 + 3}, Vector3{0, 45, 0}, Vector3{1.5, 4, 1.5}, bookshelfMat, false, true)
	createObject("Bookshelf-Corner-FR", "box", Vector3{roomWidth/2 - 4, 1.0, roomDepth/2 - 4}, Vector3{0, -30, 0}, Vector3{1, 2, 3}, bookshelfMat, false, true)

	pillarHeight := roomHeight - 0.1
	createObject("Pillar-FrontLeft", "box", Vector3{-roomWidth / 3, pillarHeight / 2, roomDepth / 3}, Vector3{}, Vector3{0.8, pillarHeight, 0.8}, pillarMat, false, true)
	createObject("Pillar-FrontRight", "box", Vector3{roomWidth / 3, pillarHeight / 2, roomDepth / 3}, Vector3{}, Vector3{0.8, pillarHeight, 0.8}, pillarMat, false, true)
	createObject("Pillar-BackLeft", "box", Vector3{-roomWidth / 3, pillarHeight / 2, -roomDepth / 3}, Vector3{}, Vector3{0.6, pillarHeight, 0.6}, pillarMat, false, true)
	createObject("Pillar-BackRight", "box", Vector3{roomWidth / 3, pillarHeight / 2, -roomDepth / 3}, Vector3{}, Vector3{0.6, pillarHeight, 0.6}, pillarMat, false, true)

	createObject("Couch-Left", "box", Vector3{-roomWidth/2 + 4, 0.5, roomDepth / 3}, Vector3{0, 90, 0}, Vector3{3, 1, 1.5}, couchMat, false, true)
	createObject("Couch-Right", "box", Vector3{roomWidth/2 - 4, 0.5, -roomDepth / 3}, Vector3{0, -90, 0}, Vector3{3, 1, 1.5}, couchMat, false, true)
	createObject("Armchair-Center", "box", Vector3{0, 0.4, -roomDepth / 4}, Vector3{0, 180, 0}, Vector3{1.2, 0.8, 1.2}, couchMat, false, true)

	createObject("PlantPot1", "box", Vector3{-roomWidth/2 + 1.5, 0.25, roomDepth/2 - 1.5}, Vector3{}, Vector3{0.5, 0.5, 0.5}, plantPotMat, false, true)
	createObject("PlantLeaves1", "sphere", Vector3{-roomWidth/2 + 1.5, 1.0, roomDepth/2 - 1.5}, Vector3{}, Vector3{0.7, 1.0, 0.7}, plantLeavesMat, false, true)
	createObject("PlantPot2", "box", Vector3{roomWidth/2 - 1.5, 0.3, -roomDepth/2 + 1.5}, Vector3{}, Vector3{0.6, 0.6, 0.6}, plantPotMat, false, true)
	createObject("PlantLeaves2", "sphere", Vector3{roomWidth/2 - 1.5, 1.2, -roomDepth/2 + 1.5}, Vector3{}, Vector3{0.8, 1.2, 0.8}, plantLeavesMat, false, true)

	createObject("LampBase1", "box", Vector3{roomWidth / 3, 0.75, 0}, Vector3{}, Vector3{0.3, 1.5, 0.3}, pillarMat, false, true)
	createObject("LampShade1", "sphere", Vector3{roomWidth / 3, 1.5 + 0.3, 0}, Vector3{}, Vector3{0.6, 0.6, 0.6}, lampMat, false, true)

	createObject("MiscBox1", "box", Vector3{5, 0.1, -10}, Vector3{0, 15, 0}, Vector3{1, 0.2, 0.5}, tableMat, false, true)
	createObject("MiscSphere1", "sphere", Vector3{-8, 0.2, 8}, Vector3{}, Vector3{0.4, 0.4, 0.4}, pillarMat, false, true)
}

func createSoundSourceAndListener() {
	sourceMat := MaterialProperties{Color: [4]float32{1, 0, 0, 1.0}}
	soundSource = createObject("SoundSource", "sphere", Vector3{0, 1.5, 5}, Vector3{}, Vector3{0.3, 0.3, 0.3}, sourceMat, false, false)
	listenerMat := MaterialProperties{Color: [4]float32{0, 0, 1, 1.0}}
	listener = createObject("Listener", "sphere", Vector3{0, 1.5, -5}, Vector3{}, Vector3{0.25, 0.25, 0.25}, listenerMat, false, false)
}
