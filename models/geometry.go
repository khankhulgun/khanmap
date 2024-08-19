package models

// Model for geometry tables
type GeometryTable struct {
	Schema         string `gorm:"column:schema"`
	TableName      string `gorm:"column:f_table_name"`
	GeometryColumn string `gorm:"column:f_geometry_column"`
	CoordDimension int    `gorm:"column:coord_dimension"`
	SRID           int    `gorm:"column:srid"`
	Type           string `gorm:"column:type"`
}

// Model for table columns
type TableColumn struct {
	ColumnName string `gorm:"column:column_name"`
	DataType   string `gorm:"column:data_type"`
}
