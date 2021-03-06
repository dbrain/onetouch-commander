package main

import (
  "github.com/dbrain/soggy"
  "encoding/json"
  "os"
  "os/exec"
  "io/ioutil"
  "io"
  "errors"
  "fmt"
)

type CommandLine struct {
  Cmd string `json:"cmd,omitempty"`
  Args []string `json:"args,omitempty"`
}

type Config struct {
  Name string
  Description string
  Host string
  Port string
  Password string
  Commands []struct {
    ShortName string `json:"shortName,omitempty"`
    Title string `json:"title,omitempty"`
    Description string `json:"description,omitempty"`
    FailOnError bool `json:"failOnError,omitempty"`
    Exec []CommandLine `json:"exec,omitempty"`
  }
}
var config *Config

func passwordAuthenticate(ctx *soggy.Context) (int, string) {
  if config.Password != "" {
    headers := ctx.Req.Header
    if (headers.Get("Authorization") != config.Password) {
      return 403, "Invalid password"
    }
  }
  ctx.Next(nil)
  return -1, ""
}

func info() interface{} {
  return map[string]interface{} {
    "passwordRequired": config.Password != "" }
}

func listCommands() interface{} {
  return map[string]interface{} { "commands": config.Commands }
}

func executeCommandLine(ctx *soggy.Context, cmdToExecute CommandLine) (err error) {
  defer func() {
    if recovered := recover(); recovered != nil { err = errors.New(recovered.(string)) }
  }()
  cmd := exec.Command(cmdToExecute.Cmd, cmdToExecute.Args...)
  stdout, err := cmd.StdoutPipe()
  stderr, err := cmd.StderrPipe()
  err = cmd.Start()
  go io.Copy(ctx.Res, stdout)
  go io.Copy(ctx.Res, stderr)
  err = cmd.Wait()
  return err
}

func executeCommand(ctx *soggy.Context, commandName string) (err error) {
  for _, command := range config.Commands {
    if command.ShortName == commandName {
      for _, cmdToExecute := range command.Exec {
        err = executeCommandLine(ctx, cmdToExecute)
        if (err != nil && command.FailOnError) {
          return err
        } else if (err != nil) {
          ctx.Res.WriteString(fmt.Sprintln(cmdToExecute.Cmd, cmdToExecute.Args, "failed with", err))
        }
      }
      return nil
    }
  }
  ctx.Res.WriteHeader(404)
  ctx.Res.WriteString("Command not found.")
  return nil
}

func startServer() {
  app, server := soggy.NewDefaultApp()

  server.Get("/commands", passwordAuthenticate, listCommands)
  server.Get("/commands/(.*)", passwordAuthenticate, executeCommand)
  server.Get("/info", info)

  server.Use(server.Router)
  app.Listen(config.Host + ":" + config.Port)
}

func main() {
  home := os.Getenv("HOME")
  configFile, err := ioutil.ReadFile(home + "/.onetouch/config.json")
  if (err != nil) { panic(err) }
  config = new(Config)
  json.Unmarshal(configFile, config)

  startServer()
}
