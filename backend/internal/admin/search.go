package admin

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func searchPattern(c *gin.Context) string {
	query := []rune(strings.TrimSpace(c.Query("q")))
	if len(query) > 100 {
		query = query[:100]
	}
	if len(query) == 0 {
		return ""
	}
	return "%" + strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(string(query)) + "%"
}
