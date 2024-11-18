package spatial

import (
	"fmt"
	"github.com/khankhulgun/khanmap/maplayer"
	"github.com/khankhulgun/khanmap/models"
	"github.com/lambda-platform/lambda/DB"
	"strings"
)

// Map relationship types to PostGIS functions
func GetRelationshipFunction(relationship string) (string, error) {
	relationshipFunctions := map[string]string{
		"contains":   "ST_Contains",
		"crosses":    "ST_Crosses",
		"disjoint":   "ST_Disjoint",
		"equals":     "ST_Equals",
		"intersects": "ST_Intersects",
		"overlaps":   "ST_Overlaps",
		"within":     "ST_Within",
		"touches":    "ST_Touches",
	}

	sqlFunction, ok := relationshipFunctions[strings.ToLower(relationship)]
	if !ok {
		return "", fmt.Errorf("invalid spatial relationship")
	}
	return sqlFunction, nil
}

// Execute the spatial query and return the results
func ExecuteSpatialQuery(query, geometry string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	if geometry == "" {
		if err := DB.DB.Raw(query).Scan(&results).Error; err != nil {
			return nil, fmt.Errorf("error executing spatial query: %w", err)
		}
	} else {
		if err := DB.DB.Raw(query, geometry).Scan(&results).Error; err != nil {
			return nil, fmt.Errorf("error executing spatial query: %w", err)
		}
	}

	return results, nil
}

// Build the full SQL query based on inputs
func BuildSpatialQuery(layerDetails models.MapLayersForTile, sqlFunction, geometry string, returnGeometry bool) string {

	if returnGeometry {
		layerDetails.ColumnSelects = layerDetails.ColumnSelects + "," + layerDetails.GeometryFieldName
	}
	query := fmt.Sprintf(`
		SELECT %s FROM %s.%s
		WHERE %s(%s, ST_GeomFromText(?, 4326))
	`, maplayer.ConstructSQLColumns(layerDetails, false), layerDetails.DbSchema, layerDetails.DbTable, sqlFunction, layerDetails.GeometryFieldName)
	return query
}

// Build the full SQL query based on inputs
func BuildSpatialQueryWithFromText(layerDetails models.MapLayersForTile, sqlFunction, geometry string, returnGeometry bool) string {

	if returnGeometry {
		layerDetails.ColumnSelects = layerDetails.ColumnSelects + "," + layerDetails.GeometryFieldName
	}
	query := fmt.Sprintf(`
		SELECT %s FROM %s.%s
		WHERE %s(%s, %s)
	`, maplayer.ConstructSQLColumns(layerDetails, false), layerDetails.DbSchema, layerDetails.DbTable, sqlFunction, layerDetails.GeometryFieldName, geometry)
	return query
}
