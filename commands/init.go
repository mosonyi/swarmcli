package commands

func init() {
	Register(HelpCommand{})
	Register(DockerStackLs{})
	// Register(DockerServiceLs{})
	// Register(InspectNode{})
}
