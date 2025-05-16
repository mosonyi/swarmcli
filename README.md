#SwarmCLI

Simple CLI for managing Docker Swarm clusters similar to k9s.

## Structure

```
├── main.go // Entry point for the application
..
├── docker // Utilities talk to the docker processes
├── cmds.go // Logic executed upon UI actions
├── model.go // Holds the app's state
├── styles.go // Defines the UI styles
├── update.go // Handles key input and updates the UI
├── utils.go // Utility functions
├── view.go // Draws the UI
```