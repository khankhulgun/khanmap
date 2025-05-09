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
	Filters     []MapFilters             `gorm:"foreignKey:MapID" json:"filters"`
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

type MapFilters struct {
	ID                  string         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	MapID               string         `gorm:"column:map_id" json:"map_id"`
	Label               string         `gorm:"column:label" json:"label"`
	ValueField          string         `gorm:"column:value_field" json:"value_field"`
	LabelField          string         `gorm:"column:label_field" json:"label_field"`
	Table               string         `gorm:"column:table" json:"table"`
	ParentFilterInTable *string        `gorm:"column:parent_filter_in_table" json:"parent_filter_in_table"`
	ParentFilterID      *string        `gorm:"column:parent_filter_id" json:"parent_filter_id"`
	FilterOrder         *int           `gorm:"column:filter_order" json:"filter_order"`
	CreatedAt           time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"column:deleted_at" json:"deleted_at"`
}

func (m *MapFilters) TableName() string {
	return "map_server.map_filters"
}
