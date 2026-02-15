package adapters

// Document 向量检索用的文档
type Document struct {
	PageContent string
	Metadata    map[string]any
}
