package scvmm

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/masterzen/winrm"
)

// ResultFunc ... function to be called after asynchronous Command result sent to winrm
type ResultFunc func(string)

//execScript ... Executes the script with params using winrm Client and creating temp file with the file name passed
func execScript(client *winrm.Client, script string, fileName string, arguments string) (string, string) {
	shell, err := client.CreateShell()
	if err != nil {
		log.Printf("[Error] While creating shell %s", err.Error())
		return "", err.Error()
	}
	var cmd *winrm.Command
	cmd, err = shell.Execute("powershell.exe")
	if err != nil {
		log.Printf("[Error] While executing powershell %s", err.Error())
		return "", err.Error()
	}

	script = strings.Replace(script, "\n", "`r`n", -1)
	script = strings.Replace(script, "\"", "`\"", -1)
	script = strings.Replace(script, "$", "`$", -1)

	var output, errorOutput string

	outputFunc := func(input string) {
		log.Println("[INFO] Output received \n" + input)
		output = input
	}
	errorFunc := func(input string) {
		log.Println("[INFO] Error Output received \n" + input)
		errorOutput = input
	}

	var wg sync.WaitGroup
	wg.Add(8)
	currentTime := time.Now().UTC()
	randomNumber := strconv.Itoa(rand.Intn(1000))
	file := randomNumber + "_" + fileName + "_" + currentTime.Format("20060102150405") + ".ps1"
	log.Println("[INFO] Shell script: " + file)
	log.Println("[INFO] Arguments: " + arguments)
	execCommand(cmd, "echo \""+script+"\" > C:\\Temp\\"+file+"\n", nil, nil, &wg)
	execCommand(cmd, "C:\\Temp\\"+file+" "+arguments+" \n", outputFunc, errorFunc, &wg)
	execCommand(cmd, "rm C:\\Temp\\"+file+"\n", nil, nil, &wg)
	execCommand(cmd, "exit\n", nil, nil, &wg)
	log.Println("[INFO] Waiting for commands to be executed.")
	cmd.Wait()
	wg.Wait()
	shell.Close()
	log.Println("[INFO] winrm Shell closed")
	return output, errorOutput
}

// execCommand ... The function to execute the command and recieve the output in the functions passed
func execCommand(cmd *winrm.Command, command string, outputFun, errorFun ResultFunc, wg *sync.WaitGroup) {
	stdin := bytes.NewBufferString(command)
	log.Printf("[INFO] Executing command %s", command)
	io.Copy(cmd.Stdin, stdin)
	go func() {
		defer (*wg).Done()
		if outputFun != nil {
			output, _ := ioutil.ReadAll(cmd.Stdout)
			for !strings.Contains(string(output), "exit") {
				output, _ = ioutil.ReadAll(cmd.Stdout)
			}
			outputFun(string(output))
		}
	}()
	go func() {
		defer (*wg).Done()
		if errorFun != nil {
			output, _ := ioutil.ReadAll(cmd.Stderr)
			errorFun(string(output))
		}
	}()
}
