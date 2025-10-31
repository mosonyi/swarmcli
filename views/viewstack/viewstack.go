package viewstack

import "swarmcli/views/view"

type Stack struct {
	stack []view.View
}

// Push a view onto the stack
func (s *Stack) Push(v view.View) {
	s.stack = append(s.stack, v)
}

// Pop returns the last view and removes it from the stack
func (s *Stack) Pop() view.View {
	if len(s.stack) == 0 {
		return nil
	}
	last := s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]
	return last
}

// PopAndPush replaces the top view with a new one.
// If the stack is empty, it just pushes the new view.
func (s *Stack) PopAndPush(v view.View) {
	if len(s.stack) > 0 {
		s.stack = s.stack[:len(s.stack)-1]
	}
	s.stack = append(s.stack, v)
}

// Peek returns the last view without removing it
func (s *Stack) Peek() view.View {
	if len(s.stack) == 0 {
		return nil
	}
	return s.stack[len(s.stack)-1]
}

// Views returns the full stack (shallow copy)
func (s *Stack) Views() []view.View {
	cpy := make([]view.View, len(s.stack))
	copy(cpy, s.stack)
	return cpy
}

// Len returns how many views are on the stack
func (s *Stack) Len() int {
	return len(s.stack)
}

// Reset clears the stack
func (s *Stack) Reset() {
	s.stack = nil
}
