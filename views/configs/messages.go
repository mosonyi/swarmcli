package configsview

import (
	"swarmcli/docker"
	"time"
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
	editConfigDoneMsg struct {
		Name      string
		Changed   bool
		OldConfig docker.ConfigWithDecodedData
		NewConfig docker.ConfigWithDecodedData
	}
	editConfigErrorMsg struct {
		err error
	}
	errorMsg error
)

type TickMsg time.Time

const PollInterval = 5 * time.Second
