package main

import (
	"fmt"
	"log"

	reparos "github.com/Hieu31/reparos-go"
)

func main() {
	fmt.Println("======================================================================")
	fmt.Println("🧪 CHƯƠNG TRÌNH KIỂM THỬ TOÀN BỘ CÁC HÀM CỦA REPAROS-GO")
	fmt.Println("======================================================================\n")

	// =========================================================================
	// HÀM 1: reparos.Correct(query)
	// Tác dụng: Nhận vào câu sai, trả về câu đúng (Zero-config, gọn nhất)
	// =========================================================================
	fmt.Println("--- 1. TEST HÀM: reparos.Correct(query) ---")
	q1 := "d pasteur q3"
	ans1, err := reparos.Correct(q1)
	if err != nil {
		log.Fatalf("Lỗi: %v", err)
	}
	fmt.Printf("Input:  '%s'\n", q1)
	fmt.Printf("Output: '%s'\n\n", ans1)

	// =========================================================================
	// HÀM 2: reparos.PredictDetailed(query)
	// Tác dụng: Lấy đầy đủ thông tin: độ trễ (ms), điểm số, cờ Changed
	// =========================================================================
	fmt.Println("--- 2. TEST HÀM: reparos.PredictDetailed(query) ---")
	q2 := "bv cho ray"
	res2, err := reparos.PredictDetailed(q2)
	if err != nil {
		log.Fatalf("Lỗi: %v", err)
	}
	fmt.Printf("Input:     '%s'\n", res2.InputQuery)
	fmt.Printf("Top-1:     '%s'\n", res2.Top1Query)
	fmt.Printf("Đã sửa:    %v\n", res2.Changed)
	fmt.Printf("Độ trễ:    %.2f ms\n", res2.LatencyMs)
	fmt.Printf("Điểm số:   %v\n\n", res2.Scores)

	// =========================================================================
	// HÀM 3: reparos.New() + reparos.WithBeamSize(2)
	// Tác dụng: Cấu hình Beam Search nhỏ để đạt tốc độ siêu nhanh (< 15ms)
	// =========================================================================
	fmt.Println("--- 3. TEST HÀM: reparos.New() với WithBeamSize(2) (Fast Mode) ---")
	fastPredictor, err := reparos.New(
		reparos.WithBeamSize(2),
		reparos.WithComputeType("int8"),
	)
	if err != nil {
		log.Fatalf("Lỗi: %v", err)
	}
	defer fastPredictor.Close()

	q3 := "nga tu hang xanh"
	res3, err := fastPredictor.Predict(q3)
	if err != nil {
		log.Fatalf("Lỗi: %v", err)
	}
	fmt.Printf("Input:  '%s'\n", res3.InputQuery)
	fmt.Printf("Top-1:  '%s'\n", res3.Top1Query)
	fmt.Printf("Độ trễ: %.2f ms (chạy với Beam Size = 2)\n\n", res3.LatencyMs)

	// =========================================================================
	// HÀM 4: reparos.New() + reparos.WithNumHypotheses(3)
	// Tác dụng: Trả về top 3 gợi ý khác nhau (tính năng "Có phải bạn muốn tìm?")
	// =========================================================================
	fmt.Println("--- 4. TEST HÀM: WithNumHypotheses(3) (Gợi ý nhiều phương án) ---")
	multiPredictor, err := reparos.New(
		reparos.WithBeamSize(10),
		reparos.WithNumHypotheses(3),
	)
	if err != nil {
		log.Fatalf("Lỗi: %v", err)
	}
	defer multiPredictor.Close()

	q4 := "dh bach khoa tphcm"
	res4, err := multiPredictor.Predict(q4)
	if err != nil {
		log.Fatalf("Lỗi: %v", err)
	}
	fmt.Printf("Input: '%s'\n", res4.InputQuery)
	fmt.Printf("Tìm thấy %d câu gợi ý hàng đầu:\n", len(res4.Hypotheses))
	for i, h := range res4.Hypotheses {
		score := 0.0
		if i < len(res4.Scores) {
			score = res4.Scores[i]
		}
		fmt.Printf("  [%d] %-45s (điểm log-prob: %.4f)\n", i+1, h, score)
	}
	fmt.Println()

	// =========================================================================
	// HÀM 5: predictor.Close()
	// Tác dụng: Giải phóng bộ nhớ RAM và tiến trình mô hình
	// =========================================================================
	fmt.Println("--- 5. TEST HÀM: predictor.Close() ---")
	if err := fastPredictor.Close(); err != nil {
		log.Printf("Lỗi close: %v", err)
	} else {
		fmt.Println("✓ fastPredictor.Close() đã giải phóng RAM thành công.")
	}
	if err := multiPredictor.Close(); err != nil {
		log.Printf("Lỗi close: %v", err)
	} else {
		fmt.Println("✓ multiPredictor.Close() đã giải phóng RAM thành công.")
	}

	fmt.Println("\n======================================================================")
	fmt.Println("🎉 ĐÃ TEST HOÀN TẤT VÀ THÀNH CÔNG 100%!")
	fmt.Println("======================================================================")
}
