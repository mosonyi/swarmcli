// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package view

// viewRegistry stores all view factories, keyed by view name.
// It is only written to during init() (single-threaded) and read-only
// after program start, so no mutex is needed.
var viewRegistry = map[string]Factory{}

// RegisterView registers a view factory under the given name.
// Must only be called from init() functions.
func RegisterView(name string, f Factory) { viewRegistry[name] = f }

// GetFactory returns the factory for the given view name.
func GetFactory(name string) (Factory, bool) { f, ok := viewRegistry[name]; return f, ok }

// AllFactories returns all registered view factories.
func AllFactories() map[string]Factory { return viewRegistry }
