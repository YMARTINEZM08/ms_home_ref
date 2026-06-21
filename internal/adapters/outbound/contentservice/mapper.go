package contentservice

import (
	"fmt"

	"github.com/YMARTINEZM08/ms_home_ref/internal/domain/home"
)

// contentServiceResponse is the top-level shape returned by the content-service
// proxy for a home/page entry. Blocks live at template.layout.blocks.
type contentServiceResponse struct {
	UID      string        `json:"uid"`
	Template csTemplate    `json:"template"`
}

type csTemplate struct {
	Layout csLayout `json:"layout"`
}

type csLayout struct {
	Blocks []any `json:"blocks"`
}

// layoutItems returns the ordered block list from template.layout.blocks,
// preserving the ordering from the content-service (Rule 18).
func (r *contentServiceResponse) layoutItems() []any {
	return r.Template.Layout.Blocks
}

// mapToRawBlocks converts a normalised content-service response into the
// domain's RawBlock slice. Fields not consumed here are preserved verbatim
// in RawBlock.Fields so the mapper never silently drops content.
func mapToRawBlocks(resp *contentServiceResponse) []home.RawBlock {
	items := normalize(resp.layoutItems())
	blocks := make([]home.RawBlock, 0, len(items))

	for _, item := range items {
		rb := mapItem(item)
		if rb == nil {
			continue
		}
		blocks = append(blocks, *rb)
	}
	return blocks
}

func mapItem(item map[string]any) *home.RawBlock {
	uid, _ := item["_content_type_uid"].(string)
	if uid == "" {
		return nil
	}

	// Copy all fields so the application layer has the full payload.
	fields := make(map[string]any, len(item))
	for k, v := range item {
		fields[k] = v
	}

	rb := &home.RawBlock{
		ID:             strField(item, "uid"),
		Type:           home.BlockType(uid),
		SourceOfData:   strField(item, "source_of_data"),
		AudienceFilter: strField(item, "audience_filter"),
		Handle:         strField(item, "handle"),
		FeatureFlagID:  featureFlagID(item),
		Enabled:        enabledField(item),
		Fields:         fields,
	}
	return rb
}

// featureFlagID extracts the feature flag identifier from a block. The legacy
// system stores it under several possible keys depending on the content type.
func featureFlagID(item map[string]any) string {
	for _, key := range []string{"feature_flag_id", "feature_flag", "flag_id"} {
		if v, ok := item[key].(string); ok && v != "" {
			return v
		}
	}
	// Fall back to the block UID so callers always have a stable toggle key.
	return fmt.Sprintf("block_%s", strField(item, "uid"))
}

// enabledField reads the optional "enabled" boolean from a block. Defaults
// to true if the field is absent — Contentstack entries are enabled by default.
func enabledField(item map[string]any) bool {
	v, ok := item["enabled"]
	if !ok {
		return true
	}
	b, ok := v.(bool)
	if !ok {
		return true
	}
	return b
}

func strField(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}
