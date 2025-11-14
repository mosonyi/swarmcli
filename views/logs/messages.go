package logsview

type InitStreamMsg struct {
	Lines chan string
	Errs  chan error
}

type LineMsg struct {
	Line string
}

type StreamErrMsg struct {
	Err error
}

type StreamDoneMsg struct{}
