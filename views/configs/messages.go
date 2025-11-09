package configsview

import "swarmcli/docker"

// Messages for async ops
type (
	configsLoadedMsg []docker.ConfigWithDecodedData
	configUpdatedMsg struct {
		Old docker.ConfigWithDecodedData
		New docker.ConfigWithDecodedData
	}
	configRotatedMsg struct {
		Old docker.ConfigWithDecodedData
		New docker.ConfigWithDecodedData
	}
	editConfigMsg struct {
		Name string
	}
	editConfigDoneMsg struct {
		updated bool
		config  docker.ConfigWithDecodedData
	}
	editConfigErrorMsg error
	errorMsg           error
)
