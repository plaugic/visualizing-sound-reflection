# 3D Sound Modeler & Acoustic Visualizer

[![3D Sound Modeler Demo Video](https://via.placeholder.com/800x450.png?text=Project+Demo+Video+Placeholder)](https://www.example.com/link_to_your_actual_video)
*(Replace the placeholder image and link above with your actual demo video/GIF and link)*

## 🌟 Overview

The 3D Sound Modeler is an interactive web-based application designed to simulate and visualize sound propagation within a defined 3D environment. Users can place a sound source and a listener, add various obstacles, and observe how sound rays travel, reflect, and interact with the scene. The application features a "learning mode" that attempts to optimize the positions of the source and listener to maximize a weighted score of sound energy reaching the listener.

This project leverages Go compiled to WebAssembly (WASM) for high-performance simulation logic, with Three.js for 3D rendering in the browser.

## ✨ Features

* **Interactive 3D Environment:**
    * Define room dimensions and add various static objects (furniture, pillars, etc.).
    * Real-time 3D rendering of the scene using Three.js.
* **Sound Propagation Simulation:**
    * Cast a configurable number of sound rays from a movable sound source.
    * Rays reflect off surfaces (objects, walls, ceiling, floor).
    * Configurable maximum number of reflections and ray attenuation per bounce.
* **Movable Sound Source & Listener:**
    * Control positions via UI sliders or by clicking and dragging in the 3D scene.
* **Scoring System:**
    * Calculates a weighted score based on rays reaching the listener.
    * Direct hits contribute a base score.
    * Indirect hits (bounced rays) contribute based on a Fibonacci sequence indexed by the number of bounces.
* **"Learning Mode" (Cooperative Maximization):**
    * Iteratively moves the sound source and listener to find positions that cooperatively maximize the listener score.
    * Configurable exploration factor to influence randomness and escape local optima.
    * Real-time feedback on learning progress and best score found.
* **Record Keeping:**
    * Automatically records settings (object positions, simulation parameters) that achieve new high scores.
    * UI to display top records and reapply any recorded setting configuration.
* **Customizable Visualization:**
    * Control initial ray opacity and wall/ceiling opacity.
    * Option to show all rays or only those reaching the listener (with a performance warning modal).
    * Dynamic ray legend indicating bounce order and listener hits.
* **User Interface:**
    * Comprehensive controls panel for adjusting simulation parameters, object positions, and learning settings.
    * Debounced UI updates for smooth interaction.

## 🛠️ Technology Stack

* **Core Logic:** [Go (Golang)](https://golang.org/) compiled to [WebAssembly (WASM)](https://webassembly.org/)
* **3D Rendering:** [Three.js](https://threejs.org/) (JavaScript 3D library using WebGL)
* **Frontend:** HTML5, CSS3, [Tailwind CSS](https://tailwindcss.com/) (for UI styling)
* **Glue/Interop:** JavaScript (managing WASM, Three.js, and DOM interactions)
* **Development Server:** A simple Go HTTP server (`server.go`)

## 🚀 Live Demo

*(Placeholder for a link to a live deployed version, if you have one)*
`[Link to Live Demo (if available)](YOUR_LIVE_DEMO_LINK_HERE)`

## 🏁 Getting Started / Local Development

Follow these instructions to get a copy of the project up and running on your local machine for development and testing purposes.

### Prerequisites

* [Go](https://golang.org/dl/) (version 1.18 or higher recommended for WASM support)
* A modern web browser that supports WebAssembly and WebGL.

### Installation & Setup

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/visualizing-sound-reflection.git
    cd visualizing-sound-reflection
    ```

2.  **Copy `wasm_exec.js`:**\
    A. This file is required to run Go WASM modules. If you are have a Go version 1.23 or higher, copy it from your Go installation's `misc/wasm` directory into the project root.
    ```bash
    cp $(go env GOROOT)/misc/wasm/wasm_exec.js .
    ```
    B. If you are have a Go version 1.23 or higher, the wasm_exec.js file that must be copied into your project will be found elsewhere. Copy it from your Go installation's `lib/wasm` directory into the project root.
    ```bash
    cp $(go env GOROOT)/lib/wasm/wasm_exec.js .
    ```

3.  **Build the WebAssembly Module:**
    Compile the Go source files into `main.wasm`. Ensure all `.go` files intended for the WASM build (e.g., `main.go`, `vecmath.go`, `scene.go`, `raycaster.go`, `optimization.go`, `records.go`) are present in the directory.
    ```bash
    GOOS=js GOARCH=wasm go build -o main.wasm .
    ```

4.  **Run the Local Development Server:**
    The provided `server.go` can be used to serve the project files.
    ```bash
    go run ./cmd/server/main.go
    ```

5.  **Open in Browser:**
    Navigate to `http://localhost:8080` in your web browser.

## 🎮 How to Use

* **Controls Panel:** Use the sliders and toggles on the right-hand side to adjust parameters for the sound source, listener, rays, environment, and learning mode.
* **3D Scene Interaction:**
    * **Orbit:** Click and drag the left mouse button on the canvas to orbit the camera around the center of the scene.
    * **Zoom:** Use the mouse wheel to zoom in and out.
    * **Move Source/Listener:** Click and drag the red (Sound Source) or blue (Listener) spheres to reposition them in the scene. Their positions will update in the controls panel.
* **Visualize Sound:** Click the "Visualize Sound (Manual)" button to perform a raycasting simulation with the current settings.
* **Learning Mode:**
    * Click "Start Learning" to begin the automated optimization process. The application will attempt to find optimal positions for the source and listener.
    * The UI will update with the current iteration and the best score found.
    * Click "Stop Learning" to halt the process. The best found settings will be applied.
* **Records:** The "Best Score Records" list shows high-scoring configurations. Click "Apply" next to a record to restore those settings.

## 📁 Project Structure (Go Code)

The Go codebase (`package main`) is organized into several files for better maintainability:

* `main.go`: Main application entry point, global state, JS interop functions, and high-level simulation orchestration.
* `vecmath.go`: `Vector3` struct and associated mathematical utility functions.
* `scene.go`: Structs for scene objects (`SceneObject`, `MaterialProperties`) and functions for creating scene elements.
* `raycaster.go`: Low-level geometric intersection tests (`performRaycast`) and ray evaluation logic.
* `optimization.go`: Logic for the learning mode (`findAndApplyBestMoveForLearning`, `runLearningCycle`).
* `records.go`: Management of best score records (`RecordManager`, `BestScoreSettings`).
* `server.go`: A simple Go HTTP server for local development (run separately).

## 💡 Key Concepts

* **Raycasting:** The process of tracing the path of rays from a source to see what they intersect. Used here to simulate sound paths.
* **Fibonacci Sphere:** An algorithm used to distribute points (and thus initial ray directions) fairly evenly on the surface of a sphere.
* **AABB (Axis-Aligned Bounding Box):** A simplified rectangular collision shape used for box objects.
* **Debouncing:** A technique to limit the rate at which a function is called, used for UI slider updates to prevent excessive recalculations.
* **Cooperative Maximization:** The learning mode where both the sound source and listener positions are adjusted to jointly achieve the highest possible score.

## 🚧 Future Development & TODOs

This project has many avenues for future development and enhancement. Key areas include:

* More advanced sound physics (e.g., diffraction, material-specific absorption).
* Sophisticated Machine Learning / Optimization strategies.
* Enhanced collision detection (e.g., OBBs, imported meshes).
* Dynamic scene editing capabilities (add/remove/modify objects via UI).
* Performance optimizations for both WASM and rendering.
* Comprehensive testing and documentation.

For a detailed list of ongoing tasks and planned features, please see the `TODO.md` file in this repository.

## 🤝 Contributing

*(Placeholder - feel free to expand if you accept contributions)*
Contributions, issues, and feature requests are welcome! Please feel free to check the `TODO.md` or open an issue to discuss what you would like to change.

## 📜 License

*(Placeholder - choose a license, e.g., MIT)*
This project is licensed under the MIT License - see the `LICENSE.md` file for details (if you add one).