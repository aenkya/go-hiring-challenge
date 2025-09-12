package catalog

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/mytheresa/go-hiring-challenge/models"
	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"
)

func TestHandleGetProducts(t *testing.T) {
	tests := []struct {
		name             string
		url              string
		mockProducts     []models.Product
		mockTotal        int64
		mockCountErr     error
		mockProductsErr  error
		mockFilters      models.ProductFilters
		wantStatus       int
		wantResponse     *GetProductsResponse
		wantErrMessage   string
		expectCount      bool
		expectPagination bool
		expectedOffset   int
		expectedLimit    int
	}{
		{
			name: "success with default pagination",
			url:  "/catalog",
			mockProducts: []models.Product{
				{
					Code:  "P1",
					Price: decimal.NewFromFloat(123.45),
					Category: models.Category{
						Code: "C1",
						Name: "Category 1",
					}},
				{
					Code:  "P2",
					Price: decimal.NewFromFloat(67.89),
					Category: models.Category{
						Code: "C2",
						Name: "Category 2",
					}},
			},
			mockTotal:   2,
			mockFilters: models.ProductFilters{},
			wantStatus:  http.StatusOK,
			wantResponse: &GetProductsResponse{
				Products: []Product{
					{
						Code:  "P1",
						Price: 123.45,
						Category: Category{
							Code: "C1",
							Name: "Category 1",
						},
					},
					{
						Code:  "P2",
						Price: 67.89,
						Category: Category{
							Code: "C2",
							Name: "Category 2",
						},
					},
				},
				Total: 2,
			},
			expectCount:      true,
			expectPagination: true,
			expectedOffset:   0,
			expectedLimit:    10,
		},
		{
			name:             "count products repo error",
			url:              "/catalog",
			mockCountErr:     errors.New("db count error"),
			mockFilters:      models.ProductFilters{},
			wantStatus:       http.StatusInternalServerError,
			wantErrMessage:   "db count error",
			expectCount:      true,
			expectPagination: false,
		},
		{
			name:             "get all products with pagination repo error",
			url:              "/catalog",
			mockTotal:        5,
			mockProductsErr:  errors.New("db pagination error"),
			mockFilters:      models.ProductFilters{},
			wantStatus:       http.StatusInternalServerError,
			wantErrMessage:   "db pagination error",
			expectCount:      true,
			expectPagination: true,
			expectedOffset:   0,
			expectedLimit:    10,
		},
		{
			name: "success with custom pagination",
			url:  "/catalog?offset=1&limit=1",
			mockProducts: []models.Product{
				{
					Code:  "P2",
					Price: decimal.NewFromFloat(67.89),
					Category: models.Category{
						Code: "C2",
						Name: "Category 2",
					},
				},
			},
			mockTotal:   2,
			mockFilters: models.ProductFilters{},
			wantStatus:  http.StatusOK,
			wantResponse: &GetProductsResponse{
				Products: []Product{
					{
						Code:  "P2",
						Price: 67.89,
						Category: Category{
							Code: "C2",
							Name: "Category 2",
						},
					},
				},
				Total: 2,
			},
			expectCount:      true,
			expectPagination: true,
			expectedOffset:   1,
			expectedLimit:    1,
		},
		{
			name:             "invalid limit defaults to 1",
			url:              "/catalog?limit=0",
			mockTotal:        0,
			mockProducts:     []models.Product{},
			mockFilters:      models.ProductFilters{},
			wantStatus:       http.StatusOK,
			wantResponse:     &GetProductsResponse{Products: []Product{}, Total: 0},
			expectCount:      true,
			expectPagination: true,
			expectedOffset:   0,
			expectedLimit:    1,
		},
		{
			name:             "limit over 100 is capped at 100",
			url:              "/catalog?limit=101",
			mockTotal:        0,
			mockProducts:     []models.Product{},
			mockFilters:      models.ProductFilters{},
			wantStatus:       http.StatusOK,
			wantResponse:     &GetProductsResponse{Products: []Product{}, Total: 0},
			expectCount:      true,
			expectPagination: true,
			expectedOffset:   0,
			expectedLimit:    100,
		},
		{
			name:             "invalid offset and limit are ignored",
			url:              "/catalog?offset=abc&limit=xyz",
			mockTotal:        0,
			mockProducts:     []models.Product{},
			mockFilters:      models.ProductFilters{},
			wantStatus:       http.StatusOK,
			wantResponse:     &GetProductsResponse{Products: []Product{}, Total: 0},
			expectCount:      true,
			expectPagination: true,
			expectedOffset:   0,
			expectedLimit:    10,
		},
		{
			name: "success with category filter",
			url:  "/catalog?category=C1",
			mockProducts: []models.Product{
				{
					Code:  "P1",
					Price: decimal.NewFromFloat(123.45),
					Category: models.Category{
						Code: "C1",
						Name: "Category 1",
					},
				},
			},
			mockTotal:   1,
			mockFilters: models.ProductFilters{Category: "C1"},
			wantStatus:  http.StatusOK,
			wantResponse: &GetProductsResponse{
				Products: []Product{
					{
						Code:  "P1",
						Price: 123.45,
						Category: Category{
							Code: "C1",
							Name: "Category 1",
						},
					},
				},
				Total: 1,
			},
			expectCount:      true,
			expectPagination: true,
			expectedOffset:   0,
			expectedLimit:    10,
		},
		{
			name: "success with price less than filter",
			url:  "/catalog?priceLessThan=100",
			mockProducts: []models.Product{
				{
					Code:  "P2",
					Price: decimal.NewFromFloat(67.89),
					Category: models.Category{
						Code: "C2",
						Name: "Category 2",
					},
				},
			},
			mockTotal:   1,
			mockFilters: models.ProductFilters{PriceLT: float64Ptr(100)},
			wantStatus:  http.StatusOK,
			wantResponse: &GetProductsResponse{
				Products: []Product{
					{
						Code:  "P2",
						Price: 67.89,
						Category: Category{
							Code: "C2",
							Name: "Category 2",
						},
					},
				},
				Total: 1,
			},
			expectCount:      true,
			expectPagination: true,
			expectedOffset:   0,
			expectedLimit:    10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			repo := models.NewMockProductFetcher(ctrl)
			if tt.expectCount {
				repo.EXPECT().CountProducts(tt.mockFilters).Return(tt.mockTotal, tt.mockCountErr)
			}

			if tt.expectPagination {
				repo.EXPECT().
					GetAllProductsWithPagination(tt.expectedOffset, tt.expectedLimit, tt.mockFilters).
					Return(tt.mockProducts, tt.mockProductsErr)
			}

			handler := &CatalogHandler{repo: repo}

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			rr := httptest.NewRecorder()

			handler.HandleGetProducts(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}

			if tt.wantResponse != nil {
				if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", ct)
				}

				var resp GetProductsResponse
				if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if !reflect.DeepEqual(resp, *tt.wantResponse) {
					t.Errorf("unexpected response body: got %+v, want %+v", resp, *tt.wantResponse)
				}
			}

			if tt.wantErrMessage != "" {
				if !bytes.Contains(rr.Body.Bytes(), []byte(tt.wantErrMessage)) {
					t.Errorf("expected error message in response, got %s", rr.Body.String())
				}
			}
		})
	}
}

func TestHandleGetProduct(t *testing.T) {
	productCode := "P1"
	productPrice := 123.45
	variantPrice := 55.55

	tests := []struct {
		name           string
		productCode    string
		mockProduct    *models.Product
		mockError      error
		wantStatus     int
		wantResponse   *ProductDetailResponse
		wantErrMessage string
	}{
		{
			name:        "variants should inherit price from product if variant price is not set",
			productCode: productCode,
			mockProduct: &models.Product{
				Code:  productCode,
				Price: decimal.NewFromFloat(productPrice),
				Category: models.Category{
					Code: "C1",
					Name: "Category 1",
				},
				Variants: []models.Variant{
					{Name: "Variant 1", SKU: "V1-SKU", Price: decimal.NewFromFloat(variantPrice)},
					{Name: "Variant 2", SKU: "V2-SKU", Price: decimal.NewFromInt(0)},
				},
			},
			wantStatus: http.StatusOK,
			wantResponse: &ProductDetailResponse{
				Code:  productCode,
				Price: productPrice,
				Category: Category{
					Code: "C1",
					Name: "Category 1",
				},
				Variants: []VariantDetail{
					{Name: "Variant 1", SKU: "V1-SKU", Price: variantPrice},
					{Name: "Variant 2", SKU: "V2-SKU", Price: productPrice},
				},
			},
		},
		{
			name:        "product not found",
			productCode: "non-existent",
			mockProduct: nil,
			mockError:   nil,
			wantStatus:  http.StatusNotFound,
		},
		{
			name:           "repository error",
			productCode:    "any-code",
			mockProduct:    nil,
			mockError:      errors.New("db error"),
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: "db error",
		},
		{
			name:        "success with no variants",
			productCode: productCode,
			mockProduct: &models.Product{
				Code:  productCode,
				Price: decimal.NewFromFloat(productPrice),
				Category: models.Category{
					Code: "C1",
					Name: "Category 1",
				},
				Variants: []models.Variant{},
			},
			wantStatus: http.StatusOK,
			wantResponse: &ProductDetailResponse{
				Code:  productCode,
				Price: productPrice,
				Category: Category{
					Code: "C1",
					Name: "Category 1",
				},
				Variants: []VariantDetail{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := models.NewMockProductFetcher(ctrl)

			repo.EXPECT().
				GetProductByCode(tt.productCode).
				Return(tt.mockProduct, tt.mockError)

			handler := &CatalogHandler{repo: repo}

			req := httptest.NewRequest(http.MethodGet, "/catalog/"+tt.productCode, nil)
			rr := httptest.NewRecorder()

			handler.HandleGetProduct(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}

			if tt.wantResponse != nil {
				if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", ct)
				}

				var resp ProductDetailResponse
				if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if !reflect.DeepEqual(resp, *tt.wantResponse) {
					t.Errorf("unexpected response body: got %+v, want %+v", resp, *tt.wantResponse)
				}
			}

			if tt.wantErrMessage != "" {
				if !bytes.Contains(rr.Body.Bytes(), []byte(tt.wantErrMessage)) {
					t.Errorf("expected error message in response, got %s", rr.Body.String())
				}
			}
		})
	}
}

func TestHandleGetCategories(t *testing.T) {
	tests := []struct {
		name           string
		mockCategories []models.Category
		mockError      error
		wantStatus     int
		wantResponse   []Category
		wantErrMessage string
	}{
		{
			name: "success",
			mockCategories: []models.Category{
				{Code: "C1", Name: "Category 1"},
				{Code: "C2", Name: "Category 2"},
			},
			mockError:  nil,
			wantStatus: http.StatusOK,
			wantResponse: []Category{
				{Code: "C1", Name: "Category 1"},
				{Code: "C2", Name: "Category 2"},
			},
		},
		{
			name:           "repository error",
			mockCategories: nil,
			mockError:      errors.New("db error"),
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: "db error",
		},
		{
			name:           "success with no categories",
			mockCategories: []models.Category{},
			mockError:      nil,
			wantStatus:     http.StatusOK,
			wantResponse:   []Category{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := models.NewMockProductFetcher(ctrl)

			repo.EXPECT().GetAllCategories().Return(tt.mockCategories, tt.mockError)

			handler := &CatalogHandler{repo: repo}

			req := httptest.NewRequest(http.MethodGet, "/categories", nil)
			rr := httptest.NewRecorder()

			handler.HandleGetCategories(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}

			if tt.wantResponse != nil {
				if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", ct)
				}

				var resp []Category
				if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if !reflect.DeepEqual(resp, tt.wantResponse) {
					t.Errorf("unexpected response body: got %+v, want %+v", resp, tt.wantResponse)
				}
			}
		})
	}
}

func TestHandleCreateCategory(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		mockError      error
		wantStatus     int
		wantResponse   *models.Category
		wantErrMessage string
	}{
		{
			name:       "success",
			body:       `{"code": "C-NEW", "name": "New Category"}`,
			mockError:  nil,
			wantStatus: http.StatusCreated,
			wantResponse: &models.Category{
				Code: "C-NEW",
				Name: "New Category",
			},
		},
		{
			name:           "invalid json body",
			body:           `{"code": "C-NEW"`,
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "Invalid request body",
		},
		{
			name:           "missing required fields - code",
			body:           `{"name": "Missing Code"}`,
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "Category code and name are required",
		},
		{
			name:           "missing required fields - name",
			body:           `{"code": "MISS-NAME"}`,
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "Category code and name are required",
		},
		{
			name:           "repository error",
			body:           `{"code": "C-FAIL", "name": "Failing Category"}`,
			mockError:      errors.New("db insert error"),
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: "Failed to create category",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := models.NewMockProductFetcher(ctrl)

			if tt.mockError != nil || (tt.wantStatus == http.StatusCreated) {
				repo.EXPECT().CreateCategory(gomock.Any()).Return(tt.mockError).AnyTimes()
			}

			handler := &CatalogHandler{repo: repo}

			req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewBufferString(tt.body))
			rr := httptest.NewRecorder()

			handler.HandleCreateCategory(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}

			if tt.wantResponse != nil {
				var resp models.Category
				if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if resp.Code != tt.wantResponse.Code || resp.Name != tt.wantResponse.Name {
					t.Errorf("unexpected response body: got %+v, want %+v", resp, *tt.wantResponse)
				}
			}

			if tt.wantErrMessage != "" {
				body := rr.Body.String()
				if !bytes.Contains([]byte(body), []byte(tt.wantErrMessage)) {
					t.Errorf("expected error message %q in response, got %q", tt.wantErrMessage, body)
				}
			}
		})
	}
}

func float64Ptr(f float64) *float64 {
	return &f
}
