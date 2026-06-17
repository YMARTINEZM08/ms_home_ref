// Package bootstrap wires adapters, services, and handlers at startup
// (compile-time DI, no reflection — skill Rule 4).
package bootstrap

import (
	"log/slog"
	"net/http"

	inhttp "ms_home/internal/adapters/inbound/http"
	"ms_home/internal/adapters/outbound/atg"
	"ms_home/internal/adapters/outbound/contentservice"
	"ms_home/internal/adapters/outbound/groupby"
	"ms_home/internal/adapters/outbound/jewel"
	"ms_home/internal/adapters/outbound/salesforce"
	"ms_home/internal/application"
	"ms_home/internal/config"
	"ms_home/internal/observability"
	"ms_home/internal/populate"
	"ms_home/internal/ports"
	"ms_home/pkg/httpclient"
)

// App holds the constructed, ready-to-serve components.
type App struct {
	Router *http.ServeMux
	Logger *slog.Logger
}

// New constructs the dependency graph from configuration.
func New(cfg config.Config) *App {
	logger := observability.NewLogger(cfg.LogLevel)

	csClient := httpclient.New(cfg.ContentService.Timeout, logger)
	contentAdapter := contentservice.New(csClient, cfg.ContentService.BaseURL)

	strategies := populate.DefaultStrategies()
	if cfg.GroupBy.SearchURL != "" {
		gbClient := httpclient.New(cfg.GroupBy.Timeout, logger)
		gbSearch := groupby.NewSearch(gbClient, cfg.GroupBy.SearchURL)
		strategies = append(strategies, populate.NewProductListGroupBy(gbSearch))
	}
	if cfg.GroupBy.RecommendationsURL != "" {
		gbClient := httpclient.New(cfg.GroupBy.Timeout, logger)
		gbRec := groupby.NewRecommendations(gbClient, cfg.GroupBy.RecommendationsURL)
		strategies = append(strategies, populate.NewProductListRecentlyViewed(gbRec))
	}
	if cfg.Jewel.URL != "" {
		jwClient := httpclient.New(cfg.Jewel.Timeout, logger)
		jwAdapter := jewel.New(jwClient, cfg.Jewel.URL)
		strategies = append(strategies, populate.NewProductListJewel(jwAdapter))
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
	handler := inhttp.NewHandler(homeService, cfg.DefaultBrand, logger)

	return &App{
		Router: inhttp.NewRouter(handler),
		Logger: logger,
	}
}
