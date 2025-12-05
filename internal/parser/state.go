package parser

type Phase int

const (
	Outside Phase = iota
	InCmd
	InPayload
)

type State struct {
	Phase         Phase
	Buffer        string
	PayloadBuffer string
	IsReply       bool
}

type Result struct {
	State    State
	Output   string
	SawReply bool
}

type RequestMeta struct {
	Model      string
	RequestID  string
	PromptHash string
	Timestamp  string
	Nonce      string
}

func NewState() State {
	return State{Phase: Outside}
}

func TransitionToCmd(isReply bool) State {
	return State{Phase: InCmd, IsReply: isReply}
}

func TransitionToPayload() State {
	return State{Phase: InPayload}
}

func TransitionToOutside() State {
	return State{Phase: Outside}
}

func withBuffer(s State, b string) State {
	return State{Phase: s.Phase, Buffer: b, PayloadBuffer: s.PayloadBuffer, IsReply: s.IsReply}
}

func withPayload(s State, p string) State {
	return State{Phase: s.Phase, Buffer: s.Buffer, PayloadBuffer: p, IsReply: s.IsReply}
}

func clearBuffer(s State) State {
	return State{Phase: s.Phase, PayloadBuffer: s.PayloadBuffer, IsReply: s.IsReply}
}
