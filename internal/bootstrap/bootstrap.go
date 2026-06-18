// Package bootstrap wires adapters, services, and handlers at startup
// (compile-time DI, no reflection — skill Rule 4).
package bootstrap

import (
	"context"
	"log/slog"
	"net/http"

	inhttp "ms_home/internal/adapters/inbound/http"
	"ms_home/internal/adapters/outbound/atg"
	"ms_home/internal/adapters/outbound/contentservice"
	"ms_home/internal/adapters/outbound/groupby"
	"ms_home/internal/adapters/outbound/jewel"
	"ms_home/internal/adapters/outbound/salesforce"
	"ms_home/internal/adapters/outbound/searchfacade"
	"ms_home/internal/application"
	"ms_home/internal/auth"
	"ms_home/internal/config"
	"ms_home/internal/observability"
	"ms_home/internal/populate"
	"ms_home/internal/ports"
	"ms_home/pkg/httpclient"
)

// App holds the constructed, ready-to-serve components.
type App struct {
	Router   http.Handler
	Logger   *slog.Logger
	Shutdown func(context.Context) error // flushes tracing on exit
}

// New constructs the dependency graph from configuration.
func New(cfg config.Config) *App {
	logger := observability.NewLogger(cfg.LogLevel)

	tracingShutdown, err := observability.InitTracing(
		context.Background(), cfg.Tracing.ServiceName, cfg.Version, cfg.Env, cfg.Tracing.Enabled)
	if err != nil {
		logger.Error("tracing init failed", "error", err.Error())
		tracingShutdown = func(context.Context) error { return nil }
	}

	csClient := httpclient.New(cfg.ContentService.Timeout, logger)
	contentAdapter := contentservice.New(csClient, cfg.ContentService.BaseURL)

	strategies := populate.DefaultStrategies()
	if cfg.GroupBy.SearchURL != "" {
		gbClient := httpclient.New(cfg.GroupBy.Timeout, logger)
		gbSearch := groupby.NewSearch(gbClient, cfg.GroupBy.SearchURL)
		strategies = append(strategies, populate.NewProductListGroupBy(gbSearch))
	}
	var gbRec ports.GroupByRecommendationsPort
	if cfg.GroupBy.RecommendationsURL != "" {
		gbClient := httpclient.New(cfg.GroupBy.Timeout, logger)
		gbRec = groupby.NewRecommendations(gbClient, cfg.GroupBy.RecommendationsURL)
		strategies = append(strategies, populate.NewProductListRecentlyViewed(gbRec))
	}
	if cfg.Jewel.URL != "" {
		jwClient := httpclient.New(cfg.Jewel.Timeout, logger)
		jwAdapter := jewel.New(jwClient, cfg.Jewel.URL)
		strategies = append(strategies, populate.NewProductListJewel(jwAdapter))
	}
	if cfg.SearchFacade.BaseURL != "" {
		sfClient := httpclient.New(cfg.SearchFacade.Timeout, logger)
		multiProduct := searchfacade.NewMultiProduct(sfClient, cfg.SearchFacade.BaseURL)
		strategies = append(strategies, populate.NewBannerProducts(multiProduct, gbRec))
	}

	// Salesforce: greeting is always registered (its non-birthday path needs no
	// Salesforce); the birthday path and the other Salesforce strategies require it.
	var sfPort ports.SalesforcePort
	if cfg.Salesforce.URL != "" {
		sfClient := httpclient.New(cfg.Salesforce.Timeout, logger)
		sfPort = salesforce.New(sfClient, cfg.Salesforce.URL)
	}
	strategies = append(strategies, populate.NewContainerGreeting(sfPort))
	if sfPort != nil {
		strategies = append(strategies,
			populate.NewProductsCards(sfPort),
			populate.NewRecommendationProductList(sfPort),
			populate.NewProductListSalesforce(sfPort),
		)
	}

	populateSvc := populate.NewService(populate.NewRegistry(strategies...), logger)

	var cartHeader ports.CartHeaderPort
	if cfg.ATG.CartHeaderURL != "" {
		atgClient := httpclient.New(cfg.ATG.Timeout, logger)
		cartHeader = atg.NewCartHeader(atgClient, cfg.ATG.CartHeaderURL)
	}

	homeService := application.NewHomeService(contentAdapter, populateSvc, cartHeader, cfg.PersonalizationEnabled, logger)

	// Auth mode by config: opaque cookie exchange (digital_bff parity) > local JWT >
	// dev (x-profile-id header). nil interface = dev mode.
	var authn inhttp.Authenticator
	switch {
	case cfg.Auth.OpaqueExchangeURL != "":
		exClient := httpclient.New(cfg.Auth.Timeout, logger)
		exchange := auth.NewExchange(exClient, cfg.Auth.OpaqueExchangeURL, cfg.Auth.CookieName)
		authn = inhttp.NewOpaqueAuthenticator(exchange, cfg.Auth.CookieName, cfg.DefaultBrand, logger)
	case cfg.Auth.JWKSURL != "":
		verifier := auth.NewVerifier(cfg.Auth.JWKSURL, cfg.Auth.Issuer, cfg.Auth.Audience, cfg.Auth.Timeout)
		authn = inhttp.NewJWTAuthenticator(verifier, cfg.Auth.ProfileClaim, logger)
	}
	handler := inhttp.NewHandler(homeService, authn, cfg.DefaultBrand, logger)

	return &App{
		Router:   inhttp.NewRouter(handler, cfg.Version),
		Logger:   logger,
		Shutdown: tracingShutdown,
	}
}
