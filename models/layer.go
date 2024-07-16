package models

type MapLayers struct {
	ID                 string  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DbTable            string  `gorm:"column:db_table" json:"db_table"`
	GeometryType       string  `gorm:"column:geometry_type" json:"geometry_type"`
	GeometryFieldName  string  `gorm:"column:geometry_fieldname" json:"geometry_fieldname"`
	IDFieldName        string  `gorm:"column:id_fieldname" json:"id_fieldname"`
	DbSchema           string  `gorm:"column:db_schema" json:"db_schema"`
	ColumnSelects      string  `gorm:"column:column_selects" json:"column_selects"`
	IsActive           bool    `gorm:"column:is_active" json:"is_active"`
	IsPublic           bool    `gorm:"column:is_public" json:"is_public"`
	IsVisible          bool    `gorm:"column:is_visible" json:"is_visible"`
	LayerOrder         int     `gorm:"column:layer_order" json:"layer_order"`
	MapLayerCategoryID string  `gorm:"column:map_layer_category_id" json:"map_layer_category_id"`
	LayerTitle         string  `gorm:"column:layer_title" json:"layer_title"`
	Description        *string `gorm:"column:description" json:"description"`
	PopupTemplate      *string `gorm:"column:popup_template" json:"popup_template"`
	UniqueValueField   *string `gorm:"column:unique_value_field" json:"unique_value_field"`
	IsOverlap          bool    `gorm:"column:is_overlap" json:"is_overlap"`
}

func (m *MapLayers) TableName() string {
	return "mapserver.map_layers"
}
