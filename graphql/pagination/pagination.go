package pagination

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

type PageInfo struct {
	HasNextPage     bool
	HasPreviousPage bool
	StartCursor     *string
	EndCursor       *string
}

type Edge struct {
	Cursor string
	Node   interface{}
}

type Connection struct {
	Edges      []*Edge
	PageInfo   *PageInfo
	TotalCount *int32
}

type CursorPaginationInput struct {
	First  *int32
	After  *string
	Last   *int32
	Before *string
}

type OffsetPaginationInput struct {
	Limit  *int32
	Offset *int32
}

func EncodeCursor(id string) string {
	return base64.StdEncoding.EncodeToString([]byte(id))
}

func DecodeCursor(cursor string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return "", fmt.Errorf("invalid cursor: %w", err)
	}
	return string(decoded), nil
}

func EncodeOffsetCursor(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("offset:%d", offset)))
}

func DecodeOffsetCursor(cursor string) (int, error) {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, fmt.Errorf("invalid cursor: %w", err)
	}
	
	parts := strings.Split(string(decoded), ":")
	if len(parts) != 2 || parts[0] != "offset" {
		return 0, fmt.Errorf("invalid offset cursor format")
	}
	
	offset, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid offset value: %w", err)
	}
	
	return offset, nil
}

func NewConnection(items []interface{}, hasMore bool, cursorFn func(interface{}) string) *Connection {
	edges := make([]*Edge, len(items))
	
	for i, item := range items {
		edges[i] = &Edge{
			Cursor: cursorFn(item),
			Node:   item,
		}
	}
	
	pageInfo := &PageInfo{
		HasNextPage: hasMore,
	}
	
	if len(edges) > 0 {
		startCursor := edges[0].Cursor
		endCursor := edges[len(edges)-1].Cursor
		pageInfo.StartCursor = &startCursor
		pageInfo.EndCursor = &endCursor
	}
	
	return &Connection{
		Edges:    edges,
		PageInfo: pageInfo,
	}
}

func NewConnectionWithTotal(items []interface{}, total int, hasMore bool, cursorFn func(interface{}) string) *Connection {
	conn := NewConnection(items, hasMore, cursorFn)
	totalInt32 := int32(total)
	conn.TotalCount = &totalInt32
	return conn
}

func ValidateCursorPagination(input *CursorPaginationInput, maxLimit int32) error {
	if input.First != nil && input.Last != nil {
		return fmt.Errorf("cannot specify both first and last")
	}
	
	if input.First != nil {
		if *input.First < 0 {
			return fmt.Errorf("first must be non-negative")
		}
		if *input.First > maxLimit {
			return fmt.Errorf("first cannot exceed %d", maxLimit)
		}
	}
	
	if input.Last != nil {
		if *input.Last < 0 {
			return fmt.Errorf("last must be non-negative")
		}
		if *input.Last > maxLimit {
			return fmt.Errorf("last cannot exceed %d", maxLimit)
		}
	}
	
	if input.After != nil && input.Before != nil {
		return fmt.Errorf("cannot specify both after and before")
	}
	
	return nil
}

func ValidateOffsetPagination(input *OffsetPaginationInput, maxLimit int32) error {
	if input.Limit != nil {
		if *input.Limit < 0 {
			return fmt.Errorf("limit must be non-negative")
		}
		if *input.Limit > maxLimit {
			return fmt.Errorf("limit cannot exceed %d", maxLimit)
		}
	}
	
	if input.Offset != nil && *input.Offset < 0 {
		return fmt.Errorf("offset must be non-negative")
	}
	
	return nil
}

func GetLimitOffset(input *CursorPaginationInput, defaultLimit int32) (limit, offset int, err error) {
	limit = int(defaultLimit)
	offset = 0
	
	if input.First != nil {
		limit = int(*input.First)
	}
	
	if input.After != nil {
		offset, err = DecodeOffsetCursor(*input.After)
		if err != nil {
			return 0, 0, err
		}
		offset++ 
	}
	
	limit++
	
	return limit, offset, nil
}

const DefaultLimit int32 = 20

const MaxLimit int32 = 100
