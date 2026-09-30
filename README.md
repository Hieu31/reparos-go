# ReparoS-Go 🇻🇳

Thư viện Go và công cụ dòng lệnh (CLI) phục vụ **sửa lỗi chính tả, bù dấu tiếng Việt và chuẩn hóa địa chỉ/truy vấn tìm kiếm** (Vietnamese Query Understanding & Spell Correction), sử dụng mô hình nơ-ron Transformer **ReparoS Base v4 (CTranslate2 INT8)**.

Mô hình siêu nhẹ chỉ **~8.6 MB**, được nhúng trực tiếp vào thư viện (**`//go:embed`**), cho độ trễ xử lý cực nhanh (**~15ms – 25ms** trên 1 core CPU).

---

## 📋 Yêu Cầu Hệ Thống (Prerequisites)

* **Go**: 1.21 trở lên.
* **Python**: 3.9+ có sẵn thư viện `ctranslate2` và `sentencepiece`:
  ```bash
  pip install ctranslate2 sentencepiece
  ```

---

## 🚀 Cách 1: Sử Dụng Trong Dự Án Go Của Bạn

### 1. Cài đặt thư viện:
Trong thư mục dự án Go của bạn, chạy lệnh:
```bash
go get github.com/Hieu31/reparos-go
```

### 2. Viết code sử dụng:
Tạo file `main.go`:
```go
package main

import (
	"fmt"
	reparos "github.com/Hieu31/reparos-go"
)

func main() {
	// Gọi trực tiếp hàm Correct (tự động nạp mô hình nhúng sẵn)
	res, err := reparos.Correct("d pasteur q3")
	if err != nil {
		panic(err)
	}

	fmt.Println(res)
	// Output: đường pasteur quận 3
}
```

### 3. Chạy chương trình:
```bash
go run main.go
```

---

## 🛠️ Cách 2: Cài Đặt Làm Công Cụ Gõ Lệnh (CLI Tool)

Bạn có thể cài đặt công cụ `reparos` vào máy tính để dùng trực tiếp trong Terminal mà không cần viết code:

### 1. Cài đặt lệnh toàn cầu:
```bash
go install github.com/Hieu31/reparos-go/cmd/reparos@latest
```

### 2. Sử dụng ở bất kỳ đâu trong Terminal:
```bash
# Sửa một câu:
reparos "bv cho ray"
# Output: bệnh viện chợ rẫy

# Xuất kết quả chi tiết kèm độ trễ dưới dạng JSON:
reparos -json "d pasteur q3"

# Mở chế độ gõ tương tác (REPL):
reparos
```

---

## ⚙️ Các Tùy Chọn Cấu Hình Nâng Cao

Nếu bạn cần tinh chỉnh sâu hơn (như lấy nhiều gợi ý hoặc tối ưu tốc độ):

```go
package main

import (
	"fmt"
	reparos "github.com/Hieu31/reparos-go"
)

func main() {
	// Khởi tạo Predictor với cấu hình tùy chọn
	predictor, err := reparos.New(
		reparos.WithBeamSize(10),     // Độ rộng Beam Search (mặc định: 10)
		reparos.WithNumHypotheses(3), // Lấy top 3 câu gợi ý tốt nhất
	)
	if err != nil {
		panic(err)
	}
	defer predictor.Close()

	res, _ := predictor.Predict("dh bach khoa tphcm")

	fmt.Printf("Top 1: %s (Độ trễ: %.2f ms)\n", res.Top1Query, res.LatencyMs)
	fmt.Println("Các gợi ý khác:", res.Hypotheses)
}
```

### Danh sách các Option:
| Hàm Option | Mặc định | Tác dụng |
| :--- | :---: | :--- |
| `reparos.WithBeamSize(n)` | `10` | Độ rộng Beam Search. Giảm xuống 2–3 để đạt tốc độ < 10ms (cho autocomplete). |
| `reparos.WithNumHypotheses(n)` | `1` | Số lượng câu gợi ý trả về trong `res.Hypotheses`. |
| `reparos.WithComputeType("int8")` | `"int8"` | Kiểu lượng hóa: `"int8"`, `"float16"`, `"float32"`. |
| `reparos.WithDevice("cpu")` | `"cpu"` | Thiết bị tính toán: `"cpu"` hoặc `"cuda"` (nếu có GPU). |

---

## ⚡ Kết Quả Thực Nghiệm

Đo đạc thực tế trên CPU đơn luồng (Intel / AMD thông thường):

| Truy vấn đầu vào | Kết quả sau khi sửa | Độ trễ (Latency) |
| :--- | :--- | :---: |
| `bv cho ray` | **bệnh viện chợ rẫy** | **14.50 ms** |
| `d pasteur q3` | **đường pasteur quận 3** | **18.20 ms** |
| `nga tu hang xanh` | **ngã tư hàng xanh** | **16.44 ms** |
| `duong so 1 binh tan` | **đường số 1 bình tân** | **27.69 ms** |
| `san bay tan son nhat` | **sân bay quốc tế tân sơn nhất** | **27.37 ms** |

---

## 📄 License
Phân phối theo giấy phép mã nguồn mở MIT.
