package handler
import "net/http"
func GetAPIKey(r *http.Request) string {
if v := r.URL.Query().Get("api_key"); v != "" { return v }
if v := r.Header.Get("X-Emby-Token"); v != "" { return v }
if v := r.Header.Get("X-MediaBrowser-Token"); v != "" { return v }
return ""
}
