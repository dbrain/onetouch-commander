package main

import (
  "github.com/dbrain/soggy"
  "encoding/json"
  "os"
  "os/exec"
  "io/ioutil"
  "io"
)

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
    Exec []struct {
      Cmd string `json:"cmd,omitempty"`
      Args []string `json:"args,omitempty"`
    } `json:"exec,omitempty"`
  }
}
var config *Config

func passwordAuthenticate(ctx *soggy.Context) (int, string) {
  if config.Password != "" {
    headers := ctx.Req.Header
    if (headers.Get("Authorization") != config.Password) {
      return 400, "Invalid password"
    }
  }
  ctx.Next(nil)
  return -1, ""
}

func listCommands() interface{} {
  return config.Commands
}

func executeCommand(ctx *soggy.Context, commandName string) (err error) {
  for _, command := range config.Commands {
    if command.ShortName == commandName {
      for _, cmdToExecute := range command.Exec {
        cmd := exec.Command(cmdToExecute.Cmd, cmdToExecute.Args...)

        stdout, err := cmd.StdoutPipe()
        if (err != nil) { return err }

        stderr, err := cmd.StderrPipe()
        if (err != nil) { return err }

        err = cmd.Start()
        if (err != nil) { return err }

        go io.Copy(ctx.Res, stdout)
        go io.Copy(ctx.Res, stderr)
        return cmd.Wait()
      }
    }
  }
  ctx.Res.WriteHeader(404)
  ctx.Res.WriteString("Command not found.")
  return nil
}

func startServer() {
  app, server := soggy.NewDefaultApp()

  server.Get("/commands", passwordAuthenticate, listCommands)
  server.Post("/commands/(.*)", passwordAuthenticate, executeCommand)

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
