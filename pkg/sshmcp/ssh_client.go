package sshmcp

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

// CreateSSHClient creates an SSH client with the given parameters
func CreateSSHClient(host string, port int, username string, authConfig *AuthConfig, timeout time.Duration) (*ssh.Client, error) {
	authMethods, err := authConfig.AuthMethod()
	if err != nil {
		return nil, fmt.Errorf("create auth method: %w", err)
	}

	config := &ssh.ClientConfig{
		User:            username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应该使用 known_hosts 验证
		Timeout:         timeout,
		Config: ssh.Config{
			KeyExchanges: []string{
				"curve25519-sha256",
				"curve25519-sha256@libssh.org",
				"ecdh-sha2-nistp256",
				"ecdh-sha2-nistp384",
				"ecdh-sha2-nistp521",
				"diffie-hellman-group14-sha256",
				"diffie-hellman-group16-sha512",
			},
			Ciphers: []string{
				"chacha20-poly1305@openssh.com",
				"aes128-gcm@openssh.com",
				"aes256-gcm@openssh.com",
				"aes128-ctr",
				"aes192-ctr",
				"aes256-ctr",
			},
		},
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("dial SSH server %s: %w", addr, err)
	}

	return client, nil
}

// AuthMethod creates SSH authentication methods based on the auth config
func (ac *AuthConfig) AuthMethod() ([]ssh.AuthMethod, error) {
	switch ac.Type {
	case AuthTypePassword:
		return []ssh.AuthMethod{ssh.Password(ac.Password)}, nil

	case AuthTypePrivateKey:
		return ac.createPrivateKeyAuth()

	case AuthTypeSSHAgent:
		return ac.createSSHAgentAuth()

	case AuthTypeKeyboard:
		return []ssh.AuthMethod{ssh.KeyboardInteractive(ac.keyboardChallenge)}, nil

	default:
		return nil, fmt.Errorf("unsupported auth type: %s", ac.Type)
	}
}

// createPrivateKeyAuth creates authentication using private key
func (ac *AuthConfig) createPrivateKeyAuth() ([]ssh.AuthMethod, error) {
	// 读取私钥文件
	var keyBytes []byte
	var err error

	// 检查是文件路径还是直接的内容
	if _, statErr := os.Stat(ac.PrivateKey); statErr == nil {
		// 是文件路径
		keyBytes, err = os.ReadFile(ac.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("read private key file: %w", err)
		}
	} else {
		// 是私钥内容
		keyBytes = []byte(ac.PrivateKey)
	}

	// 解析私钥
	var signer ssh.Signer
	if ac.Passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(ac.Passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(keyBytes)
	}
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
}

// createSSHAgentAuth creates authentication using SSH agent
func (ac *AuthConfig) createSSHAgentAuth() ([]ssh.AuthMethod, error) {
	// SSH Agent 认证暂时不支持，返回错误
	return nil, fmt.Errorf("SSH agent authentication is not yet implemented, please use password or private key authentication")
}

// keyboardChallenge handles keyboard-interactive authentication
func (ac *AuthConfig) keyboardChallenge(user, instruction string, questions []string, echos []bool) ([]string, error) {
	answers := make([]string, len(questions))
	for i := range answers {
		// 简化实现：使用密码作为所有问题的答案
		// 实际生产环境应该根据问题类型返回不同的答案
		answers[i] = ac.Password
	}
	return answers, nil
}

// TestConnection tests if an SSH connection is still alive
func TestConnection(client *ssh.Client) bool {
	if client == nil {
		return false
	}

	session, err := client.NewSession()
	if err != nil {
		return false
	}
	defer session.Close()

	// 执行一个简单的命令来测试连接
	err = session.Run("true")
	return err == nil
}

// SendKeepalive sends a keepalive request to keep the connection alive
func SendKeepalive(client *ssh.Client) error {
	if client == nil {
		return fmt.Errorf("nil client")
	}

	// 发送全局请求作为 keepalive
	_, _, err := client.SendRequest("keepalive@golang.org", true, nil)
	return err
}
