package tasksview

import (
	"swarmcli/docker"

	tea "github.com/charmbracelet/bubbletea"
)

type TasksLoadedMsg struct {
	Tasks []docker.TaskEntry
	Error error
}

func LoadTasksCmd(stackName string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := docker.GetTasksForStack(stackName)
		return TasksLoadedMsg{
			Tasks: tasks,
			Error: err,
		}
	}
}
