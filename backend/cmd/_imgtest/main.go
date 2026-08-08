package main

import (
	"fmt"
	"os"
	"path/filepath"
	"restaurant-backend/internal/images"
)

func main() {
	files, _ := filepath.Glob("internal/seed/assets/*.jpg")
	var old, w600, w1200 int64
	for _, p := range files {
		st, _ := os.Stat(p)
		old += st.Size()
		for _, w := range []int{600, 1200} {
			f, _ := os.Open(p)
			b, _, err := images.Fit(f, w)
			f.Close()
			if err != nil {
				fmt.Println("xato:", p, err)
				return
			}
			if w == 600 {
				w600 += int64(len(b))
			} else {
				w1200 += int64(len(b))
			}
		}
	}
	mb := func(n int64) string { return fmt.Sprintf("%.2f MB", float64(n)/1024/1024) }
	fmt.Printf("%d fayl\nhozir: %s\n600px: %s (-%.0f%%)\n1200px: %s (-%.0f%%)\n",
		len(files), mb(old), mb(w600), 100-100*float64(w600)/float64(old),
		mb(w1200), 100-100*float64(w1200)/float64(old))
}
