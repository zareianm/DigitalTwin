package jobService

import (
	"errors"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

func SaveFile(fh *multipart.FileHeader) (string, error) {

	viper.SetConfigName("appsettings")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("../config")
	_ = viper.ReadInConfig()

	baseUploadDir := viper.GetString("baseUploadDir")

	baseUploadDir = filepath.Clean(baseUploadDir)

	id := uuid.New().String()
	ext := safeExt(fh.Filename)
	dst := filepath.Join(baseUploadDir, id+ext)

	// Defensive: make sure dst is really inside baseUploadDir
	absBase, _ := filepath.Abs(baseUploadDir)
	absDst, _ := filepath.Abs(dst)

	var err error

	if !strings.HasPrefix(absDst, absBase+string(os.PathSeparator)) && absDst != absBase {
		err = errors.New("invalid destination path")
		return "", err
	}

	// Create the folder if someone deleted it while the app is running
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		err = errors.New("cannot create destination dir")
		return "", err
	}

	if err := saveUploadedFile(fh, dst); err != nil {
		err = errors.New("could not save file")
		return "", err
	}

	return dst, nil
}

func saveUploadedFile(fileHeader *multipart.FileHeader, dst string) error {

	src, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// Create destination file (0600 so only the server user can read/write)
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = ioCopy(out, src)
	return err
}

// Use a tiny wrapper to allow easy testing/mocking if desired.
var ioCopy = func(dst *os.File, src multipart.File) (int64, error) {
	return dst.ReadFrom(src)
}

// Extract a safe, lowercase extension from the original filename.
func safeExt(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	// Whitelist a few common ones if you want stricter control:
	// allowed := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".pdf": true, ".txt": true}
	// if !allowed[ext] { return "" }
	return ext
}
