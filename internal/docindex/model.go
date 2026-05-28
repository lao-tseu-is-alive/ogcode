package docindex

// PageEntry represents a single indexed page in a PDF document.
type PageEntry struct {
	ID        string   `json:"id"`
	DocPath   string   `json:"docPath"`
	PageNum   int      `json:"pageNum"`
	Keywords  []string `json:"keywords"`
	Labels    []string `json:"labels"`
	IndexedAt int64    `json:"indexedAt"`
}

// DocSummary holds a document's metadata for the UI listing.
// Pages is only populated by callers that explicitly attach full page data;
// the docs listing leaves it empty to keep the payload small.
type DocSummary struct {
	DocPath   string       `json:"docPath"`
	PageCount int          `json:"pageCount"`
	Pages     []*PageEntry `json:"pages,omitempty"`
	IndexedAt int64        `json:"indexedAt"`
}
