package jobService

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

func CheckCppError(f multipart.File, header *multipart.FileHeader) string {
	//save to a temp file
	tmpDir, _ := ioutil.TempDir("", "cppcheck")
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, header.Filename)
	out, err := os.Create(filePath)
	if err != nil {
		return "cannot save file"
	}
	defer out.Close()
	io.Copy(out, f)

	errorStr := checkCompileError(filePath)

	return errorStr
}

func checkCompileError(filePath string) string {

	// 3) run syntax check with g++
	syntaxErrs := runCommand("g++", "-std=c++17", "-fsyntax-only", filePath)

	if syntaxErrs != "" {
		return syntaxErrs
	}

	// 4) read source for heuristics
	contentBytes, _ := ioutil.ReadFile(filePath)
	src := string(contentBytes)

	// 5) count loops
	loopCount := countLoops(src)
	bigLoop := loopCount > 20 // arbitrary threshold

	if bigLoop {
		return "big loop detected"
	}

	// 6) scan for syscall-related keywords
	syscallKeys := scanSyscalls(src)
	syscallWarn := len(syscallKeys) > 0

	if syscallWarn {
		return "system call detected"
	}

	return ""
}

// countLoops returns the number of 'for' and 'while' occurrences.
func countLoops(src string) int {
	forRe := regexp.MustCompile(`\bfor\s*\(`)
	whileRe := regexp.MustCompile(`\bwhile\s*\(`)
	return len(forRe.FindAllString(src, -1)) + len(whileRe.FindAllString(src, -1))
}

// scanSyscalls looks for common C/C++ system-call or dangerous API keywords.
func scanSyscalls(src string) []string {
	keywords := []string{
		`\bsystem\s*\(`,
		`\bexec\s*vp?\s*\(`,
		`\bfork\s*\(`,
		`\bpopen\s*\(`,
		`\bopen\s*\(`,
		`\bioctl\s*\(`,
		`\bptrace\s*\(`,
		`\bsocket\s*\(`,
		`\bconnect\s*\(`,
		`\bbind\s*\(`,
		`\baccept\s*\(`,
	}
	found := make(map[string]struct{})
	for _, kw := range keywords {
		re := regexp.MustCompile(kw)
		if loc := re.FindStringIndex(src); loc != nil {
			// extract the bare keyword (e.g. "system(")
			match := re.FindString(src)
			found[match] = struct{}{}
		}
	}
	var list []string
	for k := range found {
		list = append(list, k)
	}
	return list
}

// runCommand executes a command and returns its stderr/stdout lines.
func runCommand(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()

	combined := stderr.String() + out.String()
	if err != nil {
		// append the Go error (e.g. "exit status 1" or "executable file not found")
		combined += "\n" + err.Error()

		return combined
	}
	return ""
}
