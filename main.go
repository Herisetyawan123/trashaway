package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fatih/color"
)

var smartTargets = map[string]string{
	"node_modules": "package.json",
	"vendor":       "composer.json",
	".dart_tool":   "pubspec.yaml",
	"build":        "pubspec.yaml",
}

func isValidProject(folder string, marker string) bool {
	_, err := os.Stat(filepath.Join(folder, marker))
	return err == nil
}

func scanTargets(root string, exclude []string) []string {
	var targets []string

	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}

		// Skip excluded folders
		for _, ex := range exclude {
			if strings.HasPrefix(path, ex) {
				return filepath.SkipDir
			}
		}

		// Skip if already inside known target
		for _, t := range targets {
			if strings.HasPrefix(path, t) {
				return filepath.SkipDir
			}
		}

		if marker, ok := smartTargets[d.Name()]; ok {
			parent := filepath.Dir(path)
			if isValidProject(parent, marker) {
				color.Green("📂 Ditemukan %s dalam proyek (%s): %s", d.Name(), marker, path)
				targets = append(targets, path)
				return filepath.SkipDir
			} else {
				color.Red("⚠️  Lewati %s (bukan proyek %s): %s", d.Name(), marker, path)
			}
		}

		return nil
	})

	return targets
}

func confirm(prompt string) bool {
	fmt.Print(prompt + " (y/n): ")
	reader := bufio.NewReader(os.Stdin)
	resp, _ := reader.ReadString('\n')
	return strings.ToLower(strings.TrimSpace(resp)) == "y"
}

func getFolderSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func deleteFolder(path string, wg *sync.WaitGroup) {
	defer wg.Done()
	err := os.RemoveAll(path)
	if err != nil {
		color.Red("❌ Gagal hapus: %s (%v)", path, err)
	} else {
		color.Green("✅ Berhasil hapus: %s", path)
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	color.White("🔍 Masukkan path folder yang ingin dipindai:")
	fmt.Print("➡️  ")
	rootInput, _ := reader.ReadString('\n')
	root := strings.TrimSpace(rootInput)

	if _, err := os.Stat(root); os.IsNotExist(err) {
		color.Red("🚫 Path tidak ditemukan.")
		return
	}

	color.White("🔒 Masukkan path folder yang ingin DIKECUALIKAN (pisahkan dengan koma, atau kosongkan):")
	fmt.Print("➡️  ")
	exInput, _ := reader.ReadString('\n')
	exInput = strings.TrimSpace(exInput)

	var excluded []string
	if exInput != "" {
		for _, p := range strings.Split(exInput, ",") {
			abs, _ := filepath.Abs(strings.TrimSpace(p))
			excluded = append(excluded, abs)
		}
	}

	absRoot, _ := filepath.Abs(root)
	color.Yellow("\n🚀 Mulai scan di folder: %s", absRoot)
	targets := scanTargets(absRoot, excluded)

	if len(targets) == 0 {
		color.Green("✅ Tidak ada folder target yang ditemukan.")
		return
	}

	color.Cyan("\n📋 Berikut folder yang akan dihapus:")
	var totalSize int64
	for i, t := range targets {
		sz := getFolderSize(t)
		fmt.Printf("  %d. %s (%.2f MB)\n", i+1, t, float64(sz)/1024/1024)
		totalSize += sz
	}

	color.Red("\n💾 Total size yang akan dihapus: %.2f MB", float64(totalSize)/1024/1024)

	if !confirm("\n❓ Lanjut hapus semua folder di atas?") {
		color.Green("❌ Dibatalkan.")
		return
	}

	color.Green("\n🔥 Menghapus dengan concurrency...\n")

	var wg sync.WaitGroup
	for _, t := range targets {
		wg.Add(1)
		go deleteFolder(t, &wg)
	}
	wg.Wait()

	color.Green("\n✅ Selesai! Semua folder berhasil dihapus (jika tidak ada error).")
}
