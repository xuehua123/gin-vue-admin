package router

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/handler"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket" // For WebSocket client in test
	"github.com/stretchr/testify/assert"
)

// TestInitNFCRelayRouter tests the InitNFCRelayRouter function.
// This covers documentation point 12.3.1.
func TestInitNFCRelayRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("SuccessfulRouteRegistration", func(t *testing.T) {
		engine := gin.New()
		rootGroup := engine.Group("")

		InitNFCRelayRouter(rootGroup)

		expectedHandlerName := runtime.FuncForPC(reflect.ValueOf(handler.WSConnectionHandler).Pointer()).Name()
		foundRoute := false
		for _, route := range engine.Routes() {
			t.Logf("Checking route: Method=%s, Path=%s, Handler=%s", route.Method, route.Path, route.Handler)
			if route.Method == "GET" && route.Path == "/nfc/relay" {
				// Compare the handler name (or a more robust way if available)
				assert.Equal(t, expectedHandlerName, route.Handler, "Handler for /nfc/relay should be handler.WSConnectionHandler")
				foundRoute = true
				break
			}
		}
		assert.True(t, foundRoute, "Expected route GET /nfc/relay not found")
	})

	t.Run("NilParentGroup", func(t *testing.T) {
		// Gin's Group method on a nil *RouterGroup will panic.
		// InitNFCRelayRouter currently does not guard against this.
		// This test verifies the current behavior (panic).
		// If InitNFCRelayRouter were changed to handle nil parentGroup gracefully (e.g., return an error or no-op),
		// this test would need to be updated accordingly.
		assert.Panics(t, func() {
			InitNFCRelayRouter(nil)
		}, "InitNFCRelayRouter should panic if parentGroup is nil")
	})
}

// TestInitNFCRelayRouter_WebSocketUpgrade tests if the route set up by InitNFCRelayRouter
// correctly attempts a WebSocket upgrade. This covers documentation point 12.3.2.
func TestInitNFCRelayRouter_WebSocketUpgrade(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	InitNFCRelayRouter(engine.Group("")) // Initialize routes on the engine

	server := httptest.NewServer(engine)
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/nfc/relay"

	// Attempt to connect. We are not testing the full WS communication here,
	// only that the handshake is attempted by the handler.
	// The handler.WSConnectionHandler might have its own dependencies (like GlobalRelayHub)
	// which might not be fully initialized in a minimal test setup, potentially leading to
	// errors *after* the HTTP 101 response, or the handler might return an HTTP error
	// if it cannot initialize itself.
	// For this router test, a successful HTTP 101 or a specific WebSocket handshake error
	// (rather than a 404) indicates the routing worked.
	dialer := websocket.Dialer{}
	conn, resp, err := dialer.Dial(wsURL, nil)

	if err != nil {
		// If Dial returns an error, check the HTTP response if available.
		// A non-404 status suggests the handler was reached.
		// e.g., handler.WSConnectionHandler itself might return 500 if GlobalRelayHub is nil.
		// Or, if CheckOrigin fails, gorilla/websocket might return an error with resp.StatusCode == http.StatusForbidden
		t.Logf("WebSocket dial error: %v", err)
		if resp != nil {
			t.Logf("HTTP response status: %s", resp.Status)
			assert.NotEqual(t, http.StatusNotFound, resp.StatusCode, "Handler should be reached, not result in 404 Not Found. Status: "+resp.Status)
			// If WSConnectionHandler is basic and only calls upgrader.Upgrade, without full hub logic,
			// it might still return 101 if upgrader.Upgrade is the first thing it does.
			// However, it's more likely that if the Hub or other dependencies aren't ready,
			// WSConnectionHandler might return an HTTP error (e.g., 500) *before* upgrading.
			// This assertion is a bit loose because the behavior depends on WSConnectionHandler's robustness
			// to uninitialized global state in a test environment.
		} else {
			// If resp is nil, it might be a connection error before HTTP response (e.g., server not listening)
			// which shouldn't happen with httptest.Server. Or a more fundamental dial issue.
			assert.Fail(t, "WebSocket dial failed without an HTTP response, error: %v", err)
		}
	} else {
		// If err is nil, the handshake was successful (HTTP 101)
		assert.NotNil(t, conn, "Connection should not be nil on successful dial")
		defer conn.Close()
		if resp != nil { // resp should be nil on successful upgrade with gorilla/websocket's Dial
			t.Logf("Unexpected HTTP response on successful dial: %s", resp.Status)
		}
		// Further check if the connection is usable could be added, but might be too much for a router test.
		// For now, successful dial implies routing to the handler and successful upgrade initiation.
	}

	// Note: The actual success of this test (especially the error case) heavily depends on
	// the implementation of handler.WSConnectionHandler and how it behaves when its
	// dependencies (like GlobalRelayHub or logging) might not be fully set up in a unit test context.
	// If handler.WSConnectionHandler always tries to upgrade first, then `err` should be `nil`
	// or a specific websocket handshake error (e.g. bad origin if CheckOrigin is strict).
}
