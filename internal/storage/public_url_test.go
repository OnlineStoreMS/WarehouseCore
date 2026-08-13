package storage

import "testing"

func TestResolveProductAndWarehouseBuckets(t *testing.T) {
	r := &PublicURLResolver{minioRoot: "https://osms.zfcycle.com/minio"}
	cases := []struct {
		in, want string
	}{
		{
			"http://192.168.3.41:9100/productcore/uploads/products/1.jpg",
			"https://osms.zfcycle.com/minio/productcore/uploads/products/1.jpg",
		},
		{
			"http://192.168.3.41:9100/warehousecore/attachments/skus/a.png",
			"https://osms.zfcycle.com/minio/warehousecore/attachments/skus/a.png",
		},
		{
			"https://osms.zfcycle.com/minio/productcore/uploads/x.jpg",
			"https://osms.zfcycle.com/minio/productcore/uploads/x.jpg",
		},
		{
			"https://example.com/foo",
			"https://example.com/foo",
		},
	}
	for _, c := range cases {
		if got := r.Resolve(c.in); got != c.want {
			t.Fatalf("Resolve(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestResolveList(t *testing.T) {
	r := &PublicURLResolver{minioRoot: "https://osms.zfcycle.com/minio"}
	in := "http://192.168.3.41:9100/productcore/uploads/a.jpg,http://192.168.3.41:9100/productcore/uploads/b.jpg"
	want := "https://osms.zfcycle.com/minio/productcore/uploads/a.jpg,https://osms.zfcycle.com/minio/productcore/uploads/b.jpg"
	if got := r.ResolveList(in); got != want {
		t.Fatalf("ResolveList()=%q want %q", got, want)
	}
}
