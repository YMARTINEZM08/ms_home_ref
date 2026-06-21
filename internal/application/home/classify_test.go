package home_test

import (
	"testing"

	apphome "github.com/YMARTINEZM08/ms_home_ref/internal/application/home"
	domain "github.com/YMARTINEZM08/ms_home_ref/internal/domain/home"
)

// classifyExported wraps the unexported classify via the exported Service
// path by exercising GetLayout end-to-end with a stub ContentPort.
// Classification logic is tested here via table tests against classify directly
// using a test-only export shim.
func TestClassify_StaticBlock(t *testing.T) {
	_ = apphome.NewService(nil, nil) // ensure package compiles

	tests := []struct {
		name     string
		raw      domain.RawBlock
		wantKind domain.BlockKind
	}{
		{
			name:     "banner is static",
			raw:      domain.RawBlock{ID: "b1", Type: domain.BlockTypeBanner},
			wantKind: domain.KindStatic,
		},
		{
			name:     "hero_banner is static",
			raw:      domain.RawBlock{ID: "b2", Type: domain.BlockTypeHeroBanner},
			wantKind: domain.KindStatic,
		},
		{
			name:     "products_list is dynamic",
			raw:      domain.RawBlock{ID: "b3", Type: domain.BlockTypeProductsList, Enabled: true},
			wantKind: domain.KindDynamic,
		},
		{
			name:     "groupby source_of_data forces dynamic",
			raw:      domain.RawBlock{ID: "b4", Type: domain.BlockTypeBanner, SourceOfData: "groupby"},
			wantKind: domain.KindDynamic,
		},
		{
			name:     "client-side handle forces dynamic",
			raw:      domain.RawBlock{ID: "b5", Type: domain.BlockTypeBanner, Handle: "client-side"},
			wantKind: domain.KindDynamic,
		},
		{
			name:     "salesforce source forces dynamic",
			raw:      domain.RawBlock{ID: "b6", Type: domain.BlockTypeCarousel, SourceOfData: "salesforce"},
			wantKind: domain.KindDynamic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := apphome.ClassifyForTest(tt.raw)
			if block.Kind != tt.wantKind {
				t.Errorf("classify(%q) kind = %q, want %q", tt.raw.Type, block.Kind, tt.wantKind)
			}
		})
	}
}

func TestClassify_DynamicBlock_Placeholder(t *testing.T) {
	raw := domain.RawBlock{
		ID:            "p1",
		Type:          domain.BlockTypeProductsList,
		FeatureFlagID: "flag-products",
		Enabled:       true,
		Fields:        map[string]any{"fallback": "skeleton"},
	}
	block := apphome.ClassifyForTest(raw)

	if block.Kind != domain.KindDynamic {
		t.Fatal("expected dynamic block")
	}
	d := block.Dynamic
	if d.ResolveEndpoint != "/home/blocks/products_list" {
		t.Errorf("ResolveEndpoint = %q, want /home/blocks/products_list", d.ResolveEndpoint)
	}
	if d.Fallback != "skeleton" {
		t.Errorf("Fallback = %q, want skeleton", d.Fallback)
	}
	if d.FeatureFlagID != "flag-products" {
		t.Errorf("FeatureFlagID = %q, want flag-products", d.FeatureFlagID)
	}
	if !d.Enabled {
		t.Error("Enabled should be true")
	}
}

func TestClassify_DisabledDynamicBlock(t *testing.T) {
	raw := domain.RawBlock{
		ID:      "d1",
		Type:    domain.BlockTypeGreeting,
		Enabled: false,
	}
	block := apphome.ClassifyForTest(raw)

	if block.Kind != domain.KindDynamic {
		t.Fatal("disabled block should still be dynamic (placeholder)")
	}
	if block.Dynamic.Enabled {
		t.Error("Enabled should be false for a disabled block")
	}
}
