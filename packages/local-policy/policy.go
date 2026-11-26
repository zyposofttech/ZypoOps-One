// Package policy implements the two mock operations authorized in the Sprint 1 harness.
package policy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"
	contracts "zyposoft.local/operations/packages/management-contracts"
	"zyposoft.local/operations/packages/signing-go"
)

type Bundle struct {
	Job      json.RawMessage   `json:"job"`
	Manifest json.RawMessage   `json:"manifest,omitempty"`
	Grants   []json.RawMessage `json:"grants,omitempty"`
	Backup   json.RawMessage   `json:"backup,omitempty"`
}
type BackupRequirement struct {
	Product, Artifact, BackupManifest, Signer string
	Categories                                []string
}
type Config struct {
	Scope                        signing.Scope
	Installation, Agent, Version string
	Trust                        signing.Trust
	TestArtifacts                map[string]string
	Preflight                    func(product string, locks []string) bool
	LocalCheck                   func(context.Context, string, []string) error
	BackupForJob                 map[string]BackupRequirement
}
type Authorized struct {
	Job                 contracts.MC03
	Digest              string
	Grants              map[string]string
	Product             string
	Locks               []string
	checkedAt, deadline time.Time
}

func Digest(wire []byte) (string, error) {
	data, err := contracts.Canonical(wire)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
func within(start, end string, now time.Time) bool {
	a, e1 := time.Parse(time.RFC3339Nano, start)
	b, e2 := time.Parse(time.RFC3339Nano, end)
	return e1 == nil && e2 == nil && !now.Before(a) && now.Before(b)
}
func meets(installed, minimum string) bool {
	a, b := strings.Split(installed, "."), strings.Split(minimum, ".")
	if len(a) != 3 || len(b) != 3 {
		return false
	}
	for i := range a {
		x, e := strconv.ParseUint(a[i], 10, 32)
		y, f := strconv.ParseUint(b[i], 10, 32)
		if e != nil || f != nil {
			return false
		}
		if x != y {
			return x > y
		}
	}
	return true
}
func verified[T any](id string, wire []byte, c Config) (T, error) {
	var v T
	_, err := signing.Verify(id, wire, c.Scope, c.Trust)
	if err != nil {
		return v, err
	}
	err = json.Unmarshal(wire, &v)
	return v, err
}
func (c Config) Authorize(b Bundle, now time.Time) (Authorized, error) {
	return c.AuthorizeContext(context.Background(), b, now)
}
func (c Config) AuthorizeContext(ctx context.Context, b Bundle, now time.Time) (Authorized, error) {
	approved, err := c.Validate(b, now)
	if err != nil || approved.Product == "" {
		return approved, err
	}
	if c.LocalCheck != nil {
		return approved, c.LocalCheck(ctx, approved.Product, slices.Clone(approved.Locks))
	}
	if c.Preflight == nil || !c.Preflight(approved.Product, slices.Clone(approved.Locks)) {
		return approved, errors.New("SITE_LOCK_ACTIVE")
	}
	return approved, nil
}

// Current rechecks the validated proof interval after external I/O or lock contention.
func (a Authorized) Current(now time.Time) bool {
	return !now.Before(a.checkedAt) && now.Before(a.deadline)
}
func (a *Authorized) limit(value string) {
	t, _ := time.Parse(time.RFC3339Nano, value)
	if t.Before(a.deadline) {
		a.deadline = t
	}
}

// Validate performs contract, signature, binding and time checks without I/O.
func (c Config) Validate(b Bundle, now time.Time) (Authorized, error) {
	denied := errors.New("LOCAL_POLICY_DENIED")
	var result Authorized
	job, err := verified[contracts.MC03]("MC-03", b.Job, c)
	if err != nil {
		return result, err
	}
	j := job.Body
	if j.InstallationId != c.Installation || j.AgentId != c.Agent {
		return result, errors.New("INSTALLATION_SCOPE_MISMATCH")
	}
	if !within(j.NotBefore, j.ExpiresAt, now) || !within(j.IssuedAt, j.ExpiresAt, now) {
		return result, errors.New("JOB_EXPIRED")
	}
	if !meets(c.Version, j.MinAgentVersion) || j.Sequence <= 0 || j.Preconditions.ObservedRevision != nil || j.Preconditions.MaintenanceWindowId != nil {
		return result, denied
	}
	hash, err := Digest(b.Job)
	if err != nil {
		return result, err
	}
	deadline, _ := time.Parse(time.RFC3339Nano, j.ExpiresAt)
	result = Authorized{Job: job, Digest: hash, Grants: map[string]string{}, checkedAt: now, deadline: deadline}
	if j.CommandType == "REQUEST_HEALTH" && len(j.ApprovalGrantIds) == 0 && len(b.Grants) == 0 && len(b.Manifest) == 0 && len(b.Backup) == 0 {
		return result, nil
	}
	if j.CommandType != "DEPLOY_RELEASE" {
		return result, denied
	}
	var p contracts.DeployReleaseParams
	if err = json.Unmarshal(j.Params, &p); err != nil {
		return result, denied
	}
	manifest, err := verified[contracts.MC05]("MC-05", b.Manifest, c)
	if err != nil {
		return result, errors.New("ARTIFACT_UNTRUSTED")
	}
	digest, err := Digest(b.Manifest)
	m := manifest.Body
	if err != nil || digest != p.ManifestSha256 || m.ProductId != p.ProductKey || c.TestArtifacts[digest] != p.ProductKey || p.ReleaseChannel != "pilot" || !meets(c.Version, m.MinAgentVersion) || !within(manifest.SentAt, m.ExpiresAt, now) {
		return result, errors.New("ARTIFACT_UNTRUSTED")
	}
	if len(j.ApprovalGrantIds) == 0 || len(b.Grants) != len(j.ApprovalGrantIds) {
		return result, errors.New("APPROVAL_REQUIRED")
	}
	result.limit(m.ExpiresAt)
	for _, raw := range b.Grants {
		grant, err := verified[contracts.MC09]("MC-09", raw, c)
		if err != nil {
			return result, errors.New("APPROVAL_REQUIRED")
		}
		g := grant.Body
		_, duplicate := result.Grants[g.GrantId]
		if duplicate || !slices.Contains(j.ApprovalGrantIds, g.GrantId) || g.TargetDigestSha256 != digest || g.Operation != j.CommandType || g.Purpose != "DEPLOYMENT" || !slices.Contains(g.ResourceScope, p.ProductKey) || !within(g.ValidFrom, g.ValidUntil, now) {
			return result, errors.New("APPROVAL_REQUIRED")
		}
		hash, err := Digest(raw)
		if err != nil {
			return result, err
		}
		result.Grants[g.GrantId] = hash
		result.limit(g.ValidUntil)
	}
	if m.BackupEvidenceRequired {
		backup, err := verified[contracts.MC14]("MC-14", b.Backup, c)
		if err != nil {
			return result, denied
		}
		body := backup.Body
		when, err := time.Parse(time.RFC3339Nano, body.LastSuccessAt)
		if err != nil || now.Before(when) || now.Sub(when) >= 24*time.Hour || !body.Encrypted || !body.IntegrityVerified || body.ErrorState != "NONE" {
			return result, denied
		}
		result.limit(when.Add(24 * time.Hour).Format(time.RFC3339Nano))
		expected, ok := c.BackupForJob[j.JobId]
		sent, e := time.Parse(time.RFC3339Nano, backup.SentAt)
		if !ok || expected.Product != p.ProductKey || expected.Artifact != digest || expected.BackupManifest != body.BackupManifestSha256 || expected.Signer != backup.Signature.Kid || len(expected.Categories) == 0 || len(expected.Categories) != len(body.DatasetCategories) || e != nil || sent.After(now) || sent.Before(when) || body.BackupAgeSeconds != int64(sent.Sub(when)/time.Second) {
			return result, errors.New("BACKUP_NOT_VERIFIED")
		}
		for _, category := range body.DatasetCategories {
			if !slices.Contains(expected.Categories, category) {
				return result, errors.New("BACKUP_NOT_VERIFIED")
			}
		}
	}
	if c.LocalCheck != nil {
		if len(m.HealthProbes) != 1 || m.HealthProbes[0].ProbeType != "HTTP" || m.HealthProbes[0].Target != "/local/v1/readiness" {
			return result, errors.New("UNSUPPORTED_PROBE")
		}
	}
	result.Product = m.ProductId
	result.Locks = m.RequiredLocks
	return result, nil
}
func (c Config) Clone() Config {
	c.Trust = maps.Clone(c.Trust)
	c.TestArtifacts = maps.Clone(c.TestArtifacts)
	c.BackupForJob = maps.Clone(c.BackupForJob)
	for id, r := range c.BackupForJob {
		r.Categories = slices.Clone(r.Categories)
		c.BackupForJob[id] = r
	}
	return c
}
