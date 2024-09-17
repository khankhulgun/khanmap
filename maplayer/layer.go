package maplayer

import (
	"github.com/dgraph-io/ristretto"
	"github.com/khankhulgun/khanmap/models"
	"github.com/lambda-platform/lambda/DB"
	"log"
	"strings"
	"time"
)

func init() {
	// Initialize the cache with Ristretto
	var err error
	layerCache, err = ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M)
		MaxCost:     1 << 30, // maximum cost of cache (1GB)
		BufferItems: 64,      // number of keys per Get buffer
	})
	if err != nil {
		log.Fatalf("Failed to initialize cache: %v", err)
	}
}

var layerCache *ristretto.Cache

func FetchLayerDetails(layerID string) (models.MapLayersForTile, error) {
	layerID = strings.TrimSpace(layerID)

	if cachedLayer, found := layerCache.Get(layerID); found {
		layerDetails, ok := cachedLayer.(models.MapLayersForTile)
		if ok {
			return layerDetails, nil
		}
	}

	var layerDetails models.MapLayersForTile
	err := DB.DB.Where("id = ?", layerID).First(&layerDetails).Error
	if err != nil {
		return layerDetails, err
	}

	layerCache.SetWithTTL(layerID, layerDetails, 1, 60*time.Minute)
	layerCache.Wait()

	return layerDetails, nil
}
func ConstructSQLColumns(layer models.MapLayersForTile, ignoreGeometry bool) string {
	sqlColumns := layer.ColumnSelects
	if sqlColumns == "" {
		return "'" + layer.IDFieldName + "'"
	}

	columns := strings.Split(sqlColumns, ",")
	columnMap := make(map[string]bool)
	idPresent := false
	uniqueValueFieldFound := false

	for _, col := range columns {
		col = strings.TrimSpace(col)
		if col != "" {
			columnMap[col] = true
			if col == layer.IDFieldName {
				idPresent = true
			}
			if layer.UniqueValueField != nil && col == *layer.UniqueValueField {
				uniqueValueFieldFound = true
			}
		}
	}

	if !idPresent {
		columnMap[layer.IDFieldName] = true
	}

	if ignoreGeometry {
		delete(columnMap, layer.GeometryFieldName)
	}

	var newColumns []string
	for col := range columnMap {
		newColumns = append(newColumns, "\""+col+"\"")
	}

	if layer.UniqueValueField != nil && !uniqueValueFieldFound {
		newColumns = append(newColumns, "\""+*layer.UniqueValueField+"\"")
	}

	return strings.Join(newColumns, ", ")
}
