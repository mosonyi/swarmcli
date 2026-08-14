// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import "context"

// ContextOps abstracts Docker context operations for testability and extensibility.
type ContextOps interface {
	ListContexts() ([]ContextInfo, error)
	UseContext(contextName string) error
	ValidateContext(ctx context.Context, contextName string) error
	InspectContext(contextName string) (string, error)
	ExportContext(contextName string) (string, error)
	ExportContextWithForce(contextName string) (string, error)
	CheckContextExportExists(contextName string) bool
	DeleteContext(contextName string) error
	ImportContext(filePath string) (string, error)
	CreateContext(name, dockerHost string) error
	CreateContextWithTLS(name, dockerHost, tlsPath string, skipTLSVerify bool) error
	CreateContextWithCertFiles(name, description, dockerHost, caFile, certFile, keyFile string, skipTLSVerify bool) error
	UpdateContextEndpoint(name, description, dockerHost string) error
}

type defaultContextOps struct{}

func (defaultContextOps) ListContexts() ([]ContextInfo, error) {
	return ListContexts()
}
func (defaultContextOps) UseContext(contextName string) error {
	return UseContext(contextName)
}
func (defaultContextOps) ValidateContext(ctx context.Context, contextName string) error {
	return ValidateContext(ctx, contextName)
}
func (defaultContextOps) InspectContext(contextName string) (string, error) {
	return InspectContext(contextName)
}
func (defaultContextOps) ExportContext(contextName string) (string, error) {
	return ExportContext(contextName)
}
func (defaultContextOps) ExportContextWithForce(contextName string) (string, error) {
	return ExportContextWithForce(contextName)
}
func (defaultContextOps) CheckContextExportExists(contextName string) bool {
	return CheckContextExportExists(contextName)
}
func (defaultContextOps) DeleteContext(contextName string) error {
	return DeleteContext(contextName)
}
func (defaultContextOps) ImportContext(filePath string) (string, error) {
	return ImportContext(filePath)
}
func (defaultContextOps) CreateContext(name, dockerHost string) error {
	return CreateContext(name, dockerHost)
}
func (defaultContextOps) CreateContextWithTLS(name, dockerHost, tlsPath string, skipTLSVerify bool) error {
	return CreateContextWithTLS(name, dockerHost, tlsPath, skipTLSVerify)
}
func (defaultContextOps) CreateContextWithCertFiles(name, description, dockerHost, caFile, certFile, keyFile string, skipTLSVerify bool) error {
	return CreateContextWithCertFiles(name, description, dockerHost, caFile, certFile, keyFile, skipTLSVerify)
}
func (defaultContextOps) UpdateContextEndpoint(name, description, dockerHost string) error {
	return UpdateContextEndpoint(name, description, dockerHost)
}
