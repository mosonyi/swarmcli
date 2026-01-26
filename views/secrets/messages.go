// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package secretsview

import (
	"swarmcli/docker"
	"time"
)

type secretsLoadedMsg []docker.SecretWithDecodedData

type errorMsg error

type TickMsg time.Time

type SpinnerTickMsg time.Time

// usedStatusUpdatedMsg carries a map of secret ID -> used boolean
type usedStatusUpdatedMsg map[string]bool

type secretDeletedMsg struct {
	Name string
}

type secretCreatedMsg struct {
	Name   string
	Secret docker.SecretWithDecodedData
}

type fileBrowserMsg struct {
	Path  string
	Files []string
}

type editorContentMsg struct {
	Content string
}

type usedByMsg struct {
	SecretName string
	UsedBy     []usedByItem
}

type secretRevealedMsg struct {
	SecretName string
	Content    string
}
