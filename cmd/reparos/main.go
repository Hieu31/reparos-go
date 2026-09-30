package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	reparos "github.com/Hieu31/reparos-go"
)

const version = "0.1.0"

func main() {
	jsonOutput := flag.Bool("json", false, "Xuất kết quả dưới định dạng JSON")
	beamSize := flag.Int("beam", 10, "Độ rộng tìm kiếm Beam Search")
	numHypo := flag.Int("n", 1, "Số lượng kết quả gợi ý trả về")
	showVersion := flag.Bool("version", false, "Hiển thị phiên bản")

	flag.Parse()

	if *showVersion {
		fmt.Printf("ReparoS-Go CLI version %s\n", version)
		return
	}

	// Khởi tạo Predictor
	predictor, err := reparos.New(
		reparos.WithBeamSize(*beamSize),
		reparos.WithNumHypotheses(*numHypo),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Lỗi khởi tạo mô hình: %v\n", err)
		os.Exit(1)
	}
	defer predictor.Close()

	args := flag.Args()

	// 1. Chế độ dòng lệnh đơn (Truyền tham số câu vào trực tiếp)
	// Ví dụ: reparos "d pasteur q3"
	if len(args) > 0 {
		query := strings.Join(args, " ")
		res, err := predictor.Predict(query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Lỗi dự đoán: %v\n", err)
			os.Exit(1)
		}

		if *jsonOutput {
			data, _ := json.MarshalIndent(res, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Println(res.Top1Query)
		}
		return
	}

	// 2. Chế độ Tương tác (Interactive REPL) khi không truyền tham số
	fmt.Printf("🇻🇳 ReparoS CLI v%s - Sửa Lỗi Chính Tả & Chuẩn Hóa Truy Vấn Tiếng Việt\n", version)
	fmt.Println("Nhập câu cần sửa và nhấn Enter (Gõ 'exit' hoặc 'quit' để thoát):")
	fmt.Println("-----------------------------------------------------------------------")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("📝 > ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			fmt.Println("Tạm biệt!")
			break
		}

		res, err := predictor.Predict(line)
		if err != nil {
			fmt.Printf("❌ Lỗi: %v\n", err)
			continue
		}

		if *jsonOutput {
			data, _ := json.MarshalIndent(res, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Printf("✨ => %s  (%.2f ms)\n", res.Top1Query, res.LatencyMs)
		}
	}
}
