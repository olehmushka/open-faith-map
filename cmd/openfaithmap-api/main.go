// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Command openfaithmap-api is the composition root for OpenFaithMap's backend
// (docs/architecture/overview.md). `serve` boots the witchcraft server. M2 added the first real
// module, registration (docs/modules/registration.md); M3 adds content (docs/modules/content.md) —
// moderation/vouching still land as each reaches its own "backend" gate, see
// docs/development-process.md.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	congregationimportadapters "github.com/olehmushka/open-faith-map/internal/congregationimport/adapters"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/adapters/connectors/arrnc"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/adapters/connectors/uaedr"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/adapters/geocoders/nominatim"
	congregationimportapplication "github.com/olehmushka/open-faith-map/internal/congregationimport/application"
	congregationimportdomain "github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
	congregationimporttransport "github.com/olehmushka/open-faith-map/internal/congregationimport/transport"
	gencongregationimport "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/congregationimport"
	gencontent "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/content"
	gendiscovery "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/discovery"
	genmoderation "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/moderation"
	genregistration "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/registration"
	genvouching "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/vouching"
	contentadapters "github.com/olehmushka/open-faith-map/internal/content/adapters"
	contentapplication "github.com/olehmushka/open-faith-map/internal/content/application"
	contentdomain "github.com/olehmushka/open-faith-map/internal/content/domain"
	contenttransport "github.com/olehmushka/open-faith-map/internal/content/transport"
	"github.com/olehmushka/open-faith-map/internal/coreintegration"
	discoveryadapters "github.com/olehmushka/open-faith-map/internal/discovery/adapters"
	discoveryapplication "github.com/olehmushka/open-faith-map/internal/discovery/application"
	discoverytransport "github.com/olehmushka/open-faith-map/internal/discovery/transport"
	moderationadapters "github.com/olehmushka/open-faith-map/internal/moderation/adapters"
	moderationapplication "github.com/olehmushka/open-faith-map/internal/moderation/application"
	moderationdomain "github.com/olehmushka/open-faith-map/internal/moderation/domain"
	moderationtransport "github.com/olehmushka/open-faith-map/internal/moderation/transport"
	"github.com/olehmushka/open-faith-map/internal/platform/config"
	regadapters "github.com/olehmushka/open-faith-map/internal/registration/adapters"
	regapplication "github.com/olehmushka/open-faith-map/internal/registration/application"
	regtransport "github.com/olehmushka/open-faith-map/internal/registration/transport"
	vouchingadapters "github.com/olehmushka/open-faith-map/internal/vouching/adapters"
	vouchingapplication "github.com/olehmushka/open-faith-map/internal/vouching/application"
	vouchingtransport "github.com/olehmushka/open-faith-map/internal/vouching/transport"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
)

// contentSiteResolver adapts contentapplication.Service's public read onto discovery's own
// ContentResolver interface — an interface-call cross-module dependency (conventions.md), even
// though the underlying discovery_site_cache.content_site_id column is a real in-schema FK
// (DS-OFM-13, docs/modules/discovery.md).
type contentSiteResolver struct {
	content *contentapplication.Service
}

func (r *contentSiteResolver) GetSiteByUnit(ctx context.Context, congregationUnitRID string) (string, bool, error) {
	site, err := r.content.GetSite(ctx, congregationUnitRID)
	if errors.Is(err, contentdomain.ErrSiteNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return site.ID, true, nil
}

// moderationVouchReporter adapts moderationapplication.Service's FileReport onto vouching's own
// ModerationReporter interface — the same in-process interface-call cross-module dependency shape
// as contentSiteResolver above. Translates vouching's own GuarantorRevokedEvent vocabulary into
// moderation's FileReportInput here, at the composition root, so internal/vouching's application
// package never imports internal/moderation's domain or application packages directly (M6's own
// decision, docs/modules/vouching.md: revocation queues moderator review, it never invalidates
// anything automatically). Uses moderation's OTHER reason code with a descriptive detail string
// rather than a new GUARANTOR_REVOKED enum value, to avoid touching M5's already-migrated CHECK
// constraint and generated SDKs for this one caller.
type moderationVouchReporter struct {
	moderation *moderationapplication.Service
}

func (r *moderationVouchReporter) ReportGuarantorRevoked(ctx context.Context, event vouchingapplication.GuarantorRevokedEvent) error {
	detail := fmt.Sprintf("guarantor_revoked: guarantor=%s claimant=%s congregation=%s reason=%s",
		event.GuarantorPersonRID, event.ClaimantPersonRID, event.CongregationUnitID, event.RevokedReason)
	_, err := r.moderation.FileReport(ctx, moderationdomain.FileReportInput{
		TargetKind: moderationdomain.TargetVouchingEdge,
		TargetRef:  event.VouchID,
		ReasonCode: moderationdomain.ReasonOther,
		Detail:     &detail,
	})
	return err
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	cmd := "serve"
	if len(args) > 0 {
		cmd = args[0]
	}

	switch cmd {
	case "serve":
		return serve()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q (known: serve)\n", cmd)
		return 2
	}
}

func serve() int {
	server := witchcraft.NewServer().
		WithInstallConfigType(config.Install{}).
		WithRuntimeConfigType(config.Runtime{}).
		WithSelfSignedCertificate().
		WithInitFunc(initServer)

	if err := server.Start(); err != nil {
		// witchcraft already logged the structured error; signal non-zero exit.
		return 1
	}
	return 0
}

// initServer is the composition root's InitFunc (docs/architecture/overview.md). Wires the
// registration module: a Postgres pool (openfaithmap schema — migrations applied out-of-band by
// docker-compose.yml's openfaithmap-migrate, never by this binary) and the transport/application/
// adapters chain, registered onto witchcraft's router.
func initServer(ctx context.Context, info witchcraft.InitInfo) (func(), error) {
	databaseURL := requireEnv("DATABASE_URL")
	oikumeneaBaseURL := requireEnv("OIKUMENEA_BASE_URL")
	rootUnitID := requireEnv("REGISTRATION_ROOT_UNIT_ID")
	congregationAdminRoleID := requireEnv("REGISTRATION_CONGREGATION_ADMIN_ROLE_ID")
	insecureSkipVerify := os.Getenv("OIKUMENEA_INSECURE_SKIP_VERIFY") == "true"

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, werror.WrapWithContextParams(ctx, err, "dial postgres")
	}

	store := regadapters.NewStore(pool)
	appSvc := regapplication.NewService(store, regapplication.Config{
		OikumeneaBaseURL:            oikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: insecureSkipVerify,
		RootUnitID:                  rootUnitID,
		CongregationAdminRoleID:     congregationAdminRoleID,
	})
	transportSvc := regtransport.NewService(appSvc, regtransport.Config{
		OikumeneaBaseURL:            oikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: insecureSkipVerify,
	})

	if err := genregistration.RegisterRoutesRegistrationService(info.Router, transportSvc); err != nil {
		pool.Close()
		return nil, werror.WrapWithContextParams(ctx, err, "register registration routes")
	}

	contentStore := contentadapters.NewStore(pool)
	contentAppSvc := contentapplication.NewService(contentStore, contentapplication.Config{
		OikumeneaBaseURL:            oikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: insecureSkipVerify,
	})
	contentTransportSvc := contenttransport.NewService(contentAppSvc, contenttransport.Config{
		OikumeneaBaseURL:            oikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: insecureSkipVerify,
	})
	contentPublicTransportSvc := contenttransport.NewPublicService(contentAppSvc)

	if err := gencontent.RegisterRoutesContentService(info.Router, contentTransportSvc); err != nil {
		pool.Close()
		return nil, werror.WrapWithContextParams(ctx, err, "register content routes")
	}
	if err := gencontent.RegisterRoutesContentPublicService(info.Router, contentPublicTransportSvc); err != nil {
		pool.Close()
		return nil, werror.WrapWithContextParams(ctx, err, "register content public routes")
	}

	// M4: the service principal's own credentials — first production use of
	// coreintegration.NewServiceClient (M1 built it, only an integration test called it until now).
	// GOOGLE_APPLICATION_CREDENTIALS is the standard convention already used by
	// scripts/bootstrap-service-principal (.env.example); audience matches that script's own
	// registration ("openfaithmap-api", also what client_integration_test.go proves against).
	discoveryStore := discoveryadapters.NewStore(pool)
	discoveryAppSvc := discoveryapplication.NewService(discoveryStore, &contentSiteResolver{content: contentAppSvc}, discoveryapplication.Config{
		OikumeneaBaseURL:            oikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: insecureSkipVerify,
		RootUnitID:                  rootUnitID,
		ServicePrincipal: coreintegration.Config{
			BaseURL:            oikumeneaBaseURL,
			CredentialsFile:    requireEnv("GOOGLE_APPLICATION_CREDENTIALS"),
			Audience:           "openfaithmap-api",
			InsecureSkipVerify: insecureSkipVerify,
		},
	})
	discoveryTransportSvc := discoverytransport.NewService(discoveryAppSvc, discoverytransport.Config{
		OikumeneaBaseURL:            oikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: insecureSkipVerify,
	})
	discoveryPublicTransportSvc := discoverytransport.NewPublicService(discoveryAppSvc)

	if err := gendiscovery.RegisterRoutesDiscoveryService(info.Router, discoveryTransportSvc); err != nil {
		pool.Close()
		return nil, werror.WrapWithContextParams(ctx, err, "register discovery routes")
	}
	if err := gendiscovery.RegisterRoutesDiscoveryPublicService(info.Router, discoveryPublicTransportSvc); err != nil {
		pool.Close()
		return nil, werror.WrapWithContextParams(ctx, err, "register discovery public routes")
	}

	// M5: platform-moderator's own root-unit-scoped Authorize check (application/authorize.go)
	// reuses RootUnitID; CheckExclusion reuses the same service-principal credentials discovery's
	// cache refresh already wires — the caller of POST /exclusion-check is anonymous, same reason.
	moderationStore := moderationadapters.NewStore(pool)
	moderationAppSvc := moderationapplication.NewService(moderationStore, moderationapplication.Config{
		OikumeneaBaseURL:            oikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: insecureSkipVerify,
		RootUnitID:                  rootUnitID,
		ServicePrincipal: coreintegration.Config{
			BaseURL:            oikumeneaBaseURL,
			CredentialsFile:    requireEnv("GOOGLE_APPLICATION_CREDENTIALS"),
			Audience:           "openfaithmap-api",
			InsecureSkipVerify: insecureSkipVerify,
		},
	})
	moderationTransportSvc := moderationtransport.NewService(moderationAppSvc, moderationtransport.Config{
		OikumeneaBaseURL:            oikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: insecureSkipVerify,
	})
	moderationPublicTransportSvc := moderationtransport.NewPublicService(moderationAppSvc)

	if err := genmoderation.RegisterRoutesModerationService(info.Router, moderationTransportSvc); err != nil {
		pool.Close()
		return nil, werror.WrapWithContextParams(ctx, err, "register moderation routes")
	}
	// M7: an in-process, per-(client IP, endpoint) rate limiter wired onto exactly this call — the
	// only two genuinely anonymous write endpoints in the whole API (D-Hardening). Every other
	// RegisterRoutes* call in this file, including ModerationService above, is untouched.
	moderationRateLimiter := moderationtransport.NewRateLimiter()
	if err := genmoderation.RegisterRoutesModerationPublicService(
		info.Router, moderationPublicTransportSvc,
		wrouter.RouteMiddleware(moderationRateLimiter.Middleware),
	); err != nil {
		pool.Close()
		return nil, werror.WrapWithContextParams(ctx, err, "register moderation public routes")
	}

	// M6: vouching has no genuinely-anonymous endpoint (unlike content/discovery/moderation), so it
	// gets a single authenticated service, and no ServicePrincipal config at all. Its
	// moderation.read/moderation.act gates reuse the same RootUnitID as moderation's own
	// requireModerate; RevokeGuarantor's moderation-report fan-out is wired through
	// moderationVouchReporter above, an in-process call into the moderationAppSvc already
	// constructed for the moderation module.
	vouchingStore := vouchingadapters.NewStore(pool)
	vouchingAppSvc := vouchingapplication.NewService(vouchingStore, &moderationVouchReporter{moderation: moderationAppSvc}, vouchingapplication.Config{
		OikumeneaBaseURL:            oikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: insecureSkipVerify,
		RootUnitID:                  rootUnitID,
	})
	vouchingTransportSvc := vouchingtransport.NewService(vouchingAppSvc, vouchingtransport.Config{
		OikumeneaBaseURL:            oikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: insecureSkipVerify,
	})

	if err := genvouching.RegisterRoutesVouchingService(info.Router, vouchingTransportSvc); err != nil {
		pool.Close()
		return nil, werror.WrapWithContextParams(ctx, err, "register vouching routes")
	}

	// congregationimport (D-CongregationImport, docs/modules/congregationimport.md): connectors are
	// a fixed registry built here, not a plugin-discovery mechanism (application.Service's own doc
	// comment). UAEDR_UO_FILE_PATH/UAEDR_SOURCE_URL are both optional and mutually exclusive (see
	// uaedr.New) — the ЄДР connector is only registered when an operator has configured one of
	// them; omitting both leaves the module running with zero connectors registered (RunConnector
	// then returns ErrRunNotFound for any sourceCode), never a boot failure. UAEDR_SOURCE_URL
	// streams directly from the remote export (no local disk landing spot needed — a cheap,
	// memory-constrained VM deployment); the exact current data.gov.ua resource URL is an
	// operator-supplied value, never hardcoded here.
	var connectors []congregationimportdomain.Connector
	if uaedrFilePath, uaedrSourceURL := os.Getenv("UAEDR_UO_FILE_PATH"), os.Getenv("UAEDR_SOURCE_URL"); uaedrFilePath != "" || uaedrSourceURL != "" {
		uaedrConnector, err := uaedr.New(uaedrFilePath, uaedrSourceURL, nil)
		if err != nil {
			pool.Close()
			return nil, werror.WrapWithContextParams(ctx, err, "construct uaedr connector")
		}
		connectors = append(connectors, uaedrConnector)
	}
	// ar-rnc (Argentina's Registro Nacional de Cultos) — same optional/never-a-boot-failure pattern
	// as uaedr above; ARRNC_FILE_PATH/ARRNC_SOURCE_URL are both optional and mutually exclusive
	// (see arrnc.New). Unlike uaedr, this source is small (3.6MB, not ~326MB) — no HTTP-streaming
	// mode exists or is needed, see arrnc's own package doc comment.
	if arrncFilePath, arrncSourceURL := os.Getenv("ARRNC_FILE_PATH"), os.Getenv("ARRNC_SOURCE_URL"); arrncFilePath != "" || arrncSourceURL != "" {
		arrncConnector, err := arrnc.New(arrncFilePath, arrncSourceURL, nil)
		if err != nil {
			pool.Close()
			return nil, werror.WrapWithContextParams(ctx, err, "construct arrnc connector")
		}
		connectors = append(connectors, arrncConnector)
	}
	// Geocoders: a fixed registry, same shape/reasoning as the connectors slice above — Nominatim
	// is free and keyless, so it's always registered (no env-gate needed to enable it), but which
	// one actually SERVES SuggestCoordinates is still an env-driven choice
	// (CONGREGATIONIMPORT_GEOCODER, application.Config.ActiveGeocoderCode), so adding a second
	// provider (LocationIQ, Google) later — and switching to it — needs no code change here beyond
	// one more append.
	geocoders := []congregationimportdomain.Geocoder{nominatim.New(nil)}

	congregationimportStore := congregationimportadapters.NewStore(pool)
	congregationimportAppSvc := congregationimportapplication.NewService(congregationimportStore, congregationimportapplication.Config{
		OikumeneaBaseURL:            oikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: insecureSkipVerify,
		RootUnitID:                  rootUnitID,
		ServicePrincipal: coreintegration.Config{
			BaseURL:            oikumeneaBaseURL,
			CredentialsFile:    requireEnv("GOOGLE_APPLICATION_CREDENTIALS"),
			Audience:           "openfaithmap-api",
			InsecureSkipVerify: insecureSkipVerify,
		},
		ActiveGeocoderCode: os.Getenv("CONGREGATIONIMPORT_GEOCODER"),
	}, connectors, geocoders)
	congregationimportTransportSvc := congregationimporttransport.NewService(congregationimportAppSvc, congregationimporttransport.Config{
		OikumeneaBaseURL:            oikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: insecureSkipVerify,
	})

	if err := gencongregationimport.RegisterRoutesCongregationImportService(info.Router, congregationimportTransportSvc); err != nil {
		pool.Close()
		return nil, werror.WrapWithContextParams(ctx, err, "register congregationimport routes")
	}

	return pool.Close, nil
}

func requireEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		panic(fmt.Sprintf("missing required env var %s", name))
	}
	return v
}
