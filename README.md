#SwarmCLI

Simple CLI for managing Docker Swarm clusters similar to k9s.

## Structure

```
├── main.go
├── model.go // Holds the app's state
├── update.go // Handles key input and updates the UI
├── view.go // Draws the UI
├── styles.go // Defines the UI styles
├── docker // Utilities talk to the docker processes
```