package sshmcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateSSHClient_InvalidHost tests creating SSH client with invalid host
func TestCreateSSHClient_InvalidHost(t *testing.T) {
	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: "testpass",
	}

	// 使用一个不太可能被占用的端口
	client, err := CreateSSHClient("localhost", 9999, "testuser", authConfig, 1)
	assert.Error(t, err, "Expected error when connecting to non-existent server")
	assert.Nil(t, client, "Expected nil client on error")
}

// TestCreateSSHClient_Timeout tests SSH client timeout
func TestCreateSSHClient_Timeout(t *testing.T) {
	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: "testpass",
	}

	// 使用一个非常短的超时时间
	client, err := CreateSSHClient("192.0.2.1", 22, "testuser", authConfig, 1)
	assert.Error(t, err, "Expected timeout error")
	assert.Nil(t, client)
}

// TestTestConnection tests TestConnection function
func TestTestConnection_NilClient(t *testing.T) {
	result := TestConnection(nil)
	assert.False(t, result, "Expected false for nil client")
}

// TestSendKeepalive tests SendKeepalive with nil client
func TestSendKeepalive_NilClient(t *testing.T) {
	err := SendKeepalive(nil)
	assert.Error(t, err, "Expected error for nil client")
}

// TestAuthConfig_PrivateKeyParsing tests private key parsing
func TestAuthConfig_PrivateKeyParsing(t *testing.T) {
	tests := []struct {
		name        string
		privateKey  string
		passphrase  string
		expectError bool
	}{
		{
			name:        "empty private key",
			privateKey:  "",
			passphrase:  "",
			expectError: true,
		},
		{
			name:        "invalid private key format",
			privateKey:  "not-a-valid-key",
			passphrase:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authConfig := &AuthConfig{
				Type:       AuthTypePrivateKey,
				PrivateKey: tt.privateKey,
				Passphrase: tt.passphrase,
			}

			methods, err := authConfig.createPrivateKeyAuth()
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, methods)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, methods)
			}
		})
	}
}

// TestAuthConfig_SSHAgentNotImplemented tests SSH agent auth not implemented
func TestAuthConfig_SSHAgentNotImplemented(t *testing.T) {
	authConfig := &AuthConfig{
		Type: AuthTypeSSHAgent,
	}

	methods, err := authConfig.createSSHAgentAuth()
	assert.Error(t, err, "Expected error: SSH agent not implemented")
	assert.Nil(t, methods)
	assert.Contains(t, err.Error(), "not yet implemented")
}

// TestAuthConfig_KeyboardChallenge tests keyboard interactive challenge
func TestAuthConfig_KeyboardChallenge(t *testing.T) {
	authConfig := &AuthConfig{
		Type:     AuthTypeKeyboard,
		Password: "test-response",
	}

	answers, err := authConfig.keyboardChallenge("user", "instruction", []string{"Password:"}, []bool{false})
	require.NoError(t, err)
	assert.Len(t, answers, 1)
	assert.Equal(t, "test-response", answers[0])
}

// BenchmarkAuthMethods benchmarks auth method creation
func BenchmarkAuthMethods(b *testing.B) {
	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: "testpass",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = authConfig.AuthMethod()
	}
}
