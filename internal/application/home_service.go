// Package application orchestrates the HOME use case. It is the Go port of
// digital_bff ContentService.getContent/fetchData and depends only on ports.
package application

import (
	"context"
	"log/slog"
	"strconv"
	"sync"

	"ms_home/internal/content"
	"ms_home/internal/domain"
	"ms_home/internal/populate"
	"ms_home/internal/ports"
)

// HomeService composes a HOME page from the Content Service proxy.
type HomeService struct {
	content                ports.ContentPort
	populate               *populate.Service
	cartHeader             ports.CartHeaderPort // may be nil when ATG is not configured
	log                    *slog.Logger
	personalizationEnabled bool // environment gate (rule #3)
}

// NewHomeService wires the service. personalizationEnabled is the env gate that,
// ANDed with the CMS feature flag, yields effective personalization. cartHeader may
// be nil (favorite store + continue-buying then unavailable).
func NewHomeService(contentPort ports.ContentPort, pop *populate.Service, cartHeader ports.CartHeaderPort, personalizationEnabled bool, log *slog.Logger) *HomeService {
	return &HomeService{content: contentPort, populate: pop, cartHeader: cartHeader, log: log, personalizationEnabled: personalizationEnabled}
}

// GetHome fetches the page (and GLOBAL for non-pocket surfaces) in parallel,
// merges feature flags, normalizes the template, applies content-type gating, and
// populates blocks. Port of ContentService.getContent.
//
// Preserved business rules (content.service.ts):
//   - #2 parallel fetch; page/global rejection is fatal.
//   - #3 personalization = envGate AND cms flag.
//   - #4 normalization: drop template.content, map layout/top_layout/bottom_layout.
//   - #5 content-type gating + rename/delete keys.
//   - #6 parallel populate of blocks/top_content/bottom_content/products, drop-on-failure.
//
// TODO(phase-1b/2): category data (Category Indexer), custom-data events,
// legacy Android welcome container, and the external-provider strategies.
func (s *HomeService) GetHome(ctx context.Context, ct domain.ContentType, locale, id string) (domain.Document, error) {
	page, err := s.fetch(ctx, ct, locale, id)
	if err != nil {
		return nil, err
	}

	// Make effective feature flags available to populate strategies via context.
	ctx = withFlags(ctx, page)

	// Resolve the favorite store (ATG) into request state before populate so
	// selected_store events and store-aware strategies can use it.
	s.loadSession(ctx)

	// #5 returned verbatim for some content types.
	if content.ReturnWithoutChanges[ct] {
		return page, nil
	}

	// The template is response.template when present, otherwise the response itself.
	template := page
	if t, ok := page["template"].(map[string]any); ok {
		template = t
	}

	// #4 normalization.
	delete(template, "content")
	for layoutKey, outKey := range content.AvailableLayouts {
		wrapper, ok := template[layoutKey].(map[string]any)
		if !ok {
			continue
		}
		blocks, _ := wrapper["blocks"].([]any)
		template[outKey] = content.NormalizeDoubleBlocks(blocks)
		delete(template, layoutKey)
	}

	// #5 error checks.
	if content.TemplatesWithUid[ct] {
		if _, ok := template["_content_type_uid"]; !ok {
			return nil, domain.ErrNoContentType
		}
	}
	if content.NeedCategoryID[ct] {
		if _, ok := page["category_id"]; !ok {
			return nil, domain.ErrNoCategoryID
		}
	}

	// #5 per-content-type key rename/delete.
	content.RenameKeys(template, content.RenameKeysFrom[ct])
	content.DeleteKeys(template, content.DeleteKeysFrom[ct])

	// #6 populate block containers in parallel, then assign (maps are not safe for
	// concurrent writes, so results are gathered before assignment).
	s.populateContainers(ctx, template, page)

	// #7 template-level custom-data events (personalization only).
	s.processTemplateEvents(ctx, template)

	// #8 legacy Android welcome container: injected into screen blocks for logged-in
	// users, after container_shortcuts.
	s.injectLegacyWelcome(ctx, ct, template)

	// #9 web personalization merge (shortcuts). `me` + cart-backed shortcuts deferred.
	s.attachWebShortcuts(ctx, ct, page)

	return page, nil
}

// fetch performs the parallel page + GLOBAL retrieval and the flag merge.
// Port of ContentService.fetchData (HOME-relevant path).
func (s *HomeService) fetch(ctx context.Context, ct domain.ContentType, locale, id string) (domain.Document, error) {
	ri := domain.RequestInfoFromContext(ctx)
	fetchGlobal := ri.Source != domain.SourcePocket && ct != domain.ContentTypeGlobal

	var (
		page, global  domain.Document
		pageErr, gErr error
		wg            sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		page, pageErr = s.content.GetContent(ctx, ct, locale, id)
	}()

	if fetchGlobal {
		wg.Add(1)
		go func() {
			defer wg.Done()
			global, gErr = s.content.GetContent(ctx, domain.ContentTypeGlobal, locale, "")
		}()
	}
	wg.Wait()

	if pageErr != nil {
		return nil, pageErr
	}
	if gErr != nil {
		return nil, gErr
	}

	if global != nil {
		s.mergeFeatureFlags(global)
		page["globalData"] = global
	}
	return page, nil
}

// populateContainers populates blocks/top_content/bottom_content (on template) and
// products (on the response) concurrently, then assigns the results sequentially.
func (s *HomeService) populateContainers(ctx context.Context, template, page domain.Document) {
	type target struct {
		wrapper map[string]any
		key     string
	}
	targets := []target{
		{template, "blocks"},
		{template, "top_content"},
		{template, "bottom_content"},
		{page, "products"},
	}

	results := make([][]any, len(targets))
	present := make([]bool, len(targets))
	var wg sync.WaitGroup
	for i, t := range targets {
		blocks, ok := t.wrapper[t.key].([]any)
		if !ok {
			continue
		}
		present[i] = true
		wg.Add(1)
		go func(i int, blocks []any) {
			defer wg.Done()
			results[i] = s.populate.PopulateAll(ctx, blocks)
		}(i, blocks)
	}
	wg.Wait()

	for i, t := range targets {
		if present[i] {
			t.wrapper[t.key] = results[i]
		}
	}
}

// withFlags extracts the effective feature flags (already gated) from the merged
// globalData and stores them on the context for populate strategies.
func withFlags(ctx context.Context, page domain.Document) context.Context {
	gd, ok := page["globalData"].(map[string]any)
	if !ok {
		return ctx
	}
	ff, ok := gd["feature_flags"].(map[string]any)
	if !ok {
		return ctx
	}
	flags := make(map[string]bool, len(ff))
	for k, v := range ff {
		if b, ok := v.(bool); ok {
			flags[k] = b
		}
	}
	ri := domain.RequestInfoFromContext(ctx)
	ri.FeatureFlags = flags
	return domain.WithRequestInfo(ctx, ri)
}

// processTemplateEvents fills BFF custom-data events at the template level when
// personalization is on (port of content.service.ts lines 137-148).
func (s *HomeService) processTemplateEvents(ctx context.Context, template domain.Document) {
	ri := domain.RequestInfoFromContext(ctx)
	if !ri.Flag("personalization") {
		return
	}
	ev, ok := template["events"]
	if !ok {
		return
	}
	if nonEmptyEvents(ev) {
		s.populate.ProcessEvents(ctx, []any{map[string]any{"events": ev}})
	} else {
		template["events"] = []any{}
	}
}

func nonEmptyEvents(ev any) bool {
	switch v := ev.(type) {
	case map[string]any:
		return len(v) > 0
	case []any:
		return len(v) > 0
	default:
		return false
	}
}

// attachWebShortcuts attaches the web personalization shortcuts block (web/page only,
// personalization on). Only the self-contained shopping-assistant shortcut is built;
// `me` and cart-backed shortcuts (continue-buying) are deferred to Phase 2c.
func (s *HomeService) attachWebShortcuts(ctx context.Context, ct domain.ContentType, page domain.Document) {
	if ct != domain.ContentTypePage {
		return
	}
	ri := domain.RequestInfoFromContext(ctx)
	if !ri.Flag("personalization") {
		return
	}

	// `me` from the memoized cart header overlaid with verified JWT claims.
	if ri.State != nil {
		if chd, ok := ri.State.CartHeader(); ok {
			page["me"] = projectMe(chd, ri.Claims)
		}
	}

	shortcuts := map[string]any{}
	if cb := continueBuyingShortcut(ri); cb != nil {
		shortcuts["continueBuying"] = cb
	}
	if sa := shoppingAssistantShortcut(ri); sa != nil {
		shortcuts["shoppingAssistant"] = sa
	}
	if len(shortcuts) > 0 {
		page["shortcuts"] = shortcuts
	}
}

// loadSession fetches the ATG cart header once per request (personalization only),
// memoizes it, and resolves the favorite store into request state.
// Port of MiddlewareService.getFavoriteStore (non-decommission path).
func (s *HomeService) loadSession(ctx context.Context) {
	if s.cartHeader == nil {
		return
	}
	ri := domain.RequestInfoFromContext(ctx)
	// personalization needs the cart header (continue-buying, me); groupby needs the
	// favorite store for banner_products multi-product pricing.
	if ri.State == nil || (!ri.Flag("personalization") && !ri.Flag("groupby")) {
		return
	}
	chd, err := s.cartHeader.GetCartHeaderDetails(ctx)
	if err != nil {
		s.log.DebugContext(ctx, "cart header fetch failed", slog.String("error", err.Error()))
		return // tolerated (favorite-store fetch is non-fatal)
	}
	ri.State.SetCartHeader(chd)

	if fav, ok := chd["favoriteStore"].(map[string]any); ok {
		id, _ := fav["id"].(string)
		name, _ := fav["storeName"].(string)
		if id != "" {
			n, _ := strconv.Atoi(id)
			ri.State.SetSelectedStore(&domain.SelectedStore{ID: n, Name: name})
		}
	}
}

// injectLegacyWelcome inserts the legacy container_welcome block into screen blocks
// for logged-in users with personalization on, positioned after container_shortcuts.
func (s *HomeService) injectLegacyWelcome(ctx context.Context, ct domain.ContentType, template domain.Document) {
	if ct != domain.ContentTypeScreen {
		return
	}
	ri := domain.RequestInfoFromContext(ctx)
	if !ri.Flag("personalization") || !ri.LoggedIn {
		return
	}
	blocks, ok := template["blocks"].([]any)
	if !ok {
		return
	}

	welcome := content.LegacyWelcomeContainer(blocks)
	insertAt := 1
	for i, b := range blocks {
		if m, ok := b.(map[string]any); ok && m["_content_type_uid"] == "container_shortcuts" {
			insertAt = i + 1
			break
		}
	}
	if insertAt > len(blocks) {
		insertAt = len(blocks)
	}

	out := make([]any, 0, len(blocks)+1)
	out = append(out, blocks[:insertAt]...)
	out = append(out, welcome)
	out = append(out, blocks[insertAt:]...)
	template["blocks"] = out
}

// mergeFeatureFlags forces personalization off unless the environment gate is on,
// otherwise honoring the CMS value (rule #3).
func (s *HomeService) mergeFeatureFlags(global domain.Document) {
	ff, ok := global["feature_flags"].(map[string]any)
	if !ok {
		return
	}
	cmsPersonalization, _ := ff["personalization"].(bool)
	ff["personalization"] = s.personalizationEnabled && cmsPersonalization
}
