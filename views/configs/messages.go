package configsview

import "swarmcli/docker"

// Messages for async ops
type (
	configsLoadedMsg []docker.ConfigWithDecodedData
	configRotatedMsg struct {
		Old docker.ConfigWithDecodedData
		New docker.ConfigWithDecodedData
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
