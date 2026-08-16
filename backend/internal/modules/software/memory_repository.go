package software

import (
	"context"
	"sort"
	"strings"
	"sync"
)

type MemoryRepository struct {
	mu       sync.RWMutex
	products map[string]Product
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{products: make(map[string]Product)}
}

func (r *MemoryRepository) List(_ context.Context) ([]Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	products := make([]Product, 0, len(r.products))
	for _, product := range r.products {
		products = append(products, product)
	}
	sort.Slice(products, func(i, j int) bool {
		if products[i].Name == products[j].Name {
			return products[i].Publisher < products[j].Publisher
		}
		return products[i].Name < products[j].Name
	})
	return products, nil
}

func (r *MemoryRepository) Create(_ context.Context, product Product) (Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.identityExists(product, "") {
		return Product{}, ErrAlreadyExists
	}
	r.products[product.ID] = product
	return product, nil
}

func (r *MemoryRepository) Update(_ context.Context, product Product) (Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.products[product.ID]
	if !exists {
		return Product{}, ErrNotFound
	}
	if r.identityExists(product, product.ID) {
		return Product{}, ErrAlreadyExists
	}
	product.CreatedAt = existing.CreatedAt
	r.products[product.ID] = product
	return product, nil
}

func (r *MemoryRepository) identityExists(candidate Product, ignoredID string) bool {
	for id, product := range r.products {
		if id == ignoredID {
			continue
		}
		if strings.EqualFold(product.Name, candidate.Name) &&
			strings.EqualFold(product.Publisher, candidate.Publisher) &&
			strings.EqualFold(product.Version, candidate.Version) {
			return true
		}
	}
	return false
}
