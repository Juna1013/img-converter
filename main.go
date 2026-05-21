package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

// resizeImage は最大幅 maxWidth に収まるようリサイズする（アスペクト比維持）
// すでに maxWidth 以下ならそのまま返す
func resizeImage(img image.Image, maxWidth int) image.Image {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if w <= maxWidth {
		return img
	}

	newW := maxWidth
	newH := h * maxWidth / w

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	return dst
}

// removeReflection は暗いピクセル（輝度が threshold 以下）を純黒に置き換える
// 画面の黒い背景に写り込んだ反射を除去する
func removeReflection(img image.Image, threshold uint8) *image.RGBA {
	bounds := img.Bounds()
	dst := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			// 輝度を計算 (ITU-R BT.601)
			luma := 0.299*float64(r8) + 0.587*float64(g8) + 0.114*float64(b8)

			if luma <= float64(threshold) {
				dst.Set(x, y, color.Black)
			} else {
				dst.Set(x, y, color.RGBA{r8, g8, b8, uint8(a >> 8)})
			}
		}
	}

	return dst
}

func compressJPEG(input, output string, quality, maxWidth int, reflectionThreshold uint8) error {
	inFile, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("open error: %w", err)
	}
	defer inFile.Close()

	img, err := jpeg.Decode(inFile)
	if err != nil {
		return fmt.Errorf("decode error: %w", err)
	}

	// リサイズ
	img = resizeImage(img, maxWidth)

	// 反射除去
	processed := removeReflection(img, reflectionThreshold)

	outFile, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("create error: %w", err)
	}
	defer outFile.Close()

	opts := &jpeg.Options{Quality: quality}
	if err := jpeg.Encode(outFile, processed, opts); err != nil {
		return fmt.Errorf("encode error: %w", err)
	}

	return nil
}

func main() {
	srcDir := filepath.Join("data", "original")
	outDir := filepath.Join("data", "compressed")
	quality := 70
	maxWidth := 1920
	// 反射除去の閾値（0-255）: この値以下の輝度のピクセルを純黒にする
	// 値を上げるほど強く除去するが、波形のデータも消える可能性あり
	var reflectionThreshold uint8 = 80

	var totalIn, totalOut int64
	count := 0

	fmt.Printf("Settings: quality=%d, maxWidth=%dpx, reflection threshold=%d\n\n",
		quality, maxWidth, reflectionThreshold)

	// data/original/ 以下を再帰的に走査
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(info.Name()))
		if ext != ".jpg" && ext != ".jpeg" {
			return nil
		}

		// 入力パスから出力パスを生成（ディレクトリ構造を維持）
		relPath, _ := filepath.Rel(srcDir, path)
		outputPath := filepath.Join(outDir, relPath)

		// 出力先のディレクトリを作成
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("mkdir error: %w", err)
		}

		fmt.Printf("Processing: %s ... ", relPath)

		if err := compressJPEG(path, outputPath, quality, maxWidth, reflectionThreshold); err != nil {
			fmt.Printf("FAILED: %v\n", err)
			return nil
		}

		// ファイルサイズを比較表示
		outInfo, _ := os.Stat(outputPath)
		inSize := info.Size()
		outSize := outInfo.Size()
		ratio := float64(outSize) / float64(inSize) * 100

		totalIn += inSize
		totalOut += outSize

		fmt.Printf("OK (%.1f MB → %.1f MB, %.0f%%)\n",
			float64(inSize)/1024/1024,
			float64(outSize)/1024/1024,
			ratio)
		count++

		return nil
	})

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("\n%d files processed\n", count)
	fmt.Printf("Total: %.1f MB → %.1f MB (%.1f MB 削減, %.0f%%)\n",
		float64(totalIn)/1024/1024,
		float64(totalOut)/1024/1024,
		float64(totalIn-totalOut)/1024/1024,
		float64(totalOut)/float64(totalIn)*100)
}
