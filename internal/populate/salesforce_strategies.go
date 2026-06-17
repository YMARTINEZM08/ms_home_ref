package populate

import (
	"context"

	"ms_home/internal/domain"
	"ms_home/internal/ports"
	"ms_home/internal/product"
)

// Salesforce action keys (salesforce.constant.ts).
const (
	sfCanLike          = "CMSTePuedeGustar"
	sfIncredibleOffers = "CMSOfertasIncreibles"
	sfBirthday         = "CMSCumpleanios"
)

// ContainerGreeting — port of ContainerGreetingPopulateStrategy. Personalization
// gated; birthday greetings require Salesforce + a matching campaign.
type ContainerGreeting struct {
	sf ports.SalesforcePort // may be nil when Salesforce is not configured
}

// NewContainerGreeting builds the strategy. sf may be nil (birthday path then drops).
func NewContainerGreeting(sf ports.SalesforcePort) ContainerGreeting {
	return ContainerGreeting{sf: sf}
}

func (ContainerGreeting) Supports(b Block) bool {
	return b["_content_type_uid"] == "container_greeting"
}

func (s ContainerGreeting) Populate(ctx context.Context, b Block) (Block, error) {
	ri := domain.RequestInfoFromContext(ctx)
	if !ri.Flag("personalization") {
		return nil, nil
	}
	isBirthday := boolDefault(b["is_birthday"], false)
	onlyLogged := boolDefault(b["only_logged"], false)

	if isBirthday && !ri.Flag("salesforce") {
		return nil, nil
	}
	if (isBirthday || onlyLogged) && !ri.LoggedIn {
		return nil, nil
	}

	if isBirthday {
		if s.sf == nil {
			return nil, nil
		}
		data, err := s.sf.GetActionFromUser(ctx, sfBirthday)
		if err != nil {
			return nil, err
		}
		b["salesforce"] = data
		if !hasBirthdayCampaign(data) {
			return nil, nil
		}
	}
	return b, nil
}

func hasBirthdayCampaign(data map[string]any) bool {
	for _, c := range sliceAt(data, "campaignResponses") {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if birthday, _ := mapAt(cm, "payload")["birthday"].(bool); birthday {
			return true
		}
	}
	return false
}

// ProductsCards — port of ProductsCardsPopulateStrategy (INCREDIBLE_OFFERS).
type ProductsCards struct{ sf ports.SalesforcePort }

func NewProductsCards(sf ports.SalesforcePort) ProductsCards { return ProductsCards{sf: sf} }

func (ProductsCards) Supports(b Block) bool { return b["_content_type_uid"] == "products_cards" }

func (s ProductsCards) Populate(ctx context.Context, b Block) (Block, error) {
	ri := domain.RequestInfoFromContext(ctx)
	if !ri.Flag("salesforce") || !ri.LoggedIn {
		return nil, nil
	}
	data, err := s.sf.GetActionFromUser(ctx, sfIncredibleOffers)
	if err != nil {
		return nil, err
	}
	b["incredibleOffersData"] = data
	return b, nil
}

// RecommendationProductList — port of RecommendationProductListPopulateStrategy (CAN_LIKE).
type RecommendationProductList struct{ sf ports.SalesforcePort }

func NewRecommendationProductList(sf ports.SalesforcePort) RecommendationProductList {
	return RecommendationProductList{sf: sf}
}

func (RecommendationProductList) Supports(b Block) bool {
	return b["_content_type_uid"] == "recommendation_product_list"
}

func (s RecommendationProductList) Populate(ctx context.Context, b Block) (Block, error) {
	ri := domain.RequestInfoFromContext(ctx)
	if !ri.Flag("salesforce") || !ri.LoggedIn {
		return nil, nil
	}
	data, err := s.sf.GetActionFromUser(ctx, sfCanLike)
	if err != nil {
		return nil, err
	}
	b["whishlistData"] = data
	return b, nil
}

// ProductListSalesforce — port of ProductListSalesforcePopulateStrategy.
type ProductListSalesforce struct{ sf ports.SalesforcePort }

func NewProductListSalesforce(sf ports.SalesforcePort) ProductListSalesforce {
	return ProductListSalesforce{sf: sf}
}

func (ProductListSalesforce) Supports(b Block) bool {
	return b["_content_type_uid"] == "products_list" && b["source_of_data"] == "salesforce"
}

func (s ProductListSalesforce) Populate(ctx context.Context, b Block) (Block, error) {
	ri := domain.RequestInfoFromContext(ctx)
	if !ri.Flag("salesforce") || !salesforceShouldPopulate(b, ri) {
		return nil, nil
	}

	resp, err := s.sf.GetActionFromUser(ctx, str(b["salesforce_carousel"]))
	if err != nil {
		return nil, err
	}
	payload := mapAt(firstMap(sliceAt(resp, "campaignResponses")), "payload")
	products := sliceAt(payload, "products")

	if len(products) < atoi(str(b["min_of_products"])) {
		return nil, nil
	}
	if max := atoi(str(b["max_of_products"])); max > 0 && len(products) > max {
		products = products[:max]
	}

	b["productsListId"] = salesforceCampaignName(b)
	mapped := make([]any, 0, len(products))
	for i, p := range products {
		pm, _ := p.(map[string]any)
		dto := product.FromSalesfroce(pm)
		dto.Index = i
		mapped = append(mapped, dto)
	}
	b["products"] = mapped
	if title := str(payload["title"]); title != "" {
		b["products_list_title"] = title
	}
	return b, nil
}

// salesforceShouldPopulate gates on audience + login + surface toggle (requires login).
func salesforceShouldPopulate(b Block, ri domain.RequestInfo) bool {
	switch str(b["audience_filter"]) {
	case "logged":
		if !ri.LoggedIn {
			return false
		}
	case "guest":
		if ri.LoggedIn {
			return false
		}
	}
	if !ri.LoggedIn {
		return false
	}
	if ri.Source == domain.SourcePocket {
		return boolDefault(b["enable_on_apps"], true)
	}
	return boolDefault(b["enable_on_web"], true)
}

// salesforceCampaignName returns block.records.campaignResponses[0].campaignName ?? "".
func salesforceCampaignName(b Block) string {
	records := mapAt(b, "records")
	first := firstMap(sliceAt(records, "campaignResponses"))
	return str(first["campaignName"])
}
