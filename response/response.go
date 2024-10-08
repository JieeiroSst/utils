package response

import (
	"github.com/gin-gonic/gin"
)

type MessageStatus struct {
	Message string      `json:"message,omitempty"`
	Error   bool        `json:"error"`
	Data    interface{} `json:"data,omitempty"`
	TraceID string      `json:"trace_id,omitempty"`
}

func ResponseStatus(c *gin.Context, code int, response MessageStatus) {
	if code != 200 {
		c.JSON(code, gin.H{
			"message":  response.Message,
			"error":    response.Error,
			"trace_id": response.TraceID,
		})
		return
	}

	c.JSON(code, gin.H{
		"trace_id": response.TraceID,
		"data":     response.Data,
	})
}
