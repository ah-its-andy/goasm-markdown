package markdown

import "testing"

func Test_Transalate(t *testing.T) {
	content := `# title 1
## 2 title 
### *italic title * `
	result := Translate([]rune(content))
	text := string(result)
	t.Log(text)
}
