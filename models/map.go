package models

import (
	"gorm.io/gorm"
	"time"
)

type Map struct {
	ID          string                   `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Map         string                   `gorm:"column:map" json:"map"`
	Description *string                  `gorm:"column:description" json:"description"`
	CreatedAt   time.Time                `gorm:"column:created_at" json:"-"`
	UpdatedAt   time.Time                `gorm:"column:updated_at" json:"-"`
	DeletedAt   gorm.DeletedAt           `gorm:"column:deleted_at" json:"-"`
	Categories  []ViewMapLayerCategories `gorm:"foreignKey:MapID" json:"categories"`
	Version     int                      `gorm:"-" json:"version"`
	Sources     map[string]VectorSource  `gorm:"-" json:"sources"`
	Sprite      string                   `gorm:"-" json:"sprite"`
	Layers      []any                    `gorm:"-" json:"layers"`
}

func (m *Map) TableName() string {
	return "map_server.map"
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
	return "map_server.view_map_layer_categories"
}

type SubMapLayerCategories struct {
	ID            string `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	MapID         string `gorm:"column:map_id;type:uuid" json:"map_id"`
	MapCategoryID string `gorm:"column:map_category_id;type:uuid" json:"map_category_id"`
	CategoryOrder int    `gorm:"column:category_order" json:"category_order"`
}

func (s *SubMapLayerCategories) TableName() string {
	return "map_server.sub_map_layer_categories"
}
