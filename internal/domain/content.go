// Package domain holds pure business types. It must not import infrastructure
// (no HTTP, no SDKs) — dependencies point inward (skill Rule 1).
package domain

// ContentType identifies a Contentstack content type as exposed by the
// Content Service proxy. Mirrors digital_bff ContentType enum (HOME-relevant subset).
type ContentType string

const (
	ContentTypePage           ContentType = "page"            // web HOME
	ContentTypeScreen         ContentType = "screen"          // pocket HOME
	ContentTypeGlobal         ContentType = "global"          // navigation, footer, feature flags
	ContentTypePageBLP        ContentType = "page-blp"        // web brand listing page
	ContentTypeScreenBLP      ContentType = "screen-blp"      // pocket brand listing page
	ContentTypeScreenServices ContentType = "screen-services" // pocket services screen
)

// IsScreen reports whether the content type belongs to the pocket (mobile) surface.
func (c ContentType) IsScreen() bool {
	return len(c) >= 6 && c[:6] == "screen"
}

// Document is a decoded CMS entry/page. Content is highly dynamic, so it is kept
// as a generic map rather than a rigid struct (preserves arbitrary CMS fields).
type Document = map[string]any

// Block is a single content block within a page/screen. Same dynamic shape as Document.
type Block = map[string]any

// Source identifies the calling surface, mirroring digital_bff SOURCE_BFF.
type Source string

const (
	SourceWeb    Source = "WEB"
	SourcePocket Source = "POCKET"
	SourceCSC    Source = "CSC"
)
