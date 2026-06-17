package content

import "ms_home/internal/domain"

// Faithful port of libs/providers/src/constant/content.constant.ts.

// ReturnWithoutChanges: content types returned verbatim (no template processing).
var ReturnWithoutChanges = map[domain.ContentType]bool{
	domain.ContentTypeScreenServices: true,
}

// TemplatesWithUid: must carry _content_type_uid after normalization.
var TemplatesWithUid = map[domain.ContentType]bool{
	domain.ContentTypePage:   true,
	domain.ContentTypeScreen: true,
}

// NeedCategoryID: must carry category_id in the response.
var NeedCategoryID = map[domain.ContentType]bool{
	domain.ContentTypePageBLP: true,
}

// RenameKeysFrom: per-content-type key renames applied to the template.
var RenameKeysFrom = map[domain.ContentType]map[string]string{
	domain.ContentTypeScreenBLP: {
		"apps_top_content": "top_content",
		"apps_products":    "products",
		"apps_article":     "article",
	},
}

// DeleteKeysFrom: per-content-type keys removed from the template.
var DeleteKeysFrom = map[domain.ContentType][]string{
	domain.ContentTypePageBLP:   {"apps_top_content", "apps_products", "apps_article"},
	domain.ContentTypeScreenBLP: {"seo", "header", "footer", "url", "blocks"},
}

// AvailableLayouts maps CMS layout wrappers to normalized output keys.
var AvailableLayouts = map[string]string{
	"layout":        "blocks",
	"top_layout":    "top_content",
	"bottom_layout": "bottom_content",
}
