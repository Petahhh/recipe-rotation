package recipes

// Recipe is a persisted recipe row.
type Recipe struct {
	ID          int64
	Name        string
	Link        string
	Ingredients string
}
