// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

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
