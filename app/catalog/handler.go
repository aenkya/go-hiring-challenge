package catalog

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/mytheresa/go-hiring-challenge/app/api"
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
	repo models.Repository
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

func NewCatalogHandler(r models.Repository) *CatalogHandler {
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
		api.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	res, err := h.repo.GetAllProductsWithPagination(offset, limit, filters)
	if err != nil {
		api.ErrorResponse(w, http.StatusInternalServerError, err.Error())
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

	response := GetProductsResponse{
		Products: products,
		Total:    total,
	}

	api.OKResponse(w, response)
}

func (h *CatalogHandler) HandleGetProduct(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Path[len("/catalog/"):]
	product, err := h.repo.GetProductByCode(code)

	if err != nil {
		api.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if product == nil {
		api.ErrorResponse(w, http.StatusNotFound, "Product not found")
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

	api.OKResponse(w, resp)
}

func (h *CatalogHandler) HandleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var newCategory models.Category
	if err := json.NewDecoder(r.Body).Decode(&newCategory); err != nil {
		api.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if newCategory.Code == "" || newCategory.Name == "" {
		api.ErrorResponse(w, http.StatusBadRequest, "Category code and name are required")
		return
	}

	if err := h.repo.CreateCategory(&newCategory); err != nil {
		api.ErrorResponse(w, http.StatusInternalServerError, "Failed to create category")
		return
	}

	w.WriteHeader(http.StatusCreated)
	api.OKResponse(w, newCategory)
}

func (h *CatalogHandler) HandleGetCategories(w http.ResponseWriter, r *http.Request) {
	res, err := h.repo.GetAllCategories()
	if err != nil {
		api.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	categories := make([]Category, len(res))
	for i, c := range res {
		categories[i] = Category{
			Code: c.Code,
			Name: c.Name,
		}
	}

	api.OKResponse(w, categories)
}
