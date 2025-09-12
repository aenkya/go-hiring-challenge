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

func TestHandleGet(t *testing.T) {
	tests := []struct {
		name             string
		url              string
		mockProducts     []models.Product
		mockTotal        int64
		mockCountErr     error
		mockProductsErr  error
		mockFilters      models.ProductFilters
		wantStatus       int
		wantResponse     *Response
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
			wantResponse: &Response{
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
			wantResponse: &Response{
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
			wantResponse:     &Response{Products: []Product{}, Total: 0},
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
			wantResponse:     &Response{Products: []Product{}, Total: 0},
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
			wantResponse:     &Response{Products: []Product{}, Total: 0},
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
			wantResponse: &Response{
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
			wantResponse: &Response{
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

			handler.HandleGet(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}

			if tt.wantResponse != nil {
				if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", ct)
				}

				var resp Response
				if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if !reflect.DeepEqual(resp, *tt.wantResponse) {
					t.Errorf("unexpected response body: got %+v, want %+v", resp, *tt.wantResponse)
				}
			} else if tt.wantErrMessage != "" {
				if !bytes.Contains(rr.Body.Bytes(), []byte(tt.wantErrMessage)) {
					t.Errorf("expected error message in response, got %s", rr.Body.String())
				}
			}
		})
	}
}

func float64Ptr(f float64) *float64 {
	return &f
}
