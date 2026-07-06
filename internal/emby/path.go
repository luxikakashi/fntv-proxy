package emby

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var windowsDrivePattern = regexp.MustCompile(`^[A-Za-z]:`)

// extractItemID 从 URL 路径提取 ItemId（支持数字 ID 和 GUID）
func extractItemID(path string) string {
	path = strings.TrimPrefix(path, "/emby")
	parts := strings.Split(path, "/")
	for i, part := range parts {
		lower := strings.ToLower(part)
		if (lower == "videos" || lower == "audio" || lower == "items") && i+1 < len(parts) {
			itemID := parts[i+1]
			if itemID != "" && !isPathSegmentNotItemID(itemID) {
				return itemID
			}
		}
	}
	return ""
}

// extractPlaybackItemID 从 PlaybackInfo 路径提取 ItemId
func extractPlaybackItemID(path string) string {
	path = strings.TrimPrefix(path, "/emby")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		if strings.EqualFold(part, "Items") && i+1 < len(parts) {
			id := parts[i+1]
			if id != "" && !strings.EqualFold(id, "PlaybackInfo") {
				return id
			}
		}
	}
	return ""
}

func isPathSegmentNotItemID(s string) bool {
	lower := strings.ToLower(s)
	switch lower {
	case "stream", "universal", "original", "playbackinfo", "subtitles":
		return true
	}
	return strings.HasPrefix(lower, "stream.") || strings.HasPrefix(lower, "master.")
}

// getAPIKey 从请求中获取 Emby API Key
func getAPIKey(r *http.Request) (name, value string) {
	if v := r.URL.Query().Get("api_key"); v != "" {
		return "api_key", v
	}
	if v := r.Header.Get("X-Emby-Token"); v != "" {
		return "api_key", v
	}
	if v := r.Header.Get("X-MediaBrowser-Token"); v != "" {
		return "api_key", v
	}
	return "api_key", ""
}

// isLocalPath 判断是否为本地媒体路径
func isLocalPath(path string) bool {
	if path == "" {
		return false
	}
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
		return true
	}
	return windowsDrivePattern.MatchString(path)
}

// isRemoteURL 判断是否为远程 URL
func isRemoteURL(path string) bool {
	u, err := url.Parse(path)
	if err != nil {
		return false
	}
	return u.Host != "" && (u.Scheme == "http" || u.Scheme == "https")
}

// redirectToOriginal 将 stream/universal 请求重定向到 original
func redirectToOriginal(w http.ResponseWriter, r *http.Request) {
	newURI := strings.Replace(r.RequestURI, "stream", "original", 1)
	newURI = strings.Replace(newURI, "universal", "original", 1)
	newURI = strings.Replace(newURI, "Stream", "original", 1)
	newURI = strings.Replace(newURI, "Universal", "original", 1)
	http.Redirect(w, r, newURI, http.StatusFound)
}
