package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"swarmcli/docker"
	"time"

	"github.com/jroimartin/gocui"
)

const (
	ModeNodes    = "nodes"
	ModeServices = "services"
	ModeStacks   = "stacks"
	Version      = "dev"
)

type State struct {
	Mode           string
	Nodes          []docker.SwarmNode
	CPUUsage       string
	MemUsage       string
	ContainerCount string
	ServiceCount   string
	TailingLogs    bool
	Paused         bool
	InInspectMode  bool
	ViewStack      []string
}

var state State

func main() {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()
	g.SetManagerFunc(layout)

	go startUsageUpdater(g)
	state.Mode = ModeNodes

	state.Nodes, err = docker.ListSwarmNodes()
	if err != nil {
		log.Println("Error fetching swarm nodes:", err)
		state.Nodes = []docker.SwarmNode{}
	}

	if err := keybindings(g); err != nil {
		log.Panicln(err)
	}
	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

func startUsageUpdater(gui *gocui.Gui) {
	for {
		state.CPUUsage = docker.GetSwarmCPUUsage()
		state.MemUsage = docker.GetSwarmMemUsage()
		state.ContainerCount = docker.GetContainerCount()
		state.ServiceCount = docker.GetServiceCount()

		if gui != nil {
			gui.Update(func(g *gocui.Gui) error {
				if g.CurrentView() != nil && g.CurrentView().Name() != "inspect" {
					if v, err := g.View("context"); err == nil {
						v.Clear()
						g.SetViewOnTop("footer")
						v.BgColor = gocui.ColorDefault
						v.FgColor = gocui.ColorWhite

						hostname, _ := os.Hostname()
						dockerVer := docker.GetDockerVersion()
						fmt.Fprintf(v, "\033[33m%-16s\033[37m%s\n", "Context:", hostname)
						fmt.Fprintf(v, "\033[33m%-16s\033[37m%s\n", "Version:", Version)
						fmt.Fprintf(v, "\033[33m%-16s\033[37m%s\n", "Docker version:", dockerVer)
						fmt.Fprintf(v, "\033[33m%-16s\033[37m%s\n", "RAM:", state.MemUsage)
						fmt.Fprintf(v, "\033[33m%-16s\033[37m%s\n", "CPU:", state.CPUUsage)
					}
				}
				return nil
			})
		}
		time.Sleep(5 * time.Second)
	}
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	// Always draw footer, even during inspect
	//log.Println("Rendering footer...")
	if v, err := g.SetView("footer", 0, maxY-3, maxX-1, maxY-1); err != nil && err != gocui.ErrUnknownView {
		return err
	} else {
		//v.Title = "FOOTER"
		//v.Frame = true

		v.Frame = false
		v.Clear()
		if state.InInspectMode {
			fmt.Fprintf(v, "\033[46m\033[30m <%s> \033[0m \033[43m\033[30m<inspect>\033[0m\n", state.Mode)

		} else {
			fmt.Fprintf(v, "\033[43m\033[30m <%s> \033[0m\n", state.Mode)
			g.Update(func(*gocui.Gui) error { return nil })

		}
	}

	if state.InInspectMode {
		return nil
	}

	if v, err := g.SetView("context", 0, 0, maxX-1, 7); err != nil && err != gocui.ErrUnknownView {
		return err
	} else if err == nil {
		v.Frame = false
		v.BgColor = gocui.ColorDefault
		v.FgColor = gocui.ColorWhite
	}

	if v, err := g.SetView("cmdbar", 0, 6, maxX-1, 8); err != nil && err != gocui.ErrUnknownView {
		return err
	} else if err == nil {
		v.Frame = false
		v.BgColor = gocui.ColorDefault
		v.FgColor = gocui.ColorYellow
		//fmt.Fprint(v, ": (nodes, services, stacks)")
	}

	mainTop := 9
	mainBottom := maxY - 4
	if _, err := g.View("cmdinput"); err == nil {
		mainTop = 10 // shift down if command input is active
	}
	if v, err := g.SetView("main", 0, mainTop, maxX-1, mainBottom); err != nil && err != gocui.ErrUnknownView {
		return err
	} else if err == nil {
		v.Title = strings.Title(state.Mode)
		v.Highlight = true
		v.SelFgColor = gocui.ColorBlack
		v.SelBgColor = gocui.ColorCyan
		v.BgColor = gocui.ColorDefault
		v.FgColor = gocui.ColorCyan
		v.Clear()

		switch state.Mode {
		case ModeNodes:
			nodes, _ := docker.ListSwarmNodes()
			for _, n := range nodes {
				fmt.Fprintln(v, n)
			}
		case ModeServices:
			services, _ := docker.ListSwarmServices()
			for _, service := range services {
				fmt.Fprintln(v, service)
			}
		case ModeStacks:
			stacks, _ := docker.ListStacks()
			for _, stack := range stacks {
				fmt.Fprintln(v, stack)
			}
		}
	}

	g.SetCurrentView("main")

	return nil
}

func inspectSelected(g *gocui.Gui, _ *gocui.View) error {
	state.ViewStack = append(state.ViewStack, state.Mode)
	v, err := g.View("main")
	if err != nil {
		return err
	}
	_, cy := v.Cursor()
	line, err := v.Line(cy)
	if err != nil || line == "" {
		return nil
	}
	item := strings.Fields(line)[0]

	var cmd *exec.Cmd
	switch state.Mode {
	case ModeNodes:
		cmd = exec.Command("docker", "node", "inspect", item)
	case ModeServices:
		cmd = exec.Command("docker", "service", "inspect", item)
	case ModeStacks:
		cmd = exec.Command("docker", "stack", "services", item)
	default:
		return nil
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("inspect error: %v %s", err, out)
	}

	v.Clear()
	state.InInspectMode = true
	g.SetCurrentView("main")
	v.Title = "Inspect (press ESC to go back)"
	v.Highlight = false
	v.SelBgColor = gocui.ColorDefault
	v.SelFgColor = gocui.ColorDefault
	v.SetCursor(0, 0)
	v.SetOrigin(0, 0)
	fmt.Fprint(v, string(out))
	v.SetCursor(0, 0)
	v.SetOrigin(0, 0)
	g.Update(func(*gocui.Gui) error { return nil })
	return nil
}

func showInspectOutput(g *gocui.Gui, output string) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("inspect", 5, 4, maxX-5, maxY-2); err != nil && err != gocui.ErrUnknownView {
		return err
	} else {
		v.Title = "Inspect"
		v.Wrap = true
		v.Autoscroll = false
		v.Editable = false
		v.Clear()
		v.Highlight = true
		v.Editable = false
		fmt.Fprint(v, output)
		g.SetCurrentView("inspect")
		v.SetCursor(0, 0)
		v.SetOrigin(0, 0)
	}
	return nil
}

func goBack(g *gocui.Gui, v *gocui.View) error {
	state.InInspectMode = false
	if len(state.ViewStack) > 0 {
		state.Mode = state.ViewStack[len(state.ViewStack)-1]
		state.ViewStack = state.ViewStack[:len(state.ViewStack)-1]
	}
	if v, err := g.View("main"); err == nil {
		v.Highlight = true
		v.SetCursor(0, 0)
		v.SetOrigin(0, 0)
	}
	return layout(g)
}

func activateCommandInput(g *gocui.Gui, v *gocui.View) error {
	maxX, _ := g.Size()
	if iv, err := g.SetView("cmdinput", 0, 8, maxX-1, 9); err != nil && err != gocui.ErrUnknownView {
		iv.Frame = true
		iv.Title = "Command"
		iv.FgColor = gocui.ColorCyan
		iv.Editable = true
		iv.Editor = gocui.DefaultEditor
		fmt.Fprint(iv, "> ")
		g.SetCurrentView("cmdinput")
		return nil
	}
	return nil
}

func cancelCommandInput(g *gocui.Gui, v *gocui.View) error {
	g.DeleteView("cmdinput")
	layout(g) // restore main view size
	g.SetCurrentView("main")
	return nil
}

func executeCommand(g *gocui.Gui, v *gocui.View) error {
	g.DeleteView("cmdinput")
	layout(g) // restore main view size
	v.Rewind()
	cmd := strings.TrimSpace(v.Buffer())
	g.DeleteView("cmdinput")
	g.SetCurrentView("main")
	if cmd == "services" || cmd == "nodes" || cmd == "stacks" {
		state.Mode = cmd
	}
	return layout(g)
}

func autocompleteCommand(g *gocui.Gui, v *gocui.View) error {
	input := strings.TrimSpace(v.Buffer())
	commands := []string{"nodes", "services", "stacks"}
	for _, c := range commands {
		if strings.HasPrefix(c, input) {
			v.Clear()
			fmt.Fprint(v, c)
			break
		}
	}
	return nil
}

func keybindings(g *gocui.Gui) error {
	g.SetKeybinding("main", gocui.KeyArrowDown, gocui.ModNone, cursorDown)
	g.SetKeybinding("main", gocui.KeyArrowUp, gocui.ModNone, cursorUp)
	g.SetKeybinding("main", gocui.KeyEsc, gocui.ModNone, goBack)
	g.SetKeybinding("main", 'b', gocui.ModNone, goBack)
	g.SetKeybinding("main", 'i', gocui.ModNone, inspectSelected)
	g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit)
	g.SetKeybinding("main", ':', gocui.ModNone, activateCommandInput)
	g.SetKeybinding("cmdinput", gocui.KeyEnter, gocui.ModNone, executeCommand)
	g.SetKeybinding("cmdinput", gocui.KeyEsc, gocui.ModNone, cancelCommandInput)
	g.SetKeybinding("cmdinput", gocui.KeyTab, gocui.ModNone, autocompleteCommand)
	g.SetKeybinding("main", 'q', gocui.ModNone, quit)
	g.SetKeybinding("main", gocui.KeyArrowDown, gocui.ModNone, cursorDown)
	g.SetKeybinding("main", gocui.KeyArrowUp, gocui.ModNone, cursorUp)
	return nil
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {

	if state.InInspectMode {
		ox, oy := v.Origin()
		v.SetOrigin(ox, oy+1)
		g.Update(func(*gocui.Gui) error { return nil })
	} else {
		_, y := v.Cursor()
		v.SetCursor(0, y+1)
	}
	return nil
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {

	if state.InInspectMode {
		ox, oy := v.Origin()
		if oy > 0 {
			v.SetOrigin(ox, oy-1)
			g.Update(func(*gocui.Gui) error { return nil })
		}
	} else {
		_, y := v.Cursor()
		if y > 0 {
			v.SetCursor(0, y-1)
		}
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
