package main

import (
	"fmt"
	"log"

	"reparos"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("🚀 REPAROS-GO: SỬA LỖI CHÍNH TẢ & CHUẨN HÓA TRUY VẤN")
	fmt.Println("==================================================")

	corrected, err := reparos.Correct("d pasteur q3")
	if err != nil {
		log.Fatalf("Lỗi: %v", err)
	}
	fmt.Printf("✓ Sửa nhanh: 'd pasteur q3' => '%s'\n\n", corrected)

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
