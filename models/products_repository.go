package models

//go:generate mockgen -source=products_repository.go -destination=mock_products_repository.go -package=models

import (
	"gorm.io/gorm"
)

type ProductFetcher interface {
	GetAllProducts() ([]Product, error)
	CountProducts() (int64, error)
	GetAllProductsWithPagination(offset, limit int) ([]Product, error)
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

func (r *ProductsRepository) CountProducts() (int64, error) {
	var count int64
	if err := r.db.Model(&Product{}).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *ProductsRepository) GetAllProductsWithPagination(offset, limit int) ([]Product, error) {
	var products []Product
	if err := r.db.Preload("Variants").Preload("Category").Offset(offset).Limit(limit).Find(&products).Error; err != nil {
		return nil, err
	}

	return products, nil
}
