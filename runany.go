package main

import "os"
import "mime"
import "os/exec"
import "path/filepath"
import "fmt"
import "strings"
import "sort"
import "flag"

func cmd(name string, arg ...string) (string, error) {
	c := exec.Command(name, arg...)
	bytes, err := c.Output()
	if err != nil {
		return "", err
	}
	out := string(bytes)
	return strings.TrimSpace(out), nil
}

func getMime(filePath string) string {
	mimeStr, err := cmd("file", "--brief", "--mime", filePath)
	if err != nil {
		return ""
	}
	mimeType, _, _ := mime.ParseMediaType(mimeStr)
	return mimeType
}

func ReadDirRecursive(path string) ([]string, error) {
	result := []string{}
	err := filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		result = append(result, path)
		return err
	})
	return result, err
}

type Game struct {
	id   string
	path string
}

type WeightedAction struct {
	name   string
	score  int
	path   string
	action Action
}

type Action func(string) error

func unzipFile(path string) error {
	_, err := cmd("unzip", "-d", filepath.Dir(path), path)
	cmd("rm", path)
	return err
}

func unzipFileAndRun(path string) error {
	err := unzipFile(path)
	if err != nil {
		return err
	}
	return runDirectory(filepath.Dir(path))
}

func unrarFile(path string) error {
	cmd := exec.Command("unrar", "x", path)
	cmd.Dir = filepath.Dir(path)
	return cmd.Run()
}

func unrarFileAndRun(path string) error {
	err := unrarFile(path)
	if err != nil {
		return err
	}
	return runDirectory(filepath.Dir(path))
}

func runJar(path string) error {
	cmd := exec.Command("java", "-jar", path)
	cmd.Dir = filepath.Dir(path)
	return cmd.Run()
}

func runWine(path string) error {
	cmd := exec.Command("wine", path)
	cmd.Dir = filepath.Dir(path)
	return cmd.Run()
}

func runLove(path string) error {
	cmd := exec.Command("love", path)
	cmd.Dir = filepath.Dir(path)
	return cmd.Run()
}

func runExecutable(path string) error {
	os.Chmod(path, 0755)
	cmd := exec.Command(path)
	cmd.Dir = filepath.Dir(path)
	return cmd.Run()
}

type ByScore []WeightedAction

func (a ByScore) Len() int           { return len(a) }
func (a ByScore) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByScore) Less(i, j int) bool { return a[i].score > a[j].score }

func doNothing(path string) error {
	return nil
}

func getWeightedAction(path string) WeightedAction {
	ext := strings.ToLower(filepath.Ext(path))
	depth := strings.Count(path, "/") // TODO: filepath.Separator to string?
	switch {
	case ext == ".zip":
		return WeightedAction{
			name:   "unzip",
			score:  10 - depth,
			path:   path,
			action: unzipFileAndRun,
		}
	case ext == ".rar":
		return WeightedAction{
			name:   "unrar",
			score:  10 - depth,
			path:   path,
			action: unrarFileAndRun,
		}
	case ext == ".jar":
		return WeightedAction{
			name:   "java",
			score:  30 - depth,
			path:   path,
			action: runJar,
		}
	case ext == ".exe":
		return WeightedAction{
			name:   "windows",
			score:  20 - depth,
			path:   path,
			action: runWine,
		}
	case ext == ".love":
		return WeightedAction{
			name:   "love",
			score:  30 - depth,
			path:   path,
			action: runLove,
		}
	case ext == ".sh":
		return WeightedAction{
			name:   "shell",
			score:  30 - depth,
			path:   path,
			action: runLove,
		}
	case (ext == "" || ext == ".x86" || ext == ".x86_64" || ext == ".bin") && getMime(path) == "application/x-executable":
		return WeightedAction{
			name:   "native",
			score:  40 - depth,
			path:   path,
			action: runExecutable,
		}
	}

	return WeightedAction{
		name:   "nothing",
		score:  0 - depth,
		path:   path,
		action: doNothing,
	}
}

func getWeightedActions(parentPath string) ([]WeightedAction, error) {
	result := []WeightedAction{}
	childPaths, err := ReadDirRecursive(parentPath)
	if err != nil {
		return result, err
	}
	for _, childPath := range childPaths {
		result = append(result, getWeightedAction(childPath))
	}
	return result, nil
}

func runDirectory(path string) error {
	actions, err := getWeightedActions(path)
	if err != nil {
		return err
	}
	sort.Sort(ByScore(actions))
	for _, action := range actions {
		fmt.Printf("%d: %s: %s\n", action.score, action.name, action.path)
		err := action.action(action.path)
		if err == nil {
			fmt.Printf("Error: %s\n", errorToString(err))
			break
		}
	}
	return nil
}

func errorToString(err error) string {
	if err != nil {
		return err.Error()
	} else {
		return "success"
	}
}

func main() {
	flag.Parse()
	packagePath, _ := filepath.Abs(flag.Args()[0])
	err := runDirectory(packagePath)
	if err != nil {
		fmt.Printf("Failed to run package: %s\n", errorToString(err))
		os.Exit(1)
	}
}
