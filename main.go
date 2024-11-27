package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/generative-ai-go/genai"
	"github.com/googleapis/gax-go/v2/apierror"
	"gorm.io/gorm"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const VERSION = "2.0.7"
const DEVFLAG = false                     // will also save the captured image to a file
const GenaiModel = "gemini-1.5-flash-002" // model to use

type App struct {
	client             *genai.Client
	model              *genai.GenerativeModel
	cs                 *genai.ChatSession
	captureImageChoice bool
	apiKey             string
	sysprompt          string
	fileUri            string
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("The Eye")
	log.Println("The Eye started")

	var err error
	var gormdb *gorm.DB

	aiapp := &App{}

	gormdb, err = InitDB()
	if err != nil {
		log.Println("Error initializing database:", err)
		return
	}
	db = gormdb

	input := widget.NewEntry()
	input.SetPlaceHolder("Enter your message here...")
	input.Wrapping = fyne.TextWrapWord

	apiKey := loadAPIKey()
	aiapp.apiKey = apiKey
	aiapp.client, err = NewClient(aiapp.apiKey, context.Background())
	if err != nil {
		dialog.ShowError(err, myWindow)
		log.Println("Error creating client:", err)
		return
	}
	defer closeClient(aiapp.client)

	aiapp.model = NewModel(aiapp.client, GenaiModel)
	aiapp.model.Tools = []*genai.Tool{FileTool}
	aiapp.sysprompt = getSysPrompt()
	aiapp.model.SystemInstruction = &genai.Content{Role: "user", Parts: []genai.Part{genai.Text(aiapp.sysprompt)}}
	var maxTokens int32 = 1000
	var temperature float32 = 0.9
	aiapp.model.MaxOutputTokens = &maxTokens
	aiapp.model.Temperature = &temperature
	aiapp.cs = aiapp.model.StartChat()

	messagesContainer := container.NewVBox()
	scrollContent := container.NewVScroll(messagesContainer)

	sendButton := widget.NewButtonWithIcon("Send", theme.MailSendIcon(), func() {
		sendMessage(aiapp, input, messagesContainer, aiapp.cs, myWindow, scrollContent)
	})

	input.OnSubmitted = func(text string) {
		sendMessage(aiapp, input, messagesContainer, aiapp.cs, myWindow, scrollContent)
	}

	var fileMsg string
	var filePickerButton *widget.Button
	var checkbox *widget.Check

	clearButton := widget.NewButtonWithIcon("", theme.ContentClearIcon(), func() {
		messagesContainer.Objects = nil
		aiapp.cs.History = nil
		messagesContainer.Refresh()
		aiapp.fileUri = ""
		filePickerButton.SetText("")
		checkbox.Enable()
		checkbox.Checked = false
		aiapp.captureImageChoice = false
	})

	filePickerButton = widget.NewButtonWithIcon(fileMsg, theme.FileIcon(), func() {
		newWindow := myApp.NewWindow("File Picker")
		newWindow.Resize(fyne.NewSize(600, 400))

		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, newWindow)
				return
			}
			if reader == nil {
				newWindow.Close()
				return
			}
			aiapp.fileUri = reader.URI().Path()
			reader.Close()
			fileName := filepath.Base(aiapp.fileUri)
			if len(fileName) > 5 {
				fileName = fileName[0:8] + ".."
			}
			filePickerButton.SetText(fileName)
			checkbox.Checked = false
			checkbox.Disable()
			aiapp.captureImageChoice = false
			newWindow.Close()
		}, newWindow)

		newWindow.Show()
	})

	settingsButton := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		showSettingsDialog(aiapp, myWindow, func(apiKey string) {
			newclient, err := NewClient(apiKey, context.Background())
			if err != nil {
				dialog.ShowError(fmt.Errorf("there was an error. Try again"), myWindow)
				return
			}

			// rebuild the client
			aiapp.client = newclient
			aiapp.model = NewModel(aiapp.client, GenaiModel)
			aiapp.model.Tools = []*genai.Tool{FileTool}
			aiapp.sysprompt = getSysPrompt()
			aiapp.model.SystemInstruction = &genai.Content{Role: "user", Parts: []genai.Part{genai.Text(aiapp.sysprompt)}}
			var maxTokens int32 = 1000
			var temperature float32 = 0.9
			aiapp.model.MaxOutputTokens = &maxTokens
			aiapp.model.Temperature = &temperature
			aiapp.cs = aiapp.model.StartChat()
		})
	})

	checkbox = widget.NewCheck("Send screen data", func(checked bool) {
		aiapp.captureImageChoice = checked
	})

	checkbox.Checked = false

	topContainer := container.NewBorder(nil, nil, nil, container.NewHBox(filePickerButton, settingsButton, clearButton))

	inputContainer := container.NewVBox(
		checkbox,
		container.NewBorder(nil, nil, nil, sendButton, input),
	)

	mainContainer := container.New(layout.NewBorderLayout(topContainer, inputContainer, nil, nil),
		scrollContent, inputContainer, topContainer)
	myWindow.SetContent(mainContainer)

	myWindow.Resize(fyne.NewSize(350, 500))
	myWindow.ShowAndRun()

}

func sendMessage(app *App, input *widget.Entry, messagesContainer *fyne.Container, cs *genai.ChatSession, myWindow fyne.Window, scrollContent *container.Scroll) {
	prompt := input.Text
	if prompt == "" {
		return
	}
	addMessage(messagesContainer, "You", prompt, scrollContent)
	input.SetText("")

	go func() {
		var res *genai.GenerateContentResponse
		var imageBytes []byte
		var err error
		var reserr error

		if app.fileUri != "" {
			fileContent, err := os.ReadFile(app.fileUri)
			if err != nil {
				dialog.ShowError(err, myWindow)
				app.fileUri = ""
				return
			}
			fileType, err := getFileMimeType(app.fileUri)
			if err != nil {
				dialog.ShowError(err, myWindow)
				app.fileUri = ""
				return
			}
			fileBlob := genai.Blob{
				MIMEType: fileType,
				Data:     fileContent, //TODO problems with txt file. maybe read the file and send raw text
			}

			res, reserr = cs.SendMessage(context.Background(), genai.Text(prompt), fileBlob)
			app.fileUri = ""
		} else if app.captureImageChoice {
			myWindow.Hide()
			imageBytes, err = captureScreen()
			myWindow.Show()
			if err != nil {
				log.Println("Error capturing screen:", err)
				return
			}
			res, reserr = cs.SendMessage(context.Background(), genai.Text(prompt), genai.ImageData("png", imageBytes))
		} else {
			res, reserr = cs.SendMessage(context.Background(), genai.Text(prompt))
		}

		if reserr != nil {
			var blockedErr *genai.BlockedError
			var apiErr *apierror.APIError

			if errors.As(reserr, &blockedErr) {
				addMessage(messagesContainer, "AI", "Error: Message blocked", scrollContent)
				return
			} else if errors.As(reserr, &apiErr) {
				addMessage(messagesContainer, "AI", "Error: API error. Could not process request. Make sure your API key is set correctly.", scrollContent)
				return
			} else {
				errortext := "Error: " + reserr.Error()
				addMessage(messagesContainer, "AI", errortext, scrollContent)
				println("Error sending message:", reserr.Error())
				return
			}

		}

		if res == nil {
			addMessage(messagesContainer, "AI", "Error: empty response", scrollContent)
			return
		}

		addMessage(messagesContainer, "AI", buildResponse(res, cs), scrollContent)

	}()

}

// buildResponse builds a string response based on content parts from candidates
func buildResponse(resp *genai.GenerateContentResponse, cs *genai.ChatSession) string {
	var response genai.Text
	funcResponse := make(map[string]interface{})
	var err error

	for _, part := range resp.Candidates[0].Content.Parts {
		functionCall, ok := part.(genai.FunctionCall)
		if ok {
			log.Println("Function call:", functionCall.Name)
			switch functionCall.Name {
			case "file_write":
				fileName, fileNameOk := functionCall.Args["fileName"].(string)
				content, contentOk := functionCall.Args["content"].(string)

				if !fileNameOk || fileName == "" {
					funcResponse["error"] = "expected non-empty string at key 'fileName'"
					break
				}
				if !contentOk || content == "" {
					funcResponse["error"] = "expected non-empty string at key 'content'"
					break
				}

				err := WriteDesktop(fileName, content)
				if err != nil {
					funcResponse["error"] = "error writing file: " + err.Error()
				} else {
					funcResponse["result"] = "file written to user Desktop."
				}

			case "file_read":
				fileName, fileNameOk := functionCall.Args["fileName"].(string)
				if !fileNameOk || fileName == "" {
					funcResponse["error"] = "expected non-empty string at key 'fileName'"
					break
				}
				fileContent, err := ReadDesktopFile(fileName)
				if err != nil {
					funcResponse["error"] = err.Error()
				} else {
					funcResponse["result"] = string(fileContent)
				}
			case "file_list":
				files, err := OutDesktopFiles()
				if err != nil {
					funcResponse["error"] = err.Error()
				} else {
					funcResponse["result"] = strings.Join(files, ", ")
				}
			case "memory_read":
				data, err := ReadMemory()
				if err != nil {
					funcResponse["error"] = err.Error()
				} else {
					funcResponse["result"] = data
				}
			case "memory_write":
				key, keyOk := functionCall.Args["title"].(string)
				value, valueOk := functionCall.Args["value"].(string)
				description, descOk := functionCall.Args["description"].(string)

				if !keyOk || key == "" {
					funcResponse["error"] = "expected non-empty string at key 'title'"
					break
				}
				if !valueOk || value == "" {
					funcResponse["error"] = "expected non-empty string at key 'value'"
					break
				}
				if !descOk || description == "" {
					funcResponse["error"] = "expected non-empty string at key 'description'"
					break
				}

				err := WriteMemory(key, value, description)
				if err != nil {
					funcResponse["error"] = err.Error()
				} else {
					funcResponse["result"] = "value written to memory"
				}
			default:
				funcResponse["error"] = "unknown function call"
			}
		}
	}

	if len(funcResponse) > 0 {
		resp, err = cs.SendMessage(context.Background(), genai.FunctionResponse{
			Name:     "Function_Call",
			Response: funcResponse,
		})
		if err != nil {
			return "Error sending message: " + err.Error()
		}
		funcResponse = nil
		return buildResponse(resp, cs)
	}

	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				res, ok := part.(genai.Text)
				if ok {
					log.Println("Response:", res)
					return string(res)
				}

			}

		}
	}
	return string(response)
}

func addMessage(container *fyne.Container, sender, content string, scrollContent *container.Scroll) {
	label := widget.NewRichTextFromMarkdown(content)
	label.Wrapping = fyne.TextWrapWord

	card := widget.NewCard(sender, "", label)
	container.Add(card)
	if sender == "You" {
		scrollContent.ScrollToBottom()
	}
}

func showSettingsDialog(app *App, window fyne.Window, onSave func(apiKey string)) {
	apiKeyEntry := widget.NewEntry()
	apiKeyEntry.SetText(app.apiKey)

	var rowsCount int64

	rowsCount, _ = CountRows(db)
	itemsStoredLabel := widget.NewLabel("Items stored in memory: " + fmt.Sprint(rowsCount))
	content := container.NewVBox(
		widget.NewLabel("API Key:"),
		apiKeyEntry,
		itemsStoredLabel,
		widget.NewButton("Clear memory", func() {
			err := DeleteData(db)
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			rowsCount, _ = CountRows(db)
			itemsStoredLabel.SetText(fmt.Sprintf("Items stored in memory: %d", rowsCount))
			dialog.ShowInformation("Memory Cleared", "Memory has been cleared", window)
		}),
		widget.NewLabel("Version: "+VERSION),
		widget.NewLabel("pilsnerbeer/the_eye_chatbot"),
	)

	dialog.ShowCustomConfirm("Settings", "Save", "Cancel", content, func(save bool) {
		if save {
			newAPIKey := apiKeyEntry.Text
			if newAPIKey != app.apiKey {
				app.apiKey = newAPIKey
				err := saveAPIKey(app.apiKey)
				if err != nil {
					log.Println("Error saving API key:", err)
					return
				}
				onSave(app.apiKey)
			}
		}
	}, window)
}

func getAppSupportDir() (string, error) {
	var appSupportDir string

	appData, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		appSupportDir = filepath.Join(appData, "AppData", "Local", "TheEye")
	} else {
		appSupportDir = filepath.Join(appData, "Library", "Application Support", "TheEye")
	}
	if err := os.MkdirAll(appSupportDir, 0700); err != nil {
		return "", err
	}
	return appSupportDir, nil
}

func loadAPIKey() string {
	key, err := GetApiKey(db)
	if err != nil {
		log.Println("Error getting API key from db:", err)
		return "apikey"
	}
	return key

}

func saveAPIKey(apiKey string) error {
	err := SaveApiKey(db, apiKey)
	if err != nil {
		log.Println("Error saving API key to db:", err)
		return err
	}
	return nil
}

func getSysPrompt() string {
	log.Println("Getting system prompt")
	basePrompt := "You are an EXTREMELY helpful assistant called The Eye who is an expert in every field and has vast knowledge about various topics. You help the user with their tasks and answer their questions. Be friendly and helpful. Utilize tools when necessary. You have access to long-term memory tool, which helps you remember things across time. write and read from it whenever necessary, when you feel that certain information might need to be remembered for later (Such as personal user information, reminders, specific instructions, etc.)."
	memoryPrompt := "Your long-term memory values are as follows: (in format: Title: Description - Value)\n"
	memVals, err := DumpRows(db)
	var mainPrompt string
	if err != nil {
		log.Println("Error dumping rows:", err)
		return basePrompt
	} else {
		mainPrompt = basePrompt + memoryPrompt + memVals
		return mainPrompt
	}

}

func getFileMimeType(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return "", err
	}

	mimeType := http.DetectContentType(buffer)
	if mimeType == "application/octet-stream" {
		ext := filepath.Ext(filePath)
		mimeType = mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}
	return mimeType, nil
}
