// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"

	"github.com/docker/docker/api/types/swarm"
)

// SecretOps abstracts secret operations for testability and extensibility.
type SecretOps interface {
	ListSecrets(ctx context.Context) ([]swarm.Secret, error)
	InspectSecret(ctx context.Context, nameOrID string) (*SecretWithDecodedData, error)
	CreateSecret(ctx context.Context, name string, data []byte, labels map[string]string) (swarm.Secret, error)
	CreateSecretVersion(ctx context.Context, baseSecret swarm.Secret, newData []byte) (swarm.Secret, error)
	RotateSecretInServices(ctx context.Context, oldSec *swarm.Secret, newSec swarm.Secret) error
	DeleteSecret(ctx context.Context, nameOrID string) error
	ListServicesUsingSecretID(ctx context.Context, secretID string) ([]swarm.Service, error)
	ListServicesUsingSecretName(ctx context.Context, name string) ([]swarm.Service, error)
}

type defaultSecretOps struct{}

func (defaultSecretOps) ListSecrets(ctx context.Context) ([]swarm.Secret, error) {
	return ListSecrets(ctx)
}
func (defaultSecretOps) InspectSecret(ctx context.Context, nameOrID string) (*SecretWithDecodedData, error) {
	return InspectSecret(ctx, nameOrID)
}
func (defaultSecretOps) CreateSecret(ctx context.Context, name string, data []byte, labels map[string]string) (swarm.Secret, error) {
	return CreateSecret(ctx, name, data, labels)
}
func (defaultSecretOps) CreateSecretVersion(ctx context.Context, baseSecret swarm.Secret, newData []byte) (swarm.Secret, error) {
	return CreateSecretVersion(ctx, baseSecret, newData)
}
func (defaultSecretOps) RotateSecretInServices(ctx context.Context, oldSec *swarm.Secret, newSec swarm.Secret) error {
	return RotateSecretInServices(ctx, oldSec, newSec)
}
func (defaultSecretOps) DeleteSecret(ctx context.Context, nameOrID string) error {
	return DeleteSecret(ctx, nameOrID)
}
func (defaultSecretOps) ListServicesUsingSecretID(ctx context.Context, secretID string) ([]swarm.Service, error) {
	return ListServicesUsingSecretID(ctx, secretID)
}
func (defaultSecretOps) ListServicesUsingSecretName(ctx context.Context, name string) ([]swarm.Service, error) {
	return ListServicesUsingSecretName(ctx, name)
}
