package compress

import (
	"strings"
	"testing"
)

func TestCompressRTKGitDiff(t *testing.T) {
	diff := `diff --git a/main.go b/main.go
index abc123..def456 100644
--- a/main.go
+++ b/main.go
@@ -10,6 +10,8 @@ import (
 	"fmt"
 )
 
+// New function
+func hello() {
+	fmt.Println("hello")
+}
+
 func main() {
 	fmt.Println("main")
 }`

	compressed, savings := CompressRTK(diff)
	if savings <= 0 {
		t.Error("expected savings > 0 for git diff")
	}
	if !strings.Contains(compressed, "+// New function") {
		t.Error("should keep added lines")
	}
	if !strings.Contains(compressed, "diff --git") {
		t.Error("should keep diff header")
	}
	// Should remove some context lines (those starting with space)
	if len(compressed) >= len(diff) {
		t.Error("compressed should be shorter than original")
	}
}

func TestCompressRTKGrep(t *testing.T) {
	grep := `main.go:10:func main() {
main.go:11:	fmt.Println("hello")
main.go:15:	fmt.Println("world")
utils.go:3:func helper() {
utils.go:4:	return nil
utils.go:8:func other() {`

	compressed, savings := CompressRTK(grep)
	if savings <= 0 {
		t.Error("expected savings > 0 for grep output")
	}
	if !strings.Contains(compressed, "main.go:") {
		t.Error("should keep file path header")
	}
}

func TestCompressRTKLs(t *testing.T) {
	ls := `-rw-r--r--  1 user group  1234 May 29 10:00 main.go
-rw-r--r--  1 user group  5678 May 29 10:01 utils.go
drwxr-xr-x  3 user group   128 May 29 09:00 src/`

	compressed, savings := CompressRTK(ls)
	if savings <= 0 {
		t.Error("expected savings > 0 for ls output")
	}
	if !strings.Contains(compressed, "main.go") {
		t.Error("should keep filenames")
	}
}

func TestCompressRTKTree(t *testing.T) {
	tree := `.
├── main.go
├── utils.go
├── src/
│   ├── handler.go
│   ├── handler.go
│   └── model.go
└── go.mod`

	compressed, _ := CompressRTK(tree)
	// Should deduplicate repeated handler.go
	if strings.Count(compressed, "handler.go") > 2 {
		t.Error("should deduplicate repeated entries")
	}
}

func TestCompressRTKFind(t *testing.T) {
	find := `./src/main.go
./src/utils.go
./src/handler.go
./src/model.go
./src/types.go
./src/config.go
./internal/db.go
./internal/cache.go`

	_, savings := CompressRTK(find)
	if savings <= 0 {
		t.Error("expected savings > 0 for find output")
	}
}

func TestCompressRTKLogs(t *testing.T) {
	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, "2026-05-29 10:00:00 INFO Starting request")
	}
	lines = append(lines, "2026-05-29 10:00:01 ERROR Connection failed")
	lines = append(lines, "2026-05-29 10:00:01 ERROR Connection failed")
	lines = append(lines, "2026-05-29 10:00:01 ERROR Connection failed")

	compressed, savings := CompressRTK(strings.Join(lines, "\n"))
	if savings <= 0 {
		t.Error("expected savings > 0 for log dump")
	}
	// Should suppress repeated lines (either "repeated" or "similar" marker)
	if !strings.Contains(compressed, "repeated") && !strings.Contains(compressed, "similar") && !strings.Contains(compressed, "suppressed") {
		t.Error("should note repeated/suppressed lines")
	}
}

func TestCompressRTKSmallContent(t *testing.T) {
	content := "hello world"
	result, savings := CompressRTK(content)
	if savings != 0 {
		t.Error("small content should not be compressed")
	}
	if result != content {
		t.Error("small content should be returned as-is")
	}
}

func TestCompressRTKSafeFallback(t *testing.T) {
	// Content that might get bigger after compression
	content := "a"
	compressed, savings := CompressRTK(content)
	if savings != 0 {
		t.Error("should not compress single char")
	}
	if compressed != content {
		t.Error("should return original if compression makes it bigger")
	}
}

func TestCompressGeneric(t *testing.T) {
	content := "line1\n\n\n\n\nline2\n\n\n\nline3"
	compressed := CompressGeneric(content)
	if strings.Count(compressed, "\n\n") > 2 {
		t.Error("should collapse multiple blank lines")
	}
}

func TestCompressGenericTruncate(t *testing.T) {
	content := strings.Repeat("x", 20000)
	compressed := CompressGeneric(content)
	if len(compressed) >= len(content) {
		t.Error("should truncate very long content")
	}
	if !strings.Contains(compressed, "truncated") {
		t.Error("should mention truncation")
	}
}
