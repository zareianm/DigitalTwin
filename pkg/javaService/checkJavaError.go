package javaService

import (
	"os"
	"regexp"
)

func CheckJavaError(filePath string, programArgs []string) (string, string) {
	stdOut, errorStr := checkCompileError(filePath, programArgs)
	return stdOut, errorStr
}

func checkCompileError(filePath string, programArgs []string) (string, string) {
	// 1) run compilation and execution check with javac/java
	stdOut, _, runErr := RunJavaInDocker(filePath, programArgs)
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

// scanDangerousAPIs looks for Java system-level or dangerous API calls.
func scanDangerousAPIs(src string) []string {
	keywords := []string{
		// Process execution
		`Runtime\.getRuntime\(\)\.exec`,
		`ProcessBuilder`,
		`\bexec\s*\(`,

		// File system operations
		`\bdelete\s*\(`,
		`Files\.delete`,
		`\.deleteOnExit\s*\(`,

		// Network operations
		`Socket`,
		`ServerSocket`,
		`URLConnection`,
		`HttpURLConnection`,

		// Reflection (can be dangerous)
		`Class\.forName`,
		`\.newInstance\s*\(`,
		`Method\.invoke`,
		`\.getDeclaredField`,
		`\.setAccessible\s*\(`,

		// Classloaders
		`ClassLoader`,
		`defineClass`,

		// Native code
		`System\.load`,
		`System\.loadLibrary`,
		`native\s+\w+`,

		// Security Manager manipulation
		`SecurityManager`,
		`System\.setSecurityManager`,

		// Threading that could cause DOS
		`Thread\.sleep`,
		`ExecutorService`,

		// Serialization (can be exploited)
		`ObjectInputStream`,
		`readObject\s*\(`,

		// Script engines
		`ScriptEngine`,
		`ScriptEngineManager`,

		// JNDI (injection risks)
		`InitialContext`,
		`lookup\s*\(`,

		// System properties manipulation
		`System\.setProperty`,
		`System\.getenv`,
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
