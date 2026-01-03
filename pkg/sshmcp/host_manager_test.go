package sshmcp

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// initTestLogger 创建一个用于测试的 logger
func initTestLogger() *zerolog.Logger {
	// 创建一个输出到 /dev/null 的 logger，避免测试输出过多日志
	logger := zerolog.New(io.Discard)
	return &logger
}

// TestHostManager_NewHostManager tests creating a new host manager
func TestHostManager_NewHostManager(t *testing.T) {
	// 创建一个临时配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// 创建初始配置
	hostsConfig := map[string]HostConfig{
		"prod": {
			Host:     "192.168.1.100",
			Port:     22,
			Username: "root",
			Password: "secret",
		},
	}

	hm := NewHostManager(hostsConfig, configPath, nil)

	assert.NotNil(t, hm)
	assert.Equal(t, 1, len(hm.ListHosts()))
}

// TestHostManager_ListHosts tests listing hosts
func TestHostManager_ListHosts(t *testing.T) {
	hostsConfig := map[string]HostConfig{
		"prod": {
			Host:     "192.168.1.100",
			Port:     22,
			Username: "root",
			Password: "secret",
		},
		"staging": {
			Host:           "staging.example.com",
			Port:           22,
			Username:       "deploy",
			PrivateKeyPath: "/path/to/key",
		},
	}

	hm := NewHostManager(hostsConfig, "", nil)

	hosts := hm.ListHosts()
	assert.Equal(t, 2, len(hosts))
	assert.Equal(t, "192.168.1.100", hosts["prod"].Host)
	assert.Equal(t, "staging.example.com", hosts["staging"].Host)
}

// TestHostManager_GetHost tests retrieving a host by name
func TestHostManager_GetHost(t *testing.T) {
	hostsConfig := map[string]HostConfig{
		"prod": {
			Host:     "192.168.1.100",
			Port:     22,
			Username: "root",
			Password: "secret",
		},
	}

	hm := NewHostManager(hostsConfig, "", nil)

	// 测试获取存在的主机
	host, err := hm.GetHost("prod")
	assert.NoError(t, err)
	assert.Equal(t, "192.168.1.100", host.Host)
	assert.Equal(t, 22, host.Port)
	assert.Equal(t, "root", host.Username)

	// 测试获取不存在的主机
	_, err = hm.GetHost("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestHostManager_HostExists tests checking if a host exists
func TestHostManager_HostExists(t *testing.T) {
	hostsConfig := map[string]HostConfig{
		"prod": {
			Host:     "192.168.1.100",
			Port:     22,
			Username: "root",
		},
	}

	hm := NewHostManager(hostsConfig, "", nil)

	assert.True(t, hm.HostExists("prod"))
	assert.False(t, hm.HostExists("staging"))
	assert.False(t, hm.HostExists(""))
}

// TestHostManager_SaveHost tests saving a new host
func TestHostManager_SaveHost(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// 创建初始配置文件
	initialConfig := `server:
  name: test
  version: 1.0.0
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	assert.NoError(t, err)

	logger := initTestLogger()
	hm := NewHostManager(map[string]HostConfig{}, configPath, logger)

	// 测试保存新主机
	hostConfig := HostConfig{
		Host:     "192.168.1.200",
		Port:     2222,
		Username: "admin",
		Password: "adminpass",
	}

	err = hm.SaveHost("newhost", hostConfig)
	assert.NoError(t, err)

	// 验证主机已保存
	assert.True(t, hm.HostExists("newhost"))

	host, err := hm.GetHost("newhost")
	assert.NoError(t, err)
	assert.Equal(t, "192.168.1.200", host.Host)
	assert.Equal(t, 2222, host.Port)
	assert.Equal(t, "admin", host.Username)

	// 测试保存重复主机名称
	err = hm.SaveHost("newhost", HostConfig{
		Host:     "another.host",
		Port:     22,
		Username: "user",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// 测试缺少必需字段
	err = hm.SaveHost("invalid", HostConfig{
		Port:     22,
		Username: "user",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "host address cannot be empty")

	err = hm.SaveHost("invalid2", HostConfig{
		Host: "some.host",
		Port: 22,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username cannot be empty")
}

// TestHostManager_RemoveHost tests removing a host
func TestHostManager_RemoveHost(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// 创建初始配置文件
	initialConfig := `server:
  name: test
  version: 1.0.0
hosts:
  prod:
    host: "192.168.1.100"
    port: 22
    username: root
    password: secret
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	assert.NoError(t, err)

	hostsConfig := map[string]HostConfig{
		"prod": {
			Host:     "192.168.1.100",
			Port:     22,
			Username: "root",
			Password: "secret",
		},
	}

	logger := initTestLogger()
	hm := NewHostManager(hostsConfig, configPath, logger)

	// 测试删除存在的主机
	err = hm.RemoveHost("prod")
	assert.NoError(t, err)
	assert.False(t, hm.HostExists("prod"))

	// 测试删除不存在的主机
	err = hm.RemoveHost("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestHostManager_Persist tests configuration persistence
func TestHostManager_Persist(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// 创建初始配置文件
	initialConfig := `server:
  name: test
  version: 1.0.0
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	assert.NoError(t, err)

	logger := initTestLogger()
	hm := NewHostManager(map[string]HostConfig{}, configPath, logger)

	// 保存多个主机
	hosts := []struct {
		name string
		cfg  HostConfig
	}{
		{
			name: "prod",
			cfg: HostConfig{
				Host:     "192.168.1.100",
				Port:     22,
				Username: "root",
				Password: "secret",
			},
		},
		{
			name: "staging",
			cfg: HostConfig{
				Host:           "staging.example.com",
				Port:           2222,
				Username:       "deploy",
				PrivateKeyPath: "/path/to/key",
				Description:    "Staging server",
			},
		},
	}

	for _, h := range hosts {
		err = hm.SaveHost(h.name, h.cfg)
		assert.NoError(t, err)
	}

	// 读取配置文件验证内容
	data, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	content := string(data)

	assert.Contains(t, content, "prod:")
	assert.Contains(t, content, "192.168.1.100")
	assert.Contains(t, content, "staging:")
	assert.Contains(t, content, "staging.example.com")
	assert.Contains(t, content, "Staging server")
}

// TestHostManager_DefaultPort tests that default port is set correctly
func TestHostManager_DefaultPort(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	err := os.WriteFile(configPath, []byte("server: {}\n"), 0644)
	assert.NoError(t, err)

	logger := initTestLogger()
	hm := NewHostManager(map[string]HostConfig{}, configPath, logger)

	// 保存不指定端口的主机
	hostConfig := HostConfig{
		Host:     "example.com",
		Port:     0,
		Username: "user",
	}

	err = hm.SaveHost("test", hostConfig)
	assert.NoError(t, err)

	host, err := hm.GetHost("test")
	assert.NoError(t, err)
	assert.Equal(t, 22, host.Port) // 默认端口应该是 22
}

// TestHostManager_ConcurrentAccess tests concurrent access to host manager
func TestHostManager_ConcurrentAccess(t *testing.T) {
	hostsConfig := map[string]HostConfig{
		"existing": {
			Host:     "192.168.1.1",
			Port:     22,
			Username: "root",
		},
	}

	hm := NewHostManager(hostsConfig, "", nil)

	// 并发读取
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_ = hm.HostExists("existing")
			_, _ = hm.GetHost("existing")
			_ = hm.ListHosts()
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证数据一致性
	assert.True(t, hm.HostExists("existing"))
	assert.Equal(t, 1, len(hm.ListHosts()))
}

// TestHostConfig_EmptyFields tests host config with various field combinations
func TestHostConfig_EmptyFields(t *testing.T) {
	tests := []struct {
		name     string
		config   HostConfig
		valid    bool
		expected HostConfig
	}{
		{
			name: "password auth",
			config: HostConfig{
				Host:     "example.com",
				Port:     22,
				Username: "user",
				Password: "pass",
			},
			valid: true,
			expected: HostConfig{
				Host:     "example.com",
				Port:     22,
				Username: "user",
				Password: "pass",
			},
		},
		{
			name: "private key auth",
			config: HostConfig{
				Host:           "example.com",
				Port:           22,
				Username:       "user",
				PrivateKeyPath: "/path/to/key",
			},
			valid: true,
			expected: HostConfig{
				Host:           "example.com",
				Port:           22,
				Username:       "user",
				PrivateKeyPath: "/path/to/key",
			},
		},
		{
			name: "with description",
			config: HostConfig{
				Host:        "example.com",
				Port:        22,
				Username:    "user",
				Password:    "pass",
				Description: "Test server",
			},
			valid: true,
			expected: HostConfig{
				Host:        "example.com",
				Port:        22,
				Username:    "user",
				Password:    "pass",
				Description: "Test server",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				assert.Equal(t, tt.expected.Host, tt.config.Host)
				assert.Equal(t, tt.expected.Port, tt.config.Port)
				assert.Equal(t, tt.expected.Username, tt.config.Username)
			}
		})
	}
}

// TestHostManager_FormatHostList tests formatting host list
func TestHostManager_FormatHostList(t *testing.T) {
	t.Run("empty hosts", func(t *testing.T) {
		hm := NewHostManager(map[string]HostConfig{}, "", nil)
		output := hm.FormatHostList()
		assert.Contains(t, output, "No predefined hosts configured")
	})

	t.Run("with hosts", func(t *testing.T) {
		hostsConfig := map[string]HostConfig{
			"prod": {
				Host:        "192.168.1.100",
				Port:        22,
				Username:    "root",
				Password:    "secret",
				Description: "Production server",
			},
			"staging": {
				Host:           "staging.example.com",
				Port:           2222,
				Username:       "deploy",
				PrivateKeyPath: "/path/to/key",
			},
		}

		hm := NewHostManager(hostsConfig, "", nil)
		output := hm.FormatHostList()

		assert.Contains(t, output, "prod:")
		assert.Contains(t, output, "192.168.1.100:22")
		assert.Contains(t, output, "Production server")
		assert.Contains(t, output, "Auth: password")

		assert.Contains(t, output, "staging:")
		assert.Contains(t, output, "staging.example.com:2222")
		assert.Contains(t, output, "Auth: private key")
	})
}
