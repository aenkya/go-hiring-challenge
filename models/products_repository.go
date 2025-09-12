package models

//go:generate mockgen -source=products_repository.go -destination=mock_products_repository.go -package=models

import (
	"gorm.io/gorm"
)

type ProductFetcher interface {
	GetAllProducts() ([]Product, error)
}

type ProductsRepository struct {
	db *gorm.DB
}

func NewProductsRepository(db *gorm.DB) ProductFetcher {
	return &ProductsRepository{
		db: db,
	}
}

func (r *ProductsRepository) GetAllProducts() ([]Product, error) {
	var products []Product
	if err := r.db.Preload("Variants").Preload("Category").Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}
