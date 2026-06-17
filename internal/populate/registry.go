package populate

// Registry holds the ordered strategy set (port of PopulateRegistry).
type Registry struct {
	strategies []Strategy
}

// NewRegistry builds a registry from the given strategies.
func NewRegistry(strategies ...Strategy) *Registry {
	return &Registry{strategies: strategies}
}

// GetStrategies returns the strategies that support the block.
func (r *Registry) GetStrategies(block Block) []Strategy {
	var out []Strategy
	for _, s := range r.strategies {
		if s.Supports(block) {
			out = append(out, s)
		}
	}
	return out
}
