package configsview

import (
	"swarmcli/docker"
	"time"

	"github.com/docker/docker/api/types/swarm"
)

// Messages for async ops
type (
	configsLoadedMsg []docker.ConfigWithDecodedData
	configRotatedMsg struct {
		Old docker.ConfigWithDecodedData
		New docker.ConfigWithDecodedData
	}
	configDeletedMsg struct {
		Name  string
		Index int
	}
	editConfigMsg struct {
		Name string
	}
	createConfigMsg struct {
		Name string
	}
	editConfigDoneMsg struct {
		Name      string
		Changed   bool
		OldConfig docker.ConfigWithDecodedData
		NewConfig docker.ConfigWithDecodedData
	}
	configCreatedMsg struct {
		Config swarm.Config
	}
	configCreateErrorMsg struct {
		err error
	}
	editorContentReadyMsg struct {
		Name string
		Data []byte
		Err  error
	}
	fileContentReadyMsg struct {
		Name     string
		FilePath string
		Data     []byte
		Err      error
	}
	editConfigErrorMsg struct {
		err error
	}
	filesLoadedMsg struct {
		Path  string
		Files []string
		Error error
	}
	usedByMsg struct {
		ConfigName string
		Stacks     []string
		Error      error
	}
	NavigateToStackMsg struct {
		StackName string
	}
	// New message for navigating directly to services in a stack
	NavigateToServicesInStackMsg struct {
		StackName string
	}
	errorMsg error
)

type TickMsg time.Time

const PollInterval = 5 * time.Second
