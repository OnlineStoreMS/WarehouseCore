package storage

import (
	"net/url"
	"strings"

	"warehousecore/internal/config"
)

// PublicURLResolver 将库内已存 MinIO 对象 URL 解析为当前公网入口。
// PublicBaseURL 形如 https://domain/minio/warehousecore，会剥掉末段 bucket，
// 把 http://内网:9100/{bucket}/... 重写为 https://domain/minio/{bucket}/...
// （商品图常来自 productcore 桶，附件在 warehousecore 桶，必须保留原 bucket。）
type PublicURLResolver struct {
	minioRoot string
}

func NewPublicURLResolver(cfg *config.StorageConfig) *PublicURLResolver {
	base := strings.TrimRight(cfg.PublicBaseURL, "/")
	root := base
	if i := strings.LastIndex(base, "/"); i > 0 {
		root = base[:i]
	}
	return &PublicURLResolver{minioRoot: root}
}

func (r *PublicURLResolver) Resolve(stored string) string {
	stored = strings.TrimSpace(stored)
	if stored == "" || r.minioRoot == "" {
		return stored
	}
	if !strings.HasPrefix(stored, "http://") && !strings.HasPrefix(stored, "https://") {
		return stored
	}
	u, err := url.Parse(stored)
	if err != nil || u.Host == "" {
		return stored
	}
	path := strings.TrimPrefix(u.Path, "/")
	if path == "" {
		return stored
	}
	// 已是公网路径 https://domain/minio/{bucket}/...
	if strings.HasPrefix(path, "minio/") {
		return r.minioRoot + "/" + strings.TrimPrefix(path, "minio/")
	}
	if !looksLikeMinIOObjectURL(u, path) {
		return stored
	}
	return r.minioRoot + "/" + path
}

func looksLikeMinIOObjectURL(u *url.URL, path string) bool {
	if strings.HasSuffix(u.Host, ":9100") || strings.Contains(strings.ToLower(u.Host), "minio") {
		return true
	}
	// path-style: {bucket}/uploads|attachments/...
	if strings.Contains(path, "/uploads/") || strings.HasPrefix(path, "uploads/") {
		return true
	}
	if strings.Contains(path, "/attachments/") || strings.HasPrefix(path, "attachments/") {
		return true
	}
	first, _, _ := strings.Cut(path, "/")
	switch first {
	case "productcore", "supplycore", "aftersalescore", "storecore", "warehousecore",
		"shippingcore", "box-edge", "materialcore", "todocenter", "selfcore", "mallcore":
		return true
	}
	return false
}

// ResolveList 重写逗号/空白分隔的多 URL（albumPics）。
func (r *PublicURLResolver) ResolveList(stored string) string {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return stored
	}
	parts := strings.FieldsFunc(stored, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	})
	if len(parts) <= 1 && !strings.ContainsAny(stored, ",;") {
		return r.Resolve(stored)
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, r.Resolve(p))
	}
	return strings.Join(out, ",")
}
