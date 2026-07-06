package handler

import "strings"

const (
	LinkTypeHLS  = "HLS (m3u8)"
	LinkTypeFile = "FILE"
)

// ClassifyDirectLink 判断直链是 HLS 还是文件型
func ClassifyDirectLink(urlStr string) string {
	lower := strings.ToLower(urlStr)
	if strings.Contains(lower, ".m3u8") ||
		strings.Contains(lower, "application/x-mpegurl") ||
		strings.Contains(lower, "application/vnd.apple.mpegurl") {
		return LinkTypeHLS
	}
	return LinkTypeFile
}

// DirectLinkMetaHint 根据直链类型给出元数据预期说明
func DirectLinkMetaHint(linkType string) string {
	if linkType == LinkTypeHLS {
		return "预计无文件元数据（时长/码率/分辨率）"
	}
	return "预计可获取文件元数据"
}
