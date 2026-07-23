package controller

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

const maxThumbnailBytes = 5 << 20

var errInvalidThumbnail = errors.New("thumbnail must be a PNG, JPEG, WEBP, or GIF image smaller than 5 MB")

func readThumbnail(c *gin.Context) ([]byte, string, error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxThumbnailBytes+(1<<20))
	file, _, err := c.Request.FormFile("thumbnail")
	if err != nil {
		return nil, "", errInvalidThumbnail
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxThumbnailBytes+1))
	if err != nil || len(content) == 0 || len(content) > maxThumbnailBytes {
		return nil, "", errInvalidThumbnail
	}
	contentType := http.DetectContentType(content)
	switch contentType {
	case "image/png", "image/jpeg", "image/webp", "image/gif":
		return bytes.Clone(content), contentType, nil
	default:
		return nil, "", errInvalidThumbnail
	}
}

func writeThumbnailError(c *gin.Context, err error) {
	if errors.Is(err, errInvalidThumbnail) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid thumbnail upload"})
}
