package main

import (
	"fmt"
	"image"
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

func compressJPEG(input, output string, quality, maxWidth int) error {
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

	outFile, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("create error: %w", err)
	}
	defer outFile.Close()

	opts := &jpeg.Options{Quality: quality}
	if err := jpeg.Encode(outFile, img, opts); err != nil {
		return fmt.Errorf("encode error: %w", err)
	}

	return nil
}

func main() {
	srcDir := filepath.Join("data", "original")
	outDir := filepath.Join("data", "compressed")
	quality := 70
	maxWidth := 1920

	var totalIn, totalOut int64
	count := 0

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

		fmt.Printf("Compressing: %s ... ", relPath)

		if err := compressJPEG(path, outputPath, quality, maxWidth); err != nil {
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

	fmt.Printf("\n%d files compressed (quality=%d, maxWidth=%dpx)\n", count, quality, maxWidth)
	fmt.Printf("Total: %.1f MB → %.1f MB (%.1f MB 削減, %.0f%%)\n",
		float64(totalIn)/1024/1024,
		float64(totalOut)/1024/1024,
		float64(totalIn-totalOut)/1024/1024,
		float64(totalOut)/float64(totalIn)*100)
}
