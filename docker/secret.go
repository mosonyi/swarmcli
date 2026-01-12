package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// ListServicesUsingSecretID returns all services that reference a secret by ID
func ListServicesUsingSecretID(ctx context.Context, secretID string) ([]swarm.Service, error) {
	client, err := GetClient()
	if err != nil {
		return nil, err
	}
	return listServicesUsingSecret(ctx, client, secretID)
}

// SecretWithDecodedData is a helper struct with the decoded data included.
// Note: Docker API doesn't return secret data for security reasons
type SecretWithDecodedData struct {
	Secret swarm.Secret
	Data   []byte // This will typically be nil/empty as secrets data cannot be retrieved
}

func (sec *SecretWithDecodedData) JSON() ([]byte, error) {
	type jsonSecret struct {
		Secret     swarm.Secret `json:"Secret"`
		DataParsed any          `json:"DataParsed,omitempty"`
	}

	obj := jsonSecret{Secret: sec.Secret}

	// Note: Secret data cannot be retrieved from Docker API
	// We only show metadata
	obj.DataParsed = "[Secret data is not available - secrets are write-only]"

	return json.Marshal(obj)
}

// PrettyJSON returns the JSON representation of the secret,
// but pretty-printed (indented) for human-readable editing.
func (sec *SecretWithDecodedData) PrettyJSON() ([]byte, error) {
	raw, err := sec.JSON()
	if err != nil {
		return nil, err
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		return nil, err
	}

	return pretty.Bytes(), nil
}

// ListSecrets retrieves all Docker Swarm secrets.
func ListSecrets(ctx context.Context) ([]swarm.Secret, error) {
	l().Debug("[ListSecrets] Listing all secrets")

	cli, err := GetClient()
	if err != nil {
		return nil, err
	}
	defer closeCli(cli)

	secrets, err := cli.SecretList(ctx, swarm.SecretListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	// SecretList doesn't populate all metadata like CreatedAt, so we need to inspect each secret
	fullSecrets := make([]swarm.Secret, len(secrets))
	for i, sec := range secrets {
		fullSec, _, err := cli.SecretInspectWithRaw(ctx, sec.ID)
		if err != nil {
			l().Warnf("[ListSecrets] Failed to inspect secret %s: %v", sec.Spec.Name, err)
			// Use the list result as fallback
			fullSecrets[i] = sec
			continue
		}
		fullSecrets[i] = fullSec
	}

	// 🔠 Sort secrets alphabetically by name
	sort.Slice(fullSecrets, func(i, j int) bool {
		return fullSecrets[i].Spec.Name < fullSecrets[j].Spec.Name
	})

	l().Infof("[ListSecrets] Found %d secrets", len(fullSecrets))
	return fullSecrets, nil
}

// InspectSecret fetches and returns the secret metadata.
// Note: Docker API does not return secret data for security reasons.
func InspectSecret(ctx context.Context, nameOrID string) (*SecretWithDecodedData, error) {
	l().Debugf("[InspectSecret] Inspecting secret: %s", nameOrID)

	cli, err := GetClient()
	if err != nil {
		return nil, err
	}
	defer closeCli(cli)

	sec, _, err := cli.SecretInspectWithRaw(ctx, nameOrID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect secret %q: %w", nameOrID, err)
	}

	l().Infof("[InspectSecret] Secret %q inspected successfully", sec.Spec.Name)
	// Note: sec.Spec.Data is not available for secrets
	return &SecretWithDecodedData{Secret: sec, Data: nil}, nil
}

// CreateSecretVersion creates a new secret, optionally using labels to mark lineage.
func CreateSecretVersion(ctx context.Context, baseSecret swarm.Secret, newData []byte) (swarm.Secret, error) {
	newName := nextSecretVersionName(baseSecret.Spec.Name)
	l().Infof("[CreateSecretVersion] Creating new secret version from %q → %q (size=%d bytes)",
		baseSecret.Spec.Name, newName, len(newData))
	spec := swarm.SecretSpec{
		Annotations: swarm.Annotations{
			Name: newName,
			Labels: map[string]string{
				"swarmcli.origin":  baseSecret.Spec.Name,
				"swarmcli.created": time.Now().UTC().Format(time.RFC3339),
			},
		},
		Data: newData,
	}

	return createSecretWithSpec(ctx, spec, "[CreateSecretVersion]")
}

// CreateSecret creates a new secret with the given name and data
func CreateSecret(ctx context.Context, name string, data []byte, labels map[string]string) (swarm.Secret, error) {
	l().Infof("[CreateSecret] Creating new secret %q (size=%d bytes)", name, len(data))

	// Merge user labels with swarmcli metadata
	allLabels := make(map[string]string)
	for k, v := range labels {
		allLabels[k] = v
	}
	allLabels["swarmcli.created"] = time.Now().UTC().Format(time.RFC3339)

	spec := swarm.SecretSpec{
		Annotations: swarm.Annotations{
			Name:   name,
			Labels: allLabels,
		},
		Data: data,
	}

	return createSecretWithSpec(ctx, spec, "[CreateSecret]")
}

// createSecretWithSpec centralizes secret creation: it calls the Docker API
// to create a secret from the provided spec, re-inspects the created secret,
// and returns the populated swarm.Secret or an error. The caller may pass a
// prefix for logging context (e.g. "[CreateSecretVersion]").
func createSecretWithSpec(ctx context.Context, spec swarm.SecretSpec, logPrefix string) (swarm.Secret, error) {
	cli, err := GetClient()
	if err != nil {
		return swarm.Secret{}, err
	}
	defer closeCli(cli)

	secName := spec.Name

	id, err := cli.SecretCreate(ctx, spec)
	if err != nil {
		l().Errorf("%s Failed to create secret %q: %v", logPrefix, secName, err)
		return swarm.Secret{}, fmt.Errorf("failed to create secret %q: %w", secName, err)
	}

	newSec, _, err := cli.SecretInspectWithRaw(ctx, id.ID)
	if err != nil {
		l().Errorf("%s Created secret %q but failed to re-inspect: %v", logPrefix, secName, err)
		return swarm.Secret{}, fmt.Errorf("failed to inspect new secret %q: %w", secName, err)
	}

	l().Infof("%s Successfully created new secret %q (ID=%s)", logPrefix, newSec.Spec.Name, newSec.ID)
	return newSec, nil
}

// RotateSecretInServices updates all services that reference oldSec to use newSec.
// If oldSec is nil, it tries to infer affected services automatically based on labels or content.
func RotateSecretInServices(ctx context.Context, oldSec *swarm.Secret, newSec swarm.Secret) error {
	if newSec.ID == "" {
		return fmt.Errorf("new secret must have a valid ID")
	}

	client, err := GetClient()
	if err != nil {
		return fmt.Errorf("failed to get docker client: %w", err)
	}
	defer closeCli(client)

	// --- 1. Find affected services
	var services []swarm.Service
	if oldSec != nil {
		services, err = listServicesUsingSecret(ctx, client, oldSec.ID)
	} else {
		services, err = listServicesUsingSecretName(ctx, client, newSec.Spec.Name)
	}
	if err != nil {
		return fmt.Errorf("failed to list services for rotation: %w", err)
	}

	if len(services) == 0 {
		l().Infof("No services found using secret %s", newSec.Spec.Name)
		return nil
	}

	// --- 2. Apply updates
	for _, svc := range services {
		updated := svc.Spec
		for i, secRef := range updated.TaskTemplate.ContainerSpec.Secrets {
			// Match by old ID or by name if oldSec is nil
			if (oldSec != nil && secRef.SecretID == oldSec.ID) ||
				(oldSec == nil && secRef.SecretName == newSec.Spec.Name) {
				updated.TaskTemplate.ContainerSpec.Secrets[i].SecretID = newSec.ID
				updated.TaskTemplate.ContainerSpec.Secrets[i].SecretName = newSec.Spec.Name
			}
		}

		if _, err := client.ServiceUpdate(ctx, svc.ID, svc.Version, updated, swarm.ServiceUpdateOptions{}); err != nil {
			return fmt.Errorf("failed to rotate secret in service %s: %w", svc.Spec.Name, err)
		}

		l().Infof("Rotated secret in service %s to %s", svc.Spec.Name, newSec.Spec.Name)
	}

	return nil
}

// DeleteSecret deletes a secret only if it's not referenced by any service.
func DeleteSecret(ctx context.Context, nameOrID string) error {
	sec, err := InspectSecret(ctx, nameOrID)
	if err != nil {
		return err
	}

	cli, err := GetClient()
	if err != nil {
		return err
	}
	defer closeCli(cli)

	svcs, err := listServicesUsingSecret(ctx, cli, sec.Secret.ID)
	if err != nil {
		return err
	}
	if len(svcs) > 0 {
		names := make([]string, len(svcs))
		for i, s := range svcs {
			names[i] = s.Spec.Name
		}
		return fmt.Errorf("cannot delete secret %q: still used by services %v", sec.Secret.Spec.Name, names)
	}

	return cli.SecretRemove(ctx, sec.Secret.ID)
}

// --- Helpers ---

var secretVersionSuffix = regexp.MustCompile(`^(.*)-v(\d+)$`)

func nextSecretVersionName(baseName string) string {
	if m := secretVersionSuffix.FindStringSubmatch(baseName); len(m) == 3 {
		prefix := m[1]
		v, _ := strconv.Atoi(m[2])
		return fmt.Sprintf("%s-v%d", prefix, v+1)
	}
	return fmt.Sprintf("%s-v2", baseName)
}

func listServicesUsingSecret(ctx context.Context, client *client.Client, secretID string) ([]swarm.Service, error) {
	services, err := client.ServiceList(ctx, swarm.ServiceListOptions{})
	if err != nil {
		return nil, err
	}
	var filtered []swarm.Service
	for _, s := range services {
		for _, sec := range s.Spec.TaskTemplate.ContainerSpec.Secrets {
			if sec.SecretID == secretID {
				filtered = append(filtered, s)
				break
			}
		}
	}
	return filtered, nil
}

// ListServicesUsingSecretName returns all services that reference a secret by name
func ListServicesUsingSecretName(ctx context.Context, name string) ([]swarm.Service, error) {
	client, err := GetClient()
	if err != nil {
		return nil, err
	}
	return listServicesUsingSecretName(ctx, client, name)
}

func listServicesUsingSecretName(ctx context.Context, client *client.Client, name string) ([]swarm.Service, error) {
	services, err := client.ServiceList(ctx, swarm.ServiceListOptions{})
	if err != nil {
		return nil, err
	}
	var filtered []swarm.Service
	for _, s := range services {
		for _, sec := range s.Spec.TaskTemplate.ContainerSpec.Secrets {
			if sec.SecretName == name {
				filtered = append(filtered, s)
				break
			}
		}
	}
	return filtered, nil
}
