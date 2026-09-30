# ReparoS-Go 🇻🇳

Thư viện **Golang** hiệu năng cao phục vụ chuẩn hóa, sửa lỗi chính tả và hiểu truy vấn tiếng Việt (**Vietnamese Query Understanding & Spell Correction**), sử dụng mô hình nơ-ron Transformer **ReparoS Base v4 (CTranslate2 INT8)**.

Mô hình siêu nhẹ chỉ **~8.6 MB**, được tối ưu hóa bằng lượng hóa INT8 và SentencePiece, mang lại độ trễ xử lý cực thấp (**~15ms – 25ms** trên 1 core CPU) mà không cần GPU.

---

## 🌟 Tính Năng Nổi Bật

* **Sửa lỗi chính tả & viết tắt phức tạp**:
  * Tự động bù dấu tiếng Việt và sửa gõ sai Telex/VNI: `tahnhf pho ho chi minh` $\rightarrow$ `thành phố hồ chí minh`
  * Chuẩn hóa từ viết tắt địa danh/hành chính: `d pasteur q3` $\rightarrow$ `đường pasteur quận 3`
  * Nhận diện và sửa tên địa điểm lớn: `bv cho ray` $\rightarrow$ `bệnh viện chợ rẫy`, `dh bach khoa tphcm` $\rightarrow$ `đại học bách khoa thành phố hồ chí minh`
* **Hiệu năng & Tối ưu**:
  * Mô hình nén INT8 chỉ nặng **8.59 MB**.
  * Tiêu tốn cực ít RAM (~25–35 MB).
  * Độ trễ trung bình trên CPU: **~20ms** (với `beam_size = 10`).
* **Sẵn sàng triển khai**:
  * Có sẵn bộ runner kiểm thử nhanh trên Windows, Linux và macOS.
  * Cung cấp mã nguồn C++ Bridge Native (`bridge/`) cho những ai muốn build shared library `.so`/`.dll` đạt hiệu năng tối đa.

---

## 📂 Cấu Trúc Dự Án

```text
reparos-go/
├── .github/workflows/      # CI/CD tự động test trên Linux & Windows
├── bridge/                 # C++ Native Bridge (CTranslate2 & SentencePiece C++ API)
│   ├── bridge.h
│   ├── bridge.cpp
│   ├── CMakeLists.txt
│   ├── build.sh
│   └── build.bat
├── example/                # Chương trình Go demo mẫu
│   └── main.go
├── internal/runner/        # Engine inference runner
│   └── runner.py
├── models/v4_int8/         # Trọng số mô hình đã huấn luyện sẵn
│   ├── model.bin           (8.59 MB - CTranslate2 INT8 model)
│   ├── tokenizer.model     (453 KB - SentencePiece)
│   ├── config.json
│   └── vocabularies...
├── go.mod
├── options.go              # Cấu hình Beam Search, Device, Quantization
├── reparos.go              # API thư viện Go chính (Predictor)
├── reparos_test.go         # Bộ unit test Go
└── README.md
```

---

## 🚀 Bắt Đầu Nhanh

### 1. Yêu Cầu Môi Trường
* **Go**: 1.21 trở lên.
* **Python**: 3.9+ (có thư viện `ctranslate2` và `sentencepiece`):
  ```bash
  pip install ctranslate2 sentencepiece
  ```

### 2. Chạy Demo
Chạy trực tiếp ví dụ trong thư mục `example/`:
```bash
go run example/main.go
```

**Kết quả mẫu trên màn hình:**
```text
==================================================
🚀 KHỞI TẠO MÔ HÌNH REPAROS-GO (BASE V4 INT8)
==================================================
✓ Nạp mô hình thành công trong 828ms

TRUY VẤN GỐC (INPUT)           | KẾT QUẢ ĐÃ SỬA (TOP-1)                        | ĐỘ TRỄ    
---------------------------------------------------------------------------------------------
d pasteur q3                   | đường pasteur quận 3                          | 23.13 ms
bv cho ray                     | bệnh viện chợ rẫy                             | 19.02 ms
nga tu hang xanh               | ngã tư hàng xanh                              | 20.07 ms
duong so 1 binh tan            | đường số 1 bình tân                           | 24.85 ms
cho ben thanh quan 1           | chợ bến thành quận 1                          | 24.19 ms
san bay tan son nhat           | sân bay quốc tế tân sơn nhất                  | 32.30 ms
dh bach khoa tphcm             | đại học bách khoa thành phố hồ chí minh       | 29.69 ms
quan 1 hcm                     | quận 1 thành phố hcm                          | 28.73 ms
ks new world q1                | ks new world quận 1                           | 20.69 ms
---------------------------------------------------------------------------------------------
✓ Kiểm tra hoàn tất 10 truy vấn. Độ trễ trung bình: 26.49 ms
```

### 3. Chạy Kiểm Thử Unit Test
```bash
go test -v ./...
```

---

## 💻 Cách Dùng Trong Dự Án Của Bạn

```go
package main

import (
	"fmt"
	"log"

	reparos "github.com/username/reparos-go"
)

func main() {
	// Khởi tạo Predictor với đường dẫn thư mục model
	predictor, err := reparos.New("models/v4_int8",
		reparos.WithBeamSize(10),       // Độ rộng tìm kiếm (mặc định: 10)
		reparos.WithComputeType("int8"), // Sử dụng lượng hóa INT8
	)
	if err != nil {
		log.Fatalf("Không thể khởi tạo mô hình: %v", err)
	}
	defer predictor.Close()

	// Sửa lỗi chính tả truy vấn
	res, err := predictor.Predict("bv cho ray")
	if err != nil {
		log.Fatalf("Lỗi dự đoán: %v", err)
	}

	fmt.Println("Truy vấn gốc:", res.InputQuery)
	fmt.Println("Kết quả sửa:", res.Top1Query)  // "bệnh viện chợ rẫy"
	fmt.Printf("Độ trễ: %.2f ms\n", res.LatencyMs)
}
```

---

## 📤 Hướng Dẫn Đẩy Lên Repo GitHub Riêng

Thư mục `reparos-go` này được thiết kế **hoàn toàn độc lập**. Bạn có thể đẩy nó lên GitHub thành một repository công khai:

```bash
cd reparos-go

# 1. Khởi tạo Git
git init

# 2. Đổi tên module trong go.mod theo username GitHub của bạn
# (Mở go.mod sửa "github.com/username/reparos-go" thành username của bạn)

# 3. Thêm các file và commit
git add .
git commit -m "feat: initial commit for reparos-go library"

# 4. Liên kết với repository GitHub của bạn
git branch -M main
git remote add origin https://github.com/<YOUR_USERNAME>/<YOUR_REPO_NAME>.git

# 5. Push lên GitHub
git push -u origin main
```

---

## ⚡ Tùy Chọn: Biên Dịch C++ Native Bridge (Tùy Chọn Cho Production)

Nếu bạn muốn nhúng trực tiếp thư viện C++ thuần qua file `.so` (Linux) hoặc `.dll` (Windows):

1. Cài đặt CTranslate2 và SentencePiece C++ libraries.
2. Chạy script biên dịch trong thư mục `bridge/`:
   * **Linux**: `./bridge/build.sh`
   * **Windows**: `.\bridge\build.bat`
3. Khi khởi tạo trong Go, truyền thêm `reparos.WithNativeLib("bridge/libreparos_bridge.so")`.

---

## 📄 Bản Quyền & Giấy Phép
Dự án được phân phối dưới giấy phép mã nguồn mở MIT.
