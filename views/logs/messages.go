package logsview

type InitStreamMsg struct {
	Lines    chan string
	Errs     chan error
	MaxLines int
}

type LineMsg struct {
	Line string
}

type StreamErrMsg struct {
	Err error
}

type StreamDoneMsg struct{}

type WrapToggledMsg struct{}
