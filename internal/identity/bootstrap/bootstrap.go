// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package bootstrap seeds the first instance admin (D-SeedBootstrap's amendment): identity is the
// one deliberate exception to deterministic seed migrations — a committed seed email in an
// open-source repo would be a universal admin backdoor, so the first admin is seeded at boot from
// operator-supplied install config instead. In ONE transaction it creates (or reuses) a person, an
// account + external identity binding the configured IdP (issuer, subject), and an instance-admin
// grant. Idempotent: it skips entirely once any active instance admin exists.
//
// Ported from ../go-oikumenea/internal/identityfederation/bootstrap/bootstrap.go, trimmed: no audit
// recording (this port has no audit log — D-DirectTokenVerification's own Consequences note), no
// Force/recover-admin break-glass path (not requested by M10.2's scope), and resolveOrCreatePerson
// drops the upstream Sex field entirely (identity_persons has no sex column — D-CorePortScope).
package bootstrap

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	authzadapters "github.com/olehmushka/open-faith-map/internal/authz/adapters"
	identityadapters "github.com/olehmushka/open-faith-map/internal/identity/adapters"
	"github.com/olehmushka/open-faith-map/internal/identity/domain"
)

// AdminSeed is the operator-supplied first-admin identity (from install config / env).
type AdminSeed struct {
	Issuer      string
	Subject     string
	Email       string
	DisplayName string
	PersonCode  string
}

// Result reports what a bootstrap run did.
type Result struct {
	Skipped        bool // an instance admin already existed
	PersonID       string
	AccountID      string
	InstanceAdmin  string // the instance-admin grant id (empty when the person was already an admin)
	CreatedPerson  bool
	CreatedAccount bool
}

// ErrInvalidSeed indicates the operator-supplied seed is incomplete (missing issuer/subject).
var ErrInvalidSeed = errors.New("bootstrap admin seed requires issuer and subject")

// ErrPlaceholderSeed indicates the seed carries the documented .env.example dummy value outside
// local/dev — refused the same way GuardSymmetricIssuers refuses HS256 outside local/dev, so a
// copy-pasted .env.example can never boot a real deployment with a pre-linked backdoor admin.
var ErrPlaceholderSeed = errors.New("bootstrap admin seed is the documented placeholder value; set a real issuer/subject for this deployment")

// PlaceholderSubject is the dummy value shipped (commented-out) in .env.example. It is never a real
// Google `sub` — Google subjects are numeric strings — so it can be recognized unambiguously.
const PlaceholderSubject = "REPLACE_ME"

// ValidateSeedForEnvironment fails closed when seed looks unset or carries the documented placeholder
// and environment is not local/dev — mirroring GuardSymmetricIssuers' fail-closed shape. Call this
// before Run in any environment other than local/dev; local/dev may seed a placeholder-shaped value
// deliberately (a fresh checkout with no real IdP configured yet).
func ValidateSeedForEnvironment(seed AdminSeed, environment string) error {
	if environment == "local" || environment == "dev" {
		return nil
	}
	if strings.TrimSpace(seed.Subject) == "" || seed.Subject == PlaceholderSubject {
		return ErrPlaceholderSeed
	}
	return nil
}

// Run executes the idempotent first-admin seed in one transaction.
func Run(ctx context.Context, pool *pgxpool.Pool, seed AdminSeed) (Result, error) {
	if strings.TrimSpace(seed.Issuer) == "" || strings.TrimSpace(seed.Subject) == "" {
		return Result{}, ErrInvalidSeed
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	authzStore := authzadapters.NewRepository(tx)
	hasAdmin, err := authzStore.HasActiveInstanceAdmin(ctx)
	if err != nil {
		return Result{}, err
	}
	if hasAdmin {
		return Result{Skipped: true}, nil // idempotent: an instance admin already exists
	}

	identityStore := identityadapters.NewStore(tx)

	person, createdPerson, err := resolveOrCreatePerson(ctx, identityStore, seed)
	if err != nil {
		return Result{}, err
	}
	res := Result{PersonID: person.ID, CreatedPerson: createdPerson}

	account, createdAccount, err := resolveOrCreateAccount(ctx, identityStore, person.ID, seed.Email)
	if err != nil {
		return Result{}, err
	}
	res.AccountID = account.ID
	res.CreatedAccount = createdAccount

	if err := linkIdentity(ctx, identityStore, account.ID, person.ID, seed); err != nil {
		return Result{}, err
	}

	adminID, err := authzStore.InsertInstanceAdmin(ctx, person.ID, "") // granted_by NULL (bootstrap)
	if err != nil {
		return Result{}, err
	}
	res.InstanceAdmin = adminID

	if err := tx.Commit(ctx); err != nil {
		return Result{}, err
	}
	return res, nil
}

// resolveOrCreatePerson reuses an existing person by code (link-to-existing) or creates one.
// DisplayName falls back to the code/subject so the NOT NULL display_name is satisfied.
func resolveOrCreatePerson(ctx context.Context, store *identityadapters.Store, seed AdminSeed) (domain.Person, bool, error) {
	if seed.PersonCode != "" {
		p, err := store.GetActivePersonByCode(ctx, seed.PersonCode)
		if err == nil {
			return p, false, nil
		}
		if !errors.Is(err, domain.ErrPersonNotFound) {
			return domain.Person{}, false, err
		}
	}
	display := firstNonEmpty(seed.DisplayName, seed.PersonCode, seed.Subject)
	created, err := store.InsertPerson(ctx, domain.Person{Code: seed.PersonCode, DisplayName: display})
	if err != nil {
		return domain.Person{}, false, err
	}
	return created, true, nil
}

func resolveOrCreateAccount(ctx context.Context, store *identityadapters.Store, personID, email string) (domain.Account, bool, error) {
	account, err := store.GetActiveAccountByPerson(ctx, personID)
	if err == nil {
		return account, false, nil
	}
	if !errors.Is(err, domain.ErrAccountNotFound) {
		return domain.Account{}, false, err
	}
	created, err := store.InsertAccount(ctx, personID, email)
	if err != nil {
		return domain.Account{}, false, err
	}
	return created, true, nil
}

// linkIdentity links the configured (issuer, subject) to the account, tolerating an idempotent
// re-run. Pre-checks existence rather than catching a unique-violation: a constraint error would
// poison the surrounding transaction, so the follow-up query could not run. An identity that already
// maps to THIS person is a no-op; one mapping elsewhere is refused.
func linkIdentity(ctx context.Context, store *identityadapters.Store, accountID, personID string, seed AdminSeed) error {
	existing, err := store.ResolveBySubject(ctx, seed.Issuer, seed.Subject)
	switch {
	case err == nil:
		if existing.PersonID == personID {
			return nil // already linked to this person — idempotent
		}
		return domain.ErrIdentityConflict // linked to a different person — refuse
	case !errors.Is(err, domain.ErrIdentityNotFound):
		return err
	}
	_, err = store.InsertIdentity(ctx, accountID, seed.Issuer, seed.Subject)
	return err
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
