package models

import "time"

type Category struct {
	ID        uint   `gorm:"primarykey"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (c *Category) TableName() string {
	return "categories"
}
