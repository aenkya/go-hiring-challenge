package catalog

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mytheresa/go-hiring-challenge/models"
	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"
)

func TestHandleGet(t *testing.T) {
	tests := []struct {
		name           string
		products       []models.Product
		repoErr        error
		wantStatus     int
		wantProducts   []Product
		wantErrMessage string
	}{
		{
			name: "success",
			products: []models.Product{
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
			repoErr:    nil,
			wantStatus: http.StatusOK,
			wantProducts: []Product{
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
		},
		{
			name:           "repo error",
			products:       nil,
			repoErr:        errors.New("db error"),
			wantStatus:     http.StatusInternalServerError,
			wantErrMessage: "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := models.NewMockProductFetcher(ctrl)
			repo.EXPECT().GetAllProducts().Return(tt.products, tt.repoErr)

			handler := &CatalogHandler{repo: repo}

			req := httptest.NewRequest(http.MethodGet, "/catalog", nil)
			rr := httptest.NewRecorder()

			handler.HandleGet(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}

			if tt.wantStatus == http.StatusOK {
				if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", ct)
				}

				var resp Response
				if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if len(resp.Products) != len(tt.wantProducts) {
					t.Errorf("expected %d products, got %d", len(tt.wantProducts), len(resp.Products))
				}

				for i, p := range tt.wantProducts {
					if resp.Products[i].Code != p.Code || resp.Products[i].Price != p.Price {
						t.Errorf("unexpected product[%d]: %+v", i, resp.Products[i])
					}
				}
			} else if tt.wantErrMessage != "" {
				if !bytes.Contains(rr.Body.Bytes(), []byte(tt.wantErrMessage)) {
					t.Errorf("expected error message in response, got %s", rr.Body.String())
				}
			}
		})
	}
}
