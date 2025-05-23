# Project TODO & Improvement Roadmap

## I. Core Simulation Engine
### A. Scoring System ⚙️
  - [x] Direct ray hit: score by base amount
  - [x] Indirect ray hit: score by Fibonacci sequence index (bounce-based)
  - [ ] **UI Configuration:**
    - [ ] Allow `BASE_DIRECT_HIT_SCORE` to be set via UI slider
    - [ ] Allow `FIBONACCI_SCORE_CAP_INDEX` to be set via UI slider
  - [ ] **Advanced Scoring Models:**
    - [ ] Introduce scoring penalties (e.g., for very long ray paths, excessive bounces beyond a soft cap)
    - [ ] Experiment with alternative scoring functions (e.g., exponential decay, custom curves for bounces)
    - [ ] Weight scores by angle of incidence on listener (optional)

### B. Sound Propagation Physics 🔊
  - [ ] **Material Interactions:**
    - [ ] Define acoustic properties for materials (absorption, reflection coefficients).
    - [ ] Model frequency-dependent absorption/reflection (simple bands: low, mid, high).
    - [ ] Ray energy attenuation based on material properties and distance.
  - [ ] **Wave Phenomena (Basic):**
    - [ ] Implement basic diffraction around sharp edges of static objects.
    - [ ] Simulate sound transmission through designated "thin" objects with significant attenuation.
  - [ ] **Advanced Ray Properties:**
    - [ ] Track phase information for rays (for interference - highly advanced).

### C. Scene & Object Representation 🧊
  - [ ] **Object Primitives:**
    - [ ] Add support for more ray-intersectable primitives (e.g., cylinders, capsules).
  - [ ] **Imported Geometry (Major Task):**
    - [ ] Allow importing simple mesh files (e.g., .obj, .stl) for static obstacles.
    - [ ] Implement efficient ray-mesh intersection for imported models (BVH or similar).

## II. Object Interactions & Collision Detection
### A. Static & Main Object Intersection Prevention 🛡️
  - [x] Learning Mode: Prevent moving object (sound/listener) from intersecting:
    - [x] Other main object (listener/sound)
    - [x] Box-shaped static objects
  - [ ] **Manual Dragging Enhancements:**
    - [ ] Implement real-time collision prevention for sound source & listener against:
      - [ ] Other main object
      - [ ] All static objects (boxes, spheres, new primitives, imported meshes)
    - [ ] Visual feedback on impending collision (e.g., object outline highlights red).
    - [ ] Option to "snap" to the last valid position or a nearby valid spot.
  - [ ] **Learning Mode Collision Robustness:**
    - [ ] Extend collision checks to include sphere-shaped static objects.
    - [ ] Implement Oriented Bounding Box (OBB) checks for rotated static box objects for more accuracy.
    - [ ] Add support for new primitive collision checks (cylinders, etc.) during learning.
  - [ ] **Collision Margins:**
    - [ ] Allow defining a small "padding" or collision margin around objects.

## III. Optimization & Machine Learning System 🧠
### A. Current "Cooperative Maximize" Learning Algorithm
  - [x] Randomness Control: "Exploration Factor" slider and integration.
  - [ ] **Adaptive Parameters:**
    - [ ] Adaptive `OPTIMIZATION_STEP_SIZE`: Gradually decrease step size as solution appears to converge.
    - [ ] Implement a simulated annealing schedule for `randomJumpProbability` and/or `explorationFactor`.
  - [ ] **Movement Strategies:**
    - [ ] Incorporate momentum into object movement choices to potentially pass small local optima.
  - [ ] **Initialization & Restart:**
    - [ ] Implement multi-start optimization: option to run several shorter learning cycles from different random initial positions.
  - [ ] **Constraint Handling:**
    - [ ] Investigate more sophisticated methods for handling boundary and inter-object constraints during optimization.

### B. New Optimization / Machine Learning Strategies 🤖
  - [ ] **Alternative Objective Functions:**
    - [ ] Implement "Cooperative Minimize Hits" mode (e.g., find acoustically "dead" spots for the listener).
  - [ ] **Adversarial / Competitive Modes:**
    - [ ] One object (e.g., source) tries to maximize score, while the other (e.g., listener) tries to minimize its own score or a related metric.
  - [ ] **Population-Based Methods:**
    - [ ] Explore Genetic Algorithms or Evolutionary Strategies for optimizing object positions and potentially other parameters (e.g., `numRays`, `maxReflections`).
  - [ ] **Reinforcement Learning (RL) - Experimental:**
    - [ ] Define a clear state space, action space, and reward function for object placement.
    - [ ] Experiment with simple RL agents (e.g., Q-learning, Deep Q-Network if feasible) for guiding one or both objects.
  - [ ] **Surrogate Modeling / Bayesian Optimization:**
    - [ ] If full ray evaluations are very costly, investigate building a surrogate model of the score landscape to guide search more efficiently.

### C. Topological Point Cloud / Spatial Awareness (Efficiency Enhancement) 🗺️
  - [ ] **Data Structure Implementation:**
    - [ ] Design and implement an Octree or a 3D Grid for spatial partitioning of the room.
    - [ ] Efficiently update occupancy status (empty, listener, source, both, static obstacle) upon object movement.
  - [ ] **Algorithm Integration:**
    - [ ] Use occupancy data to quickly prune invalid candidate positions for learning agents.
    - [ ] Derive a "gradient" or "potential field" from occupancy changes to guide movement.
    - [ ] Prioritize exploration of less-explored, valid regions.
  - [ ] **Visualization:**
    - [ ] Implement 2D slice views of the occupancy grid/Octree.
    - [ ] Create a color-coded 3D representation of explored/unexplored/high-potential zones, possibly in a separate viewport or overlay.

### D. Machine Learning Model Evaluation & Introspection 📊
  - [ ] **Performance Metrics:**
    - [ ] Track and display convergence speed of the learning algorithm.
    - [ ] Measure the diversity of solutions found if using population-based methods.
    - [ ] Log average evaluation time per step in learning.
  - [ ] **Visualization for ML:**
    - [ ] Plot score over iterations/time.
    - [ ] Plot key parameters (e.g., exploration factor, step size if adaptive) over iterations.
    - [ ] Visualize movement paths of source/listener during learning.
  - [ ] **Comparative Analysis:**
    - [ ] Implement a simple A/B testing framework or script to compare different optimization algorithms or parameter sets.
  - [ ] **Sensitivity Analysis:**
    - [ ] Tools to analyze how sensitive the optimal score is to small changes in object positions or key simulation parameters.

## IV. User Interface & Experience (UI/UX) ✨
### A. Record Management
  - [x] Logic: Store, retrieve, and apply best-score settings.
  - [x] File Structure: Abstract record logic into `records.go`.
  - [x] UI: Display list of records (score, iteration, key parameters).
  - [x] UI: "Apply" button to restore selected record's settings.
  - [ ] **Enhancements:**
    - [ ] Allow users to name or add notes/tags to saved records.
    - [ ] Display more detailed view of a record's parameters on hover or click.
    - [ ] Implement Export/Import functionality for record lists (e.g., JSON).

### B. Ray Visualization
  - [x] Toggle: "Show only rays reaching listener".
  - [x] Modal: Custom "Are you sure?" confirmation modal for showing all rays.
  - [ ] **Interactive Filtering & Analysis:**
    - [ ] UI to filter visualized rays by number of bounces.
    - [ ] UI to filter rays that hit specific (named) objects.
    - [ ] Ability to select an individual ray (or ray path) in the  meninas to view its detailed properties (length, bounces, hit objects).
  - [ ] **Advanced Listener Visualization:**
    - [ ] Display a "heatmap" or directional indicators on the listener sphere surface showing intensity/angle of arrival of rays.

### C. Scene Configuration & Control
  - [ ] **Dynamic Scene Editing:**
    - [ ] UI to add new primitive objects (cubes, spheres, cylinders) to the scene.
    - [ ] UI to select any scene object (static or main).
    - [ ] Gizmos/UI for moving, rotating, and scaling selected objects.
    - [ ] UI to delete objects from the scene.
    - [ ] UI to modify material properties (color, and later, acoustic properties) of selected objects.
  - [ ] **Environment Controls:**
    - [ ] UI sliders/inputs to dynamically change room dimensions (`roomWidth`, `roomDepth`, `roomHeight`).
  - [ ] **Camera Enhancements:**
    - [ ] Implement preset camera views (top, front, side, isometric).
    - [ ] "Focus on selected object" camera functionality.
    - [ ] More intuitive orbit/pan/zoom controls.
  - [ ] **State Management:**
    - [ ] Implement Undo/Redo functionality for scene manipulations and parameter changes.
    - [ ] Save/Load entire scene configurations (object placements, simulation parameters) to/from local files or browser storage.

### D. User Feedback & Assistance
  - [ ] **Real-time Performance Metrics:**
    - [ ] Display current FPS in the UI.
    - [ ] Show estimated WASM execution time for `visualizeSoundPropagation` or learning steps.
  - [ ] **Help & Guidance:**
    - [ ] Implement tooltips for all UI elements and parameter sliders.
    - [ ] Create a simple in-app help section or link to an external tutorial/documentation.
    - [ ] Provide clearer status messages during learning (e.g., "Exploring...", "Converged?", "Stuck in local optimum?").

## V. Code Structure & Maintainability 🛠️
### A. Core Code Refactoring
  - [x] Refactor Go code into separate, logical files (`vecmath.go`, `scene.go`, etc.).
  - [x] `server.go` functional for local development.
### B. Testing Framework 🧪
  - [ ] **Unit Tests:**
    - [ ] `vecmath.go`: For all vector operations.
    - [ ] `raycaster.go`: For primitive intersection tests (sphere, box) and ray evaluation logic.
    - [ ] `records.go`: For `RecordManager` add/sort/retrieve logic.
    - [ ] `optimization.go`: For movement validation and basic step logic.
    - [ ] Scoring logic: Test direct vs. indirect Fibonacci scoring.
  - [ ] **Integration Tests:**
    - [ ] Test ray propagation through a simple, fixed scene with expected hit counts/scores.
    - [ ] Test learning mode for a trivial case to ensure basic convergence.
  - [ ] **End-to-End (E2E) Test Stubs (if using a framework like Playwright/Puppeteer):**
    - [ ] Basic UI interaction tests (e.g., moving a slider updates visualization).
### C. Documentation 📚
  - [ ] **Code Comments:**
    - [ ] Add GoDoc comments for all public functions, structs, and important global variables.
    - [ ] Comment complex algorithms or non-obvious logic blocks.
  - [ ] **Project Documentation:**
    - [ ] Update/Maintain `DESIGN_DOCUMENT.md` (or similar) with new features, architectural decisions, and ML strategies.
    - [ ] Create a `DEVELOPER_GUIDE.md` for setting up the environment, building, running, and extending the project.
### D. Code Quality & Best Practices
  - [ ] **Linting & Formatting:**
    - [ ] Integrate `golangci-lint` or similar into the development workflow.
    - [ ] Enforce consistent code style (e.g., `gofmt`, `goimports`).
  - [ ] **Configuration Management:**
    - [ ] Move more "magic numbers" and default constants into a well-defined configuration struct or external config file (if app grows).
  - [ ] **Error Handling:**
    - [ ] Perform a thorough review of error handling, especially around JS interop and potentially failing Go operations.
    - [ ] Provide more informative error messages to the user/console.
  - [ ] **Dependency Management:**
    - [ ] Keep Go module dependencies (`go.mod`, `go.sum`) clean and up-to-date.

## VI. Performance Optimizations 🚀
### A. Go / WebAssembly Core
  - [ ] **Algorithmic Optimizations for Raycasting:**
    - [ ] Further optimize `performRaycast` with more aggressive early exits and optimized mathematical operations.
    - [ ] If spatial partitioning (Octree/Grid from III.C) is implemented, use it to accelerate ray-object intersection queries significantly.
  - [ ] **Concurrency & Parallelism (Advanced - requires careful JS interop):**
    - [ ] Explore if batches of rays (e.g., within `visualizeSoundPropagation` or `calculateListenerScore`) can be processed concurrently using Go goroutines before results are aggregated for JS.
    - [ ] **(Major)** Investigate moving the entire Go WASM simulation and learning loop to a Web Worker to prevent blocking the main browser thread, carefully managing data synchronization.
  - [ ] **Memory Management:**
    - [ ] Profile Go WASM execution for memory allocations in performance-critical loops (raycasting, learning steps).
    - [ ] Minimize allocations by reusing objects/slices where possible (e.g., sync.Pool for temporary ray data).
### B. JavaScript & Three.js Rendering
  - [ ] **Draw Call Reduction:**
    - [ ] Use `THREE.InstancedMesh` for rendering large numbers of identical static objects (if applicable to scene design).
    - [ ] For complex static parts of the scene, consider merging geometries to reduce draw calls.
  - [ ] **Level of Detail (LOD):**
    - [ ] If complex imported meshes are supported, implement LODs to render simpler versions for distant objects.
  - [ ] **Throttling & Debouncing:**
    - [ ] Ensure `renderSceneJS` is not called excessively or unnecessarily from Go, especially during rapid iterations in learning mode. Implement more aggressive throttling for UI updates during learning.
  - [ ] **Data Transfer Go <-> JS:**
    - [ ] Analyze and optimize the format and method of transferring scene and ray data. Explore direct use of `ArrayBuffer`s passed via `js.TypedArrayOf` if performance profiling indicates this as a bottleneck.
### C. Overall System Responsiveness
  - [ ] **Progressive Detailing / Adaptive Performance:**
    - [ ] During fast interactions (e.g., dragging objects, rapid learning iterations), automatically reduce `numRays` or visualization complexity (e.g., fewer bounces shown).
    - [ ] Restore full detail when the system is paused or interaction stops.
  - [ ] **Caching:**
    - [ ] Cache results of expensive calculations if inputs haven't changed significantly (e.g., part of the score for a static configuration).