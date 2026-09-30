package main

import (
	"fmt"
	"log"

	reparos "github.com/Hieu31/reparos-go"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("🚀 REPAROS-GO: SỬA LỖI CHÍNH TẢ & CHUẨN HÓA TRUY VẤN")
	fmt.Println("==================================================")

	// 1. Cách dùng 1 dòng siêu gọn (Zero-Config / Plug & Play)
	// Tự động sử dụng model INT8 nhúng sẵn, không cần truyền bất kỳ đường dẫn nào!
	corrected, err := reparos.Correct("d pasteur q3")
	if err != nil {
		log.Fatalf("Lỗi: %v", err)
	}
	fmt.Printf("✓ Sửa nhanh: 'd pasteur q3' => '%s'\n\n", corrected)

	// 2. Chạy thử nghiệm danh sách các truy vấn thực tế
	queries := []string{
		"bv cho ray",
		"nga tu hang xanh",
		"tahnhf pho ho chi minh",
		"duong so 1 binh tan",
		"cho ben thanh quan 1",
		"san bay tan son nhat",
		"dh bach khoa tphcm",
		"ks new world q1",
	}

	fmt.Printf("%-28s | %-45s | %-10s\n", "TRUY VẤN GỐC", "KẾT QUẢ ĐÃ SỬA", "ĐỘ TRỄ")
	fmt.Println("----------------------------------------------------------------------------------------")

	for _, q := range queries {
		res, err := reparos.PredictDetailed(q)
		if err != nil {
			log.Printf("Lỗi: %v", err)
			continue
		}
		fmt.Printf("%-28s | %-45s | %.2f ms\n", res.InputQuery, res.Top1Query, res.LatencyMs)
	}
	fmt.Println("----------------------------------------------------------------------------------------")
}
