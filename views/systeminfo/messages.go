package systeminfoview

import "time"

type Msg struct {
	context    string
	cpu        string
	mem        string
	containers int
	services   int
}

type SlowStatusMsg struct {
	cpu string
	mem string
}

type TickMsg time.Time

type SpinnerTickMsg time.Time
