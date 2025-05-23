package logicfunction

import (
	"database/sql"
	"encoding/json"
)

type TagLevel struct {
	Tag   string `json:"tag"`
	Level int    `json:"level"`
}

type UserTagLevels struct {
	ID        string     `json:"id"`
	TagLevels []TagLevel `json:"taglevels"`
}

type PostRow struct {
	ID       string   `json:"id"`
	PosterID string   `json:"posterid"`
	Tags     []string `json:"tags"`
}

func RowsToJSON(rows *sql.Rows) ([]byte, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}
		results = append(results, rowMap)
	}

	return json.Marshal(results)
}

func RowsToJSONObject(rows *sql.Rows) (map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}
		results = append(results, rowMap)
	}

	return map[string]interface{}{"results": results}, nil
}

func RowsToObjectList(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}
		results = append(results, rowMap)
	}

	return results, nil
}

// Converts SQL rows to a slice of TagLevel structs.
// Expects each row to have columns "tag" (string) and "level" (int or int16 or float64).
func RowsToTagLevelList(rows *sql.Rows) ([][]TagLevel, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results [][]TagLevel
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		for i, col := range columns {
			if col == "taglevels" {
				switch v := values[i].(type) {
				case string:
					var tagLevels []TagLevel
					if err := json.Unmarshal([]byte(v), &tagLevels); err == nil {
						results = append(results, tagLevels)
					}
				case []byte:
					var tagLevels []TagLevel
					if err := json.Unmarshal(v, &tagLevels); err == nil {
						results = append(results, tagLevels)
					}
				}
			}
		}
	}

	return results, nil
}

// Converts SQL rows to a slice of UserTagLevels.
// Each row should have an "id" column and a "taglevels" column (JSON array of TagLevel objects).
func RowsToUserTagLevelList(rows *sql.Rows) ([]UserTagLevels, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []UserTagLevels
	for rows.Next() {
		var id string
		var skillRaw []byte
		scanArgs := make([]interface{}, len(columns))
		for i, col := range columns {
			switch col {
			case "id":
				scanArgs[i] = &id
			case "skill":
				scanArgs[i] = &skillRaw
			default:
				var discard interface{}
				scanArgs[i] = &discard
			}
		}
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}

		tagLevels := []TagLevel{}
		if len(skillRaw) > 0 {
			if err := json.Unmarshal(skillRaw, &tagLevels); err != nil {
				tagLevels = []TagLevel{}
			}
		}
		results = append(results, UserTagLevels{ID: id, TagLevels: tagLevels})
	}
	return results, nil
}

// Converts SQL rows to a slice of PostRow structs.
// Each row should have columns "id", "poster_id", and "tags" (tags as []string, []byte, or comma-separated string).
func RowsToPostRowList(rows *sql.Rows) ([]PostRow, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []PostRow
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		var id, posterID string
		var tags []string

		for i, col := range columns {
			switch col {
			case "id":
				switch v := values[i].(type) {
				case string:
					id = v
				case []byte:
					id = string(v)
				}
			case "poster_id":
				switch v := values[i].(type) {
				case string:
					posterID = v
				case []byte:
					posterID = string(v)
				}
			case "tags":
				switch v := values[i].(type) {
				case []string:
					tags = v
				case []interface{}:
					for _, t := range v {
						if s, ok := t.(string); ok {
							tags = append(tags, s)
						}
					}
				case string:
					// comma-separated string
					if v != "" {
						tags = append(tags, v)
					}
				case []byte:
					// comma-separated string
					str := string(v)
					if str != "" {
						tags = append(tags, str)
					}
				}
			}
		}
		results = append(results, PostRow{
			ID:       id,
			PosterID: posterID,
			Tags:     tags,
		})
	}

	return results, nil
}
