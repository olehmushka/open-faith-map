// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package vouching owns the web-of-trust guarantor verification mechanism (D-Vouching, M6): an
// immutable vouching_edges log plus a mutable guarantor_status overlay. See
// docs/modules/vouching.md for the entity model. Subpackages follow the same
// transport → application → domain → adapters layering as every other module
// (docs/architecture/overview.md).
package vouching
