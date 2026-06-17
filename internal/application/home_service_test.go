package application

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"

	"ms_home/internal/domain"
	"ms_home/internal/populate"
)

func testPopulate() *populate.Service {
	return populate.NewService(populate.NewRegistry(populate.DefaultStrategies()...), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

type fakeContent struct {
	mu    sync.Mutex
	docs  map[domain.ContentType]domain.Document
	errs  map[domain.ContentType]error
	calls []domain.ContentType
}

func (f *fakeContent) GetContent(_ context.Context, ct domain.ContentType, _, _ string) (domain.Document, error) {
	f.mu.Lock()
	f.calls = append(f.calls, ct)
	f.mu.Unlock()
	if e := f.errs[ct]; e != nil {
		return nil, e
	}
	return cloneDoc(f.docs[ct]), nil
}

func (f *fakeContent) called(ct domain.ContentType) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.calls {
		if c == ct {
			return true
		}
	}
	return false
}

// cloneDoc shallow-clones top level + feature_flags so per-test mutation is isolated.
func cloneDoc(d domain.Document) domain.Document {
	if d == nil {
		return nil
	}
	out := make(domain.Document, len(d))
	for k, v := range d {
		if ff, ok := v.(map[string]any); ok {
			cp := make(map[string]any, len(ff))
			for fk, fv := range ff {
				cp[fk] = fv
			}
			out[k] = cp
			continue
		}
		out[k] = v
	}
	return out
}

func globalWith(personalization bool) domain.Document {
	return domain.Document{"feature_flags": map[string]any{"personalization": personalization}}
}

type fakeCart struct{ chd map[string]any }

func (f *fakeCart) GetCartHeaderDetails(context.Context) (map[string]any, error) { return f.chd, nil }

func TestGetHomeResolvesFavoriteStoreAndContinueBuying(t *testing.T) {
	discard := slog.New(slog.NewTextHandler(io.Discard, nil))
	page := domain.Document{
		"_content_type_uid": "page",
		"layout": map[string]any{"blocks": []any{
			map[string]any{"container": map[string]any{
				"_metadata": map[string]any{"uid": "c1"},
				"events":    []any{map[string]any{"customData": []any{map[string]any{"source": "bff", "type": "selected_store.name"}}}},
			}},
		}},
	}
	fc := &fakeContent{
		docs: map[domain.ContentType]domain.Document{
			domain.ContentTypePage:   page,
			domain.ContentTypeGlobal: globalWith(true),
		},
		errs: map[domain.ContentType]error{},
	}
	cart := &fakeCart{chd: map[string]any{
		"isLoggedIn":        true,
		"lastCartAddedItem": "https://img/last.jpg",
		"favoriteStore":     map[string]any{"id": "42", "storeName": "Centro"},
	}}
	svc := NewHomeService(fc, testPopulate(), cart, true, discard)
	ri := domain.RequestInfo{
		Source: domain.SourceWeb, LoggedIn: true, ProfileID: "u1",
		State: domain.NewRequestState(),
	}
	ctx := domain.WithRequestInfo(context.Background(), ri)

	doc, err := svc.GetHome(ctx, domain.ContentTypePage, "es-mx", "/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// selected_store.name event resolved from the favorite store.
	block := doc["blocks"].([]any)[0].(map[string]any)
	value := block["events"].([]any)[0].(map[string]any)["customData"].([]any)[0].(map[string]any)["value"]
	if value != "Centro" {
		t.Errorf("selected_store.name = %v, want Centro", value)
	}

	// continue-buying shortcut attached.
	shortcuts, ok := doc["shortcuts"].(map[string]any)
	if !ok || shortcuts["continueBuying"] == nil {
		t.Errorf("continueBuying shortcut missing: %v", doc["shortcuts"])
	}
}

func TestGetHomeNormalizesAndPopulates(t *testing.T) {
	discard := slog.New(slog.NewTextHandler(io.Discard, nil))
	page := domain.Document{
		"_content_type_uid": "page",
		"content":           "should-be-deleted",
		"layout": map[string]any{
			"blocks": []any{
				map[string]any{"container": map[string]any{"_metadata": map[string]any{"uid": "c1"}}},
				map[string]any{"countdown": map[string]any{"_metadata": map[string]any{"uid": "x"}, "is_active": false}},
			},
		},
	}
	fc := &fakeContent{
		docs: map[domain.ContentType]domain.Document{
			domain.ContentTypePage:   page,
			domain.ContentTypeGlobal: globalWith(true),
		},
		errs: map[domain.ContentType]error{},
	}
	svc := NewHomeService(fc, testPopulate(), nil, true, discard)
	ctx := domain.WithRequestInfo(context.Background(), domain.RequestInfo{Source: domain.SourceWeb})

	doc, err := svc.GetHome(ctx, domain.ContentTypePage, "es-mx", "/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := doc["content"]; ok {
		t.Error("content should be deleted")
	}
	if _, ok := doc["layout"]; ok {
		t.Error("layout should be mapped to blocks and removed")
	}
	blocks, ok := doc["blocks"].([]any)
	if !ok {
		t.Fatalf("blocks missing: %v", doc["blocks"])
	}
	// container kept (countdown inactive dropped) => 1 block.
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block after populate, got %d: %v", len(blocks), blocks)
	}
	if blocks[0].(map[string]any)["columns_mobile_small"] != 2 {
		t.Errorf("container default not applied: %v", blocks[0])
	}
}

func TestGetHomeMissingContentTypeUID(t *testing.T) {
	discard := slog.New(slog.NewTextHandler(io.Discard, nil))
	fc := &fakeContent{
		docs: map[domain.ContentType]domain.Document{
			domain.ContentTypePage:   {"title": "no-uid"},
			domain.ContentTypeGlobal: globalWith(false),
		},
		errs: map[domain.ContentType]error{},
	}
	svc := NewHomeService(fc, testPopulate(), nil, false, discard)
	ctx := domain.WithRequestInfo(context.Background(), domain.RequestInfo{Source: domain.SourceWeb})

	if _, err := svc.GetHome(ctx, domain.ContentTypePage, "es-mx", "/"); err != domain.ErrNoContentType {
		t.Fatalf("expected ErrNoContentType, got %v", err)
	}
}

func TestGetHome(t *testing.T) {
	discard := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name                string
		source              domain.Source
		ct                  domain.ContentType
		envGate             bool
		cmsPersonalization  bool
		pageErr             error
		globalErr           error
		wantErr             bool
		wantGlobalFetched   bool
		wantPersonalization bool
		wantGlobalData      bool
	}{
		{
			name: "web page env+cms on", source: domain.SourceWeb, ct: domain.ContentTypePage,
			envGate: true, cmsPersonalization: true,
			wantGlobalFetched: true, wantPersonalization: true, wantGlobalData: true,
		},
		{
			name: "web page env off forces personalization false", source: domain.SourceWeb, ct: domain.ContentTypePage,
			envGate: false, cmsPersonalization: true,
			wantGlobalFetched: true, wantPersonalization: false, wantGlobalData: true,
		},
		{
			name: "pocket screen skips global fetch", source: domain.SourcePocket, ct: domain.ContentTypeScreen,
			envGate: true, cmsPersonalization: true,
			wantGlobalFetched: false, wantGlobalData: false,
		},
		{
			name: "page fetch error is fatal", source: domain.SourceWeb, ct: domain.ContentTypePage,
			pageErr: errors.New("boom"), wantErr: true,
		},
		{
			name: "global fetch error is fatal", source: domain.SourceWeb, ct: domain.ContentTypePage,
			globalErr: errors.New("boom"), wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &fakeContent{
				docs: map[domain.ContentType]domain.Document{
					tt.ct:                    {"_content_type_uid": string(tt.ct), "title": "home"},
					domain.ContentTypeGlobal: globalWith(tt.cmsPersonalization),
				},
				errs: map[domain.ContentType]error{tt.ct: tt.pageErr, domain.ContentTypeGlobal: tt.globalErr},
			}
			svc := NewHomeService(fc, testPopulate(), nil, tt.envGate, discard)
			ctx := domain.WithRequestInfo(context.Background(), domain.RequestInfo{Source: tt.source})

			doc, err := svc.GetHome(ctx, tt.ct, "es-mx", "/")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := fc.called(domain.ContentTypeGlobal); got != tt.wantGlobalFetched {
				t.Errorf("global fetched = %v, want %v", got, tt.wantGlobalFetched)
			}
			_, hasGlobalData := doc["globalData"]
			if hasGlobalData != tt.wantGlobalData {
				t.Errorf("globalData present = %v, want %v", hasGlobalData, tt.wantGlobalData)
			}
			if tt.wantGlobalData {
				gd := doc["globalData"].(domain.Document)
				ff := gd["feature_flags"].(map[string]any)
				if ff["personalization"].(bool) != tt.wantPersonalization {
					t.Errorf("personalization = %v, want %v", ff["personalization"], tt.wantPersonalization)
				}
			}
		})
	}
}
