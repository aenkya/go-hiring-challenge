package catalog

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/mytheresa/go-hiring-challenge/models"
)

type GetProductsResponse struct {
	Products []Product `json:"products"`
	Total    int64     `json:"total"`
}

type Product struct {
	Code     string   `json:"code"`
	Price    float64  `json:"price"`
	Category Category `json:"category"`
}

type Category struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type CatalogHandler struct {
	repo models.ProductFetcher
}

type ProductDetailResponse struct {
	Code     string          `json:"code"`
	Price    float64         `json:"price"`
	Category Category        `json:"category"`
	Variants []VariantDetail `json:"variants"`
}

type VariantDetail struct {
	Name  string  `json:"name"`
	SKU   string  `json:"sku"`
	Price float64 `json:"price"`
}

func NewCatalogHandler(r models.ProductFetcher) *CatalogHandler {
	return &CatalogHandler{
		repo: r,
	}
}

func (h *CatalogHandler) HandleGetProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	offset, limit := 0, 10

	if o := q.Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil {
			offset = n
		}
	}

	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			if n < 1 {
				n = 1
			} else if n > 100 {
				n = 100
			}

			limit = n
		}
	}

	filters := models.ProductFilters{
		Category: q.Get("category"),
	}

	if p := q.Get("priceLessThan"); p != "" {
		if price, err := strconv.ParseFloat(p, 64); err == nil {
			filters.PriceLT = &price
		}
	}

	total, err := h.repo.CountProducts(filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	res, err := h.repo.GetAllProductsWithPagination(offset, limit, filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Map response
	products := make([]Product, len(res))
	for i, p := range res {
		products[i] = Product{
			Code:  p.Code,
			Price: p.Price.InexactFloat64(),
			Category: Category{
				Code: p.Category.Code,
				Name: p.Category.Name,
			},
		}
	}

	// Return the products as a JSON response
	w.Header().Set("Content-Type", "application/json")

	response := GetProductsResponse{
		Products: products,
		Total:    total,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *CatalogHandler) HandleGetProduct(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Path[len("/catalog/"):]
	product, err := h.repo.GetProductByCode(code)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if product == nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	variants := make([]VariantDetail, len(product.Variants))
	prodPrice := product.Price.InexactFloat64()
	for i, v := range product.Variants {
		price := v.Price.InexactFloat64()
		if v.Price.IsZero() {
			price = prodPrice
		}

		variants[i] = VariantDetail{
			Name:  v.Name,
			SKU:   v.SKU,
			Price: price,
		}
	}

	resp := ProductDetailResponse{
		Code:     product.Code,
		Price:    prodPrice,
		Category: Category{Code: product.Category.Code, Name: product.Category.Name},
		Variants: variants,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *CatalogHandler) HandleGetCategories(w http.ResponseWriter, r *http.Request) {
	res, err := h.repo.GetAllCategories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	categories := make([]Category, len(res))
	for i, c := range res {
		categories[i] = Category{
			Code: c.Code,
			Name: c.Name,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(categories); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
