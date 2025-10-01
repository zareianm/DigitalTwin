package pythonService

import (
	"io/ioutil"
	"regexp"
)

func CheckPythonError(filePath string, programArgs []string) (string, string) {
	stdOut, errorStr := checkPythonError(filePath, programArgs)
	return stdOut, errorStr
}

func checkPythonError(filePath string, programArgs []string) (string, string) {
	// 1) run syntax check and execution with python in Docker
	stdOut, _, runErr := RunPythonInDocker(filePath, programArgs)
	if runErr != nil {
		return "", runErr.Error()
	}

	// 2) read source for heuristics
	contentBytes, _ := ioutil.ReadFile(filePath)
	src := string(contentBytes)

	// 3) count loops
	loopCount := countLoops(src)
	bigLoop := loopCount > 20 // arbitrary threshold
	if bigLoop {
		return "", "big loop detected"
	}

	// 4) scan for dangerous system calls and imports
	dangerousKeys := scanDangerousCalls(src)
	if len(dangerousKeys) > 0 {
		return "", "dangerous system call or import detected"
	}

	return stdOut, ""
}

// countLoops returns the number of 'for' and 'while' occurrences.
func countLoops(src string) int {
	// Match 'for' loops (for ... in ...:)
	forRe := regexp.MustCompile(`\bfor\s+\w+\s+in\s+`)
	// Match 'while' loops (while ...:)
	whileRe := regexp.MustCompile(`\bwhile\s+.+:`)
	return len(forRe.FindAllString(src, -1)) + len(whileRe.FindAllString(src, -1))
}

// scanDangerousCalls looks for common Python dangerous functions and imports.
func scanDangerousCalls(src string) []string {
	keywords := []string{
		// System execution
		`\bos\.system\s*\(`,
		`\bsubprocess\.call\s*\(`,
		`\bsubprocess\.run\s*\(`,
		`\bsubprocess\.Popen\s*\(`,
		`\bos\.popen\s*\(`,
		`\bos\.spawn`,
		`\bos\.exec`,
		`\beval\s*\(`,
		`\bexec\s*\(`,
		`\b__import__\s*\(`,
		`\bcompile\s*\(`,

		// File system operations (potentially dangerous)
		`\bos\.remove\s*\(`,
		`\bos\.rmdir\s*\(`,
		`\bos\.unlink\s*\(`,
		`\bshutil\.rmtree\s*\(`,

		// Network operations
		`\bsocket\.socket\s*\(`,
		`\bsocket\.connect\s*\(`,
		`\bsocket\.bind\s*\(`,
		`\bsocket\.accept\s*\(`,
		`\burllib\.request`,
		`\brequests\.`,

		// Dangerous imports
		`\bimport\s+os\b`,
		`\bfrom\s+os\s+import`,
		`\bimport\s+subprocess\b`,
		`\bfrom\s+subprocess\s+import`,
		`\bimport\s+socket\b`,
		`\bfrom\s+socket\s+import`,
		`\bimport\s+ctypes\b`,
		`\bfrom\s+ctypes\s+import`,

		// Code injection
		`\bpickle\.loads\s*\(`,
		`\byaml\.load\s*\(`,
		`\bmarshall\.loads\s*\(`,
	}

	found := make(map[string]struct{})
	for _, kw := range keywords {
		re := regexp.MustCompile(kw)
		if loc := re.FindStringIndex(src); loc != nil {
			// extract the match
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
