package proc

import (
	"bytes"
	"strings"
)

func IsJvm(cmdline []byte) bool {
	idx := bytes.Index(cmdline, []byte{0})
	if idx < 0 {
		return false
	}

	// Extract the executable name (first part before null byte)
	executablePath := string(cmdline[:idx])

	// Get just the executable name without path
	parts := strings.Split(executablePath, "/")
	executable := parts[len(parts)-1]

	// Check for various Java runtime executables
	javaRuntimes := []string{
		"java",                   // Standard Oracle/OpenJDK Java
		"openjdk",                // OpenJDK variants
		"javac",                  // Java compiler (sometimes used to run apps)
		"scala",                  // Scala runtime
		"kotlin",                 // Kotlin runtime
		"kotlinc",                // Kotlin compiler
		"groovy",                 // Groovy runtime
		"clojure",                // Clojure runtime
		"jruby",                  // JRuby runtime
		"jython",                 // Jython runtime
		"graalvm",                // GraalVM runtime
		"native-image",           // GraalVM native image
		"gu",                     // GraalVM updater
		"polyglot",               // GraalVM polyglot
		"native-image-configure", // GraalVM native image configuration
	}

	// Check for exact matches
	for _, runtime := range javaRuntimes {
		if executable == runtime {
			return true
		}
	}

	// Check for runtime names with version suffixes (e.g., java11, java17, openjdk8)
	for _, runtime := range []string{"java", "openjdk", "graalvm"} {
		if strings.HasPrefix(executable, runtime) {
			// Allow for version numbers after the base name
			suffix := strings.TrimPrefix(executable, runtime)
			if suffix == "" || isVersionSuffix(suffix) {
				return true
			}
		}
	}

	// Check if the command line contains JVM-specific parameters or classes
	// Convert cmdline to string for easier searching
	cmdlineStr := strings.ReplaceAll(string(cmdline), "\x00", " ")

	// Look for JVM-specific indicators in the command line
	// Only include highly specific indicators to avoid false positives
	jvmIndicators := []string{
		"-classpath ",          // Classpath (very Java-specific)
		"-jar ",                // JAR execution (highly Java-specific)
		"-Xmx",                 // Heap size (Java-specific)
		"-Xms",                 // Initial heap size (Java-specific)
		"-XX:",                 // JVM options (Java-specific)
		"-javaagent:",          // Java agents (Java-specific)
		"-Xbootclasspath",      // Bootstrap classpath (Java-specific)
		"-Xrunjdwp:",           // Debug transport (Java-specific)
		"com.sun.",             // Sun/Oracle packages (Java-specific)
		"java.lang.",           // Core Java packages (Java-specific)
		"scala.tools.",         // Scala tools (Scala-specific)
		"kotlin.compiler",      // Kotlin compiler (Kotlin-specific)
		"groovy.lang.",         // Groovy packages (Groovy-specific)
		"clojure.main",         // Clojure main (Clojure-specific)
		"org.jruby.",           // JRuby packages (JRuby-specific)
		"org.python.",          // Jython packages (Jython-specific)
		"org.springframework.", // Spring Framework (Java-specific)
		"org.junit.",           // JUnit testing (Java-specific)
		"org.gradle.",          // Gradle build tool (Java-specific)
		"org.maven.",           // Maven packages (Java-specific)
	}

	for _, indicator := range jvmIndicators {
		if strings.Contains(cmdlineStr, indicator) {
			return true
		}
	}

	return false
}

// isVersionSuffix checks if a string looks like a version suffix (e.g., "8", "11", "17", "-11", "_8")
func isVersionSuffix(suffix string) bool {
	if len(suffix) == 0 {
		return true
	}

	// Remove common separators
	suffix = strings.TrimPrefix(suffix, "-")
	suffix = strings.TrimPrefix(suffix, "_")
	suffix = strings.TrimPrefix(suffix, ".")

	// Check if remaining part is numeric (version number)
	if len(suffix) == 0 {
		return true
	}

	// Simple check for numeric version (1-2 digits typically)
	for _, char := range suffix {
		if char < '0' || char > '9' {
			return false
		}
	}

	return len(suffix) <= 3 // Reasonable version number length
}
