package models

import (
	"gorm.io/gorm"
	"time"
)

type MapLayersForTile struct {
	ID                 string                   `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	DbTable            string                   `gorm:"column:db_table" json:"db_table"`
	GeometryType       string                   `gorm:"column:geometry_type" json:"geometry_type"`
	GeometryFieldName  string                   `gorm:"column:geometry_fieldname" json:"geometry_fieldname"`
	IDFieldName        string                   `gorm:"column:id_fieldname" json:"id_fieldname"`
	DbSchema           string                   `gorm:"column:db_schema" json:"db_schema"`
	ColumnSelects      string                   `gorm:"column:column_selects" json:"column_selects"`
	IsActive           bool                     `gorm:"column:is_active" json:"is_active"`
	IsPublic           bool                     `gorm:"column:is_public" json:"is_public"`
	IsVisible          bool                     `gorm:"column:is_visible" json:"is_visible"`
	LayerOrder         int                      `gorm:"column:layer_order" json:"layer_order"`
	MapLayerCategoryID string                   `gorm:"column:map_layer_category_id;type:uuid" json:"map_layer_category_id"`
	LayerTitle         string                   `gorm:"column:layer_title" json:"layer_title"`
	Description        *string                  `gorm:"column:description" json:"description"`
	PopupTemplate      *string                  `gorm:"column:popup_template" json:"popup_template"`
	UniqueValueField   *string                  `gorm:"column:unique_value_field" json:"unique_value_field"`
	IsOverlap          bool                     `gorm:"column:is_overlap" json:"is_overlap"`
	IsPermission       bool                     `gorm:"column:is_permission" json:"is_permission"`
	SoumIDField        *string                  `gorm:"column:soum_id_field" json:"soum_id_field"`
	BaghIDField        *string                  `gorm:"column:bagh_id_field" json:"bagh_id_field"`
	Permissions        []SubMapLayerPermissions `gorm:"foreignKey:LayerID" json:"permissions"`
	Filters            []SubMapLayerFilters     `gorm:"foreignKey:LayerID" json:"filters"`
}

func (m *MapLayersForTile) TableName() string {
	return "map_server.map_layers"
}

type MapLayerCategory struct {
	ID            string      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Icon          string      `gorm:"column:icon" json:"icon"`
	IsActive      bool        `gorm:"column:is_active" json:"is_active"`
	IsVisible     bool        `gorm:"column:is_visible" json:"is_visible"`
	LayerCategory string      `gorm:"column:layer_category" json:"layer_category"`
	Layers        []MapLayers `gorm:"foreignKey:MapLayerCategoryID" json:"layers"`
}

func (m *MapLayerCategory) TableName() string {
	return "map_server.map_layer_category"
}

type MapLayers struct {
	ID                 string                    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	DbTable            string                    `gorm:"column:db_table" json:"-"`
	GeometryType       string                    `gorm:"column:geometry_type" json:"geometry_type"`
	GeometryFieldname  string                    `gorm:"column:geometry_fieldname" json:"geometry_fieldname"`
	IDFieldname        string                    `gorm:"column:id_fieldname" json:"id_fieldname"`
	DbSchema           string                    `gorm:"column:db_schema" json:"-"`
	ColumnSelects      string                    `gorm:"column:column_selects" json:"-"`
	IsActive           bool                      `gorm:"column:is_active" json:"-"`
	IsPublic           bool                      `gorm:"column:is_public" json:"is_public"`
	IsVisible          bool                      `gorm:"column:is_visible" json:"is_visible"`
	LayerOrder         int                       `gorm:"column:layer_order" json:"layer_order"`
	MapLayerCategoryID string                    `gorm:"column:map_layer_category_id" json:"map_layer_category_id"`
	LayerTitle         string                    `gorm:"column:layer_title" json:"layer_title"`
	Description        *string                   `gorm:"column:description" json:"description"`
	PopupTemplate      *string                   `gorm:"column:popup_template" json:"popup_template"`
	UniqueValueField   *string                   `gorm:"column:unique_value_field" json:"unique_value_field"`
	IsOverlap          bool                      `gorm:"column:is_overlap" json:"is_overlap"`
	IsPermission       bool                      `gorm:"column:is_permission" json:"is_permission"`
	SoumIDField        *string                   `gorm:"column:soum_id_field" json:"soum_id_field"`
	BaghIDField        *string                   `gorm:"column:bagh_id_field" json:"bagh_id_field"`
	Layer              *interface{}              `gorm:"-" json:"layer"`
	Legends            []MapLayerLegends         `gorm:"foreignKey:LayerID" json:"legends"`
	AdminFilters       []SubMapLayerAdminFilters `gorm:"foreignKey:LayerID" json:"admin_filters"`
}

func (m *MapLayers) TableName() string {
	return "map_server.map_layers"
}

type MapLayerLegends struct {
	ID               string  `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	LayerID          string  `gorm:"column:layer_id;type:uuid" json:"layer_id"`
	GeometryType     string  `gorm:"column:geometry_type" json:"geometry_type"`
	FillColor        *string `gorm:"column:fill_color" json:"fill_color"`
	Marker           *string `gorm:"column:marker" json:"marker"`
	PolygonType      *string `gorm:"column:polygon_type" json:"polygon_type"`
	LineType         *string `gorm:"column:line_type" json:"line_type"`
	UniqueValue      *string `gorm:"column:unique_value" json:"unique_value"`
	UniqueValueLabel *string `gorm:"column:unique_value_label" json:"unique_value_label"`
	UniqueVisible    bool    `gorm:"column:unique_visible" json:"unique_visible"`
	StrokeColor      *string `gorm:"column:stroke_color" json:"stroke_color"`
	LegendOrder      *string `gorm:"column:legend_order" json:"legend_order"`
}

func (m *MapLayerLegends) TableName() string {
	return "map_server.map_layer_legends"
}

type SubMapLayerPermissions struct {
	ID      string `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	LayerID string `gorm:"column:layer_id;type:uuid" json:"layer_id"`
	RoleID  int    `gorm:"column:role_id" json:"role_id"`
}

func (s *SubMapLayerPermissions) TableName() string {
	return "map_server.sub_map_layer_permissions"
}

type SubMapLayerFilters struct {
	ID          string `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	LayerID     string `gorm:"column:layer_id" json:"layer_id"`
	UserColumn  string `gorm:"column:user_column" json:"user_column"`
	TableColumn string `gorm:"column:table_column" json:"table_column"`
}

func (s *SubMapLayerFilters) TableName() string {
	return "map_server.sub_map_layer_filters"
}

type SubMapLayerAdminFilters struct {
	ID         string         `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	LayerID    string         `gorm:"column:layer_id" json:"layer_id"`
	FilterID   string         `gorm:"column:filter_id" json:"filter_id"`
	LayerField string         `gorm:"column:layer_field" json:"layer_field"`
	CreatedAt  time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at" json:"deleted_at"`
}

func (s *SubMapLayerAdminFilters) TableName() string {
	return "map_server.sub_map_layer_admin_filters"
}
