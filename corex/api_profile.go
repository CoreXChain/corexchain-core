// Copyright 2026 CoreX Team
// This file is part of the corex-chain library.
//
// The corex-chain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The corex-chain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the corex-chain library. If not, see <http://www.gnu.org/licenses/>.

package corex

import "com.corexkey/corex/corexconfig"

// ProfileSummary exposes the high-level CoreXChain profile reflected in README and configs.
type ProfileSummary struct {
	Chain         string                          `json:"chain"`
	Developer     string                          `json:"developer"`
	Consensus     string                          `json:"consensus"`
	Communication corexconfig.CommunicationConfig `json:"communication"`
	Governance    corexconfig.GovernanceConfig    `json:"governance"`
	Trust         corexconfig.TrustConfig         `json:"trust"`
	Asset         corexconfig.AssetConfig         `json:"asset"`
	NodeRole      corexconfig.NodeRoleConfig      `json:"nodeRole"`
}

// ProfileAPI provides read-only access to the CoreXChain profile model.
type ProfileAPI struct {
	e *CoreXChain
}

// NewProfileAPI creates a new CoreXChain profile API instance.
func NewProfileAPI(e *CoreXChain) *ProfileAPI {
	return &ProfileAPI{e: e}
}

// Summary returns the high-level CoreXChain profile.
func (api *ProfileAPI) Summary() ProfileSummary {
	return ProfileSummary{
		Chain:         "CoreXChain",
		Developer:     "CoreX Team",
		Consensus:     "PoV",
		Communication: api.e.config.Communication,
		Governance:    api.e.config.Governance,
		Trust:         api.e.config.Trust,
		Asset:         api.e.config.Asset,
		NodeRole:      api.e.config.NodeRole,
	}
}

// Communication returns the communication layer profile.
func (api *ProfileAPI) Communication() corexconfig.CommunicationConfig {
	return api.e.config.Communication
}

// Governance returns the governance layer profile.
func (api *ProfileAPI) Governance() corexconfig.GovernanceConfig {
	return api.e.config.Governance
}

// Trust returns the trust layer profile.
func (api *ProfileAPI) Trust() corexconfig.TrustConfig {
	return api.e.config.Trust
}

// Asset returns the asset model profile.
func (api *ProfileAPI) Asset() corexconfig.AssetConfig {
	return api.e.config.Asset
}

// NodeRole returns the node role profile.
func (api *ProfileAPI) NodeRole() corexconfig.NodeRoleConfig {
	return api.e.config.NodeRole
}
