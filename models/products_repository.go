package models

//go:generate mockgen -source=products_repository.go -destination=mock_products_repository.go -package=models

import (
	"gorm.io/gorm"
)

type ProductFilters struct {
	Category string
	PriceLT  *float64
}

type ProductFetcher interface {
	GetAllProducts() ([]Product, error)
	CountProducts(filters ProductFilters) (int64, error)
	GetAllProductsWithPagination(offset, limit int, filters ProductFilters) ([]Product, error)
	GetProductByCode(code string) (*Product, error)
	GetAllCategories() ([]Category, error)
	CreateCategory(category *Category) error
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

func (r *ProductsRepository) CountProducts(filters ProductFilters) (int64, error) {
	var count int64
	query := r.db.Model(&Product{})
	query = applyFilters(query, filters)
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *ProductsRepository) GetAllProductsWithPagination(offset, limit int, filters ProductFilters) ([]Product, error) {
	var products []Product
	query := r.db.Preload("Variants").Preload("Category")
	query = applyFilters(query, filters)
	if err := query.Offset(offset).Limit(limit).Find(&products).Error; err != nil {
		return nil, err
	}

	return products, nil
}

func applyFilters(query *gorm.DB, filters ProductFilters) *gorm.DB {
	if filters.Category != "" {
		query = query.Joins("Category").Where(`"Category"."code" = ?`, filters.Category)
	}
	if filters.PriceLT != nil {
		query = query.Where("price < ?", *filters.PriceLT)
	}
	return query
}

func (r *ProductsRepository) GetProductByCode(code string) (*Product, error) {
	var product Product
	if err := r.db.Preload("Variants").Preload("Category").Where("code = ?", code).First(&product).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}

		return nil, err
	}

	return &product, nil
}

func (r *ProductsRepository) GetAllCategories() ([]Category, error) {
	var categories []Category
	if err := r.db.Find(&categories).Error; err != nil {
		return nil, err
	}

	return categories, nil
}

func (r *ProductsRepository) CreateCategory(category *Category) error {
	return r.db.Create(category).Error
}
