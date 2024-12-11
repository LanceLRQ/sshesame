package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
)

type readLiner interface {
	ReadLine() (string, error)
}

type commandContext struct {
	args           []string
	stdin          readLiner
	stdout, stderr io.Writer
	pty            bool
	user           string
}

type command interface {
	execute(context commandContext, ctx *sessionContext) (uint32, error)
}

var commands = map[string]command{
	"sh":     cmdShell{},
	"true":   cmdTrue{},
	"false":  cmdFalse{},
	"echo":   cmdEcho{},
	"cat":    cmdCat{},
	"su":     cmdSu{},
	"whoami": cmdWhoami{},
	"pwd":    cmdPwd{},
	"huahuo": cmdHuahuo{},
	"never":  cmdNeverGonnaGiveYouUp{},
	"uname":  cmdUname{},
	"cd":     cmdCd{},
	"ls":     cmdLs{},
	"ll":     cmdLs{},
}

var shellProgram = []string{"sh"}

func executeProgram(context commandContext, ctx *sessionContext) (uint32, error) {
	if len(context.args) == 0 {
		return 0, nil
	}
	cmd := strings.TrimRight(context.args[0], ";")
	command := commands[cmd]
	if command == nil {
		_, err := fmt.Fprintf(context.stderr, "%v: command not found\n", context.args[0])
		return 127, err
	}
	return command.execute(context, ctx)
}

type cmdShell struct{}

func (cmdShell) execute(context commandContext, ctx *sessionContext) (uint32, error) {
	var prompt string
	if context.pty {
		switch context.user {
		case "root":
			prompt = "# "
		default:
			prompt = "$ "
		}
	}
	var lastStatus uint32
	var line string
	var err error
	for {
		_, err = fmt.Fprint(context.stdout, prompt)
		if err != nil {
			return lastStatus, err
		}
		line, err = context.stdin.ReadLine()
		if err != nil {
			return lastStatus, err
		}
		args := strings.Fields(line)
		if len(args) == 0 {
			continue
		}
		if args[0] == "exit" || args[0] == "exit;" {
			var err error
			var status uint64 = uint64(lastStatus)
			if len(args) > 1 {
				status, err = strconv.ParseUint(args[1], 10, 32)
				if err != nil {
					status = 255
				}
			}
			return uint32(status), nil
		}
		newContext := context
		newContext.args = args
		if lastStatus, err = executeProgram(newContext, ctx); err != nil {
			return lastStatus, err
		}
	}
}

type cmdTrue struct{}

func (cmdTrue) execute(context commandContext, ctx *sessionContext) (uint32, error) {
	return 0, nil
}

type cmdFalse struct{}

func (cmdFalse) execute(context commandContext, ctx *sessionContext) (uint32, error) {
	return 1, nil
}

type cmdEcho struct{}

func (cmdEcho) execute(context commandContext, ctx *sessionContext) (uint32, error) {
	_, err := fmt.Fprintln(context.stdout, strings.Join(context.args[1:], " "))
	return 0, err
}

type cmdHuahuo struct{}

func (cmdHuahuo) execute(context commandContext, ctx *sessionContext) (uint32, error) {
	_, err := fmt.Fprintln(context.stdout, "哟，小灰毛，玩的开心吗？玩的开心就好。")
	return 0, err
}

type cmdWhoami struct{}

func (cmdWhoami) execute(context commandContext, ctx *sessionContext) (uint32, error) {
	_, err := fmt.Fprintln(context.stdout, "花斯卡，火斯卡，小~花~火！")
	return 0, err
}

type cmdUname struct{}

func (cmdUname) execute(context commandContext, ctx *sessionContext) (uint32, error) {
	_, err := fmt.Fprintln(context.stdout, "Linux never-gonna-give-you-up-server 5.4.0-187-generic #207-Ubuntu SMP Mon Jun 10 08:16:10 UTC 2024 x86_64 x86_64 x86_64 GNU/Linux")
	return 0, err
}

type cmdCd struct{}

func (cmdCd) execute(context commandContext, ctx *sessionContext) (uint32, error) {
	if len(context.args[1]) > 1024 {
		ctx.virtualPath = context.args[1][:1024]
	} else {
		ctx.virtualPath = context.args[1]
	}
	return 0, nil
}

type cmdPwd struct{}

func (cmdPwd) execute(context commandContext, ctx *sessionContext) (uint32, error) {
	_, err := fmt.Fprintln(context.stdout, ctx.virtualPath)
	return 0, err
}

type cmdLs struct{}

func (cmdLs) execute(context commandContext, ctx *sessionContext) (uint32, error) {
	param := ""
	if len(context.args) > 1 {
		param = context.args[1]
	}
	if context.args[0] == "ll" {
		param += "l"
	}
	files := fakeFileList(rand.Intn(100))
	if strings.Index(param, "l") > -1 {
		showHidden := strings.Index(param, "a") > -1
		for _, file := range files {
			if file.isHidden && !showHidden {
				continue
			}
			dText := "-"
			if file.IsDir {
				dText = "d"
			}
			_, err := fmt.Fprintf(
				context.stdout,
				"%s%s %1d %8s %8s %8d %s %s\n",
				dText,
				file.Perm,
				1,
				file.Owner,
				file.OwnerGroup,
				file.FileSize,
				file.ModTime.Format("Jan 02 15:04"),
				file.FileName,
			)
			if err != nil {
				return 0, err
			}
		}
		_, err := fmt.Fprintf(context.stdout, "total %d\n", len(files))
		return 0, err
	}
	showHidden := strings.Index(param, "a") > -1
	for _, file := range files {
		if file.isHidden && !showHidden {
			continue
		}
		_, err := fmt.Fprintf(context.stdout, "%s\t", file.FileName)
		if err != nil {
			return 0, err
		}
	}
	_, err := fmt.Fprintln(context.stdout, "")
	return 0, err
}

type cmdNeverGonnaGiveYouUp struct{}

func (cmdNeverGonnaGiveYouUp) execute(context commandContext, ctx *sessionContext) (uint32, error) {
	_, err := fmt.Fprintln(context.stdout, "Never gonna give you up, Never gonna let you down, Never gonna run around and desert you, Never gonna make you cry, Never gonna say goodbye, Never gonna tell a lie and hurt you")
	return 0, err
}

type cmdCat struct{}

func (cmdCat) execute(context commandContext, ctx *sessionContext) (uint32, error) {
	if len(context.args) > 1 {
		for _, file := range context.args[1:] {
			funnyFileName := strings.Replace(file, "/", "_", -1)
			fp, err := os.OpenFile(path.Join(ctx.cfg.WorkDir, "./funny_files/cat/", funnyFileName), os.O_RDONLY, 0644)
			if err == nil {
				fileData, err := io.ReadAll(fp)
				if err == nil {
					_, err := fmt.Fprintln(context.stdout, string(fileData))
					return 0, err
				}
			}
			if _, err := fmt.Fprintf(context.stderr, "%v: %v: No such file or directory\n", context.args[0], file); err != nil {
				return 0, err
			}
		}
		return 1, nil
	}
	var line string
	var err error
	for err == nil {
		line, err = context.stdin.ReadLine()
		if err == nil {
			_, err = fmt.Fprintln(context.stdout, line)
		}
	}
	return 0, err
}

type cmdSu struct{}

func (cmdSu) execute(context commandContext, ctx *sessionContext) (uint32, error) {
	newContext := context
	newContext.user = "root"
	if len(context.args) > 1 {
		newContext.user = context.args[1]
	}
	newContext.args = shellProgram
	return executeProgram(newContext, ctx)
}
