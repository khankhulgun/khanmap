package models

import (
	"gorm.io/gorm"
	"time"
)

type Map struct {
	ID          string                   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Map         string                   `gorm:"column:map" json:"map"`
	Description *string                  `gorm:"column:description" json:"description"`
	CreatedAt   time.Time                `gorm:"column:created_at" json:"-"`
	UpdatedAt   time.Time                `gorm:"column:updated_at" json:"-"`
	DeletedAt   gorm.DeletedAt           `gorm:"column:deleted_at" json:"-"`
	Categories  []ViewMapLayerCategories `gorm:"foreignKey:MapID" json:"categories"`
}

func (m *Map) TableName() string {
	return "mapserver.map"
}

type ViewMapLayerCategories struct {
	ID            string      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	MapID         string      `gorm:"column:map_id" json:"map_id"`
	CategoryOrder int         `gorm:"column:category_order" json:"category_order"`
	Icon          string      `gorm:"column:icon" json:"icon"`
	IsActive      bool        `gorm:"column:is_active" json:"-"`
	IsVisible     bool        `gorm:"column:is_visible" json:"is_visible"`
	LayerCategory string      `gorm:"column:layer_category" json:"layer_category"`
	Layers        []MapLayers `gorm:"foreignKey:MapLayerCategoryID" json:"layers"`
}

func (v *ViewMapLayerCategories) TableName() string {
	return "mapserver.view_map_layer_categories"
}

type SubMapLayerCategories struct {
	ID            string `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	MapID         string `gorm:"column:map_id" json:"map_id"`
	MapCategoryID string `gorm:"column:map_category_id" json:"map_category_id"`
	CategoryOrder int    `gorm:"column:category_order" json:"category_order"`
}

func (s *SubMapLayerCategories) TableName() string {
	return "mapserver.sub_map_layer_categories"
}
