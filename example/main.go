package main

import (
	"fmt"
	"log"
	"time"

	reparos "github.com/username/reparos-go"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("🚀 KHỞI TẠO MÔ HÌNH REPAROS-GO (BASE V4 INT8)")
	fmt.Println("==================================================")

	started := time.Now()
	// Khởi tạo Predictor trỏ tới thư mục models/v4_int8
	predictor, err := reparos.New("models/v4_int8",
		reparos.WithBeamSize(10),
		reparos.WithNumHypotheses(1),
		reparos.WithComputeType("int8"),
	)
	if err != nil {
		log.Fatalf("Lỗi khởi tạo mô hình: %v", err)
	}
	defer predictor.Close()

	fmt.Printf("✓ Nạp mô hình thành công trong %v\n\n", time.Since(started).Round(time.Millisecond))

	testQueries := []string{
		"d pasteur q3",
		"bv cho ray",
		"tahnhf pho ho chi minh",
		"nga tu hang xanh",
		"duong so 1 binh tan",
		"cho ben thanh quan 1",
		"san bay tan son nhat",
		"dh bach khoa tphcm",
		"quan 1 hcm",
		"ks new world q1",
	}

	fmt.Printf("%-30s | %-45s | %-10s\n", "TRUY VẤN GỐC (INPUT)", "KẾT QUẢ ĐÃ SỬA (TOP-1)", "ĐỘ TRỄ")
	fmt.Println("---------------------------------------------------------------------------------------------")

	var totalLatency float64
	for _, q := range testQueries {
		res, err := predictor.Predict(q)
		if err != nil {
			log.Printf("Lỗi xử lý câu '%s': %v", q, err)
			continue
		}

		totalLatency += res.LatencyMs
		fmt.Printf("%-30s | %-45s | %.2f ms\n", res.InputQuery, res.Top1Query, res.LatencyMs)
	}

	avgLatency := totalLatency / float64(len(testQueries))
	fmt.Println("---------------------------------------------------------------------------------------------")
	fmt.Printf("✓ Kiểm tra hoàn tất %d truy vấn. Độ trễ trung bình: %.2f ms\n", len(testQueries), avgLatency)
}
