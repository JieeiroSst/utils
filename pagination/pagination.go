package pagination

import (
	"math"
	"strings"

	"gorm.io/gorm"
)

type Pagination struct {
	Limit      int                    `json:"limit,omitempty" query:"limit"`
	Page       int                    `json:"page,omitempty" query:"page"`
	Sort       string                 `json:"sort,omitempty" query:"sort"`
	TotalRows  int64                  `json:"total_rows"`
	TotalPages int                    `json:"total_pages"`
	Rows       interface{}            `json:"rows"`
	Filters    map[string]interface{} `json:"filters,omitempty" query:"-"` // Dynamic filters
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

func (p *Pagination) GetSort() string {
	if p.Sort == "" {
		p.Sort = "id desc"
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

func ApplyDynamicSort(db *gorm.DB, sort string) *gorm.DB {
	if sort == "" {
		return db
	}

	sortFields := strings.Split(sort, ",")
	for _, field := range sortFields {
		field = strings.TrimSpace(field)
		if field != "" {
			direction := "ASC"
			if strings.HasPrefix(field, "-") {
				direction = "DESC"
				field = field[1:] 
			}

			db = db.Order(field + " " + direction)
		}
	}
	return db
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
