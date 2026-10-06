package admin

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSearchPatternEscapesWildcardsAndCapsLength(t *testing.T) {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/?q=%25_%5C", nil)
	if got := searchPattern(context); got != `%\%\_\\%` {
		t.Fatalf("pattern=%q", got)
	}
	context, _ = gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/?q="+strings.Repeat("x", 150), nil)
	if got := searchPattern(context); len([]rune(got)) != 102 {
		t.Fatalf("pattern length=%d", len([]rune(got)))
	}
}
