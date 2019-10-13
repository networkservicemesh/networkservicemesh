package utils

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

// ParseVariable - parses var=value variable format.
func ParseVariable(variable string) (string, string, error) {
	pos := strings.Index(variable, "=")
	if pos == -1 {
		return "", "", errors.Errorf("variable passed are invalid")
	}
	return variable[:pos], variable[pos+1:], nil
}

// ParseCommandLine - parses command line with support of "" and escaping.
func ParseCommandLine(cmdLine string) []string {
	pos := 0
	current := strings.Builder{}

	count := len(cmdLine)
	result := []string{}

	for pos < count {

		charAt := cmdLine[pos]

		if charAt == '\\' {
			pos++
			if pos < count {
				// Write one more symbol
				_ = current.WriteByte(cmdLine[pos])
			}
		} else if charAt == '"' {
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
			pos++
			// Read until next " with escaping support
			str := ""
			str, pos = readStringEscaping(pos, count, cmdLine, '"')
			result = append(result, str)
		} else {
			//Add skiping spaces.
			if charAt != ' ' && charAt != '\t' {
				_ = current.WriteByte(charAt)
			} else if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
		}

		pos++
	}
	if current.Len() > 0 {
		result = append(result, current.String())
		current.Reset()
	}

	return result

}

// SubstituteVariable - perform a substitution of all ${var} $(arg) in passed string and return substitution results and error
func SubstituteVariable(variable string, vars, args map[string]string) (string, error) {

	pos := 0
	result := strings.Builder{}

	count := len(variable)

	for pos < count {

		charAt := variable[pos]

		if charAt == '$' {
			if pos+1 < count {
				// We have more symbols to check
				nextChar := variable[pos+1]

				if nextChar == '{' {
					// This is variable substitution
					pos += 2
					var varName string
					varName, pos = readString(pos, count, variable, '}')

					// We found variable or reached end of string
					if varValue, ok := vars[varName]; ok {
						_, _ = result.WriteString(varValue)
					} else {
						return "", errors.Errorf("failed to find variable %v in passed variables", varName)
					}

				} else if nextChar == '(' {
					// This is parameter substitution
					pos += 2
					var varName string
					varName, pos = readString(pos, count, variable, ')')

					// We found variable or reached end of string
					if argValue, ok := args[varName]; ok {
						_, _ = result.WriteString(argValue)
					} else {
						return "", errors.Errorf("failed to find argument %v in passed arguments", varName)
					}
				}

			} else {
				// End of string just add symbol to result
				_ = result.WriteByte(charAt)
			}
		} else {
			_ = result.WriteByte(charAt)
		}

		pos++
	}
	return result.String(), nil

}

func readString(pos, count int, variable string, delim uint8) (string, int) {
	varName := strings.Builder{}
	for pos < count {
		tChar := variable[pos]
		if tChar == delim {
			break
		} else {
			_ = varName.WriteByte(tChar)
		}
		pos++
	}
	return varName.String(), pos
}

func readStringEscaping(pos, count int, variable string, delim uint8) (string, int) {
	varName := strings.Builder{}
	for pos < count {
		tChar := variable[pos]
		if tChar == '\\' {
			pos++
			if pos < count {
				// Write one more symbol
				_ = varName.WriteByte(variable[pos])
			}
		} else if tChar == delim {
			break
		} else {
			_ = varName.WriteByte(tChar)
		}
		pos++
	}
	return varName.String(), pos
}

// ParseScript - parse multi line script and return individual commands.
func ParseScript(s string) []string {
	return strings.Split(strings.TrimSpace(s), "\n")
}

// RunCommand - run shell command and put output into file, command variables are substituted.
func RunCommand(context context.Context, cmd, dir string, logger func(str string), writer *bufio.Writer, env []string, args map[string]string, returnStdout bool) (string, error) {
	finalEnv := append(os.Environ(), env...)
	environment := map[string]string{}
	for _, k := range finalEnv {
		key, value, err := ParseVariable(k)
		if err != nil {
			return "", err
		}
		environment[key] = value
	}

	finalCmd, err := SubstituteVariable(cmd, environment, args)
	if err != nil {
		return "", err
	}

	cmdLine := ParseCommandLine(finalCmd)
	proc, err := ExecProc(context, dir, cmdLine, finalEnv)
	if err != nil {
		return "", errors.Wrapf(err, "failed to run %s", finalCmd)
	}

	builder := strings.Builder{}
	var wg sync.WaitGroup
	wg.Add(2)
	processOutput(proc.Stdout, writer, logger, "StdOut", &builder, returnStdout, &wg)
	processOutput(proc.Stderr, writer, logger, "StdErr", nil, false, &wg)
	wg.Wait()
	code := proc.ExitCode()
	if code != 0 {
		return "", errors.Errorf("failed to run %v ExitCode: %v", finalCmd, code)
	}
	if returnStdout {
		return builder.String(), nil
	}
	return "", nil
}

func processOutput(stream io.Reader, writer *bufio.Writer, logger func(str string), pattern string, builder io.StringWriter, returnStdout bool, wg *sync.WaitGroup) {
	go func() {
		defer wg.Done()
		reader := bufio.NewReader(stream)
		for {
			s, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			_, _ = writer.WriteString(s)
			_ = writer.Flush()
			if len(strings.TrimSpace(s)) > 0 {
				logger(fmt.Sprintf("%s => %v", pattern, s))
			}
			if returnStdout {
				_, _ = builder.WriteString(strings.TrimSpace(s) + "\n")
			}
		}
	}()
}
