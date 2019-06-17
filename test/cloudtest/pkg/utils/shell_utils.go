package utils

import (
	"bufio"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
)

func ParseVariable(variable string) (string, string, error) {
	pos := strings.Index(variable, "=")
	if pos == -1 {
		return "", "", fmt.Errorf("Variable passed are invalid...")
	}
	return variable[:pos], variable[pos+1:], nil
}

func ParseCommandLine(cmdLine string) []string {
	pos := 0
	current := strings.Builder{}

	count := len(cmdLine)
	result := []string{}

	for ; pos < count; {

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

func SubstituteVariable(variable string, vars, args map[string]string) (string, error) {

	pos := 0
	result := strings.Builder{}

	count := len(variable)

	for ; pos < count; {

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
						result.WriteString(varValue)
					} else {
						return "", fmt.Errorf("Failed to find variable %v in passed variables", varName)
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
						return "", fmt.Errorf("Failed to find argument %v in passed arguments", varName)
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
	for ; pos < count; {
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
	for ; pos < count; {
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

func ParseScript(s string) []string {
	return strings.Split(strings.TrimSpace(s), "\n")
}

func RunCommand(id string, context context.Context, cmd, operation string, writer *bufio.Writer, env []string, args map[string]string) error {
	finalEnv := append(os.Environ(), env...)

	environment := map[string]string{}
	for _, k := range finalEnv {
		key, value, err := ParseVariable(k)
		if err != nil {
			return err
		}
		environment[key] = value
	}

	finalCmd, err := SubstituteVariable(cmd, environment, args)
	if err != nil {
		return err
	}

	cmdLine := ParseCommandLine(finalCmd)

	proc, err := ExecProc(context, cmdLine, finalEnv)
	if err != nil {
		return fmt.Errorf("Failed to run %s %v", cmdLine, err)
	}
	go func() {
		reader := bufio.NewReader(proc.Stdout)
		for {
			s, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			_, _ = writer.WriteString(s)
			_ = writer.Flush()
			if (len(strings.TrimSpace(s)) > 0) {
				logrus.Infof("Output: %s => %s %v", id, operation, s)
			}
		}
	}()
	go func() {
		reader := bufio.NewReader(proc.Stderr)
		for {
			s, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			_, _ = writer.WriteString(s)
			_ = writer.Flush()
			if (len(strings.TrimSpace(s)) > 0) {
				logrus.Infof("StdErr: %s => %s %v", id, operation, s)
			}
		}
	}()
	if code := proc.ExitCode(); code != 0 {
		logrus.Errorf("Failed to run %s ExitCode: %v. Logs inside %v", cmdLine, code, operation)
		return fmt.Errorf("failed to run %s ExitCode: %v. Logs inside %v", cmdLine, code, operation)
	}
	return nil
}
