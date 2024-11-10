package main

import (
	"github.com/google/generative-ai-go/genai"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var fileWriteSchema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"fileName": {
			Type:        genai.TypeString,
			Description: "The name of the file to write to. Do not include extension, it will be automatically added (.txt)",
		},
		"content": {
			Type:        genai.TypeString,
			Description: "The text content to write to the file",
		},
	},
	Required: []string{"fileName", "content"},
}

var fileReadSchema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"fileName": {
			Type:        genai.TypeString,
			Description: "The name of the file to read (including extension)",
		},
	},
	Required: []string{"fileName"},
}

var memoryWriteSchema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"title": {
			Type:        genai.TypeString,
			Description: "The title of the memory value, for example username.",
		},
		"description": {
			Type:        genai.TypeString,
			Description: "The description of the memory value, for example Name of the User.",
		},
		"value": {
			Type:        genai.TypeString,
			Description: "The value of the memory value, for example Thomas.",
		},
	},
	Required: []string{"title", "description", "value"},
}

var FileTool = &genai.Tool{
	FunctionDeclarations: []*genai.FunctionDeclaration{
		{
			Name:        "file_write", //TODO add modes, write, append
			Description: "write a text file to user local file system with specified name and content.",
			Parameters:  fileWriteSchema,
		},
		{
			Name:        "file_read",
			Description: "read a file from user local file system with specified name.",
			Parameters:  fileReadSchema,
		},
		{
			Name:        "file_list",
			Description: "get list of user file names, separated by comma",
		},
		{
			Name:        "memory_read",
			Description: "returns the long term memory database with all values",
		},
		{
			Name:        "memory_write",
			Description: "write a value to the long term memory database to remember it forever",
			Parameters:  memoryWriteSchema,
		},
	},
}

func WriteDesktop(fileName string, content string) error {
	// TODO if the file is txt, convert markdown to txt
	fileName = fileName + ".txt"
	home, _ := os.UserHomeDir()
	fullPath := filepath.Join(home, "Desktop", fileName)

	formattedContent := strings.ReplaceAll(content, "\\n", "\n")

	err := os.WriteFile(fullPath, []byte(formattedContent), 0644)
	if err != nil {
		return err
	}
	return nil
}

func ReadDesktopFile(filename string) ([]byte, error) {

	path, err := getDesktopdir()
	if err != nil {
		return nil, errors.New("failed to get desktop directory")
	}

	fullPath := filepath.Join(path, filename)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, errors.New("failed to read file")
	}

	return content, nil
}

// OutDesktopFiles returns a list of files on the desktop
func OutDesktopFiles() ([]string, error) {
	path, err := getDesktopdir()
	if err != nil {
		return nil, err
	}
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var fileNames []string
	for _, file := range files {
		if strings.HasPrefix(file.Name(), ".") {
			continue
		}
		if !file.IsDir() {
			fileNames = append(fileNames, file.Name())
		}
	}
	return fileNames, nil
}

// getDesktopdir returns the path to the desktop directory
func getDesktopdir() (string, error) {
	var desktopDir string
	if runtime.GOOS == "windows" {
		// For Windows, use the Local AppData directory
		appData, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		desktopDir = filepath.Join(appData, "Desktop")
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		desktopDir = filepath.Join(homeDir, "Desktop")
	}
	return desktopDir, nil
}

func WriteMemory(title string, description string, value string) error {
	var (
		db  *gorm.DB
		err error
	)
	db, err = GetDB()
	if err != nil {
		return err
	}
	err = InsertData(db, title, description, value)
	if err != nil {
		return err
	}
	return nil
}

func ReadMemory() (string, error) {
	var (
		db  *gorm.DB
		err error
	)
	db, err = GetDB()
	if err != nil {
		return "", err
	}
	data, err := ReadDataAsJSON(db)
	if err != nil {
		return "", err
	}
	return data, nil
}
