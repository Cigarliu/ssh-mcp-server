package sshmcp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
)

// UploadFile uploads a file to the remote host
func (s *Session) UploadFile(localPath, remotePath string, createDirs, overwrite bool) (*FileTransferResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LastUsedAt = time.Now()

	startTime := time.Now()

	// 检查本地文件
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return &FileTransferResult{Error: fmt.Errorf("stat local file: %w", err)}, err
	}

	// 如果是目录，递归上传
	if fileInfo.IsDir() {
		return s.uploadDirectory(localPath, remotePath, createDirs, overwrite)
	}

	// 检查远程文件是否存在
	remoteFileExists := false
	if fi, err := s.SFTPClient.Stat(remotePath); err == nil {
		remoteFileExists = true
		if !overwrite && fi != nil {
			return &FileTransferResult{
				Error: fmt.Errorf("remote file already exists: %s (use overwrite=true to overwrite)", remotePath),
			}, fmt.Errorf("file exists")
		}
	}

	// 如果需要，创建远程目录
	if createDirs {
		remoteDir := filepath.Dir(remotePath)
		if err := s.SFTPClient.MkdirAll(remoteDir); err != nil {
			return &FileTransferResult{Error: fmt.Errorf("create remote directory: %w", err)}, err
		}
	}

	// 打开本地文件
	localFile, err := os.Open(localPath)
	if err != nil {
		return &FileTransferResult{Error: fmt.Errorf("open local file: %w", err)}, err
	}
	defer localFile.Close()

	// 创建远程文件
	var remoteFile *sftp.File
	if remoteFileExists && overwrite {
		remoteFile, err = s.SFTPClient.OpenFile(remotePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE)
	} else {
		remoteFile, err = s.SFTPClient.Create(remotePath)
	}
	if err != nil {
		return &FileTransferResult{Error: fmt.Errorf("create remote file: %w", err)}, err
	}
	defer remoteFile.Close()

	// 复制文件内容
	bytesTransferred, err := io.Copy(remoteFile, localFile)
	if err != nil {
		return &FileTransferResult{Error: fmt.Errorf("copy file content: %w", err)}, err
	}

	duration := time.Since(startTime)

	return &FileTransferResult{
		Status:           "success",
		BytesTransferred: bytesTransferred,
		Duration:         duration.String(),
	}, nil
}

// uploadDirectory uploads a directory recursively
func (s *Session) uploadDirectory(localPath, remotePath string, createDirs, overwrite bool) (*FileTransferResult, error) {
	var totalBytes int64
	startTime := time.Now()

	// 遍历本地目录
	err := filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(localPath, path)
		if err != nil {
			return err
		}

		remoteFilePath := filepath.Join(remotePath, relPath)

		if info.IsDir() {
			// 创建远程目录
			if err := s.SFTPClient.MkdirAll(remoteFilePath); err != nil {
				return fmt.Errorf("create remote directory %s: %w", remoteFilePath, err)
			}
			return nil
		}

		// 上传文件
		result, err := s.UploadFile(path, remoteFilePath, false, overwrite)
		if err != nil {
			return err
		}
		totalBytes += result.BytesTransferred

		return nil
	})

	if err != nil {
		return &FileTransferResult{Error: err}, err
	}

	duration := time.Since(startTime)

	return &FileTransferResult{
		Status:           "success",
		BytesTransferred: totalBytes,
		Duration:         duration.String(),
	}, nil
}

// DownloadFile downloads a file from the remote host
func (s *Session) DownloadFile(remotePath, localPath string, createDirs, overwrite bool) (*FileTransferResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LastUsedAt = time.Now()

	startTime := time.Now()

	// 检查远程文件
	fileInfo, err := s.SFTPClient.Stat(remotePath)
	if err != nil {
		return &FileTransferResult{Error: fmt.Errorf("stat remote file: %w", err)}, err
	}

	// 如果是目录，递归下载
	if fileInfo.IsDir() {
		return s.downloadDirectory(remotePath, localPath, createDirs, overwrite)
	}

	// 检查本地文件是否存在
	localFileExists := false
	if _, err := os.Stat(localPath); err == nil {
		localFileExists = true
		if !overwrite {
			return &FileTransferResult{
				Error: fmt.Errorf("local file already exists: %s (use overwrite=true to overwrite)", localPath),
			}, fmt.Errorf("file exists")
		}
	}

	// 如果需要，创建本地目录
	if createDirs {
		localDir := filepath.Dir(localPath)
		if err := os.MkdirAll(localDir, 0755); err != nil {
			return &FileTransferResult{Error: fmt.Errorf("create local directory: %w", err)}, err
		}
	}

	// 打开远程文件
	remoteFile, err := s.SFTPClient.Open(remotePath)
	if err != nil {
		return &FileTransferResult{Error: fmt.Errorf("open remote file: %w", err)}, err
	}
	defer remoteFile.Close()

	// 创建本地文件
	var localFile *os.File
	if localFileExists && overwrite {
		localFile, err = os.OpenFile(localPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	} else {
		localFile, err = os.Create(localPath)
	}
	if err != nil {
		return &FileTransferResult{Error: fmt.Errorf("create local file: %w", err)}, err
	}
	defer localFile.Close()

	// 复制文件内容
	bytesTransferred, err := io.Copy(localFile, remoteFile)
	if err != nil {
		return &FileTransferResult{Error: fmt.Errorf("copy file content: %w", err)}, err
	}

	duration := time.Since(startTime)

	return &FileTransferResult{
		Status:           "success",
		BytesTransferred: bytesTransferred,
		Duration:         duration.String(),
	}, nil
}

// downloadDirectory downloads a directory recursively
func (s *Session) downloadDirectory(remotePath, localPath string, createDirs, overwrite bool) (*FileTransferResult, error) {
	var totalBytes int64
	startTime := time.Now()

	// 遍历远程目录
	walker := s.SFTPClient.Walk(remotePath)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			return &FileTransferResult{Error: err}, err
		}

		path := walker.Path()
		info := walker.Stat()

		// 计算相对路径
		relPath, err := filepath.Rel(remotePath, path)
		if err != nil {
			return &FileTransferResult{Error: err}, err
		}

		localFilePath := filepath.Join(localPath, relPath)

		if info.IsDir() {
			// 创建本地目录
			if err := os.MkdirAll(localFilePath, 0755); err != nil {
				return &FileTransferResult{Error: fmt.Errorf("create local directory %s: %w", localFilePath, err)}, err
			}
			continue
		}

		// 下载文件
		result, err := s.DownloadFile(path, localFilePath, false, overwrite)
		if err != nil {
			return &FileTransferResult{Error: err}, err
		}
		totalBytes += result.BytesTransferred
	}

	duration := time.Since(startTime)

	return &FileTransferResult{
		Status:           "success",
		BytesTransferred: totalBytes,
		Duration:         duration.String(),
	}, nil
}

// ListDirectory lists the contents of a remote directory
func (s *Session) ListDirectory(remotePath string, recursive bool) ([]FileInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LastUsedAt = time.Now()

	var files []FileInfo

	if recursive {
		// 递归列出
		walker := s.SFTPClient.Walk(remotePath)
		for walker.Step() {
			if err := walker.Err(); err != nil {
				return nil, fmt.Errorf("walk remote directory: %w", err)
			}

			path := walker.Path()
			info := walker.Stat()

			// 跳过根目录
			if path == remotePath {
				continue
			}

			fileInfo := FileInfo{
				Name:     filepath.Base(path),
				Type:     getFileType(info),
				Size:     info.Size(),
				Mode:     info.Mode().String(),
				Modified: info.ModTime(),
			}
			files = append(files, fileInfo)
		}
	} else {
		// 非递归列出
		entries, err := s.SFTPClient.ReadDir(remotePath)
		if err != nil {
			return nil, fmt.Errorf("read remote directory: %w", err)
		}

		for _, entry := range entries {
			fileInfo := FileInfo{
				Name:     entry.Name(),
				Type:     getFileType(entry),
				Size:     entry.Size(),
				Mode:     entry.Mode().String(),
				Modified: entry.ModTime(),
			}
			files = append(files, fileInfo)
		}
	}

	return files, nil
}

// getFileType returns the file type as a string
func getFileType(info os.FileInfo) string {
	if info.IsDir() {
		return "directory"
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "symlink"
	}
	return "file"
}

// MakeDirectory creates a remote directory
func (s *Session) MakeDirectory(remotePath string, recursive bool, mode os.FileMode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LastUsedAt = time.Now()

	if recursive {
		return s.SFTPClient.MkdirAll(remotePath)
	}

	return s.SFTPClient.Mkdir(remotePath)
}

// RemoveFile removes a remote file or directory
func (s *Session) RemoveFile(remotePath string, recursive bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LastUsedAt = time.Now()

	// 检查文件类型
	info, err := s.SFTPClient.Stat(remotePath)
	if err != nil {
		return fmt.Errorf("stat remote path: %w", err)
	}

	if info.IsDir() {
		// 删除目录
		if recursive {
			return s.SFTPClient.RemoveAll(remotePath)
		}
		return s.SFTPClient.RemoveDirectory(remotePath)
	}

	// 删除文件
	return s.SFTPClient.Remove(remotePath)
}

// GetFileInfo gets information about a remote file
func (s *Session) GetFileInfo(remotePath string) (*FileInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LastUsedAt = time.Now()

	info, err := s.SFTPClient.Stat(remotePath)
	if err != nil {
		return nil, fmt.Errorf("stat remote file: %w", err)
	}

	return &FileInfo{
		Name:     filepath.Base(remotePath),
		Type:     getFileType(info),
		Size:     info.Size(),
		Mode:     info.Mode().String(),
		Modified: info.ModTime(),
	}, nil
}
