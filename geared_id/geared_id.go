package geared_id

import (
	"strconv"

	"github.com/bwmarrin/snowflake"
)

func GearedIntID() int {
	n, err := snowflake.NewNode(1)
	if err != nil {
		return 0
	}
	id, err := strconv.Atoi(n.Generate().String())
	if err != nil {
		return 0
	}
	return id
}

func GearedStringID() string {
	n, err := snowflake.NewNode(1)
	if err != nil {
		return ""
	}
	return n.Generate().String()
}
