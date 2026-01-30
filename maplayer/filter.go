package maplayer

import (
	"fmt"
	"strings"
	"sync"

	"github.com/lambda-platform/lambda/DB"
)

var schemaCache sync.Map

// getTableSchema retrieves and caches column types for a given table
func getTableSchema(schema, table string) (map[string]string, error) {
	cacheKey := fmt.Sprintf("%s.%s", schema, table)

	if cached, ok := schemaCache.Load(cacheKey); ok {
		return cached.(map[string]string), nil
	}

	columnTypes := make(map[string]string)

	// Query to get column names and their data types
	query := `
		SELECT 
			a.attname AS column_name,
			format_type(a.atttypid, a.atttypmod) AS data_type
		FROM 
			pg_attribute a
		JOIN 
			pg_class c ON a.attrelid = c.oid
		JOIN 
			pg_namespace n ON c.relnamespace = n.oid
		WHERE 
			c.relname = ?      
			AND n.nspname = ?   
			AND a.attnum > 0 
			AND NOT a.attisdropped
	`

	rows, err := DB.DB.Raw(query, table, schema).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var colName, dataType string
		if err := rows.Scan(&colName, &dataType); err == nil {
			columnTypes[colName] = dataType
		}
	}

	schemaCache.Store(cacheKey, columnTypes)
	return columnTypes, nil
}

// BuildFilterConditions generates SQL WHERE clauses and arguments from query parameters
func BuildFilterConditions(filters map[string]string, schema, table string) ([]string, []interface{}) {
	var conditions []string
	var args []interface{}

	// Attempt to get schema info, but don't fail hard if it fails (just fallback to default string handling usually)
	// However, for array logic we strictly need it. If it fails, we assume standard behavior.
	colTypes, _ := getTableSchema(schema, table)

	for key, value := range filters {
		// Skip metadata keys or empty values
		if key == "search_columns" || value == "" {
			continue
		}

		// Handle generic global search
		if key == "search" {
			searchCols := filters["search_columns"]
			if searchCols == "" {
				// Default searchable columns if none specified.
				// These are common columns in the schemas.
				searchCols = "name,org_name,title,description"
			}
			cols := strings.Split(searchCols, ",")
			var searchParts []string

			for _, col := range cols {
				col = strings.TrimSpace(col)
				if col == "" {
					continue
				}
				// Basic sanitization
				col = strings.ReplaceAll(col, "\"", "")
				col = strings.ReplaceAll(col, "'", "")

				searchParts = append(searchParts, fmt.Sprintf("\"%s\" ILIKE ?", col))
			}

			if len(searchParts) > 0 {
				conditions = append(conditions, fmt.Sprintf("AND (%s)", strings.Join(searchParts, " OR ")))
				// Append the search value for each column condition
				for range searchParts {
					args = append(args, "%"+value+"%")
				}
			}
			continue
		}

		safeKey := strings.ReplaceAll(key, "\"", "")

		// Check if it is an array column
		isBoundsArray := false
		if colTypes != nil {
			if dtype, ok := colTypes[key]; ok {
				if strings.HasSuffix(dtype, "[]") {
					isBoundsArray = true
				}
			}
		}

		// Handle Array Columns Dynamically
		if isBoundsArray {
			cleanedValue := strings.Trim(value, "[]")
			if cleanedValue == "" {
				continue
			}

			// Determine appropriate cast based on column type
			castType := "int[]" // default
			singleCast := "int" // default

			if dtype, ok := colTypes[key]; ok {
				switch dtype {
				case "smallint[]":
					castType = "smallint[]"
					singleCast = "smallint"
				case "bigint[]":
					castType = "bigint[]"
					singleCast = "bigint"
				case "integer[]", "int[]":
					castType = "int[]"
					singleCast = "int"
				case "text[]", "character varying[]":
					castType = "text[]"
					singleCast = "text"
				}
			} else {
				// Fallback generic logic or explicit overrides if schema missing
				if key == "service_type_ids" {
					castType = "smallint[]"
					singleCast = "smallint"
				} else if key == "food_country_ids" {
					castType = "int[]"
					singleCast = "int"
				}
			}

			if strings.Contains(cleanedValue, ",") {
				parts := strings.Split(cleanedValue, ",")
				var placeholders []string
				for _, part := range parts {
					placeholders = append(placeholders, "?")
					args = append(args, strings.TrimSpace(part))
				}

				conditions = append(conditions, fmt.Sprintf("AND \"%s\" && ARRAY[%s]::%s", safeKey, strings.Join(placeholders, ","), castType))
			} else {
				conditions = append(conditions, fmt.Sprintf("AND ?::%s = ANY(\"%s\")", singleCast, safeKey))
				args = append(args, cleanedValue)
			}
			continue
		}

		// Handle Standard Array/IN (supports "1,2,3" and "[1,2,3]") for non-array columns (e.g. status IN (1,2))
		if strings.Contains(value, ",") {
			cleanedValue := strings.Trim(value, "[]")
			parts := strings.Split(cleanedValue, ",")
			var placeholders []string
			for _, part := range parts {
				placeholders = append(placeholders, "?")
				args = append(args, strings.TrimSpace(part))
			}
			conditions = append(conditions, fmt.Sprintf("AND \"%s\" IN (%s)", safeKey, strings.Join(placeholders, ",")))
			continue
		}

		// Handle explicit LIKE
		if strings.HasSuffix(key, "__like") {
			realKey := strings.TrimSuffix(key, "__like")
			safeKey := strings.ReplaceAll(realKey, "\"", "")
			conditions = append(conditions, fmt.Sprintf("AND \"%s\" ILIKE ?", safeKey))
			args = append(args, "%"+value+"%")
			continue
		}

		// Standard Equality
		conditions = append(conditions, fmt.Sprintf("AND \"%s\" = ?", safeKey))
		args = append(args, value)
	}

	return conditions, args
}
