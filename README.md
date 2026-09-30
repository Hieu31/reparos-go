# ReparoS-Go 🇻🇳

Thư viện & Công cụ dòng lệnh **Golang** hiệu năng cao phục vụ chuẩn hóa, sửa lỗi chính tả và hiểu truy vấn tiếng Việt (**Vietnamese Query Understanding & Spell Correction**), sử dụng mô hình mạng nơ-ron Transformer **ReparoS Base v4 (CTranslate2 INT8)**.

Mô hình siêu nhẹ chỉ **~8.6 MB**, được nhúng trực tiếp vào thư viện (**Zero-Config `//go:embed`**), mang lại độ trễ xử lý cực thấp (**~15ms – 25ms** trên 1 core CPU) mà không cần cấu hình đường dẫn file phức tạp.

---

## 🌟 Tính Năng Nổi Bật

* **Zero-Config (Plug & Play)**: Nhúng sẵn mô hình 8.6MB qua `//go:embed`. Gọi 1 hàm là chạy ngay, không cần tự copy file model.
* **Cung cấp CLI Tool tiện lợi**: Cài đặt bằng `go install` để sửa lỗi chính tả trực tiếp ngay trong Terminal.
* **Sửa lỗi chính tả & viết tắt phức tạp**:
  * Tự động bù dấu tiếng Việt và sửa lỗi Telex/VNI: `d pasteur q3` $\rightarrow$ `đường pasteur quận 3`
  * Chuẩn hóa từ viết tắt địa danh/hành chính: `bv cho ray` $\rightarrow$ `bệnh viện chợ rẫy`, `dh bach khoa tphcm` $\rightarrow$ `đại học bách khoa thành phố hồ chí minh`
* **Hiệu năng cao**: Độ trễ ~20ms trên CPU với Beam Search 10 nhánh, tiêu tốn chỉ ~25-35MB RAM.

---

## 💻 Cách Dùng Trong Code Go (Cực Kỳ Đơn Giản)

### Cách 1: Dùng 1 dòng (Zero-Config)
Người dùng tải về không cần truyền bất kỳ đường dẫn file nào:

```go
package main

import (
    "fmt"
    "reparos"
)

func main() {
    // Tự động sử dụng mô hình INT8 nhúng sẵn
    res, _ := reparos.Correct("d pasteur q3")
    fmt.Println(res) // "đường pasteur quận 3"
}
```

### Cách 2: Tùy biến tham số (Beam Size, Hypotheses)
```go
predictor, err := reparos.New(
    reparos.WithBeamSize(10),       // Độ rộng Beam Search
    reparos.WithNumHypotheses(3),   // Lấy top 3 câu gợi ý tốt nhất
)
defer predictor.Close()

res, _ := predictor.Predict("bv cho ray")
fmt.Printf("Top 1: %s (Độ trễ: %.2f ms)\n", res.Top1Query, res.LatencyMs)
```

---

## 🛠️ Cài Đặt Dưới Dạng Công Cụ Dòng Lệnh (CLI Tool)

Bạn có thể cài đặt công cụ `reparos` vào máy tính chỉ bằng 1 lệnh Go duy nhất:

```bash
go install https://github.com/Hieu31/reparos-go/cmd/reparos@latest
```

Sau khi cài đặt, bạn có thể gọi lệnh `reparos` ở bất kỳ đâu trong Terminal:

```bash
# Sửa một câu trực tiếp:
reparos "bv cho ray"
# Output: bệnh viện chợ rẫy

# Xuất kết quả kèm độ trễ dưới dạng JSON:
reparos -json "d pasteur q3"

# Mở chế độ gõ tương tác (REPL):
reparos
```
