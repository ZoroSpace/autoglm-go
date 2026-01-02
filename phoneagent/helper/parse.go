package helper

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	logs "github.com/sirupsen/logrus"
)

type Action map[string]any

type ActionResult struct {
	Success              bool
	ShouldFinish         bool
	Message              string
	RequiresConfirmation bool
}

func ParseAction(rawActionStr string) (Action, error) {
	logs.Debugf("begin to parse action: %s", rawActionStr)

	rawActionStr = strings.TrimSpace(rawActionStr)

	// case 1: do(action=...)
	if strings.HasPrefix(rawActionStr, "do(") {
		action, err := parseDoCall(rawActionStr)
		if err != nil {
			logs.Errorf("failed to parse do() action, rawActionStr: %s, err: %v", rawActionStr, err)
			return nil, fmt.Errorf("failed to parse do() action: %w", err)
		}
		return action, nil
	}

	// case 2: finish(message="...")
	if strings.HasPrefix(rawActionStr, "finish") {
		msg, err := parseFinishMessage(rawActionStr)
		if err != nil {
			return nil, err
		}

		return Action{
			"_metadata": "finish",
			"message":   msg,
		}, nil
	}
	return nil, fmt.Errorf("failed to parse action: %s", rawActionStr)
}

func parseDoCall(expr string) (Action, error) {
	// 去掉 do( 和 )
	if !strings.HasPrefix(expr, "do(") || !strings.HasSuffix(expr, ")") {
		return nil, errors.New("invalid do() syntax")
	}

	body := strings.TrimSuffix(strings.TrimPrefix(expr, "do("), ")")

	action := Action{
		"_metadata": "do",
	}

	if strings.TrimSpace(body) == "" {
		return action, nil
	}

	parts := strings.Split(body, ", ")

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid argument: %s", part)
		}

		key := strings.TrimSpace(kv[0])
		valStr := strings.TrimSpace(kv[1])

		val, err := parseLiteral(valStr)
		if err != nil {
			return nil, fmt.Errorf("invalid value for %s: %w", key, err)
		}

		action[key] = val
	}
	return action, nil
}

var messageRe = regexp.MustCompile(`message="((?:\\.|[^"])*)"`)

func parseFinishMessage(s string) (string, error) {
	matches := messageRe.FindStringSubmatch(s)
	if len(matches) < 2 {
		return "", errors.New("message not found")
	}
	return matches[1], nil
}

func parseLiteral(s string) (any, error) {
	logs.Debugf("begin to parse literal: %s", s)
	// string
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		return s[1 : len(s)-1], nil
	}

	// bool
	if s == "true" {
		return true, nil
	}
	if s == "false" {
		return false, nil
	}

	// int[]
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		content := strings.TrimSpace(s[1 : len(s)-1])
		if content == "" {
			return []int{}, nil
		}

		parts := strings.Split(content, ",")
		result := make([]int, 0, len(parts))

		for _, p := range parts {
			v, err := strconv.Atoi(strings.TrimSpace(p))
			if err != nil {
				return nil, fmt.Errorf("invalid int in array: %s", p)
			}
			result = append(result, v)
		}
		return result, nil
	}

	// int
	if i, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return i, nil
	}

	// float
	if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
		return f, nil
	}

	return nil, fmt.Errorf("unsupported literal: %s", s)
}
