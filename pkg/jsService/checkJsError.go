package jsService

import (
	"os"
	"regexp"
)

func CheckJsError(filePath string, programArgs []string) (string, string) {
	stdOut, errorStr := checkExecutionError(filePath, programArgs)
	return stdOut, errorStr
}

func checkExecutionError(filePath string, programArgs []string) (string, string) {
	// 1) run execution check with node
	stdOut, _, runErr := RunJsInDocker(filePath, programArgs)
	if runErr != nil {
		return "", runErr.Error()
	}

	// 2) read source for heuristics
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", "failed to read source file"
	}
	src := string(contentBytes)

	// 3) count loops
	loopCount := countLoops(src)
	bigLoop := loopCount > 20 // arbitrary threshold
	if bigLoop {
		return "", "big loop detected"
	}

	// 4) scan for dangerous API calls
	dangerousKeys := scanDangerousAPIs(src)
	if len(dangerousKeys) > 0 {
		return "", "dangerous API call detected"
	}

	return stdOut, ""
}

// countLoops returns the number of 'for', 'while', and 'do-while' occurrences.
func countLoops(src string) int {
	forRe := regexp.MustCompile(`\bfor\s*\(`)
	whileRe := regexp.MustCompile(`\bwhile\s*\(`)
	doWhileRe := regexp.MustCompile(`\bdo\s*\{`)

	return len(forRe.FindAllString(src, -1)) +
		len(whileRe.FindAllString(src, -1)) +
		len(doWhileRe.FindAllString(src, -1))
}

// scanDangerousAPIs looks for JavaScript/Node.js dangerous API calls.
func scanDangerousAPIs(src string) []string {
	keywords := []string{
		// Process execution
		`require\s*\(\s*['"]child_process['"]`,
		`\.exec\s*\(`,
		`\.execSync\s*\(`,
		`\.spawn\s*\(`,
		`\.spawnSync\s*\(`,
		`\.fork\s*\(`,
		`\.execFile\s*\(`,

		// File system operations (dangerous ones)
		`require\s*\(\s*['"]fs['"]`,
		`\.unlink\s*\(`,
		`\.unlinkSync\s*\(`,
		`\.rmdir\s*\(`,
		`\.rmdirSync\s*\(`,
		`\.rm\s*\(`,
		`\.rmSync\s*\(`,

		// Network operations
		`require\s*\(\s*['"]net['"]`,
		`require\s*\(\s*['"]http['"]`,
		`require\s*\(\s*['"]https['"]`,
		`\.createServer\s*\(`,
		`\.connect\s*\(`,
		`\.request\s*\(`,

		// Eval and code execution
		`\beval\s*\(`,
		`Function\s*\(`,
		`setTimeout\s*\(.*['"]`,
		`setInterval\s*\(.*['"]`,
		`new\s+Function\s*\(`,

		// VM module (can execute arbitrary code)
		`require\s*\(\s*['"]vm['"]`,
		`\.runInContext\s*\(`,
		`\.runInNewContext\s*\(`,
		`\.runInThisContext\s*\(`,

		// Worker threads
		`require\s*\(\s*['"]worker_threads['"]`,
		`new\s+Worker\s*\(`,

		// Cluster module
		`require\s*\(\s*['"]cluster['"]`,

		// DNS module
		`require\s*\(\s*['"]dns['"]`,

		// Process manipulation
		`process\.exit\s*\(`,
		`process\.kill\s*\(`,
		`process\.abort\s*\(`,

		// Dynamic imports
		`import\s*\(`,

		// Debugging/profiling (can be exploited)
		`require\s*\(\s*['"]inspector['"]`,
		`debugger`,

		// Crypto module (if used for mining or DOS)
		`require\s*\(\s*['"]crypto['"]`,

		// OS module (system information)
		`require\s*\(\s*['"]os['"]`,

		// WebAssembly
		`WebAssembly\.compile`,
		`WebAssembly\.instantiate`,
	}

	found := make(map[string]struct{})
	for _, kw := range keywords {
		re := regexp.MustCompile(kw)
		if loc := re.FindStringIndex(src); loc != nil {
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
