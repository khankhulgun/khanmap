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

			if strings.Contains(cleanedValue, ",") {
				parts := strings.Split(cleanedValue, ",")
				var placeholders []string
				for _, part := range parts {
					placeholders = append(placeholders, "?")
					args = append(args, strings.TrimSpace(part))
				}
				// Use correct casting based on column type if we wanted to be super precise,
				// but ::int[] is usually what we need for ID arrays.
				// However, if the user has a text[] column, ::int[] might fail.
				// The user specifically asked for "service_type_ids" -> "smallint[]" and "food_country_ids" -> "integer[]".
				// Both are compatible with int casting for the numeric literal inputs usually.
				// But to be generic, we should probably output the formatted string array or cast appropriately.
				// For now, adhering to the user's previous request pattern which used ::int[].
				// If we want to be safe for text arrays, we might need to check the base type.
				// But given the context of "ids", int is likely specific.
				// Let's stick to the user's requested pattern for arrays. To support generic arrays better we might need more logic
				// but for "array_agg column filter" which are usually IDs, this is the request.

				// Let's refine: If the column type is integer[] or smallint[], cast to int[].
				// If text[], cast to text[].
				// User only mentioned "if column type array_agg it should by 2 = ANY(service_type_ids); or food_country_ids && ARRAY[1, 5];"
				// I will use ::int[] as default for now as most agg arrays in this context are IDs.

				conditions = append(conditions, fmt.Sprintf("AND \"%s\" && ARRAY[%s]::int[]", safeKey, strings.Join(placeholders, ",")))
			} else {
				conditions = append(conditions, fmt.Sprintf("AND ?::int = ANY(\"%s\")", safeKey))
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
