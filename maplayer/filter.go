package maplayer

import (
	"fmt"
	"strings"
)

// BuildFilterConditions generates SQL WHERE clauses and arguments from query parameters
func BuildFilterConditions(filters map[string]string) ([]string, []interface{}) {
	var conditions []string
	var args []interface{}

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

		// Handle Array/IN (supports "1,2,3" and "[1,2,3]")
		if strings.Contains(value, ",") {
			cleanedValue := strings.Trim(value, "[]")
			parts := strings.Split(cleanedValue, ",")
			var placeholders []string
			for _, part := range parts {
				placeholders = append(placeholders, "?")
				args = append(args, strings.TrimSpace(part))
			}
			// Sanitize key
			safeKey := strings.ReplaceAll(key, "\"", "")
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
		safeKey := strings.ReplaceAll(key, "\"", "")
		conditions = append(conditions, fmt.Sprintf("AND \"%s\" = ?", safeKey))
		args = append(args, value)
	}

	return conditions, args
}
