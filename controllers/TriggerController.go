package controllers

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/khankhulgun/khanmap/models"
	"github.com/lambda-platform/lambda/DB"
	"github.com/lambda-platform/lambda/datagrid"
	"gorm.io/gorm"
	"os"
	"strings"
)

func AfterSaveLayer(datePre interface{}) {
	GenerateMapServerConfig()
}

func DeleteLayer(id interface{}, grid datagrid.Datagrid, query *gorm.DB, c *fiber.Ctx) (interface{}, *gorm.DB, bool, bool) {
	GenerateMapServerConfig()

	return id, query, true, false
}

func GenerateMapServerConfig() {
	fmt.Println("generate map server config")

	var layers []models.MapLayers

	DB.DB.Where("is_active = ?", true).Find(&layers)

	var providers, mapLayers string
	for _, layer := range layers {

		var provider, mapLayer = getProvider(layer)

		providers = providers + provider
		mapLayers = mapLayers + mapLayer
	}

	var mapServerConfig string = getConfigTemplate(providers, mapLayers)

	err := os.WriteFile("mapserver/config/config.toml", []byte(mapServerConfig), 0644)
	if err != nil {
		fmt.Println(err.Error())
	}

}

func getProvider(layer models.MapLayers) (string, string) {
	provider := `
[[providers.layers]]
name = "%s"
geometry_type = "%s"
geometry_fieldname = "%s"
id_fieldname = "%s"
sql = """
    SELECT ST_AsMVTGeom(%s, !BBOX!) as %s,
    %s
    FROM %s.%s
    WHERE %s && !BBOX!"""
`
	maplayer := `
[[maps.layers]]
provider_layer = "map.%s"
min_zoom = 0
max_zoom = 22

`

	sqlColumns := layer.ColumnSelects
	if sqlColumns == "" {
		sqlColumns = layer.IDFieldname
	} else {
		// Process to remove duplicates and ensure 'id' is included if not present
		columns := strings.Split(sqlColumns, ",")
		columnMap := make(map[string]bool)
		idPresent := false
		uniqueValueFieldFound := false

		for _, col := range columns {
			col = strings.TrimSpace(col) // Clean up whitespace
			if col != "" {
				columnMap[col] = true
				if col == layer.IDFieldname {
					idPresent = true
				}
				if layer.UniqueValueField != nil {
					if col == *layer.UniqueValueField {
						uniqueValueFieldFound = true
					}
				}

			}
		}

		// Ensure the ID fieldname is included if it's not present
		if !idPresent {
			columnMap[layer.IDFieldname] = true
		}

		// Remove the geometry fieldname if it's present in the selects
		delete(columnMap, layer.GeometryFieldname)

		// Rebuild the sqlColumns string without duplicates and with 'id' included
		var newColumns []string
		for col := range columnMap {
			newColumns = append(newColumns, col)
		}

		if layer.UniqueValueField != nil && !uniqueValueFieldFound {
			newColumns = append(newColumns, *layer.UniqueValueField)
		}
		sqlColumns = strings.Join(newColumns, ", ")
	}

	return fmt.Sprintf(provider,
		layer.ID,
		layer.GeometryType,
		layer.GeometryFieldname,
		layer.IDFieldname,
		layer.GeometryFieldname, // Assuming geometry field is used in SQL
		layer.GeometryFieldname, // Alias as same as geometry field name
		sqlColumns,              // Additional columns to select
		layer.DbSchema,          // Schema name
		layer.DbTable,           // Table name
		layer.GeometryFieldname), fmt.Sprintf(maplayer, layer.ID)
}

func getConfigTemplate(providers, mapLayers string) string {
	var template string = `
[webserver]
port = ":8089"

[webserver.headers]
Cache-Control = "s-maxage=3600"

[cache]
type = "redis"
address = "${REDIS_HOST}"
password = "${REDIS_PASSWORD}"
ttl = 10
max_zoom = 18
ssl = "${TEGOLA_REDIS_SSL}"

[[providers]]
name = "map"
type = "mvt_postgis"
host = "${DB_HOST}"
port = "${DB_PORT}"
uri = "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=prefer" # PostGIS connection string (required)
database = "${DB_NAME}"
user = "${DB_USER}"
password = "${DB_PASSWORD}"
srid = 4326
ssl_mode = "${TEGOLA_POSTGIS_SSL}"

%s

[[maps]]
name = "mapserver"
center = [106.8723431, 47.8837957, 11.0]

%s
`

	return fmt.Sprintf(template, providers, mapLayers)
}
