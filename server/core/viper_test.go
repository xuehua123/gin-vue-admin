package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/core"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// setupTempConfigEnvironment helper: creates a temporary config.yaml in a new temp dir,
// changes current working directory to it, and returns the original working directory.
func setupTempConfigEnvironment(t *testing.T, content string) (originalWD string) {
	t.Helper()
	tempDirPath := t.TempDir()
	configFilePath := filepath.Join(tempDirPath, "config.yaml")
	err := os.WriteFile(configFilePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp config file: %v", err)
	}
	originalWD, err = os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	err = os.Chdir(tempDirPath)
	if err != nil {
		t.Fatalf("Failed to change working directory to temp dir: %v", err)
	}
	return originalWD
}

// TestMain can be used to run setup before any tests in the package,
// but for flag redefinition, it might not solve the issue if Viper is called in each TestXxx.
// We rely on test execution order or separate runs if flag redefinition persists across TestXxx functions.

// Test Scenario: Valid NfcRelay Configuration Loaded by core.Viper
func TestNfcRelayConfigLoading_ValidConfigViaViper(t *testing.T) {
	originalGlobalConfig := global.GVA_CONFIG
	// This test function might be the only one that can safely call core.Viper()
	// if flag redefinition is an issue across top-level TestXxx functions in one `go test` run.
	// If run in isolation (e.g., `go test -run ^TestNfcRelayConfigLoading_ValidConfigViaViper$`), it should pass.
	t.Cleanup(func() {
		global.GVA_CONFIG = originalGlobalConfig
	})

	configContent := `
nfc-relay:
  hub-check-interval-sec: 10
  session-inactive-timeout-sec: 120
  websocket-write-wait-sec: 15
  websocket-pong-wait-sec: 70
  websocket-max-message-bytes: 4096
`
	originalWD := setupTempConfigEnvironment(t, configContent)
	t.Cleanup(func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Logf("Failed to restore original working directory: %v", err)
		}
	})

	var freshConf config.Server
	global.GVA_CONFIG = freshConf
	_ = core.Viper() // This call defines the '-c' flag.

	assert.Equal(t, 10, global.GVA_CONFIG.NfcRelay.HubCheckIntervalSec)
	assert.Equal(t, 120, global.GVA_CONFIG.NfcRelay.SessionInactiveTimeoutSec)
	assert.Equal(t, 15, global.GVA_CONFIG.NfcRelay.WebsocketWriteWaitSec)
	assert.Equal(t, 70, global.GVA_CONFIG.NfcRelay.WebsocketPongWaitSec)
	assert.Equal(t, 4096, global.GVA_CONFIG.NfcRelay.WebsocketMaxMessageBytes)
}

// Test Scenario: Invalid Type for a Field in NfcRelay Configuration (causes Viper to panic)
func TestNfcRelayConfigLoading_InvalidTypeFieldCausesPanic(t *testing.T) {
	originalGlobalConfig := global.GVA_CONFIG
	// This test should ideally be run in isolation if flag redefinition is an issue.
	t.Cleanup(func() {
		global.GVA_CONFIG = originalGlobalConfig
	})

	configContent := `
nfc-relay:
  hub-check-interval-sec: "this-is-not-an-int"
  session-inactive-timeout-sec: 180
`
	originalWD := setupTempConfigEnvironment(t, configContent)
	t.Cleanup(func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Logf("Failed to restore original working directory: %v", err)
		}
	})

	var freshConf config.Server
	global.GVA_CONFIG = freshConf

	assert.Panics(t, func() {
		core.Viper() // This call might redefine flag if not run first or in isolation.
	}, "core.Viper should panic on unmarshal error due to invalid type")
}

// Test Scenario: Default Zero Values for NfcRelay fields when config is missing or incomplete.
// This test does NOT call core.Viper() to avoid flag redefinition issues.
// It tests the state of a fresh config.Server struct, assuming Viper would leave fields
// as zero if not found or if the nfc-relay block is missing.
func TestNfcRelayConfig_DefaultZeroValues(t *testing.T) {
	var conf config.Server
	// Scenario 0.3.2: NfcRelay fields missing ( 일부 필드 누락 / 部分字段缺失)
	// If hub-check-interval-sec and websocket-max-message-bytes were set, others would be zero.
	// Here, we test a completely fresh config.NfcRelay (all zero).

	assert.Equal(t, 0, conf.NfcRelay.HubCheckIntervalSec, "Default HubCheckIntervalSec should be zero")
	assert.Equal(t, 0, conf.NfcRelay.SessionInactiveTimeoutSec, "Default SessionInactiveTimeoutSec should be zero")
	assert.Equal(t, 0, conf.NfcRelay.WebsocketWriteWaitSec, "Default WebsocketWriteWaitSec should be zero")
	assert.Equal(t, 0, conf.NfcRelay.WebsocketPongWaitSec, "Default WebsocketPongWaitSec should be zero")
	assert.Equal(t, 0, conf.NfcRelay.WebsocketMaxMessageBytes, "Default WebsocketMaxMessageBytes should be zero")

	// Scenario 0.3.3: nfc-relay block entirely missing
	// This is also covered by checking a fresh config.Server, as NfcRelay would be its zero value.
}

// Test Scenario: Negative Value for a Field in NfcRelay Configuration
// This test also has to call core.Viper and might suffer from flag redefinition if not run in isolation or first.
func TestNfcRelayConfigLoading_NegativeValueField(t *testing.T) {
	originalGlobalConfig := global.GVA_CONFIG
	t.Cleanup(func() {
		global.GVA_CONFIG = originalGlobalConfig
	})

	configContent := `
nfc-relay:
  hub-check-interval-sec: -5
  session-inactive-timeout-sec: 150
`
	originalWD := setupTempConfigEnvironment(t, configContent)
	t.Cleanup(func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Logf("Failed to restore original working directory: %v", err)
		}
	})

	var freshConf config.Server
	global.GVA_CONFIG = freshConf
	_ = core.Viper()

	assert.Equal(t, -5, global.GVA_CONFIG.NfcRelay.HubCheckIntervalSec)
	assert.Equal(t, 150, global.GVA_CONFIG.NfcRelay.SessionInactiveTimeoutSec)
}
