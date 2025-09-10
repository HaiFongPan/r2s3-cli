package utils

import (
	"fmt"
	"os/exec"
	"runtime"
)

func CopyToClipboard(content string) error {
	switch runtime.GOOS {
	case "darwin": // macOS
		cmd := exec.Command("pbcopy")
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return err
		}

		go func() {
			defer stdin.Close()
			stdin.Write([]byte(content))
		}()

		return cmd.Run()

	case "linux": // Linux
		// 尝试多种Linux剪贴板工具
		for _, tool := range []string{"xclip", "xsel"} {
			if _, err := exec.LookPath(tool); err == nil {
				cmd := exec.Command(tool, "-selection", "clipboard")
				stdin, err := cmd.StdinPipe()
				if err != nil {
					continue
				}

				go func() {
					defer stdin.Close()
					stdin.Write([]byte(content))
				}()

				return cmd.Run()
			}
		}
		return fmt.Errorf("未找到可用的剪贴板工具 (xclip 或 xsel)")

	case "windows": // Windows
		cmd := exec.Command("clip")
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return err
		}

		go func() {
			defer stdin.Close()
			stdin.Write([]byte(content))
		}()

		return cmd.Run()

	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}
