package deliveryhandler

import (
	"context"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"zentora-service/internal/domain/delivery"
	"zentora-service/internal/middleware"
)

type fakeRepository struct {
	calls    int
	conflict bool
}

func (r *fakeRepository) Get(context.Context) (*delivery.Policy, error) {
	return &delivery.Policy{}, nil
}
func (r *fakeRepository) Update(_ context.Context, u delivery.Update, adminID int64) (*delivery.Policy, error) {
	r.calls++
	if r.conflict {
		return nil, delivery.ErrConflict
	}
	return &delivery.Policy{Version: u.ExpectedVersion + 1}, nil
}

func TestAdminDeliveryValidationAndPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name, role, body string
		conflict         bool
		status, calls    int
	}{
		{"customer", "customer", `{"expected_version":1,"nairobi_indicative_fee":300}`, false, 403, 0},
		{"unknown field", "admin", `{"expected_version":1,"nairobi_indicative_fee":300,"included_in_order_total":true}`, false, 400, 0},
		{"negative", "admin", `{"expected_version":1,"nairobi_indicative_fee":-1}`, false, 400, 0},
		{"fraction", "admin", `{"expected_version":1,"nairobi_indicative_fee":3.5}`, false, 400, 0},
		{"multiple bodies", "admin", `{"expected_version":1,"nairobi_indicative_fee":300} {}`, false, 400, 0},
		{"stale", "admin", `{"expected_version":1,"nairobi_indicative_fee":300}`, true, 409, 1},
		{"valid", "admin", `{"expected_version":1,"nairobi_indicative_fee":300}`, false, 200, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepository{conflict: tt.conflict}
			h := NewHandler(repo, zap.NewNop())
			router := gin.New()
			auth := middleware.NewAuthMiddleware(nil)
			router.PUT("/", func(c *gin.Context) { c.Set("identity_id", int64(42)); c.Set("roles", []string{tt.role}) }, auth.RequireRole("admin", "super_admin"), h.Update)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			if w.Code != tt.status || repo.calls != tt.calls {
				t.Fatalf("status=%d calls=%d body=%s", w.Code, repo.calls, w.Body.String())
			}
		})
	}
}
