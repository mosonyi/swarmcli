package systeminfoview

type Msg struct {
	host       string
	cpu        string
	mem        string
	containers int
	services   int
}
