package pagination

import (
	"math"
	"strings"

	"gorm.io/gorm"
)

type SortOrder struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type Pagination struct {
	Limit      int                    `json:"limit,omitempty" query:"limit"`
	Page       int                    `json:"page,omitempty" query:"page"`
	Sort       []SortOrder            `json:"sort,omitempty" query:"-"`
	TotalRows  int64                  `json:"total_rows"`
	TotalPages int                    `json:"total_pages"`
	Rows       interface{}            `json:"rows"`
	Filters    map[string]interface{} `json:"filters,omitempty" query:"-"` 
}

func (p *Pagination) GetOffset() int {
	return (p.GetPage() - 1) * p.GetLimit()
}

func (p *Pagination) GetLimit() int {
	if p.Limit == 0 {
		p.Limit = 10
	}
	return p.Limit
}

func (p *Pagination) GetPage() int {
	if p.Page == 0 {
		p.Page = 1
	}
	return p.Page
}

func (p *Pagination) GetSort() []SortOrder {
	if p.Sort == nil || len(p.Sort) == 0 {
		p.Sort = []SortOrder{{Field: "id", Direction: "desc"}}
	}
	return p.Sort
}

func ApplyFilters(db *gorm.DB, filters map[string]interface{}) *gorm.DB {
	if filters == nil {
		return db
	}

	tx := db
	for key, value := range filters {
		switch v := value.(type) {
		case string:
			if v != "" {
				tx = tx.Where(key+" LIKE ?", "%"+v+"%")
			}
		case []interface{}:
			if len(v) > 0 {
				tx = tx.Where(key+" IN ?", v)
			}
		case map[string]interface{}:
			for op, val := range v {
				switch op {
				case "eq":
					tx = tx.Where(key+" = ?", val)
				case "neq":
					tx = tx.Where(key+" != ?", val)
				case "gt":
					tx = tx.Where(key+" > ?", val)
				case "gte":
					tx = tx.Where(key+" >= ?", val)
				case "lt":
					tx = tx.Where(key+" < ?", val)
				case "lte":
					tx = tx.Where(key+" <= ?", val)
				case "like":
					tx = tx.Where(key+" LIKE ?", "%"+val.(string)+"%")
				case "in":
					tx = tx.Where(key+" IN ?", val)
				case "between":
					values := val.([]interface{})
					if len(values) == 2 {
						tx = tx.Where(key+" BETWEEN ? AND ?", values[0], values[1])
					}
				}
			}
		default:
			tx = tx.Where(key+" = ?", v)
		}
	}
	return tx
}

func ApplyDynamicSort(db *gorm.DB, sorts []SortOrder) *gorm.DB {
	if sorts == nil || len(sorts) == 0 {
		return db
	}

	tx := db
	for _, sort := range sorts {
		direction := strings.ToUpper(sort.Direction)
		if direction != "ASC" && direction != "DESC" {
			direction = "ASC"
		}

		tx = tx.Order(sort.Field + " " + direction)
	}
	return tx
}

func Paginate(value interface{}, pagination *Pagination, db *gorm.DB, preloads ...string) func(db *gorm.DB) *gorm.DB {
	var totalRows int64

	countQuery := db.Model(value)
	if pagination.Filters != nil && len(pagination.Filters) > 0 {
		countQuery = ApplyFilters(countQuery, pagination.Filters)
	}
	countQuery.Count(&totalRows)

	pagination.TotalRows = totalRows
	totalPages := int(math.Ceil(float64(totalRows) / float64(pagination.GetLimit())))
	pagination.TotalPages = totalPages

	return func(db *gorm.DB) *gorm.DB {
		tx := db

		if pagination.Filters != nil && len(pagination.Filters) > 0 {
			tx = ApplyFilters(tx, pagination.Filters)
		}

		tx = tx.Offset(pagination.GetOffset()).Limit(pagination.GetLimit())

		if len(preloads) > 0 {
			for _, preload := range preloads {
				tx = tx.Preload(preload)
			}
		}

		return ApplyDynamicSort(tx, pagination.GetSort())
	}
}


func ParseSortFromQuery(query string) []SortOrder {
	if query == "" {
		return nil
	}

	sortParams := strings.Split(query, ",")
	sortOrders := make([]SortOrder, 0, len(sortParams))

	for _, param := range sortParams {
		parts := strings.Split(strings.TrimSpace(param), ":")

		if len(parts) == 2 {
			field := parts[0]
			direction := strings.ToLower(parts[1])

			if direction != "asc" && direction != "desc" {
				direction = "asc" 
			}

			sortOrders = append(sortOrders, SortOrder{
				Field:     field,
				Direction: direction,
			})
		} else if len(parts) == 1 && parts[0] != "" {
			sortOrders = append(sortOrders, SortOrder{
				Field:     parts[0],
				Direction: "asc",
			})
		}
	}

	return sortOrders
}
